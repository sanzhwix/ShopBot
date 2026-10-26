package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var ErrEmptyCart = errors.New("cart is empty for this shop")

const (
	StatusPendingPayment = "pending_payment"
	StatusPaid           = "paid"
	StatusAccepted       = "accepted"
	StatusReady          = "ready"
	StatusDelivered      = "delivered"
	StatusCancelled      = "cancelled"
)

// ValidStatus lists the statuses an owner can set.
var ValidStatus = map[string]bool{
	StatusPaid: true, StatusAccepted: true, StatusReady: true, StatusDelivered: true, StatusCancelled: true,
}

type Order struct {
	ID            int64  `json:"id"`
	ShopID        int64  `json:"shop_id"`
	UserID        int64  `json:"user_id"`
	Status        string `json:"status"`
	Total         int    `json:"total"`
	DeliveryType  string `json:"delivery_type"`
	Address       string `json:"address"`
	Phone         string `json:"phone"`
	Comment       string `json:"comment"`
	Source        string `json:"source"`
	PaymentMethod string `json:"payment_method"`
	CreatedAt     int64  `json:"created_at"`
	PaidAt        *int64 `json:"paid_at"`
	UpdatedAt     int64  `json:"updated_at"`
}

type OrderItem struct {
	ProductID int64  `json:"product_id"`
	Name      string `json:"name"`
	Image     string `json:"image"`
	Photo     string `json:"photo"`
	Size      string `json:"size"`
	Color     string `json:"color"`
	Qty       int    `json:"qty"`
	Price     int    `json:"price"`
}

type OrderDetail struct {
	Order
	ShopName     string      `json:"shop_name"`
	ShopLogo     string      `json:"shop_logo"`
	ShopCategory string      `json:"shop_category"`
	AccountLogin string      `json:"account_login"`
	Customer     User        `json:"customer"`
	Items        []OrderItem `json:"items"`
}

type CheckoutInput struct {
	DeliveryType string `json:"delivery_type"`
	Address      string `json:"address"`
	Phone        string `json:"phone"`
	Comment      string `json:"comment"`
}

