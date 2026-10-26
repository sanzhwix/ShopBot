// Package bot is the Telegram side of ShopBot: the /start menu that opens the
// Mini App, owner linking, order notifications and status buttons.
package bot

import (
	"context"
	"fmt"
	"html"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"shopbot/internal/config"
	"shopbot/internal/orders"
	"shopbot/internal/store"
)

type Bot struct {
	api    *tg.Bot
	cfg    *config.Config
	store  *store.Store
	orders *orders.Service

	username string

	mu        sync.Mutex
	ownerCode map[int64]time.Time // chat → when the correct /owner code was entered
	replyTo   map[int64]int64     // owner chat → order id they are replying about
}

// New connects to Telegram. Extra options are for tests (for example a fake server URL).
func New(cfg *config.Config, st *store.Store, svc *orders.Service, extra ...tg.Option) (*Bot, error) {
	b := &Bot{cfg: cfg, store: st, orders: svc, ownerCode: map[int64]time.Time{}, replyTo: map[int64]int64{}}
	opts := append([]tg.Option{
		tg.WithDefaultHandler(b.onMessage),
		tg.WithCallbackQueryDataHandler("", tg.MatchTypePrefix, b.onCallback),
		tg.WithErrorsHandler(func(err error) { log.Printf("telegram: %v", err) }),
	}, extra...)
	api, err := tg.New(cfg.BotToken, opts...)
	if err != nil {
		return nil, err
	}
	b.api = api
	for cmd, h := range map[string]tg.HandlerFunc{
		"start": b.onStart, "help": b.onHelp, "map": b.onOpen("/map"), "search": b.onOpen("/search"),
		"orders": b.onOpen("/orders"), "owner": b.onOwner, "lang": b.onLang,
	} {
		api.RegisterHandler(tg.HandlerTypeMessageText, cmd, tg.MatchTypeCommandStartOnly, h)
	}
	me, err := api.GetMe(context.Background())
	if err != nil {
		return nil, err
	}
	b.username = me.Username
	return b, nil
}

func (b *Bot) Username() string { return b.username }

// Start polls Telegram for updates until ctx is cancelled.
func (b *Bot) Start(ctx context.Context) {
	log.Printf("bot: @%s started", b.username)
	b.api.Start(ctx)
}

// SetupMenu sets the profile texts, the command lists and the "ShopBot" button
// next to the message field. It runs again whenever the public URL changes.
func (b *Bot) SetupMenu(ctx context.Context) {
	b.setupProfile(ctx)
	if url := b.appURL("/map"); url != "" {
		_, err := b.api.SetChatMenuButton(ctx, &tg.SetChatMenuButtonParams{MenuButton: models.MenuButtonWebApp{
			Type: models.MenuButtonTypeWebApp, Text: "ShopBot", WebApp: models.WebAppInfo{URL: url},
		}})
		if err != nil {
			log.Printf("bot: set menu button: %v", err)
		}
	}
}

// appURL returns the Mini App address for a route, or "" if there is no HTTPS address yet.
// Telegram only opens Mini Apps over HTTPS.
func (b *Bot) appURL(path string) string {
	base := b.cfg.PublicURL()
	if !strings.HasPrefix(base, "https://") {
		return ""
	}
	return base + path
}

func webAppButton(text, url string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{Text: text, WebApp: &models.WebAppInfo{URL: url}}
}

func (b *Bot) userLang(id int64) string {
	if u, err := b.store.GetUser(id); err == nil {
		return u.Lang
	}
	return "en"
}

// rememberUser saves the Telegram profile so later messages use the right language.
func (b *Bot) rememberUser(u *models.User) store.User {
	lang := "en"
	if u.LanguageCode == "ru" || u.LanguageCode == "kk" {
		lang = "ru"
	}
	saved, err := b.store.UpsertUser(store.User{ID: u.ID, FirstName: u.FirstName, LastName: u.LastName, Username: u.Username, Lang: lang})
	if err != nil {
		log.Printf("bot: save user: %v", err)
		return store.User{ID: u.ID, FirstName: u.FirstName, Lang: lang}
	}
	return saved
}

