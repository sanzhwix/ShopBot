package bot

import (
	"context"
	"fmt"
	"html"
	"log"
	"strings"
	"time"

	tg "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"shopbot/internal/seed"
	"shopbot/internal/store"
)

// The Bot implements orders.Notifier.

// OrderPaid shows the order to the shop and sends a receipt to the customer.
// In Kaspi link mode the shop already has a card for this order, so it is updated instead.
func (b *Bot) OrderPaid(ctx context.Context, order store.OrderDetail) {
	if !b.updateOwnerCards(ctx, order) {
		b.sendOwnerCard(ctx, order)
	}
	b.notifyCustomer(ctx, order)
}

// OrderAwaitingPayment sends the order card with a "Mark as paid" button (Kaspi link mode).
func (b *Bot) OrderAwaitingPayment(ctx context.Context, order store.OrderDetail) {
	b.sendOwnerCard(ctx, order)
}

// StatusChanged updates the order card in the shop's chat and tells the customer.
func (b *Bot) StatusChanged(ctx context.Context, order store.OrderDetail) {
	b.updateOwnerCards(ctx, order)
	b.notifyCustomer(ctx, order)
}

// updateOwnerCards edits the order cards already sent to the shop. It reports
// false if no card was sent yet.
func (b *Bot) updateOwnerCards(ctx context.Context, order store.OrderDetail) bool {
	msgs, err := b.store.OwnerMessages(order.ID)
	if err != nil {
		log.Printf("bot: owner messages: %v", err)
	}
	for _, m := range msgs {
		lang := b.userLang(m.ChatID)
		detail, err := b.store.GetOrder(order.ID, lang)
		if err != nil {
			continue
		}
		edit := func(profileLink bool) error {
			_, err := b.api.EditMessageText(ctx, &tg.EditMessageTextParams{
				ChatID: m.ChatID, MessageID: m.MessageID, Text: b.orderCard(detail, lang),
				ParseMode: models.ParseModeHTML, ReplyMarkup: b.orderKeyboard(detail, lang, profileLink),
			})
			if err != nil && strings.Contains(err.Error(), "message is not modified") {
				return nil
			}
			return err
		}
		if err := edit(true); err != nil {
			// Same privacy rule as in sendOwnerCard: fall back to the relay button.
			if err := edit(false); err != nil {
				log.Printf("bot: edit order card: %v", err)
			}
		}
	}
	return len(msgs) > 0
}

func (b *Bot) sendOwnerCard(ctx context.Context, order store.OrderDetail) {
	shop, err := b.store.GetShop(order.ShopID, "en")
	if err != nil || shop.NotifyChatID == nil {
		log.Printf("bot: shop %d has no Telegram chat linked, order #%d not sent", order.ShopID, order.ID)
		return
	}
	chatID := *shop.NotifyChatID
	lang := b.userLang(chatID)
	detail, err := b.store.GetOrder(order.ID, lang)
	if err != nil {
		return
	}
	text := b.orderCard(detail, lang)
	msg, err := b.send(ctx, chatID, text, b.orderKeyboard(detail, lang, true))
	if err != nil {
		// Telegram refuses a tg://user link when the customer's privacy settings hide them.
		// Then the owner can still answer through the bot.
		log.Printf("bot: order card with profile link failed (%v), sending with relay button", err)
		msg, err = b.send(ctx, chatID, text, b.orderKeyboard(detail, lang, false))
	}
	if err != nil {
		log.Printf("bot: send order card: %v", err)
		return
	}
	_ = b.store.SaveOwnerMessage(order.ID, chatID, msg.ID)
}

