package api

import (
	"errors"
	"net/http"
	"strings"

	"shopbot/internal/store"
)

type cartGroup struct {
	Shop    store.Shop       `json:"shop"`
	Items   []store.CartItem `json:"items"`
	Total   int              `json:"total"`
	Session shopSession      `json:"session"`
}

func (s *Server) handleCart(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	l := lang(r)
	items, err := s.Store.ListCart(u.ID, l)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	groups := []cartGroup{}
	index := map[int64]int{}
	count := 0
	for _, it := range items {
		i, ok := index[it.ShopID]
		if !ok {
			shop, err := s.Store.GetShop(it.ShopID, l)
			if err != nil {
				writeStoreError(w, err)
				return
			}
			i = len(groups)
			index[it.ShopID] = i
			groups = append(groups, cartGroup{Shop: shop, Items: []store.CartItem{}, Session: s.shopSessionInfo(u.ID, shop)})
		}
		groups[i].Items = append(groups[i].Items, it)
		if it.InStock {
			groups[i].Total += it.Qty * it.Price
		}
		count += it.Qty
	}
	writeJSON(w, 200, map[string]any{"groups": groups, "count": count})
}

// handleAddToCart needs a session at the product's shop. Without one it answers
// 401 SHOP_AUTH_REQUIRED, and the Mini App signs in (or uses the saved account) and retries.
func (s *Server) handleAddToCart(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ProductID int64  `json:"product_id"`
		Size      string `json:"size"`
		Color     string `json:"color"`
		Qty       int    `json:"qty"`
		Source    string `json:"source"`
	}
	if err := readJSON(r, &in); err != nil || in.Qty < 1 || in.Qty > 20 {
		writeError(w, 400, "qty must be 1–20")
		return
	}
	u := userFrom(r)
	l := lang(r)
	p, err := s.Store.GetProduct(in.ProductID, l)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if !p.InStock {
		writeJSON(w, 409, apiError{Error: "out of stock", Code: "OUT_OF_STOCK"})
		return
	}
	shop, err := s.Store.GetShop(p.ShopID, l)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if _, err := s.Store.ShopSession(u.ID, shop.ID); err != nil {
		info := s.shopSessionInfo(u.ID, shop)
		writeJSON(w, 401, map[string]any{"error": "sign in to " + shop.Name, "code": "SHOP_AUTH_REQUIRED", "session": info})
		return
	}
	if in.Source != "map" && in.Source != "search" {
		in.Source = "direct"
	}
	if err := s.Store.AddToCart(u.ID, p.ID, in.Size, in.Color, in.Qty, in.Source); err != nil {
		writeStoreError(w, err)
		return
	}
	_ = s.Store.AddEvent(u.ID, store.Event{Type: "add_to_cart", ShopID: shop.ID, ProductID: p.ID, Source: in.Source})
	count, _ := s.Store.CartCount(u.ID)
	writeJSON(w, 200, map[string]int{"cart_count": count})
}

func (s *Server) handleCartQty(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Qty int `json:"qty"`
	}
	if err := readJSON(r, &in); err != nil || in.Qty < 0 || in.Qty > 20 {
		writeError(w, 400, "qty must be 0–20")
		return
	}
	if err := s.Store.SetCartQty(userFrom(r).ID, idParam(r, "id"), in.Qty); err != nil {
		writeStoreError(w, err)
		return
	}
	s.handleCart(w, r)
}

func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ShopID int64 `json:"shop_id"`
		store.CheckoutInput
	}
	if err := readJSON(r, &in); err != nil {
		writeError(w, 400, "bad JSON")
		return
	}
	in.Address = strings.TrimSpace(in.Address)
	if in.DeliveryType == "delivery" && in.Address == "" {
		writeJSON(w, 400, apiError{Error: "address is required for delivery", Code: "ADDRESS_REQUIRED"})
		return
	}
	u := userFrom(r)
	shop, err := s.Store.GetShop(in.ShopID, lang(r))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	acc, err := s.Store.ShopSession(u.ID, shop.ID)
	if err != nil {
		writeJSON(w, 401, map[string]any{"error": "sign in to " + shop.Name, "code": "SHOP_AUTH_REQUIRED", "session": s.shopSessionInfo(u.ID, shop)})
		return
	}
	order, err := s.Orders.Checkout(u.ID, shop.ID, acc.ID, in.CheckoutInput)
	if errors.Is(err, store.ErrEmptyCart) {
		writeJSON(w, 400, apiError{Error: err.Error(), Code: "EMPTY_CART"})
		return
	}
	if err != nil {
		writeStoreError(w, err)
		return
	}
	redirect := s.Payment.Start(order, shop)
	if redirect.External {
		s.Orders.AwaitingPayment(r.Context(), order)
	}
	writeJSON(w, 201, map[string]any{"order": order, "payment": redirect})
}

func (s *Server) handleOrders(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListUserOrders(userFrom(r).ID, lang(r))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"orders": list})
}

// userOrder loads an order that belongs to the current user.
func (s *Server) userOrder(w http.ResponseWriter, r *http.Request) (store.OrderDetail, bool) {
	order, err := s.Store.GetOrder(idParam(r, "id"), lang(r))
	if err == nil && order.UserID != userFrom(r).ID {
		err = store.ErrNotFound
	}
	if err != nil {
		writeStoreError(w, err)
		return order, false
	}
	return order, true
}

func (s *Server) handleOrder(w http.ResponseWriter, r *http.Request) {
	order, ok := s.userOrder(w, r)
	if !ok {
		return
	}
	writeJSON(w, 200, map[string]any{"order": order})
}

func (s *Server) handlePayment(w http.ResponseWriter, r *http.Request) {
	order, ok := s.userOrder(w, r)
	if !ok {
		return
	}
	shop, err := s.Store.GetShop(order.ShopID, lang(r))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{
		"order":       order,
		"shop":        shop,
		"payment":     s.Payment.Start(order, shop),
		"can_confirm": s.Payment.CanConfirm(shop),
	})
}

func (s *Server) handlePaymentConfirm(w http.ResponseWriter, r *http.Request) {
	order, ok := s.userOrder(w, r)
	if !ok {
		return
	}
	shop, err := s.Store.GetShop(order.ShopID, lang(r))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if !s.Payment.CanConfirm(shop) {
		writeJSON(w, 403, apiError{Error: "this shop confirms payments itself", Code: "MANUAL_CONFIRMATION"})
		return
	}
	if order.Status != store.StatusPendingPayment && order.Status != store.StatusPaid {
		writeJSON(w, 409, apiError{Error: "order is already " + order.Status, Code: "BAD_STATUS"})
		return
	}
	if _, err := s.Orders.MarkPaid(r.Context(), order.ID); err != nil {
		writeStoreError(w, err)
		return
	}
	order, _ = s.Store.GetOrder(order.ID, lang(r))
	writeJSON(w, 200, map[string]any{"order": order})
}
