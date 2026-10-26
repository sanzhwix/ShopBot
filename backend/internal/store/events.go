package store

import "database/sql"

// EventTypes are the tracking events the Mini App may send.
var EventTypes = map[string]bool{
	"app_open": true, "map_view": true, "pin_tap": true, "shop_view": true, "product_view": true,
	"search": true, "add_to_cart": true, "checkout": true, "paid": true, "invite": true,
}

type Event struct {
	Type      string `json:"type"`
	ShopID    int64  `json:"shop_id"`
	ProductID int64  `json:"product_id"`
	Source    string `json:"source"`
	Meta      string `json:"meta"`
}

func nullID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

func (s *Store) AddEvent(userID int64, e Event) error {
	_, err := s.DB.Exec(`INSERT INTO events (user_id, type, shop_id, product_id, source, meta, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, e.Type, nullID(e.ShopID), nullID(e.ProductID), e.Source, e.Meta, now())
	return err
}

type Invite struct {
	OsmID    string  `json:"osm_id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

func (s *Store) AddInvite(userID int64, in Invite) error {
	_, err := s.DB.Exec(`INSERT OR IGNORE INTO invites (user_id, osm_id, name, category, lat, lng, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, in.OsmID, in.Name, in.Category, in.Lat, in.Lng, now())
	return err
}

func (s *Store) InviteCount(osmID string) (int, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM invites WHERE osm_id = ?`, osmID).Scan(&n)
	return n, err
}

func (s *Store) AddFeedback(userID int64, rating int, comment string) error {
	_, err := s.DB.Exec(`INSERT INTO feedback (user_id, rating, comment, created_at) VALUES (?, ?, ?, ?)`, userID, rating, comment, now())
	return err
}

// GetCache returns a cached OSM response if it is newer than maxAge seconds.
func (s *Store) GetCache(key string, maxAge int64) (string, bool) {
	var body string
	err := s.DB.QueryRow(`SELECT body FROM osm_cache WHERE key = ? AND fetched_at > ?`, key, now()-maxAge).Scan(&body)
	if err == sql.ErrNoRows || err != nil {
		return "", false
	}
	return body, true
}

func (s *Store) PutCache(key, body string) error {
	_, err := s.DB.Exec(`INSERT OR REPLACE INTO osm_cache (key, body, fetched_at) VALUES (?, ?, ?)`, key, body, now())
	return err
}