func (b *Bot) send(ctx context.Context, chatID int64, text string, kb *models.InlineKeyboardMarkup) (*models.Message, error) {
	p := &tg.SendMessageParams{ChatID: chatID, Text: text, ParseMode: models.ParseModeHTML,
		LinkPreviewOptions: &models.LinkPreviewOptions{IsDisabled: tg.True()}}
	if kb != nil {
		p.ReplyMarkup = kb
	}
	return b.api.SendMessage(ctx, p)
}

// --- main menu ---

func (b *Bot) menuKeyboard(lang string) *models.InlineKeyboardMarkup {
	row := func(buttons ...models.InlineKeyboardButton) []models.InlineKeyboardButton { return buttons }
	cat := func(key, cat string) models.InlineKeyboardButton {
		return webAppButton(t(lang, key), b.appURL("/map?cat="+cat))
	}
	var rows [][]models.InlineKeyboardButton
	if b.appURL("/") != "" {
		rows = append(rows,
			row(webAppButton(t(lang, "btn_map"), b.appURL("/map"))),
			row(cat("btn_clothes", "clothes"), cat("btn_bakery", "bakery")),
			row(cat("btn_mall", "mall"), cat("btn_grocery", "grocery")),
			row(cat("btn_pharmacy", "pharmacy"), cat("btn_electro", "electronics")),
			row(cat("btn_flowers", "flowers"), webAppButton(t(lang, "btn_search"), b.appURL("/search"))),
			row(webAppButton(t(lang, "btn_cart"), b.appURL("/cart")), webAppButton(t(lang, "btn_orders"), b.appURL("/orders"))),
			row(webAppButton(t(lang, "btn_accounts"), b.appURL("/accounts"))),
		)
	}
	rows = append(rows, row(
		models.InlineKeyboardButton{Text: t(lang, "btn_lang"), CallbackData: "lang"},
		models.InlineKeyboardButton{Text: t(lang, "btn_owners"), CallbackData: "owners"},
	))
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (b *Bot) menuText(u store.User) string {
	text := t(u.Lang, "welcome", html.EscapeString(u.FirstName))
	if b.appURL("/") == "" {
		text += t(u.Lang, "no_url")
	}
	return text
}

func (b *Bot) onStart(ctx context.Context, _ *tg.Bot, upd *models.Update) {
	u := b.rememberUser(upd.Message.From)
	if _, err := b.send(ctx, upd.Message.Chat.ID, b.menuText(u), b.menuKeyboard(u.Lang)); err != nil {
		log.Printf("bot: start: %v", err)
	}
}

func (b *Bot) onHelp(ctx context.Context, _ *tg.Bot, upd *models.Update) {
	u := b.rememberUser(upd.Message.From)
	b.send(ctx, upd.Message.Chat.ID, t(u.Lang, "help"), nil)
}

// onOpen answers /map, /search, /orders with one button that opens that screen.
func (b *Bot) onOpen(path string) tg.HandlerFunc {
	keys := map[string]string{"/map": "btn_map", "/search": "btn_search", "/orders": "btn_orders"}
	return func(ctx context.Context, _ *tg.Bot, upd *models.Update) {
		u := b.rememberUser(upd.Message.From)
		url := b.appURL(path)
		if url == "" {
			b.send(ctx, upd.Message.Chat.ID, b.menuText(u), b.menuKeyboard(u.Lang))
			return
		}
		b.send(ctx, upd.Message.Chat.ID, "👇", &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{webAppButton(t(u.Lang, keys[path]), url)},
		}})
	}
}

func (b *Bot) onLang(ctx context.Context, _ *tg.Bot, upd *models.Update) {
	u := b.rememberUser(upd.Message.From)
	u.Lang = map[string]string{"en": "ru", "ru": "en"}[u.Lang]
	_ = b.store.SetUserLang(u.ID, u.Lang)
	b.send(ctx, upd.Message.Chat.ID, b.menuText(u), b.menuKeyboard(u.Lang))
}

// --- owners ---

