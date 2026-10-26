// Package seed fills an empty database with the demo malls, shops, catalogues,
// owner accounts and 180 days of made-up order history (so even the 90-day
// dashboard view has a previous period to compare with).
package seed

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

//go:embed catalog.json
var catalogJSON []byte

// photos.json maps "shop-slug|English product name" to a photo in the Mini App
// (webapp/public/products) and its author and license.
//
//go:embed photos.json
var photosJSON []byte

// DemoPassword is the password of every demo owner and demo shop account.
const DemoPassword = "demo123"

// DemoShopLogin is a customer account that exists in every demo shop.
const DemoShopLogin = "demo@shopbot.kz"

type catalog struct {
	Malls []struct {
		Slug      string  `json:"slug"`
		NameEn    string  `json:"name_en"`
		NameRu    string  `json:"name_ru"`
		AddressEn string  `json:"address_en"`
		AddressRu string  `json:"address_ru"`
		Lat       float64 `json:"lat"`
		Lng       float64 `json:"lng"`
	} `json:"malls"`
	Shops []shopSeed `json:"shops"`
}

type shopSeed struct {
	Slug          string  `json:"slug"`
	Name          string  `json:"name"`
	Category      string  `json:"category"`
	Mall          string  `json:"mall"`
	Lat           float64 `json:"lat"`
	Lng           float64 `json:"lng"`
	AddressEn     string  `json:"address_en"`
	AddressRu     string  `json:"address_ru"`
	DescriptionEn string  `json:"description_en"`
	DescriptionRu string  `json:"description_ru"`
	Hours         string  `json:"hours"`
	Rating        float64 `json:"rating"`
	Logo          string  `json:"logo"`
	BaseOrders    float64 `json:"base_orders"`
	Categories    []struct {
		NameEn   string `json:"name_en"`
		NameRu   string `json:"name_ru"`
		Products []struct {
			En       string   `json:"en"`
			Ru       string   `json:"ru"`
			DescEn   string   `json:"desc_en"`
			DescRu   string   `json:"desc_ru"`
			Price    int      `json:"price"`
			Image    string   `json:"image"`
			Sizes    []string `json:"sizes"`
			Colors   []string `json:"colors"`
			Material string   `json:"material"`
		} `json:"products"`
	} `json:"categories"`
}

var ownerNames = []string{
	"Aigerim Sadykova", "Daniyar Nurlanov", "Madina Bekova", "Arman Seitkali", "Dana Zhaksylykova",
	"Yerlan Abenov", "Asel Tulegenova", "Timur Kassymov", "Saule Omarova", "Nurlan Akhmetov",
	"Zhanna Iskakova", "Ruslan Mukhamedov", "Kamila Utegenova", "Bauyrzhan Serikbayev",
}

