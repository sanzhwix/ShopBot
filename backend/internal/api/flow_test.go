package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"shopbot/internal/analytics"
	"shopbot/internal/auth"
	"shopbot/internal/config"
	"shopbot/internal/db"
	"shopbot/internal/orders"
	"shopbot/internal/osm"
	"shopbot/internal/payment"
	"shopbot/internal/seed"
	"shopbot/internal/store"
	"shopbot/internal/vault"
)

const testToken = "123456:TEST-TOKEN"

type fakeNotifier struct {
	mu      sync.Mutex
	paid    []int64
	changed []string
	done    chan struct{}
}

func (f *fakeNotifier) OrderPaid(_ context.Context, o store.OrderDetail) {
	f.mu.Lock()
	f.paid = append(f.paid, o.ID)
	f.mu.Unlock()
	f.done <- struct{}{}
}

func (f *fakeNotifier) OrderAwaitingPayment(context.Context, store.OrderDetail) {}

func (f *fakeNotifier) StatusChanged(_ context.Context, o store.OrderDetail) {
	f.mu.Lock()
	f.changed = append(f.changed, o.Status)
	f.mu.Unlock()
	f.done <- struct{}{}
}

func (f *fakeNotifier) wait(t *testing.T) {
	t.Helper()
	select {
	case <-f.done:
	case <-time.After(2 * time.Second):
		t.Fatal("notifier was not called")
	}
}

func newTestServer(t *testing.T) (*httptest.Server, *fakeNotifier) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := seed.Run(conn); err != nil {
		t.Fatal(err)
	}
	st := store.New(conn)
	svc := orders.New(st)
	notifier := &fakeNotifier{done: make(chan struct{}, 10)}
	svc.SetNotifier(notifier)
	v, _ := vault.New(bytes.Repeat([]byte{1}, 32))
	cfg := &config.Config{BotToken: testToken, PaymentMode: "demo", AdminPassword: "admin", DevAuth: true, UploadDir: t.TempDir()}
	srv := &Server{
		Cfg: cfg, Store: st, Orders: svc, Analytics: &analytics.Service{DB: conn}, Vault: v,
		Payment: payment.New("demo"), OSM: osm.New(st), Sessions: auth.Sessions{Secret: bytes.Repeat([]byte{2}, 32)},
	}
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)
	return ts, notifier
}

type client struct {
	t        *testing.T
	base     string
	initData string
	cookie   string
}

func telegramClient(t *testing.T, base string, userID int64) *client {
	v := url.Values{
		"auth_date": {strconv.FormatInt(time.Now().Unix(), 10)},
		"user":      {fmt.Sprintf(`{"id":%d,"first_name":"Tester","username":"tester%d","language_code":"en"}`, userID, userID)},
	}
	return &client{t: t, base: base, initData: auth.SignInitData(v, testToken)}
}

