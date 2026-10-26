// Package orders holds the order rules that both the API and the bot use:
// checkout, payment and status changes. Every change notifies Telegram
// (through Notifier) and the live dashboard (through Hub).
package orders

import (
	"context"
	"errors"
	"log"
	"sync"

	"shopbot/internal/store"
)

var ErrBadStatus = errors.New("unknown order status")

// Notifier sends Telegram messages. The bot implements it; tests use a fake.
type Notifier interface {
	OrderPaid(ctx context.Context, order store.OrderDetail)
	// OrderAwaitingPayment: the shopper pays with the shop's own Kaspi link,
	// so the shop must see the order now and confirm the payment by hand.
	OrderAwaitingPayment(ctx context.Context, order store.OrderDetail)
	StatusChanged(ctx context.Context, order store.OrderDetail)
}

type noopNotifier struct{}

func (noopNotifier) OrderPaid(context.Context, store.OrderDetail)            {}
func (noopNotifier) OrderAwaitingPayment(context.Context, store.OrderDetail) {}
func (noopNotifier) StatusChanged(context.Context, store.OrderDetail)        {}

type Service struct {
	Store *store.Store
	Hub   *Hub

	mu       sync.RWMutex
	notifier Notifier
}

func New(st *store.Store) *Service {
	return &Service{Store: st, Hub: NewHub(), notifier: noopNotifier{}}
}

// SetNotifier is called after the bot starts. The bot needs the service and
// the service needs the bot, so one of them is connected later.
func (s *Service) SetNotifier(n Notifier) {
	s.mu.Lock()
	s.notifier = n
	s.mu.Unlock()
}

func (s *Service) notify() Notifier {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.notifier
}

func (s *Service) Checkout(userID, shopID, accountID int64, in store.CheckoutInput) (store.OrderDetail, error) {
	if in.DeliveryType != "delivery" {
		in.DeliveryType = "pickup"
		in.Address = ""
	}
	if in.Phone != "" {
		_ = s.Store.SetUserPhone(userID, in.Phone)
	}
	id, err := s.Store.CreateOrderFromCart(userID, shopID, accountID, in)
	if err != nil {
		return store.OrderDetail{}, err
	}
	_ = s.Store.AddEvent(userID, store.Event{Type: "checkout", ShopID: shopID})
	return s.Store.GetOrder(id, "en")
}

// AwaitingPayment tells the shop about an order that will be paid outside ShopBot.
func (s *Service) AwaitingPayment(ctx context.Context, order store.OrderDetail) {
	go s.notify().OrderAwaitingPayment(context.WithoutCancel(ctx), order)
}

// MarkPaid confirms payment once. A second call returns the order without notifying again.
func (s *Service) MarkPaid(ctx context.Context, orderID int64) (store.OrderDetail, error) {
	changed, err := s.Store.MarkPaid(orderID)
	if err != nil {
		return store.OrderDetail{}, err
	}
	order, err := s.Store.GetOrder(orderID, "en")
	if err != nil {
		return order, err
	}
	if changed {
		_ = s.Store.AddEvent(order.UserID, store.Event{Type: "paid", ShopID: order.ShopID, Source: order.Source})
		s.Hub.Publish(order.ShopID, HubEvent{Type: "order_paid", Order: order})
		go s.notify().OrderPaid(context.WithoutCancel(ctx), order)
		log.Printf("order #%d paid (%d ₸, shop %d)", order.ID, order.Total, order.ShopID)
	}
	return order, nil
}

func (s *Service) SetStatus(ctx context.Context, orderID int64, status string) (store.OrderDetail, error) {
	if !store.ValidStatus[status] {
		return store.OrderDetail{}, ErrBadStatus
	}
	if status == store.StatusPaid {
		return s.MarkPaid(ctx, orderID)
	}
	if err := s.Store.SetOrderStatus(orderID, status); err != nil {
		return store.OrderDetail{}, err
	}
	order, err := s.Store.GetOrder(orderID, "en")
	if err != nil {
		return order, err
	}
	s.Hub.Publish(order.ShopID, HubEvent{Type: "order_updated", Order: order})
	go s.notify().StatusChanged(context.WithoutCancel(ctx), order)
	return order, nil
}
