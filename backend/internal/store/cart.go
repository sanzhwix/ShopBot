package store

import "fmt"

type CartItem struct {
	ID        int64  `json:"id"`
	ProductID int64  `json:"product_id"`
	ShopID    int64  `json:"shop_id"`
	Name      string `json:"name"`
	Image     string `json:"image"`
	Photo     string `json:"photo"`
	Size      string `json:"size"`
	Color     string `json:"color"`
	Qty       int    `json:"qty"`
	Price     int    `json:"price"`
	InStock   bool   `json:"in_stock"`
}

func (s *Store) ListCart(userID int64, lang string) ([]CartItem, error) {
	rows, err := s.DB.Query(fmt.Sprintf(`
		SELECT c.id, c.product_id, p.shop_id, p.%s, p.image, p.photo, c.size, c.color, c.qty, p.price, p.in_stock
		FROM cart_items c JOIN products p ON p.id = c.product_id
		WHERE c.user_id = ? ORDER BY p.shop_id, c.id`, loc(lang, "name")), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CartItem{}
	for rows.Next() {
		var c CartItem
		if err := rows.Scan(&c.ID, &c.ProductID, &c.ShopID, &c.Name, &c.Image, &c.Photo, &c.Size, &c.Color, &c.Qty, &c.Price, &c.InStock); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// AddToCart adds qty to the line with the same product, size and color.
func (s *Store) AddToCart(userID, productID int64, size, color string, qty int, source string) error {
	_, err := s.DB.Exec(`
		INSERT INTO cart_items (user_id, product_id, size, color, qty, source, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, product_id, size, color) DO UPDATE SET qty = qty + excluded.qty`,
		userID, productID, size, color, qty, source, now())
	return err
}

// SetCartQty changes a line's quantity. Zero removes the line.
func (s *Store) SetCartQty(userID, itemID int64, qty int) error {
	var err error
	if qty <= 0 {
		_, err = s.DB.Exec(`DELETE FROM cart_items WHERE id = ? AND user_id = ?`, itemID, userID)
	} else {
		_, err = s.DB.Exec(`UPDATE cart_items SET qty = ? WHERE id = ? AND user_id = ?`, qty, itemID, userID)
	}
	return err
}

func (s *Store) CartCount(userID int64) (int, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COALESCE(SUM(qty), 0) FROM cart_items WHERE user_id = ?`, userID).Scan(&n)
	return n, err
}
