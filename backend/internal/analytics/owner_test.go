package analytics

import (
	"path/filepath"
	"testing"
	"time"

	"shopbot/internal/db"
	"shopbot/internal/seed"
)

func TestOwnerStatsMatchesOrders(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "a.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := seed.Run(conn); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	s := &Service{DB: conn}
	stats, err := s.OwnerStats(1, 30, "en", now)
	if err != nil {
		t.Fatal(err)
	}

	start := midnight(now.In(seed.Astana)).AddDate(0, 0, -29).Unix()
	var orders, revenue int
	conn.QueryRow(`SELECT COUNT(*), COALESCE(SUM(total), 0) FROM orders
		WHERE shop_id = 1 AND created_at >= ? AND status NOT IN ('pending_payment', 'cancelled')`, start).Scan(&orders, &revenue)

	if int(stats.KPIs.Orders.Value) != orders || int(stats.KPIs.Revenue.Value) != revenue {
		t.Fatalf("KPIs = %v orders / %v ₸, SQL = %d / %d", stats.KPIs.Orders.Value, stats.KPIs.Revenue.Value, orders, revenue)
	}
	if orders == 0 || stats.KPIs.Orders.Prev == 0 {
		t.Fatal("demo history must fill the current and the previous period")
	}
	if len(stats.Series) != 30 {
		t.Fatalf("series has %d days, want 30", len(stats.Series))
	}
	sum := 0
	for _, p := range stats.Series {
		sum += p.Revenue
	}
	if sum != revenue {
		t.Fatalf("series sum %d != revenue %d", sum, revenue)
	}
	for i := 1; i < len(stats.Funnel); i++ {
		if stats.Funnel[i].Value > stats.Funnel[i-1].Value {
			t.Fatalf("funnel grows at %s: %v", stats.Funnel[i].Name, stats.Funnel)
		}
	}
}
