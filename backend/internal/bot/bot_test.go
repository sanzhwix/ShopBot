package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	tg "github.com/go-telegram/bot"

	"shopbot/internal/config"
	"shopbot/internal/db"
	"shopbot/internal/orders"
	"shopbot/internal/seed"
	"shopbot/internal/store"
)

// fakeTelegram answers like the Bot API and records every call.
type fakeTelegram struct {
	mu          sync.Mutex
	calls       []call
	nextID      int
	failPrivacy bool // refuse tg://user buttons, like Telegram does for hidden profiles
}

type call struct {
	method string
	form   url.Values
}

func (f *fakeTelegram) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	method := path.Base(r.URL.Path)
	form := url.Values{}
	if err := r.ParseMultipartForm(1 << 20); err == nil {
		form = url.Values(r.MultipartForm.Value)
	}
	f.mu.Lock()
	f.calls = append(f.calls, call{method, form})
	f.nextID++
	id := f.nextID
	fail := f.failPrivacy && strings.Contains(form.Get("reply_markup"), "tg://user")
	f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	switch {
	case method == "getMe":
		fmt.Fprint(w, `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"ShopBot","username":"shopbot_test_bot"}}`)
	case fail:
		w.WriteHeader(400)
		fmt.Fprint(w, `{"ok":false,"error_code":400,"description":"Bad Request: BUTTON_USER_PRIVACY_RESTRICTED"}`)
	case method == "sendMessage" || method == "editMessageText":
		fmt.Fprintf(w, `{"ok":true,"result":{"message_id":%d,"date":0,"chat":{"id":%s,"type":"private"},"text":"ok"}}`, id, form.Get("chat_id"))
	default:
		fmt.Fprint(w, `{"ok":true,"result":true}`)
	}
}

func (f *fakeTelegram) find(method string, chatID int64) []call {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []call
	for _, c := range f.calls {
		if c.method == method && c.form.Get("chat_id") == fmt.Sprint(chatID) {
			out = append(out, c)
		}
	}
	return out
}

func setup(t *testing.T) (*Bot, *store.Store, *fakeTelegram) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "bot.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := seed.Run(conn); err != nil {
		t.Fatal(err)
	}
	st := store.New(conn)
	fake := &fakeTelegram{}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	cfg := &config.Config{BotToken: "123:TEST"}
	cfg.SetPublicURL("https://shopbot.example")
	b, err := New(cfg, st, orders.New(st), tg.WithServerURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	return b, st, fake
}

// paidOrder places a paid order for one croissant from Nan & Co (shop 1).
func paidOrder(t *testing.T, st *store.Store, customer store.User) store.OrderDetail {
	t.Helper()
	if _, err := st.UpsertUser(customer); err != nil {
		t.Fatal(err)
	}
	acc, err := st.FindShopAccount(1, seed.DemoShopLogin)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AddToCart(customer.ID, 6, "", "", 2, "map"); err != nil {
		t.Fatal(err)
	}
	id, err := st.CreateOrderFromCart(customer.ID, 1, acc.ID, store.CheckoutInput{DeliveryType: "pickup", Phone: "+7 701 000 00 00"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.MarkPaid(id); err != nil {
		t.Fatal(err)
	}
	order, err := st.GetOrder(id, "en")
	if err != nil {
		t.Fatal(err)
	}
	return order
}

func TestOrderNotifications(t *testing.T) {
	b, st, fake := setup(t)
	ctx := context.Background()
	const ownerChat = 777
	st.UpsertUser(store.User{ID: ownerChat, FirstName: "Owner", Lang: "en"})
	if err := st.LinkOwnerTelegram(1, ownerChat, ownerChat); err != nil {
		t.Fatal(err)
	}

	order := paidOrder(t, st, store.User{ID: 555, FirstName: "Aigerim", Username: "aigerim_k", Lang: "ru"})
	b.OrderPaid(ctx, order)

	cards := fake.find("sendMessage", ownerChat)
	if len(cards) != 1 {
		t.Fatalf("owner got %d messages, want 1", len(cards))
	}
	text, markup := cards[0].form.Get("text"), cards[0].form.Get("reply_markup")
	for _, want := range []string{fmt.Sprintf("New order #%d", order.ID), "Butter croissant", "1 300 ₸", "@aigerim_k"} {
		if !strings.Contains(text, want) {
			t.Errorf("order card has no %q:\n%s", want, text)
		}
	}
	for _, want := range []string{"https://t.me/aigerim_k", fmt.Sprintf("st:%d:accepted", order.ID), fmt.Sprintf("st:%d:cancelled", order.ID)} {
		if !strings.Contains(markup, want) {
			t.Errorf("order buttons have no %q: %s", want, markup)
		}
	}

	receipts := fake.find("sendMessage", 555)
	if len(receipts) != 1 || !strings.Contains(receipts[0].form.Get("text"), "оплачен") {
		t.Fatalf("customer receipt (in Russian) missing: %+v", receipts)
	}
	if !strings.Contains(receipts[0].form.Get("reply_markup"), fmt.Sprintf("https://shopbot.example/orders/%d", order.ID)) {
		t.Errorf("receipt has no track-order button: %s", receipts[0].form.Get("reply_markup"))
	}

	// The owner accepts: the same card is edited and the customer is told.
	st.SetOrderStatus(order.ID, store.StatusAccepted)
	accepted, _ := st.GetOrder(order.ID, "en")
	b.StatusChanged(ctx, accepted)
	edits := fake.find("editMessageText", ownerChat)
	if len(edits) != 1 {
		t.Fatalf("owner card edited %d times, want 1", len(edits))
	}
	if !strings.Contains(edits[0].form.Get("text"), "Accepted") || !strings.Contains(edits[0].form.Get("reply_markup"), fmt.Sprintf("st:%d:ready", order.ID)) {
		t.Errorf("edited card is wrong: %s / %s", edits[0].form.Get("text"), edits[0].form.Get("reply_markup"))
	}
	if got := fake.find("sendMessage", 555); len(got) != 2 || !strings.Contains(got[1].form.Get("text"), "принял") {
		t.Errorf("customer did not get the 'accepted' update: %+v", got)
	}
}

func TestPrivacyFallback(t *testing.T) {
	b, st, fake := setup(t)
	const ownerChat = 888
	st.LinkOwnerTelegram(1, ownerChat, ownerChat)
	fake.failPrivacy = true

	// No username: the button would be tg://user?id=..., which Telegram refuses here.
	order := paidOrder(t, st, store.User{ID: 556, FirstName: "Dias", Lang: "en"})
	b.OrderPaid(context.Background(), order)

	sent := fake.find("sendMessage", ownerChat)
	if len(sent) != 2 {
		t.Fatalf("want a failed try and a retry, got %d calls", len(sent))
	}
	if !strings.Contains(sent[1].form.Get("reply_markup"), fmt.Sprintf("reply:%d", order.ID)) {
		t.Errorf("retry has no 'Reply via ShopBot' button: %s", sent[1].form.Get("reply_markup"))
	}
}

func TestMenuOpensMiniApp(t *testing.T) {
	b, _, _ := setup(t)
	kb, _ := json.Marshal(b.menuKeyboard("en"))
	for _, want := range []string{"https://shopbot.example/map", "https://shopbot.example/map?cat=bakery", "https://shopbot.example/search", `"callback_data":"owners"`} {
		if !strings.Contains(string(kb), want) {
			t.Errorf("menu has no %s", want)
		}
	}
	if b.Username() != "shopbot_test_bot" {
		t.Errorf("username = %q", b.Username())
	}
}