func (b *Bot) orderCard(o store.OrderDetail, lang string) string {
	var sb strings.Builder
	line := func(s string) {
		sb.WriteString(s)
		sb.WriteByte('\n')
	}
	esc := html.EscapeString

	line(t(lang, "new_order", o.ID, esc(o.ShopName)))
	line("")
	name := strings.TrimSpace(o.Customer.FirstName + " " + o.Customer.LastName)
	if o.Customer.Username != "" {
		name += " (@" + o.Customer.Username + ")"
	}
	line(t(lang, "customer", esc(name)))
	if o.Phone != "" {
		line(t(lang, "phone", esc(o.Phone)))
	}
	if o.AccountLogin != "" {
		line(t(lang, "account", esc(o.AccountLogin)))
	}
	line("")
	for _, it := range o.Items {
		s := fmt.Sprintf("• %d × %s — %s", it.Qty, esc(it.Name), money(it.Qty*it.Price))
		var opts []string
		if it.Size != "" {
			opts = append(opts, it.Size)
		}
		if it.Color != "" {
			opts = append(opts, it.Color)
		}
		if len(opts) > 0 {
			s += " <i>(" + esc(strings.Join(opts, ", ")) + ")</i>"
		}
		line(s)
	}
	line("")
	payment := t(lang, "paid_kaspi")
	if o.Status == store.StatusPendingPayment {
		payment = t(lang, "pending_kaspi")
	}
	line(t(lang, "total", money(o.Total)) + " · " + payment)
	if o.DeliveryType == "delivery" {
		line(t(lang, "delivery", esc(o.Address)))
	} else {
		line(t(lang, "pickup"))
	}
	if o.Comment != "" {
		line(t(lang, "comment", esc(o.Comment)))
	}
	line(t(lang, "source", t(lang, "src_"+o.Source)))
	line("🕒 " + time.Unix(o.CreatedAt, 0).In(seed.Astana).Format("02.01 15:04"))
	line("")
	sb.WriteString(t(lang, "status_line", t(lang, "st_"+o.Status)))
	return sb.String()
}

// orderKeyboard shows only the next sensible statuses. profileLink=false
// replaces the "Message customer" link with "Reply via ShopBot".
func (b *Bot) orderKeyboard(o store.OrderDetail, lang string, profileLink bool) *models.InlineKeyboardMarkup {
	btn := func(key, status string) models.InlineKeyboardButton {
		return models.InlineKeyboardButton{Text: t(lang, key), CallbackData: fmt.Sprintf("st:%d:%s", o.ID, status)}
	}
	var rows [][]models.InlineKeyboardButton
	if profileLink && o.Customer.ID > 0 {
		url := fmt.Sprintf("tg://user?id=%d", o.Customer.ID)
		if o.Customer.Username != "" {
			url = "https://t.me/" + o.Customer.Username
		}
		rows = append(rows, []models.InlineKeyboardButton{{Text: t(lang, "btn_msg"), URL: url}})
	} else {
		rows = append(rows, []models.InlineKeyboardButton{{Text: t(lang, "btn_reply"), CallbackData: fmt.Sprintf("reply:%d", o.ID)}})
	}
	switch o.Status {
	case store.StatusPendingPayment:
		rows = append(rows, []models.InlineKeyboardButton{btn("btn_markpaid", store.StatusPaid), btn("btn_cancel", store.StatusCancelled)})
	case store.StatusPaid:
		rows = append(rows, []models.InlineKeyboardButton{btn("btn_accept", store.StatusAccepted), btn("btn_cancel", store.StatusCancelled)})
	case store.StatusAccepted:
		rows = append(rows, []models.InlineKeyboardButton{btn("btn_ready", store.StatusReady), btn("btn_cancel", store.StatusCancelled)})
	case store.StatusReady:
		rows = append(rows, []models.InlineKeyboardButton{btn("btn_delivered", store.StatusDelivered)})
	}
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (b *Bot) notifyCustomer(ctx context.Context, order store.OrderDetail) {
	if order.UserID <= 0 {
		return // demo customer
	}
	lang := b.userLang(order.UserID)
	shop, err := b.store.GetShop(order.ShopID, lang)
	if err != nil {
		return
	}
	var text string
	var kb *models.InlineKeyboardMarkup
	button := func(key, path string) *models.InlineKeyboardMarkup {
		if url := b.appURL(path); url != "" {
			return &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{webAppButton(t(lang, key), url)}}}
		}
		return nil
	}
	name := html.EscapeString(shop.Name)
	switch order.Status {
	case store.StatusPaid:
		text = t(lang, "cust_paid", order.ID, name, money(order.Total))
		kb = button("btn_track", fmt.Sprintf("/orders/%d", order.ID))
	case store.StatusAccepted:
		text = t(lang, "cust_accepted", name, order.ID)
	case store.StatusReady:
		if order.DeliveryType == "delivery" {
			text = t(lang, "cust_ready_delivery", order.ID)
		} else {
			text = t(lang, "cust_ready_pickup", order.ID, name, html.EscapeString(shop.Address))
		}
	case store.StatusDelivered:
		text = t(lang, "cust_delivered", order.ID, name)
		kb = button("btn_rate", "/feedback")
	case store.StatusCancelled:
		text = t(lang, "cust_cancelled", name, order.ID)
	default:
		return
	}
	if _, err := b.send(ctx, order.UserID, text, kb); err != nil {
		log.Printf("bot: notify customer %d: %v", order.UserID, err)
	}
}
