// Package osm loads real shops near the user from OpenStreetMap (Overpass API).
// These are not ShopBot partners yet: the map shows them as grey pins with an
// "Invite this shop" button.
package osm

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"shopbot/internal/store"
)

const (
	radius     = 2000 // meters around the rounded point
	cacheTTL   = 24 * 60 * 60
	liveWait   = 5 * time.Second
	retryAfter = 2 * time.Minute
)

// Overpass is a free public service and its servers are often overloaded,
// so we ask several mirrors at the same time and take the first answer.
var endpoints = []string{
	"https://overpass-api.de/api/interpreter",
	"https://overpass.private.coffee/api/interpreter",
	"https://overpass.kumi.systems/api/interpreter",
	"https://maps.mail.ru/osm/tools/overpass/api/interpreter",
}

// astanaSnapshot is a saved Overpass answer for central Astana (real OSM data).
// It is the last fallback when every mirror is down, so the demo map always has real shops.
//
//go:embed astana_snapshot.json
var astanaSnapshot []byte

type Place struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Address  string  `json:"address"`
}

// Source tells where the answer came from: live, cache or snapshot.
type Source string

type Client struct {
	Store    *store.Store
	HTTP     *http.Client
	snapshot []Place

	mu       sync.Mutex
	failedAt map[string]time.Time // key → when Overpass last failed for it
}

func New(st *store.Store) *Client {
	c := &Client{Store: st, HTTP: &http.Client{Timeout: 12 * time.Second}, failedAt: map[string]time.Time{}}
	var err error
	if c.snapshot, err = parse(strings.NewReader(string(astanaSnapshot))); err != nil {
		log.Printf("osm: bad snapshot: %v", err)
	}
	return c
}

// categoryOf maps OSM tags to ShopBot categories.
var categoryOf = map[string]string{
	"bakery": "bakery", "pastry": "bakery", "confectionery": "bakery",
	"clothes": "clothes", "shoes": "clothes", "boutique": "clothes", "fashion": "clothes",
	"mall": "mall", "department_store": "mall",
	"supermarket": "grocery", "convenience": "grocery", "greengrocer": "grocery", "butcher": "grocery",
	"electronics": "electronics", "mobile_phone": "electronics", "computer": "electronics",
	"florist": "flowers",
	"chemist": "pharmacy", "pharmacy": "pharmacy",
}

// Nearby returns real shops around (lat, lng). Points are rounded to about 1 km,
// so users near each other share one cached answer and Overpass is not overloaded.
// Order of tries: fresh cache → live Overpass → old cache → built-in snapshot.
func (c *Client) Nearby(ctx context.Context, lat, lng float64) ([]Place, Source, error) {
	lat, lng = math.Round(lat*100)/100, math.Round(lng*100)/100
	key := fmt.Sprintf("nearby:%.2f,%.2f", lat, lng)
	if body, ok := c.Store.GetCache(key, cacheTTL); ok {
		places, err := decode(body)
		return places, "cache", err
	}
	// After a recent failure, answer from the fallbacks at once instead of waiting again.
	c.mu.Lock()
	recentFail := time.Since(c.failedAt[key]) < retryAfter
	c.mu.Unlock()
	if recentFail {
		return c.fallback(key, lat, lng, fmt.Errorf("overpass failed recently"))
	}

	// The live request keeps running even if the user leaves the map, and
	// fills the cache when it finishes. The user waits at most liveWait.
	type result struct {
		places []Place
		err    error
	}
	done := make(chan result, 1)
	go func() {
		places, err := c.fetch(context.WithoutCancel(ctx), lat, lng)
		if err == nil {
			body, _ := json.Marshal(places)
			_ = c.Store.PutCache(key, string(body))
		} else {
			log.Printf("osm: all Overpass mirrors failed: %v", err)
			c.mu.Lock()
			c.failedAt[key] = time.Now()
			c.mu.Unlock()
		}
		done <- result{places, err}
	}()
	var err error
	select {
	case r := <-done:
		if r.err == nil {
			return r.places, "live", nil
		}
		err = r.err
	case <-time.After(liveWait):
		err = fmt.Errorf("overpass is slower than %s", liveWait)
	case <-ctx.Done():
		return nil, "", ctx.Err()
	}
	return c.fallback(key, lat, lng, err)
}