func (c *client) do(method, path string, body any) (int, map[string]any) {
	c.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, c.base+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if c.initData != "" {
		req.Header.Set("X-Telegram-Init-Data", c.initData)
	}
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	if sc := resp.Header.Get("Set-Cookie"); sc != "" {
		c.cookie = sc[:bytes.IndexByte([]byte(sc), ';')]
	}
	out := map[string]any{}
	json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func TestShoppingFlow(t *testing.T) {
	ts, notifier := newTestServer(t)

	anon := &client{t: t, base: ts.URL}
	if code, body := anon.do("GET", "/api/me", nil); code != 200 || body["user"].(map[string]any)["username"] != "demo_shopper" {
		t.Fatalf("browser without Telegram in dev mode: %d %v", code, body)
	}

	forged := telegramClient(t, ts.URL, 556)
	forged.initData = strings.Replace(forged.initData, "Tester", "Hacker", 1)
	if code, _ := forged.do("GET", "/api/me", nil); code != 401 {
		t.Fatalf("forged initData: got %d, want 401", code)
	}

	c := telegramClient(t, ts.URL, 555)
	const croissant = 6 // "Butter croissant" in Nan & Co (shop 1)

	// First visit: sign in with the demo account and save it.
	if code, body := c.do("POST", "/api/shops/1/login", map[string]any{"login": seed.DemoShopLogin, "password": "wrong"}); code != 401 {
		t.Fatalf("wrong password: %d %v", code, body)
	}
	if code, body := c.do("POST", "/api/shops/1/login", map[string]any{"login": seed.DemoShopLogin, "password": seed.DemoPassword, "save": true}); code != 200 || body["signed_in"] != true {
		t.Fatalf("login: %d %v", code, body)
	}
	c.do("POST", "/api/shops/1/logout", nil)

	// Signed out: add to cart asks for sign-in and tells the app a saved account exists.
	code, body := c.do("POST", "/api/cart", map[string]any{"product_id": croissant, "qty": 2, "source": "map"})
	if code != 401 || body["code"] != "SHOP_AUTH_REQUIRED" || body["session"].(map[string]any)["has_saved_account"] != true {
		t.Fatalf("add without session: %d %v", code, body)
	}
	if code, body := c.do("POST", "/api/shops/1/login/vault", nil); code != 200 || body["login"] != seed.DemoShopLogin {
		t.Fatalf("vault login: %d %v", code, body)
	}
	if code, body := c.do("POST", "/api/cart", map[string]any{"product_id": croissant, "qty": 2, "source": "map"}); code != 200 || body["cart_count"] != 2.0 {
		t.Fatalf("add to cart: %d %v", code, body)
	}

	code, body = c.do("POST", "/api/checkout", map[string]any{"shop_id": 1, "delivery_type": "pickup", "phone": "+7 701 000 00 00"})
	if code != 201 {
		t.Fatalf("checkout: %d %v", code, body)
	}
	order := body["order"].(map[string]any)
	orderID := int64(order["id"].(float64))
	if order["status"] != store.StatusPendingPayment || order["total"] != 1300.0 {
		t.Fatalf("new order: %v", order)
	}
	if pay := body["payment"].(map[string]any); pay["url"] != fmt.Sprintf("/pay/%d", orderID) {
		t.Fatalf("payment redirect: %v", pay)
	}
	if _, cart := c.do("GET", "/api/cart", nil); cart["count"] != 0.0 {
		t.Fatalf("cart must be empty after checkout: %v", cart)
	}

	// Another user cannot see or pay this order.
	other := telegramClient(t, ts.URL, 777)
	if code, _ := other.do("POST", fmt.Sprintf("/api/payments/%d/confirm", orderID), nil); code != 404 {
		t.Fatalf("foreign order confirm: %d", code)
	}

	// Pay twice: the shop is notified once.
	for range 2 {
		if code, body := c.do("POST", fmt.Sprintf("/api/payments/%d/confirm", orderID), nil); code != 200 || body["order"].(map[string]any)["status"] != store.StatusPaid {
			t.Fatalf("confirm: %d %v", code, body)
		}
	}
	notifier.wait(t)
	time.Sleep(50 * time.Millisecond)
	if len(notifier.paid) != 1 || notifier.paid[0] != orderID {
		t.Fatalf("OrderPaid calls: %v", notifier.paid)
	}

	// The owner sees the order and accepts it.
	owner := &client{t: t, base: ts.URL}
	if code, body := owner.do("POST", "/api/owner/login", map[string]any{"email": "owner@nan-and-co.kz", "password": seed.DemoPassword}); code != 200 {
		t.Fatalf("owner login: %d %v", code, body)
	}
	_, list := owner.do("GET", "/api/owner/orders?status=active", nil)
	found := false
	for _, o := range list["orders"].([]any) {
		if int64(o.(map[string]any)["id"].(float64)) == orderID {
			found = true
		}
	}
	if !found {
		t.Fatal("owner does not see the new order")
	}
	if code, body := owner.do("PATCH", fmt.Sprintf("/api/owner/orders/%d", orderID), map[string]any{"status": "accepted"}); code != 200 {
		t.Fatalf("accept: %d %v", code, body)
	}
	notifier.wait(t)
	if len(notifier.changed) != 1 || notifier.changed[0] != store.StatusAccepted {
		t.Fatalf("StatusChanged calls: %v", notifier.changed)
	}

	// An owner of another shop cannot change it.
	stranger := &client{t: t, base: ts.URL}
	stranger.do("POST", "/api/owner/login", map[string]any{"email": "owner@steppe-wear.kz", "password": seed.DemoPassword})
	if code, _ := stranger.do("PATCH", fmt.Sprintf("/api/owner/orders/%d", orderID), map[string]any{"status": "cancelled"}); code != 404 {
		t.Fatalf("foreign owner changed the order: %d", code)
	}
}

// upload sends a file as multipart form field "photo".
func (c *client) upload(path string, data []byte) (int, map[string]any) {
	c.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("photo", "photo.bin")
	fw.Write(data)
	mw.Close()
	req, _ := http.NewRequest("POST", c.base+path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Cookie", c.cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	out := map[string]any{}
	json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func TestOwnerPhotoUpload(t *testing.T) {
	ts, _ := newTestServer(t)
	owner := &client{t: t, base: ts.URL}
	owner.do("POST", "/api/owner/login", map[string]any{"email": "owner@nan-and-co.kz", "password": seed.DemoPassword})

	// A real 1×1 PNG.
	png, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==")
	code, body := owner.upload("/api/owner/products/6/photo", png)
	if code != 200 {
		t.Fatalf("upload: %d %v", code, body)
	}
	photo := body["photo"].(string)
	resp, err := http.Get(ts.URL + photo)
	if err != nil || resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("GET %s: %v %v", photo, resp.Status, resp.Header.Get("Content-Type"))
	}
	if _, p := owner.do("GET", "/api/owner/catalogue", nil); !strings.Contains(fmt.Sprint(p["products"]), photo) {
		t.Fatal("the product does not show the new photo")
	}

	// Text that pretends to be a photo is refused.
	if code, _ := owner.upload("/api/owner/products/6/photo", []byte("<svg onload=alert(1)>")); code != 415 {
		t.Fatalf("svg/text upload: got %d, want 415", code)
	}
	// A product of another shop cannot be changed.
	if code, _ := owner.upload("/api/owner/products/60/photo", png); code != 404 {
		t.Fatalf("foreign product: got %d, want 404", code)
	}
	// Paths cannot leave the upload folder.
	if resp, _ := http.Get(ts.URL + "/uploads/..%2f..%2fgo.mod"); resp.StatusCode != 404 {
		t.Fatalf("path traversal: got %d", resp.StatusCode)
	}
}
