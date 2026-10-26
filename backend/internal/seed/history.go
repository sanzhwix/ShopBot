package seed

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand/v2"
	"time"
)

// Astana uses UTC+5 all year. A fixed zone works even without tzdata in the container.
var Astana = time.FixedZone("Astana", 5*3600)

const historyDays = 180

func midnight(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

type genProduct struct {
	id             int64
	nameEn, nameRu string
	price          int
	sizes, colors  []string
}

type genShop struct {
	id       int64
	category string
	base     float64 // average paid orders per day at the end of the period
	products []genProduct
}

// profile describes how customers of one shop category behave.
type profile struct {
	weekend, friday float64 // demand multipliers
	hours           [24]float64
	maxQty          int
	maxLines        int
	searchShare     float64 // share of orders that came from the shared search
	deliveryShare   float64
}

func hourWeights(open, close int, peaks ...float64) [24]float64 {
	var w [24]float64
	for h := open; h < close; h++ {
		w[h] = 0.25
		for _, p := range peaks {
			w[h] += math.Exp(-math.Pow(float64(h)-p, 2) / 3)
		}
	}
	return w
}

var profiles = map[string]profile{
	"bakery":      {weekend: 1.25, friday: 1.05, hours: hourWeights(7, 22, 8.5, 18), maxQty: 3, maxLines: 3, searchShare: 0.2, deliveryShare: 0.3},
	"clothes":     {weekend: 1.55, friday: 1.15, hours: hourWeights(10, 23, 13, 20), maxQty: 1, maxLines: 2, searchShare: 0.45, deliveryShare: 0.5},
	"grocery":     {weekend: 1.2, friday: 1.1, hours: hourWeights(8, 23, 10, 19.5), maxQty: 3, maxLines: 5, searchShare: 0.15, deliveryShare: 0.6},
	"pharmacy":    {weekend: 0.9, friday: 1.0, hours: hourWeights(8, 23, 11, 20), maxQty: 2, maxLines: 3, searchShare: 0.35, deliveryShare: 0.55},
	"electronics": {weekend: 1.4, friday: 1.1, hours: hourWeights(10, 22, 14, 19), maxQty: 1, maxLines: 2, searchShare: 0.5, deliveryShare: 0.4},
	"flowers":     {weekend: 1.3, friday: 1.45, hours: hourWeights(8, 22, 11, 17.5), maxQty: 1, maxLines: 2, searchShare: 0.3, deliveryShare: 0.7},
}

var firstNames = []string{
	"Aigerim", "Dias", "Madina", "Arman", "Aruzhan", "Nursultan", "Dana", "Yerlan", "Asel", "Timur",
	"Kamila", "Alibek", "Zarina", "Olzhas", "Aliya", "Sanzhar", "Diana", "Ruslan", "Togzhan", "Askar",
	"Elena", "Dmitry", "Anna", "Sergey", "Ainur", "Bekzat", "Moldir", "Azamat", "Saltanat", "Ilyas",
	"Gulnara", "Maksat", "Akbota", "Daulet", "Inkar", "Yerkebulan", "Malika", "Abylay", "Aisha", "Nurlan",
}

var lastInitials = []string{"A.", "B.", "D.", "Ye.", "Zh.", "K.", "M.", "N.", "O.", "S.", "T.", "U."}

var addresses = []string{
	"Kabanbay Batyr Ave 40, apt 112", "Turan Ave 19, apt 57", "Syganak St 25, apt 9", "Kerey-Zhanibek Khans St 12, apt 81",
	"Dostyk St 18, apt 203", "Mangilik El Ave 55, apt 14", "Respublika Ave 7, apt 66", "Abay Ave 63, apt 31",
	"Kenesary St 40, apt 18", "Saryarka Ave 6, apt 140", "Kunaev St 33, apt 47", "Akmesht St 8, apt 92",
}

var comments = []string{
	"Please call before delivery", "Позвоните за 10 минут", "Leave at the door", "Нужен пакет", "Gift wrapping please", "",
}

type historyStats struct{ orders, customers int }

type customer struct {
	id    int64
	phone string
}

func poisson(r *rand.Rand, lambda float64) int {
	l := math.Exp(-lambda)
	k, p := 0, 1.0
	for {
		p *= r.Float64()
		if p <= l {
			return k
		}
		k++
	}
}

func pickWeighted(r *rand.Rand, w []float64) int {
	total := 0.0
	for _, x := range w {
		total += x
	}
	x := r.Float64() * total
	for i, v := range w {
		x -= v
		if x <= 0 {
			return i
		}
	}
	return len(w) - 1
}

func pick[T any](r *rand.Rand, list []T) T { return list[r.IntN(len(list))] }

// generateHistory writes 180 days of demo orders and funnel stats. The random
// generator has a fixed seed, so every fresh database gets the same numbers.
func generateHistory(tx *sql.Tx, shops []genShop, now time.Time) (historyStats, error) {
	r := rand.New(rand.NewPCG(42, 2026))
	now = now.In(Astana)
	first := midnight(now).AddDate(0, 0, -historyDays)

	// Demo customers have negative ids, so they never clash with Telegram ids.
	pool := make([]customer, 4000)
	insUser, err := tx.Prepare(`INSERT INTO users (id, first_name, last_name, username, lang, phone, is_demo, created_at, last_seen_at) VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?)`)
	if err != nil {
		return historyStats{}, err
	}
	defer insUser.Close()
	for i := range pool {
		name := pick(r, firstNames)
		// No @username: a made-up handle could belong to a real Telegram user.
		c := customer{id: -int64(i + 1), phone: fmt.Sprintf("+7 70%d %03d %02d %02d", r.IntN(8), r.IntN(1000), r.IntN(100), r.IntN(100))}
		lang := "ru"
		if r.Float64() < 0.3 {
			lang = "en"
		}
		if _, err := insUser.Exec(c.id, name, pick(r, lastInitials), "", lang, c.phone, first.Unix(), now.Unix()); err != nil {
			return historyStats{}, err
		}
		pool[i] = c
	}

	insOrder, err := tx.Prepare(`
		INSERT INTO orders (shop_id, user_id, account_id, status, total, delivery_type, address, phone, comment, source, payment_method, is_demo, created_at, paid_at, updated_at)
		VALUES (?, ?, NULL, ?, ?, ?, ?, ?, ?, ?, 'kaspi', 1, ?, ?, ?)`)
	if err != nil {
		return historyStats{}, err
	}
	defer insOrder.Close()
	insItem, err := tx.Prepare(`INSERT INTO order_items (order_id, product_id, name_en, name_ru, size, color, qty, price) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return historyStats{}, err
	}
	defer insItem.Close()
	insStats, err := tx.Prepare(`INSERT INTO daily_stats (shop_id, day, pin_taps, shop_views, product_views, add_to_carts, checkouts) VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return historyStats{}, err
	}
	defer insStats.Close()

	// Product popularity follows a Zipf-like curve: a few best-sellers, a long tail.
	popularity := map[int64][]float64{}
	for _, s := range shops {
		w := make([]float64, len(s.products))
		for i, rank := range r.Perm(len(s.products)) {
			w[i] = 1 / math.Pow(float64(rank+1), 0.9)
		}
		popularity[s.id] = w
	}

	stats := historyStats{customers: len(pool)}
	regulars := map[int64][]customer{}
	newcomers := map[int64][]int{} // per shop: pool indexes in random order, used once each
	for _, s := range shops {
		newcomers[s.id] = r.Perm(len(pool))
	}

	for d := 0; d <= historyDays; d++ {
		day := first.AddDate(0, 0, d)
		// The shops grow over the period, faster in the last months (more users find them on the map).
		growth := 0.55 + 0.45*math.Pow(float64(d)/historyDays, 1.3)
		for _, s := range shops {
			p := profiles[s.category]
			demand := s.base * growth * (0.85 + 0.3*r.Float64())
			switch day.Weekday() {
			case time.Saturday, time.Sunday:
				demand *= p.weekend
			case time.Friday:
				demand *= p.friday
			}
			n := poisson(r, demand)
			placed := 0
			for range n {
				h := pickWeighted(r, p.hours[:])
				at := day.Add(time.Duration(h)*time.Hour + time.Duration(r.IntN(3600))*time.Second)
				if at.After(now) {
					continue
				}

				var c customer
				if past := regulars[s.id]; len(past) > 5 && r.Float64() < 0.45 || len(newcomers[s.id]) == 0 {
					c = pick(r, past)
				} else {
					c = pool[newcomers[s.id][0]]
					newcomers[s.id] = newcomers[s.id][1:]
					regulars[s.id] = append(regulars[s.id], c)
				}

				source := "map"
				switch x := r.Float64(); {
				case x < p.searchShare:
					source = "search"
				case x > 0.8:
					source = "direct"
				}
				delivery, address := "pickup", ""
				if r.Float64() < p.deliveryShare {
					delivery, address = "delivery", pick(r, addresses)
				}
				comment := ""
				if r.Float64() < 0.12 {
					comment = pick(r, comments)
				}
				status := historyStatus(r, now.Sub(at))

				lines := 1 + r.IntN(p.maxLines)
				type line struct {
					p           genProduct
					qty         int
					size, color string
				}
				var items []line
				total := 0
				used := map[int64]bool{}
				for range lines {
					pr := s.products[pickWeighted(r, popularity[s.id])]
					if used[pr.id] {
						continue
					}
					used[pr.id] = true
					l := line{p: pr, qty: 1 + r.IntN(p.maxQty)}
					if len(pr.sizes) > 0 {
						l.size = pick(r, pr.sizes)
					}
					if len(pr.colors) > 0 {
						l.color = pick(r, pr.colors)
					}
					total += l.qty * pr.price
					items = append(items, l)
				}

				paidAt := at.Add(time.Duration(40+r.IntN(140)) * time.Second)
				updated := paidAt.Add(time.Duration(10+r.IntN(180)) * time.Minute)
				if updated.After(now) {
					updated = now
				}
				res, err := insOrder.Exec(s.id, c.id, status, total, delivery, address, c.phone, comment, source, at.Unix(), paidAt.Unix(), updated.Unix())
				if err != nil {
					return stats, err
				}
				orderID, _ := res.LastInsertId()
				for _, l := range items {
					if _, err := insItem.Exec(orderID, l.p.id, l.p.nameEn, l.p.nameRu, l.size, l.color, l.qty, l.p.price); err != nil {
						return stats, err
					}
				}
				placed++
				stats.orders++
			}

			// Funnel numbers (unique visits per step) that lead to the orders placed this day.
			checkouts := ceilDiv(placed, 0.82+0.08*r.Float64())
			carts := ceilDiv(checkouts, 0.45+0.1*r.Float64())
			productViews := ceilDiv(carts, 0.22+0.08*r.Float64())
			shopViews := ceilDiv(productViews, 0.55+0.15*r.Float64())
			pinTaps := ceilDiv(shopViews, 0.6+0.15*r.Float64())
			if _, err := insStats.Exec(s.id, day.Unix(), pinTaps, shopViews, productViews, carts, checkouts); err != nil {
				return stats, err
			}
		}
	}
	return stats, nil
}

// historyStatus: old orders are finished; orders from the last hours are still in progress.
func historyStatus(r *rand.Rand, age time.Duration) string {
	x := r.Float64()
	switch {
	case age > 36*time.Hour:
		if x < 0.05 {
			return "cancelled"
		}
		return "delivered"
	case age > 6*time.Hour:
		switch {
		case x < 0.05:
			return "cancelled"
		case x < 0.2:
			return "ready"
		default:
			return "delivered"
		}
	default:
		switch {
		case x < 0.45:
			return "paid"
		case x < 0.8:
			return "accepted"
		default:
			return "ready"
		}
	}
}

func ceilDiv(n int, ratio float64) int {
	return int(math.Ceil(float64(n) / ratio))
}