// CreateOrderFromCart moves the user's cart lines for one shop into a new order.
func (s *Store) CreateOrderFromCart(userID, shopID, accountID int64, in CheckoutInput) (int64, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	rows, err := tx.Query(`
		SELECT c.product_id, p.name_en, p.name_ru, c.size, c.color, c.qty, p.price, c.source
		FROM cart_items c JOIN products p ON p.id = c.product_id
		WHERE c.user_id = ? AND p.shop_id = ? AND p.in_stock = 1 ORDER BY c.id`, userID, shopID)
	if err != nil {
		return 0, err
	}
	type line struct {
		productID           int64
		nameEn, nameRu      string
		size, color, source string
		qty, price          int
	}
	var lines []line
	total := 0
	for rows.Next() {
		var l line
		if err := rows.Scan(&l.productID, &l.nameEn, &l.nameRu, &l.size, &l.color, &l.qty, &l.price, &l.source); err != nil {
			rows.Close()
			return 0, err
		}
		total += l.qty * l.price
		lines = append(lines, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(lines) == 0 {
		return 0, ErrEmptyCart
	}

	t := now()
	res, err := tx.Exec(`
		INSERT INTO orders (shop_id, user_id, account_id, status, total, delivery_type, address, phone, comment, source, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		shopID, userID, accountID, StatusPendingPayment, total, in.DeliveryType, in.Address, in.Phone, in.Comment, lines[0].source, t, t)
	if err != nil {
		return 0, err
	}
	orderID, _ := res.LastInsertId()
	for _, l := range lines {
		if _, err := tx.Exec(`INSERT INTO order_items (order_id, product_id, name_en, name_ru, size, color, qty, price) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			orderID, l.productID, l.nameEn, l.nameRu, l.size, l.color, l.qty, l.price); err != nil {
			return 0, err
		}
	}
	if _, err := tx.Exec(`DELETE FROM cart_items WHERE user_id = ? AND product_id IN (SELECT id FROM products WHERE shop_id = ?)`, userID, shopID); err != nil {
		return 0, err
	}
	return orderID, tx.Commit()
}

func (s *Store) loadOrders(lang, where string, args ...any) ([]OrderDetail, error) {
	rows, err := s.DB.Query(`
		SELECT o.id, o.shop_id, o.user_id, o.status, o.total, o.delivery_type, o.address, o.phone, o.comment,
		       o.source, o.payment_method, o.created_at, o.paid_at, o.updated_at,
		       sh.name, sh.logo, sh.category, COALESCE(a.login, ''),
		       u.id, u.first_name, u.last_name, u.username, u.lang, u.phone
		FROM orders o
		JOIN shops sh ON sh.id = o.shop_id
		JOIN users u ON u.id = o.user_id
		LEFT JOIN shop_accounts a ON a.id = o.account_id
		WHERE `+where, args...)
	if err != nil {
		return nil, err
	}
	out := []OrderDetail{}
	index := map[int64]int{}
	for rows.Next() {
		var d OrderDetail
		var paid sql.NullInt64
		if err := rows.Scan(&d.ID, &d.ShopID, &d.UserID, &d.Status, &d.Total, &d.DeliveryType, &d.Address, &d.Phone, &d.Comment,
			&d.Source, &d.PaymentMethod, &d.CreatedAt, &paid, &d.UpdatedAt,
			&d.ShopName, &d.ShopLogo, &d.ShopCategory, &d.AccountLogin,
			&d.Customer.ID, &d.Customer.FirstName, &d.Customer.LastName, &d.Customer.Username, &d.Customer.Lang, &d.Customer.Phone); err != nil {
			rows.Close()
			return nil, err
		}
		if paid.Valid {
			d.PaidAt = &paid.Int64
		}
		d.Items = []OrderItem{}
		index[d.ID] = len(out)
		out = append(out, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return out, nil
	}

	ids := make([]string, 0, len(out))
	for _, d := range out {
		ids = append(ids, fmt.Sprint(d.ID))
	}
	itemRows, err := s.DB.Query(fmt.Sprintf(`
		SELECT oi.order_id, oi.product_id, oi.%s, COALESCE(p.image, ''), COALESCE(p.photo, ''), oi.size, oi.color, oi.qty, oi.price
		FROM order_items oi LEFT JOIN products p ON p.id = oi.product_id
		WHERE oi.order_id IN (%s) ORDER BY oi.id`, loc(lang, "name"), strings.Join(ids, ",")))
	if err != nil {
		return nil, err
	}
	defer itemRows.Close()
	for itemRows.Next() {
		var orderID int64
		var it OrderItem
		if err := itemRows.Scan(&orderID, &it.ProductID, &it.Name, &it.Image, &it.Photo, &it.Size, &it.Color, &it.Qty, &it.Price); err != nil {
			return nil, err
		}
		d := &out[index[orderID]]
		d.Items = append(d.Items, it)
	}
	return out, itemRows.Err()
}

func (s *Store) GetOrder(id int64, lang string) (OrderDetail, error) {
	list, err := s.loadOrders(lang, "o.id = ?", id)
	if err != nil {
		return OrderDetail{}, err
	}
	if len(list) == 0 {
		return OrderDetail{}, ErrNotFound
	}
	return list[0], nil
}

func (s *Store) ListUserOrders(userID int64, lang string) ([]OrderDetail, error) {
	return s.loadOrders(lang, "o.user_id = ? ORDER BY o.id DESC LIMIT 50", userID)
}

// ListShopOrders returns the newest orders of a shop. status "" means all,
// "active" means paid, accepted or ready.
func (s *Store) ListShopOrders(shopID int64, status string, limit int, lang string) ([]OrderDetail, error) {
	switch status {
	case "":
		return s.loadOrders(lang, "o.shop_id = ? AND o.status != ? ORDER BY o.created_at DESC LIMIT ?", shopID, StatusPendingPayment, limit)
	case "active":
		return s.loadOrders(lang, "o.shop_id = ? AND o.status IN (?, ?, ?) ORDER BY o.created_at DESC LIMIT ?",
			shopID, StatusPaid, StatusAccepted, StatusReady, limit)
	default:
		return s.loadOrders(lang, "o.shop_id = ? AND o.status = ? ORDER BY o.created_at DESC LIMIT ?", shopID, status, limit)
	}
}

// MarkPaid moves an order from pending_payment to paid. It reports false
// if the order was already paid, so a double click does nothing.
func (s *Store) MarkPaid(id int64) (bool, error) {
	t := now()
	res, err := s.DB.Exec(`UPDATE orders SET status = ?, paid_at = ?, updated_at = ? WHERE id = ? AND status = ?`,
		StatusPaid, t, t, id, StatusPendingPayment)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *Store) SetOrderStatus(id int64, status string) error {
	_, err := s.DB.Exec(`UPDATE orders SET status = ?, updated_at = ? WHERE id = ?`, status, now(), id)
	return err
}

type OwnerMessage struct {
	ChatID    int64
	MessageID int
}

func (s *Store) SaveOwnerMessage(orderID, chatID int64, messageID int) error {
	_, err := s.DB.Exec(`INSERT OR REPLACE INTO owner_messages (order_id, chat_id, message_id) VALUES (?, ?, ?)`, orderID, chatID, messageID)
	return err
}

func (s *Store) OwnerMessages(orderID int64) ([]OwnerMessage, error) {
	rows, err := s.DB.Query(`SELECT chat_id, message_id FROM owner_messages WHERE order_id = ?`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OwnerMessage
	for rows.Next() {
		var m OwnerMessage
		if err := rows.Scan(&m.ChatID, &m.MessageID); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
