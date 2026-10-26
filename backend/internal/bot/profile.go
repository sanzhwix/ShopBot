package bot

import (
	"bytes"
	"context"
	"fmt"
	"log"

	tg "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// What people see on the bot's profile page and in an empty chat with the bot.
// "" is the default for every language; "ru" is for users with Russian Telegram.
type profileTexts struct {
	name        string // chat title, up to 64 characters
	short       string // profile "About" and link previews, up to 120 characters
	description string // "What can this bot do?" in an empty chat, up to 512 characters
	commands    []models.BotCommand
}

var profiles = map[string]profileTexts{
	"": {
		name:  "ShopBot",
		short: "Shops near you in Astana: map, catalogues, one search for all shops and Kaspi payment, right in Telegram.",
		description: "🛍 ShopBot helps you buy from local shops in Astana without installing anything.\n\n" +
			"🗺 Map of malls, bakeries, clothing stores, pharmacies and more around you\n" +
			"🔍 Find one product in many shops and compare prices\n" +
			"🔐 Sign in to a shop once, ShopBot remembers your accounts\n" +
			"💳 Pay with Kaspi and follow your order in this chat\n\n" +
			"🏪 Own a shop? Get orders in Telegram and see your sales analytics.",
		commands: []models.BotCommand{
			{Command: "start", Description: "Main menu"},
			{Command: "map", Description: "Shops near me"},
			{Command: "search", Description: "Find a product in all shops"},
			{Command: "orders", Description: "My orders"},
			{Command: "owner", Description: "For shop owners"},
			{Command: "lang", Description: "English / Русский"},
		},
	},
	"ru": {
		name:  "ShopBot",
		short: "Магазины рядом в Астане: карта, каталоги, поиск по всем магазинам и оплата Kaspi прямо в Telegram.",
		description: "🛍 ShopBot помогает покупать в магазинах Астаны без установки приложений.\n\n" +
			"🗺 Карта ТРЦ, пекарен, магазинов одежды, аптек и других мест рядом\n" +
			"🔍 Найдите товар сразу во многих магазинах и сравните цены\n" +
			"🔐 Войдите в магазин один раз, ShopBot запомнит аккаунт\n" +
			"💳 Оплата Kaspi и статус заказа прямо в этом чате\n\n" +
			"🏪 У вас магазин? Получайте заказы в Telegram и смотрите аналитику продаж.",
		commands: []models.BotCommand{
			{Command: "start", Description: "Главное меню"},
			{Command: "map", Description: "Магазины рядом"},
			{Command: "search", Description: "Найти товар во всех магазинах"},
			{Command: "orders", Description: "Мои заказы"},
			{Command: "owner", Description: "Для владельцев магазинов"},
			{Command: "lang", Description: "English / Русский"},
		},
	},
}

// ApplyProfile sets the name, descriptions and command lists in all languages.
// Telegram limits how often a name may change, so each text is sent only when it differs.
func ApplyProfile(ctx context.Context, api *tg.Bot) error {
	for lang, p := range profiles {
		if cur, err := api.GetMyName(ctx, &tg.GetMyNameParams{LanguageCode: lang}); err != nil || cur.Name != p.name {
			if _, err := api.SetMyName(ctx, &tg.SetMyNameParams{Name: p.name, LanguageCode: lang}); err != nil {
				return fmt.Errorf("set name (%q): %w", lang, err)
			}
		}
		if cur, err := api.GetMyShortDescription(ctx, &tg.GetMyShortDescriptionParams{LanguageCode: lang}); err != nil || cur.ShortDescription != p.short {
			if _, err := api.SetMyShortDescription(ctx, &tg.SetMyShortDescriptionParams{ShortDescription: p.short, LanguageCode: lang}); err != nil {
				return fmt.Errorf("set short description (%q): %w", lang, err)
			}
		}
		if cur, err := api.GetMyDescription(ctx, &tg.GetMyDescriptionParams{LanguageCode: lang}); err != nil || cur.Description != p.description {
			if _, err := api.SetMyDescription(ctx, &tg.SetMyDescriptionParams{Description: p.description, LanguageCode: lang}); err != nil {
				return fmt.Errorf("set description (%q): %w", lang, err)
			}
		}
		if _, err := api.SetMyCommands(ctx, &tg.SetMyCommandsParams{Commands: p.commands, LanguageCode: lang}); err != nil {
			return fmt.Errorf("set commands (%q): %w", lang, err)
		}
	}
	return nil
}

// ApplyPhoto uploads a new profile photo (JPEG). Every call adds a photo, so it is
// run by hand with `make bot-profile`, not at every start.
func ApplyPhoto(ctx context.Context, api *tg.Bot, jpeg []byte) error {
	_, err := api.SetMyProfilePhoto(ctx, &tg.SetMyProfilePhotoParams{
		Photo: models.InputProfilePhotoStatic{Photo: "attach://avatar.jpg", MediaAttachment: bytes.NewReader(jpeg)},
	})
	return err
}

func (b *Bot) setupProfile(ctx context.Context) {
	if err := ApplyProfile(ctx, b.api); err != nil {
		log.Printf("bot: profile: %v", err)
	}
}
