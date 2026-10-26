package analytics

import (
	"sort"
	"strings"
	"time"

	"shopbot/internal/seed"
)

type DayCount struct {
	Day   int64 `json:"day"`
	Value int   `json:"value"`
}

type FeedbackItem struct {
	Rating    int    `json:"rating"`
	Comment   string `json:"comment"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"created_at"`
}

type InviteStat struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// Founder holds real tester data only (no seeded rows). It feeds the HADI
// cycle and the beta-testing part of the report.
type Founder struct {
	Users        int            `json:"users"`
	NewUsers7d   int            `json:"new_users_7d"`
	ActiveToday  int            `json:"active_today"`
	Orders       int            `json:"orders"`
	GMV          int            `json:"gmv"`
	DAU          []DayCount     `json:"dau"`
	Funnel       []NamedValue   `json:"funnel"` // unique users per step
	TopSearches  []NamedValue   `json:"top_searches"`
	Invites      []InviteStat   `json:"invites"`
	FeedbackAvg  float64        `json:"feedback_avg"`
	FeedbackList []FeedbackItem `json:"feedback"`
}

func (s *Service) FounderStats(now time.Time) (Founder, error) {
	var f Founder
	now = now.In(seed.Astana)
	today := midnight(now)
	from := today.AddDate(0, 0, -13)

	err := s.DB.QueryRow(`SELECT COUNT(*), COALESCE(SUM(created_at >= ?), 0) FROM users WHERE is_demo = 0`,
		today.AddDate(0, 0, -6).Unix()).Scan(&f.Users, &f.NewUsers7d)
	if err != nil {
		return f, err
	}
	if err := s.DB.QueryRow(`SELECT COUNT(DISTINCT user_id) FROM events WHERE created_at >= ?`, today.Unix()).Scan(&f.ActiveToday); err != nil {
		return f, err
	}
	if err := s.DB.QueryRow(`SELECT COUNT(*), COALESCE(SUM(total), 0) FROM orders WHERE is_demo = 0 AND status NOT IN ('pending_payment', 'cancelled')`).
		Scan(&f.Orders, &f.GMV); err != nil {
		return f, err
	}

	dau := map[int64]int{}
	rows, err := s.DB.Query(`SELECT created_at, user_id FROM events WHERE created_at >= ?`, from.Unix())
	if err != nil {
		return f, err
	}
	seen := map[[2]int64]bool{}
	for rows.Next() {
		var at, uid int64
		if err := rows.Scan(&at, &uid); err != nil {
			rows.Close()
			return f, err
		}
		day := midnight(time.Unix(at, 0).In(seed.Astana)).Unix()
		if !seen[[2]int64{day, uid}] {
			seen[[2]int64{day, uid}] = true
			dau[day]++
		}
	}
	rows.Close()
	for d := from; !d.After(today); d = d.AddDate(0, 0, 1) {
		f.DAU = append(f.DAU, DayCount{Day: d.Unix(), Value: dau[d.Unix()]})
	}

	steps := []string{"app_open", "map_view", "pin_tap", "shop_view", "product_view", "add_to_cart", "checkout", "paid"}
	counts := map[string]int{}
	rows, err = s.DB.Query(`SELECT type, COUNT(DISTINCT user_id) FROM events GROUP BY type`)
	if err != nil {
		return f, err
	}
	for rows.Next() {
		var typ string
		var n int
		if err := rows.Scan(&typ, &n); err != nil {
			rows.Close()
			return f, err
		}
		counts[typ] = n
	}
	rows.Close()
	for _, st := range steps {
		f.Funnel = append(f.Funnel, NamedValue{st, counts[st]})
	}

	searches := map[string]int{}
	rows, err = s.DB.Query(`SELECT meta FROM events WHERE type = 'search' AND meta != ''`)
	if err != nil {
		return f, err
	}
	for rows.Next() {
		var q string
		if err := rows.Scan(&q); err != nil {
			rows.Close()
			return f, err
		}
		searches[strings.ToLower(strings.TrimSpace(q))]++
	}
	rows.Close()
	f.TopSearches = topN(searches, 10)

	rows, err = s.DB.Query(`SELECT name, category, COUNT(*) AS n FROM invites GROUP BY osm_id ORDER BY n DESC LIMIT 10`)
	if err != nil {
		return f, err
	}
	for rows.Next() {
		var in InviteStat
		if err := rows.Scan(&in.Name, &in.Category, &in.Count); err != nil {
			rows.Close()
			return f, err
		}
		f.Invites = append(f.Invites, in)
	}
	rows.Close()

	if err := s.DB.QueryRow(`SELECT COALESCE(AVG(rating), 0) FROM feedback`).Scan(&f.FeedbackAvg); err != nil {
		return f, err
	}
	rows, err = s.DB.Query(`SELECT f.rating, f.comment, COALESCE(u.first_name, ''), f.created_at
		FROM feedback f LEFT JOIN users u ON u.id = f.user_id ORDER BY f.id DESC LIMIT 30`)
	if err != nil {
		return f, err
	}
	defer rows.Close()
	for rows.Next() {
		var it FeedbackItem
		if err := rows.Scan(&it.Rating, &it.Comment, &it.Name, &it.CreatedAt); err != nil {
			return f, err
		}
		f.FeedbackList = append(f.FeedbackList, it)
	}
	return f, rows.Err()
}

func topN(m map[string]int, n int) []NamedValue {
	out := make([]NamedValue, 0, len(m))
	for k, v := range m {
		out = append(out, NamedValue{k, v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Value > out[j].Value })
	if len(out) > n {
		out = out[:n]
	}
	return out
}