// fallback answers without Overpass: an old cached answer, else the built-in snapshot.
func (c *Client) fallback(key string, lat, lng float64, err error) ([]Place, Source, error) {
	if body, ok := c.Store.GetCache(key, 30*cacheTTL); ok {
		places, err := decode(body)
		return places, "cache", err
	}
	if near := c.near(lat, lng); len(near) > 0 {
		return near, "snapshot", nil
	}
	return nil, "", err
}

// near returns snapshot places within the search radius, closest first.
func (c *Client) near(lat, lng float64) []Place {
	var out []Place
	for _, p := range c.snapshot {
		if DistanceM(lat, lng, p.Lat, p.Lng) <= radius {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return DistanceM(lat, lng, out[i].Lat, out[i].Lng) < DistanceM(lat, lng, out[j].Lat, out[j].Lng)
	})
	if len(out) > 250 {
		out = out[:250]
	}
	return out
}

func decode(body string) ([]Place, error) {
	var out []Place
	err := json.Unmarshal([]byte(body), &out)
	return out, err
}

func (c *Client) fetch(ctx context.Context, lat, lng float64) ([]Place, error) {
	shopTypes := []string{}
	for k := range categoryOf {
		if k != "pharmacy" {
			shopTypes = append(shopTypes, k)
		}
	}
	sort.Strings(shopTypes)
	query := fmt.Sprintf(`[out:json][timeout:10];
(
  nwr["shop"~"^(%s)$"]["name"](around:%d,%f,%f);
  nwr["amenity"="pharmacy"]["name"](around:%d,%f,%f);
);
out center tags 250;`, strings.Join(shopTypes, "|"), radius, lat, lng, radius, lat, lng)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // stops the slower mirrors once one has answered
	type result struct {
		places []Place
		err    error
	}
	results := make(chan result, len(endpoints))
	for _, e := range endpoints {
		go func() {
			places, err := c.fetchFrom(ctx, e, query)
			results <- result{places, err}
		}()
	}
	var lastErr error
	for range endpoints {
		r := <-results
		if r.err == nil {
			return r.places, nil
		}
		lastErr = r.err
	}
	return nil, lastErr
}

func (c *Client) fetchFrom(ctx context.Context, endpoint, query string) ([]Place, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(url.Values{"data": {query}}.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Overpass answers 406 to requests without a User-Agent.
	req.Header.Set("User-Agent", "ShopBot-MVP/0.1 (student project, Astana)")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("%s: %s", endpoint, resp.Status)
	}
	return parse(resp.Body)
}

// parse turns an Overpass JSON answer into places with ShopBot categories.
func parse(r io.Reader) ([]Place, error) {
	var data struct {
		Elements []struct {
			Type   string                      `json:"type"`
			ID     int64                       `json:"id"`
			Lat    float64                     `json:"lat"`
			Lon    float64                     `json:"lon"`
			Center *struct{ Lat, Lon float64 } `json:"center"`
			Tags   map[string]string           `json:"tags"`
		} `json:"elements"`
	}
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return nil, err
	}
	places := []Place{}
	for _, e := range data.Elements {
		cat := categoryOf[e.Tags["shop"]]
		if e.Tags["amenity"] == "pharmacy" {
			cat = "pharmacy"
		}
		if cat == "" || e.Tags["name"] == "" {
			continue
		}
		p := Place{ID: fmt.Sprintf("%s/%d", e.Type, e.ID), Name: e.Tags["name"], Category: cat, Lat: e.Lat, Lng: e.Lon}
		if e.Center != nil {
			p.Lat, p.Lng = e.Center.Lat, e.Center.Lon
		}
		p.Address = strings.TrimSpace(e.Tags["addr:street"] + " " + e.Tags["addr:housenumber"])
		places = append(places, p)
	}
	return places, nil
}

// DistanceM is the distance between two points in meters (haversine formula).
func DistanceM(lat1, lng1, lat2, lng2 float64) float64 {
	const earth = 6371000.0
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earth * math.Asin(math.Sqrt(a))
}
