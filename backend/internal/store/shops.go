package store

import (
	"database/sql"
	"fmt"
	"slices"
	"strings"
)

type Mall struct {
	ID        int64   `json:"id"`
	Slug      string  `json:"slug"`
	Name      string  `json:"name"`
	Address   string  `json:"address"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	ShopCount int     `json:"shop_count"`
}

type Shop struct {
	ID           int64   `json:"id"`
	Slug         string  `json:"slug"`
	Name         string  `json:"name"`
	Category     string  `json:"category"`
	MallID       *int64  `json:"mall_id"`
	MallName     string  `json:"mall_name"`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	Address      string  `json:"address"`
	Description  string  `json:"description"`
	Hours        string  `json:"hours"`
	Rating       float64 `json:"rating"`
	Logo         string  `json:"logo"`
	KaspiPayURL  string  `json:"kaspi_pay_url"`
	NotifyChatID *int64  `json:"-"`
}

type Category struct {
	ID     int64  `json:"id"`
	NameEn string `json:"name_en"`
	NameRu string `json:"name_ru"`
	Name   string `json:"name"`
	Sort   int    `json:"sort"`
}

type Product struct {
	ID          int64    `json:"id"`
	ShopID      int64    `json:"shop_id"`
	CategoryID  int64    `json:"category_id"`
	Name        string   `json:"name"`
	NameEn      string   `json:"name_en"`
	NameRu      string   `json:"name_ru"`
	Description string   `json:"description"`
	Price       int      `json:"price"`
	Image       string   `json:"image"` // emoji, used when there is no photo
	Photo       string   `json:"photo"`
	PhotoCredit string   `json:"photo_credit"`
	Sizes       []string `json:"sizes"`
	Colors      []string `json:"colors"`
	Material    string   `json:"material"`
	InStock     bool     `json:"in_stock"`
}

func (s *Store) ListMalls(lang string) ([]Mall, error) {
	rows, err := s.DB.Query(fmt.Sprintf(`
		SELECT m.id, m.slug, m.%s, m.%s, m.lat, m.lng,
		       (SELECT COUNT(*) FROM shops WHERE mall_id = m.id)
		FROM malls m ORDER BY m.id`, loc(lang, "name"), loc(lang, "address")))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Mall{}
	for rows.Next() {
		var m Mall
		if err := rows.Scan(&m.ID, &m.Slug, &m.Name, &m.Address, &m.Lat, &m.Lng, &m.ShopCount); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) GetMall(id int64, lang string) (Mall, error) {
	var m Mall
	err := s.DB.QueryRow(fmt.Sprintf(`
		SELECT m.id, m.slug, m.%s, m.%s, m.lat, m.lng,
		       (SELECT COUNT(*) FROM shops WHERE mall_id = m.id)
		FROM malls m WHERE m.id = ?`, loc(lang, "name"), loc(lang, "address")), id).
		Scan(&m.ID, &m.Slug, &m.Name, &m.Address, &m.Lat, &m.Lng, &m.ShopCount)
	return m, notFound(err)
}

func (s *Store) shopSelect(lang string) string {
	return fmt.Sprintf(`
		SELECT s.id, s.slug, s.name, s.category, s.mall_id, COALESCE(m.%s, ''), s.lat, s.lng,
		       s.%s, s.%s, s.hours, s.rating, s.logo, s.kaspi_pay_url, s.notify_chat_id
		FROM shops s LEFT JOIN malls m ON m.id = s.mall_id`,
		loc(lang, "name"), loc(lang, "address"), loc(lang, "description"))
}

func scanShop(row interface{ Scan(...any) error }) (Shop, error) {
	var sh Shop
	var mallID, chatID sql.NullInt64
	err := row.Scan(&sh.ID, &sh.Slug, &sh.Name, &sh.Category, &mallID, &sh.MallName, &sh.Lat, &sh.Lng,
		&sh.Address, &sh.Description, &sh.Hours, &sh.Rating, &sh.Logo, &sh.KaspiPayURL, &chatID)
	if mallID.Valid {
		sh.MallID = &mallID.Int64
	}
	if chatID.Valid {
		sh.NotifyChatID = &chatID.Int64
	}
	return sh, err
}

// ListShops returns partner shops. category "" means all; mallID 0 means any mall.
func (s *Store) ListShops(lang, category string, mallID int64) ([]Shop, error) {
	q := s.shopSelect(lang) + ` WHERE (? = '' OR s.category = ?) AND (? = 0 OR s.mall_id = ?) ORDER BY s.id`
	rows, err := s.DB.Query(q, category, category, mallID, mallID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Shop{}
	for rows.Next() {
		sh, err := scanShop(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sh)
	}
	return out, rows.Err()
}

func (s *Store) GetShop(id int64, lang string) (Shop, error) {
	sh, err := scanShop(s.DB.QueryRow(s.shopSelect(lang)+` WHERE s.id = ?`, id))
	return sh, notFound(err)
}

func (s *Store) UpdateShopSettings(id int64, hours, kaspiURL string) error {
	_, err := s.DB.Exec(`UPDATE shops SET hours = ?, kaspi_pay_url = ? WHERE id = ?`, hours, kaspiURL, id)
	return err
}

func (s *Store) SetShopNotifyChat(shopID int64, chatID *int64) error {
	_, err := s.DB.Exec(`UPDATE shops SET notify_chat_id = ? WHERE id = ?`, chatID, shopID)
	return err
}

func (s *Store) ShopCategories(shopID int64, lang string) ([]Category, error) {
	rows, err := s.DB.Query(fmt.Sprintf(`
		SELECT id, name_en, name_ru, %s, sort FROM product_categories
		WHERE shop_id = ? ORDER BY sort, id`, loc(lang, "name")), shopID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.NameEn, &c.NameRu, &c.Name, &c.Sort); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) AddCategory(shopID int64, nameEn, nameRu string) (int64, error) {
	res, err := s.DB.Exec(`
		INSERT INTO product_categories (shop_id, name_en, name_ru, sort)
		VALUES (?, ?, ?, (SELECT COALESCE(MAX(sort), 0) + 1 FROM product_categories WHERE shop_id = ?))`,
		shopID, nameEn, nameRu, shopID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) productSelect(lang string) string {
	return fmt.Sprintf(`
		SELECT p.id, p.shop_id, p.category_id, p.%s, p.name_en, p.name_ru, p.%s, p.price, p.image,
		       p.photo, p.photo_credit, p.sizes, p.colors, p.material, p.in_stock
		FROM products p`, loc(lang, "name"), loc(lang, "description"))
}

func scanProduct(row interface{ Scan(...any) error }) (Product, error) {
	var p Product
	var sizes, colors string
	err := row.Scan(&p.ID, &p.ShopID, &p.CategoryID, &p.Name, &p.NameEn, &p.NameRu, &p.Description,
		&p.Price, &p.Image, &p.Photo, &p.PhotoCredit, &sizes, &colors, &p.Material, &p.InStock)
	p.Sizes = decodeList(sizes)
	p.Colors = decodeList(colors)
	return p, err
}

func (s *Store) queryProducts(q string, args ...any) ([]Product, error) {
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) ShopProducts(shopID int64, lang string) ([]Product, error) {
	return s.queryProducts(s.productSelect(lang)+` WHERE p.shop_id = ? ORDER BY p.category_id, p.id`, shopID)
}

func (s *Store) AllProducts(lang string) ([]Product, error) {
	return s.queryProducts(s.productSelect(lang) + ` WHERE p.in_stock = 1 ORDER BY p.id`)
}

func (s *Store) GetProduct(id int64, lang string) (Product, error) {
	p, err := scanProduct(s.DB.QueryRow(s.productSelect(lang)+` WHERE p.id = ?`, id))
	return p, notFound(err)
}

type ProductInput struct {
	CategoryID    int64    `json:"category_id"`
	NameEn        string   `json:"name_en"`
	NameRu        string   `json:"name_ru"`
	DescriptionEn string   `json:"description_en"`
	DescriptionRu string   `json:"description_ru"`
	Price         int      `json:"price"`
	Image         string   `json:"image"`
	Sizes         []string `json:"sizes"`
	Colors        []string `json:"colors"`
	Material      string   `json:"material"`
}

func (s *Store) AddProduct(shopID int64, in ProductInput) (int64, error) {
	res, err := s.DB.Exec(`
		INSERT INTO products (shop_id, category_id, name_en, name_ru, description_en, description_ru,
		                      price, image, sizes, colors, material, in_stock, created_at)
		SELECT ?, id, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ? FROM product_categories WHERE id = ? AND shop_id = ?`,
		shopID, in.NameEn, in.NameRu, in.DescriptionEn, in.DescriptionRu, in.Price, in.Image,
		encodeList(in.Sizes), encodeList(in.Colors), in.Material, now(), in.CategoryID, shopID)
	if err != nil {
		return 0, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return 0, ErrNotFound
	}
	return res.LastInsertId()
}

// SetProductPhoto stores a new photo path. Only products of the given shop can be changed.
func (s *Store) SetProductPhoto(shopID, productID int64, photo, credit string) error {
	res, err := s.DB.Exec(`UPDATE products SET photo = ?, photo_credit = ? WHERE id = ? AND shop_id = ?`, photo, credit, productID, shopID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateProduct changes price and stock. Only products of the given shop can be changed.
func (s *Store) UpdateProduct(shopID, productID int64, price int, inStock bool) error {
	res, err := s.DB.Exec(`UPDATE products SET price = ?, in_stock = ? WHERE id = ? AND shop_id = ?`,
		price, inStock, productID, shopID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

type SearchParams struct {
	Query    string
	Category string
	Size     string
	Material string
	MinPrice int
	MaxPrice int
}

type SearchHit struct {
	Product
	Shop Shop `json:"shop"`
}

// Search filters in Go, not SQL: SQLite LOWER() does not handle Cyrillic,
// and the catalogue is small enough to scan.
func (s *Store) Search(lang string, p SearchParams) ([]SearchHit, error) {
	products, err := s.AllProducts(lang)
	if err != nil {
		return nil, err
	}
	shops, err := s.ListShops(lang, p.Category, 0)
	if err != nil {
		return nil, err
	}
	byID := map[int64]Shop{}
	for _, sh := range shops {
		byID[sh.ID] = sh
	}
	q := strings.ToLower(strings.TrimSpace(p.Query))
	out := []SearchHit{}
	for _, pr := range products {
		sh, ok := byID[pr.ShopID]
		if !ok {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(pr.NameEn), q) && !strings.Contains(strings.ToLower(pr.NameRu), q) {
			continue
		}
		if p.MinPrice > 0 && pr.Price < p.MinPrice || p.MaxPrice > 0 && pr.Price > p.MaxPrice {
			continue
		}
		if p.Material != "" && pr.Material != p.Material {
			continue
		}
		if p.Size != "" && !slices.Contains(pr.Sizes, p.Size) {
			continue
		}
		out = append(out, SearchHit{Product: pr, Shop: sh})
	}
	return out, nil
}
