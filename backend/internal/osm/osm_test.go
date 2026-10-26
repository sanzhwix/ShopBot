package osm

import "testing"

func TestSnapshotFallback(t *testing.T) {
	c := New(nil)
	if len(c.snapshot) < 500 {
		t.Fatalf("snapshot has %d places, want the saved Astana data", len(c.snapshot))
	}
	near := c.near(51.13, 71.43) // around Baiterek
	if len(near) == 0 {
		t.Fatal("no snapshot places near the Astana demo area")
	}
	cats := map[string]bool{}
	for _, p := range near {
		if p.Name == "" || p.Category == "" {
			t.Fatalf("place without name or category: %+v", p)
		}
		if d := DistanceM(51.13, 71.43, p.Lat, p.Lng); d > radius {
			t.Fatalf("%s is %.0f m away, outside the radius", p.Name, d)
		}
		cats[p.Category] = true
	}
	for _, want := range []string{"grocery", "clothes", "pharmacy"} {
		if !cats[want] {
			t.Errorf("no %s shops near the center", want)
		}
	}
}
