// Package analytics computes dashboard numbers.
//
// Owner analytics mixes the seeded demo history with real orders and events,
// so the dashboard looks alive. Founder metrics use real tester data only.
package analytics

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"shopbot/internal/seed"
	"shopbot/internal/store"
)

type KPI struct {
	Value float64 `json:"value"`
	Prev  float64 `json:"prev"`
}

type DayPoint struct {
	Day     int64 `json:"day"` // unix time of local midnight
	Revenue int   `json:"revenue"`
	Orders  int   `json:"orders"`
}

type ProductStat struct {
	Name    string `json:"name"`
	Image   string `json:"image"`
	Photo   string `json:"photo"`
	Qty     int    `json:"qty"`
	Revenue int    `json:"revenue"`
}

type NamedValue struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type Owner struct {
	RangeDays int `json:"range_days"`
	KPIs      struct {
		Revenue      KPI `json:"revenue"`
		Orders       KPI `json:"orders"`
		AvgOrder     KPI `json:"avg_order"`
		NewCustomers KPI `json:"new_customers"`
		Conversion   KPI `json:"conversion"` // paid orders / shop views, in %
	} `json:"kpis"`
	Series      []DayPoint     `json:"series"`
	Statuses    []NamedValue   `json:"statuses"`
	TopProducts []ProductStat  `json:"top_products"`
	Categories  []NamedValue   `json:"categories"`
	Heatmap     [7][24]int     `json:"heatmap"` // [weekday Mon..Sun][hour]
	Funnel      []NamedValue   `json:"funnel"`
	Sources     []NamedValue   `json:"sources"`
	Customers   map[string]int `json:"customers"` // new, returning
}

type Service struct {
	DB *sql.DB
}

func midnight(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, seed.Astana)
}

func weekdayMon(t time.Time) int { return (int(t.Weekday()) + 6) % 7 }

// OwnerStats returns the dashboard of one shop for the last rangeDays days (today included),
// compared with the period of the same length before it.
func (s *Service) OwnerStats(shopID int64, rangeDays int, lang string, now time.Time) (Owner, error) {
	now = now.In(seed.Astana)
	start := midnight(now).AddDate(0, 0, -(rangeDays - 1))
	prevStart := start.AddDate(0, 0, -rangeDays)
	out := Owner{RangeDays: rangeDays, Customers: map[string]int{}}

	firstOrder := map[int64]int64{}
	rows, err := s.DB.Query(`SELECT user_id, MIN(created_at) FROM orders WHERE shop_id = ? AND status NOT IN (?, ?) GROUP BY user_id`,
		shopID, store.StatusPendingPayment, store.StatusCancelled)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var uid, at int64
		if err := rows.Scan(&uid, &at); err != nil {
			rows.Close()
			return out, err
		}
		firstOrder[uid] = at
	}
	rows.Close()

	series := map[int64]*DayPoint{}
	for d := start; !d.After(now); d = d.AddDate(0, 0, 1) {
		p := &DayPoint{Day: d.Unix()}
		series[p.Day] = p
		out.Series = append(out.Series, *p)
	}

	statuses := map[string]int{}
	sources := map[string]int{}
	var cur, prev struct{ revenue, orders, newCustomers int }
	seenCustomer := map[int64]bool{}

	rows, err = s.DB.Query(`SELECT user_id, status, total, source, created_at FROM orders
		WHERE shop_id = ? AND created_at >= ? AND status != ?`, shopID, prevStart.Unix(), store.StatusPendingPayment)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var uid, at int64
		var status, source string
		var total int
		if err := rows.Scan(&uid, &status, &total, &source, &at); err != nil {
			rows.Close()
			return out, err
		}
		inCurrent := at >= start.Unix()
		bucket := &prev
		if inCurrent {
			bucket = &cur
			statuses[status]++
		}
		if status == store.StatusCancelled {
			continue
		}
		bucket.revenue += total
		bucket.orders++
		isNew := firstOrder[uid] == at
		if isNew {
			bucket.newCustomers++
		}
		if !inCurrent {
			continue
		}
		t := time.Unix(at, 0).In(seed.Astana)
		if p := series[midnight(t).Unix()]; p != nil {
			p.Revenue += total
			p.Orders++
		}
		out.Heatmap[weekdayMon(t)][t.Hour()]++
		sources[source]++
		if !seenCustomer[uid] {
			seenCustomer[uid] = true
			if firstOrder[uid] >= start.Unix() {
				out.Customers["new"]++
			} else {
				out.Customers["returning"]++
			}
		}
	}
	rows.Close()
	for i := range out.Series {
		out.Series[i] = *series[out.Series[i].Day]
	}

	out.KPIs.Revenue = KPI{float64(cur.revenue), float64(prev.revenue)}
	out.KPIs.Orders = KPI{float64(cur.orders), float64(prev.orders)}
	out.KPIs.AvgOrder = KPI{avg(cur.revenue, cur.orders), avg(prev.revenue, prev.orders)}
	out.KPIs.NewCustomers = KPI{float64(cur.newCustomers), float64(prev.newCustomers)}

	curFunnel, err := s.funnel(shopID, start.Unix(), now.Unix()+1)
	if err != nil {
		return out, err
	}
	prevFunnel, err := s.funnel(shopID, prevStart.Unix(), start.Unix())
	if err != nil {
		return out, err
	}
	out.KPIs.Conversion = KPI{pct(cur.orders, curFunnel["shop_views"]), pct(prev.orders, prevFunnel["shop_views"])}
	out.Funnel = []NamedValue{
		{"pin_taps", curFunnel["pin_taps"]},
		{"shop_views", curFunnel["shop_views"]},
		{"product_views", curFunnel["product_views"]},
		{"add_to_carts", curFunnel["add_to_carts"]},
		{"checkouts", curFunnel["checkouts"]},
		{"paid", cur.orders},
	}

	for _, k := range []string{"paid", "accepted", "ready", "delivered", "cancelled"} {
		out.Statuses = append(out.Statuses, NamedValue{k, statuses[k]})
	}
	for _, k := range []string{"map", "search", "direct"} {
		out.Sources = append(out.Sources, NamedValue{k, sources[k]})
	}

	out.TopProducts, out.Categories, err = s.productStats(shopID, start.Unix(), lang)
	return out, err
}