func (b *Bot) ownerScreen(lang string, tgID int64) (string, *models.InlineKeyboardMarkup) {
	owner, err := b.store.OwnerByTelegram(tgID)
	if err != nil {
		var rows [][]models.InlineKeyboardButton
		if url := b.appURL("/owner/login"); url != "" {
			rows = append(rows, []models.InlineKeyboardButton{webAppButton(t(lang, "btn_dashboard"), url)})
		}
		rows = append(rows, []models.InlineKeyboardButton{{Text: t(lang, "btn_back"), CallbackData: "menu"}})
		return t(lang, "owner_intro"), &models.InlineKeyboardMarkup{InlineKeyboard: rows}
	}
	shop, _ := b.store.GetShop(owner.ShopID, lang)
	var rows [][]models.InlineKeyboardButton
	if url := b.appURL("/owner"); url != "" {
		rows = append(rows, []models.InlineKeyboardButton{webAppButton(t(lang, "btn_dashboard"), url)})
	}
	rows = append(rows,
		[]models.InlineKeyboardButton{{Text: t(lang, "btn_unlink"), CallbackData: "unlink"}},
		[]models.InlineKeyboardButton{{Text: t(lang, "btn_back"), CallbackData: "menu"}},
	)
	return t(lang, "owner_linked", html.EscapeString(shop.Name)), &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// onOwner handles "/owner CODE". The right code lets this chat pick a demo shop.
func (b *Bot) onOwner(ctx context.Context, _ *tg.Bot, upd *models.Update) {
	u := b.rememberUser(upd.Message.From)
	chatID := upd.Message.Chat.ID
	_, code, _ := strings.Cut(strings.TrimSpace(upd.Message.Text), " ")
	code = strings.TrimSpace(code)
	if code == "" {
		text, kb := b.ownerScreen(u.Lang, u.ID)
		b.send(ctx, chatID, text, kb)
		return
	}
	if code != b.cfg.OwnerCode {
		b.send(ctx, chatID, t(u.Lang, "owner_badcode"), nil)
		return
	}
	b.mu.Lock()
	b.ownerCode[chatID] = time.Now()
	b.mu.Unlock()

	shops, err := b.store.ListShops(u.Lang, "", 0)
	if err != nil {
		log.Printf("bot: list shops: %v", err)
		return
	}
	var rows [][]models.InlineKeyboardButton
	for i := 0; i < len(shops); i += 2 {
		row := []models.InlineKeyboardButton{{Text: shops[i].Logo + " " + shops[i].Name, CallbackData: fmt.Sprintf("link:%d", shops[i].ID)}}
		if i+1 < len(shops) {
			row = append(row, models.InlineKeyboardButton{Text: shops[i+1].Logo + " " + shops[i+1].Name, CallbackData: fmt.Sprintf("link:%d", shops[i+1].ID)})
		}
		rows = append(rows, row)
	}
	b.send(ctx, chatID, t(u.Lang, "owner_pick"), &models.InlineKeyboardMarkup{InlineKeyboard: rows})
}

// --- callbacks ---

func (b *Bot) onCallback(ctx context.Context, _ *tg.Bot, upd *models.Update) {
	cq := upd.CallbackQuery
	u := b.rememberUser(&cq.From)
	answer := ""
	defer func() {
		b.api.AnswerCallbackQuery(ctx, &tg.AnswerCallbackQueryParams{CallbackQueryID: cq.ID, Text: answer})
	}()
	msg := cq.Message.Message
	if msg == nil {
		return
	}
	chatID := msg.Chat.ID
	edit := func(text string, kb *models.InlineKeyboardMarkup) {
		_, err := b.api.EditMessageText(ctx, &tg.EditMessageTextParams{ChatID: chatID, MessageID: msg.ID, Text: text,
			ParseMode: models.ParseModeHTML, ReplyMarkup: kb})
		if err != nil {
			log.Printf("bot: edit: %v", err)
		}
	}

	action, arg, _ := strings.Cut(cq.Data, ":")
	switch action {
	case "menu":
		edit(b.menuText(u), b.menuKeyboard(u.Lang))
	case "lang":
		u.Lang = map[string]string{"en": "ru", "ru": "en"}[u.Lang]
		_ = b.store.SetUserLang(u.ID, u.Lang)
		answer = t(u.Lang, "lang_set")
		edit(b.menuText(u), b.menuKeyboard(u.Lang))
	case "owners":
		edit(b.ownerScreen(u.Lang, u.ID))
	case "unlink":
		_ = b.store.UnlinkOwnerTelegram(u.ID)
		answer = t(u.Lang, "owner_unlinked")
		edit(b.ownerScreen(u.Lang, u.ID))
	case "link":
		b.mu.Lock()
		at, ok := b.ownerCode[chatID]
		b.mu.Unlock()
		if !ok || time.Since(at) > 10*time.Minute {
			answer = t(u.Lang, "owner_expired")
			return
		}
		shopID, _ := strconv.ParseInt(arg, 10, 64)
		shop, err := b.store.GetShop(shopID, u.Lang)
		if err != nil {
			return
		}
		if err := b.store.LinkOwnerTelegram(shopID, u.ID, chatID); err != nil {
			log.Printf("bot: link owner: %v", err)
			return
		}
		_, kb := b.ownerScreen(u.Lang, u.ID)
		edit(t(u.Lang, "owner_done", html.EscapeString(shop.Name)), kb)
	case "st":
		answer = b.onStatusButton(ctx, u, arg)
	case "reply":
		orderID, _ := strconv.ParseInt(arg, 10, 64)
		order, err := b.store.GetOrder(orderID, u.Lang)
		if err != nil || !b.ownsShop(u.ID, order.ShopID) {
			answer = t(u.Lang, "not_owner")
			return
		}
		b.mu.Lock()
		b.replyTo[chatID] = orderID
		b.mu.Unlock()
		b.send(ctx, chatID, t(u.Lang, "reply_prompt", orderID), nil)
	}
}

func (b *Bot) ownsShop(tgID, shopID int64) bool {
	owner, err := b.store.OwnerByTelegram(tgID)
	return err == nil && owner.ShopID == shopID
}

// onStatusButton handles "st:<order>:<status>" from the order message.
func (b *Bot) onStatusButton(ctx context.Context, u store.User, arg string) string {
	idStr, status, _ := strings.Cut(arg, ":")
	orderID, _ := strconv.ParseInt(idStr, 10, 64)
	order, err := b.store.GetOrder(orderID, u.Lang)
	if err != nil || !b.ownsShop(u.ID, order.ShopID) {
		return t(u.Lang, "not_owner")
	}
	if _, err := b.orders.SetStatus(ctx, orderID, status); err != nil {
		log.Printf("bot: set status: %v", err)
		return "Error"
	}
	return t(u.Lang, "status_set", t(u.Lang, "st_"+status))
}

// onMessage handles plain text. If the owner pressed "Reply via ShopBot",
// the text goes to the customer; otherwise the user gets the menu.
func (b *Bot) onMessage(ctx context.Context, _ *tg.Bot, upd *models.Update) {
	if upd.Message == nil || upd.Message.From == nil {
		return
	}
	u := b.rememberUser(upd.Message.From)
	chatID := upd.Message.Chat.ID

	b.mu.Lock()
	orderID, replying := b.replyTo[chatID]
	delete(b.replyTo, chatID)
	b.mu.Unlock()

	if replying && upd.Message.Text != "" {
		order, err := b.store.GetOrder(orderID, "en")
		if err != nil {
			return
		}
		custLang := b.userLang(order.UserID)
		text := t(custLang, "from_shop", html.EscapeString(order.ShopName), order.ID, html.EscapeString(upd.Message.Text))
		if _, err := b.send(ctx, order.UserID, text, nil); err != nil {
			b.send(ctx, chatID, t(u.Lang, "reply_failed"), nil)
			return
		}
		b.send(ctx, chatID, t(u.Lang, "reply_sent"), nil)
		return
	}
	b.send(ctx, chatID, b.menuText(u), b.menuKeyboard(u.Lang))
}