// Run seeds the database if it has no shops yet.
func Run(db *sql.DB) error {
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM shops`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		if err := Refresh(db); err != nil {
			return err
		}
		return syncPhotos(db)
	}

	var cat catalog
	if err := json.Unmarshal(catalogJSON, &cat); err != nil {
		return fmt.Errorf("catalog.json: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(DemoPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	start := time.Now()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	t := time.Now().Unix()
	mallIDs := map[string]int64{}
	for _, m := range cat.Malls {
		res, err := tx.Exec(`INSERT INTO malls (slug, name_en, name_ru, address_en, address_ru, lat, lng) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			m.Slug, m.NameEn, m.NameRu, m.AddressEn, m.AddressRu, m.Lat, m.Lng)
		if err != nil {
			return err
		}
		mallIDs[m.Slug], _ = res.LastInsertId()
	}

	var shops []genShop
	for i, s := range cat.Shops {
		var mallID any
		if s.Mall != "" {
			mallID = mallIDs[s.Mall]
		}
		res, err := tx.Exec(`
			INSERT INTO shops (slug, name, category, mall_id, lat, lng, address_en, address_ru, description_en, description_ru, hours, rating, logo, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			s.Slug, s.Name, s.Category, mallID, s.Lat, s.Lng, s.AddressEn, s.AddressRu, s.DescriptionEn, s.DescriptionRu, s.Hours, s.Rating, s.Logo, t)
		if err != nil {
			return err
		}
		shopID, _ := res.LastInsertId()
		gs := genShop{id: shopID, category: s.Category, base: s.BaseOrders}

		if _, err := tx.Exec(`INSERT INTO shop_owners (shop_id, email, password_hash, name) VALUES (?, ?, ?, ?)`,
			shopID, "owner@"+s.Slug+".kz", string(hash), ownerNames[i%len(ownerNames)]); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO shop_accounts (shop_id, login, password_hash, name, created_at) VALUES (?, ?, ?, ?, ?)`,
			shopID, DemoShopLogin, string(hash), "Demo Customer", t); err != nil {
			return err
		}

		for ci, c := range s.Categories {
			res, err := tx.Exec(`INSERT INTO product_categories (shop_id, name_en, name_ru, sort) VALUES (?, ?, ?, ?)`, shopID, c.NameEn, c.NameRu, ci)
			if err != nil {
				return err
			}
			catID, _ := res.LastInsertId()
			for _, p := range c.Products {
				sizes, _ := json.Marshal(orEmpty(p.Sizes))
				colors, _ := json.Marshal(orEmpty(p.Colors))
				res, err := tx.Exec(`
					INSERT INTO products (shop_id, category_id, name_en, name_ru, description_en, description_ru, price, image, sizes, colors, material, created_at)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					shopID, catID, p.En, p.Ru, p.DescEn, p.DescRu, p.Price, p.Image, string(sizes), string(colors), p.Material, t)
				if err != nil {
					return err
				}
				pid, _ := res.LastInsertId()
				gs.products = append(gs.products, genProduct{id: pid, nameEn: p.En, nameRu: p.Ru, price: p.Price, sizes: p.Sizes, colors: p.Colors})
			}
		}
		shops = append(shops, gs)
	}

	stats, err := generateHistory(tx, shops, time.Now())
	if err != nil {
		return fmt.Errorf("history: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	log.Printf("seed: %d malls, %d shops, %d orders, %d demo customers in %s",
		len(cat.Malls), len(cat.Shops), stats.orders, stats.customers, time.Since(start).Round(time.Millisecond))
	return syncPhotos(db)
}

// syncPhotos gives demo products their photos. It only fills products that have
// no photo yet, so a photo uploaded by the owner is never replaced.
func syncPhotos(db *sql.DB) error {
	var photos map[string]struct {
		Photo  string `json:"photo"`
		Credit string `json:"credit"`
	}
	if err := json.Unmarshal(photosJSON, &photos); err != nil {
		return fmt.Errorf("photos.json: %w", err)
	}
	filled := 0
	for key, p := range photos {
		slug, name, _ := strings.Cut(key, "|")
		res, err := db.Exec(`UPDATE products SET photo = ?, photo_credit = ?
			WHERE photo = '' AND name_en = ? AND shop_id = (SELECT id FROM shops WHERE slug = ?)`, p.Photo, p.Credit, name, slug)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		filled += int(n)
	}
	if filled > 0 {
		log.Printf("seed: added photos to %d products", filled)
	}
	return nil
}

func orEmpty(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

// Refresh keeps the demo history current. If the app is started some days
// after seeding, all demo rows move forward by those days, so "last 7 days"
// on the dashboard is never empty. Rows that end up in the future are removed.
func Refresh(db *sql.DB) error {
	var last sql.NullInt64
	if err := db.QueryRow(`SELECT MAX(created_at) FROM orders WHERE is_demo = 1`).Scan(&last); err != nil || !last.Valid {
		return err
	}
	now := time.Now().In(Astana)
	days := int(midnight(now).Sub(midnight(time.Unix(last.Int64, 0).In(Astana))).Hours() / 24)
	if days <= 0 {
		return nil
	}
	shift := int64(days) * 86400
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	steps := []struct {
		q   string
		arg int64
	}{
		{`UPDATE orders SET created_at = created_at + ?1, paid_at = paid_at + ?1, updated_at = updated_at + ?1 WHERE is_demo = 1`, shift},
		{`UPDATE daily_stats SET day = day + ?`, shift},
		{`DELETE FROM order_items WHERE order_id IN (SELECT id FROM orders WHERE is_demo = 1 AND created_at > ?)`, now.Unix()},
		{`DELETE FROM orders WHERE is_demo = 1 AND created_at > ?`, now.Unix()},
		{`DELETE FROM daily_stats WHERE day > ?`, now.Unix()},
	}
	for _, s := range steps {
		if _, err := tx.Exec(s.q, s.arg); err != nil {
			return err
		}
	}
	log.Printf("seed: moved demo history forward by %d days", days)
	return tx.Commit()
}