// funnel adds the seeded daily stats and the real events of the period.
func (s *Service) funnel(shopID, from, to int64) (map[string]int, error) {
	f := map[string]int{}
	var pin, shop, product, cart, checkout int
	err := s.DB.QueryRow(`SELECT COALESCE(SUM(pin_taps),0), COALESCE(SUM(shop_views),0), COALESCE(SUM(product_views),0),
		COALESCE(SUM(add_to_carts),0), COALESCE(SUM(checkouts),0) FROM daily_stats WHERE shop_id = ? AND day >= ? AND day < ?`,
		shopID, from, to).Scan(&pin, &shop, &product, &cart, &checkout)
	if err != nil {
		return f, err
	}
	f["pin_taps"], f["shop_views"], f["product_views"], f["add_to_carts"], f["checkouts"] = pin, shop, product, cart, checkout

	names := map[string]string{"pin_tap": "pin_taps", "shop_view": "shop_views", "product_view": "product_views", "add_to_cart": "add_to_carts", "checkout": "checkouts"}
	rows, err := s.DB.Query(`SELECT type, COUNT(*) FROM events WHERE shop_id = ? AND created_at >= ? AND created_at < ? GROUP BY type`, shopID, from, to)
	if err != nil {
		return f, err
	}
	defer rows.Close()
	for rows.Next() {
		var typ string
		var n int
		if err := rows.Scan(&typ, &n); err != nil {
			return f, err
		}
		if key, ok := names[typ]; ok {
			f[key] += n
		}
	}
	return f, rows.Err()
}

func (s *Service) productStats(shopID, from int64, lang string) ([]ProductStat, []NamedValue, error) {
	l := store.Lang(lang)
	rows, err := s.DB.Query(fmt.Sprintf(`
		SELECT oi.name_%s, COALESCE(p.image, ''), COALESCE(p.photo, ''), COALESCE(pc.name_%s, '—'), oi.qty, oi.price
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		LEFT JOIN products p ON p.id = oi.product_id
		LEFT JOIN product_categories pc ON pc.id = p.category_id
		WHERE o.shop_id = ? AND o.created_at >= ? AND o.status NOT IN (?, ?)`, l, l),
		shopID, from, store.StatusPendingPayment, store.StatusCancelled)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	products := map[string]*ProductStat{}
	categories := map[string]int{}
	for rows.Next() {
		var name, image, photo, category string
		var qty, price int
		if err := rows.Scan(&name, &image, &photo, &category, &qty, &price); err != nil {
			return nil, nil, err
		}
		p := products[name]
		if p == nil {
			p = &ProductStat{Name: name, Image: image, Photo: photo}
			products[name] = p
		}
		p.Qty += qty
		p.Revenue += qty * price
		categories[category] += qty * price
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	top := make([]ProductStat, 0, len(products))
	for _, p := range products {
		top = append(top, *p)
	}
	sort.Slice(top, func(i, j int) bool { return top[i].Revenue > top[j].Revenue })
	if len(top) > 8 {
		top = top[:8]
	}
	cats := make([]NamedValue, 0, len(categories))
	for name, v := range categories {
		cats = append(cats, NamedValue{name, v})
	}
	sort.Slice(cats, func(i, j int) bool { return cats[i].Value > cats[j].Value })
	return top, cats, nil
}

func avg(sum, n int) float64 {
	if n == 0 {
		return 0
	}
	return float64(sum) / float64(n)
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) * 100 / float64(b)
}
