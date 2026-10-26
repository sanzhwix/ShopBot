package bot

import (
	"fmt"
	"strconv"
	"strings"
)

var texts = map[string]map[string]string{
	"en": {
		"welcome": "👋 <b>Welcome to ShopBot, %s!</b>\n\n" +
			"Find shops near you in Astana, look through their catalogues and order in one tap, right here in Telegram.\n\n" +
			"🗺 <b>Map</b>: malls, bakeries, clothes and more around you\n" +
			"🔍 <b>Find a product</b>: compare one item across many shops\n" +
			"💳 Pay with Kaspi and follow your order in this chat",
		"no_url":        "\n\n⚠️ The Mini App address is not ready yet (the tunnel is starting). Send /start again in a few seconds.",
		"btn_map":       "🗺 Open map",
		"btn_clothes":   "👗 Clothes",
		"btn_bakery":    "🥐 Bakeries",
		"btn_mall":      "🏬 Malls",
		"btn_grocery":   "🛒 Groceries",
		"btn_pharmacy":  "💊 Pharmacy",
		"btn_electro":   "📱 Electronics",
		"btn_flowers":   "💐 Flowers",
		"btn_search":    "🔍 Find a product",
		"btn_cart":      "🧺 Cart",
		"btn_orders":    "📦 My orders",
		"btn_accounts":  "🔐 My shop accounts",
		"btn_lang":      "🌐 Русский",
		"btn_owners":    "🏪 For shop owners",
		"btn_dashboard": "📊 Open dashboard",
		"btn_unlink":    "🔓 Stop managing this shop",
		"btn_back":      "⬅️ Menu",
		"menu_button":   "Open ShopBot",
		"lang_set":      "Language: English",
		"owner_intro": "🏪 <b>ShopBot for business</b>\n\n" +
			"Open your shop in Telegram in one day: catalogue, Kaspi payments, orders in this chat and a sales dashboard. No website and no IT team needed.\n\n" +
			"<b>Demo:</b> send <code>/owner CODE</code> to manage one of the demo shops. You will get its orders here.",
		"owner_linked":   "🏪 You manage <b>%s</b>.\nNew orders arrive in this chat. Statistics and catalogue are in the dashboard.",
		"owner_pick":     "Choose the demo shop you want to manage:",
		"owner_badcode":  "❌ Wrong code. Ask the ShopBot team for the demo code.",
		"owner_expired":  "The code has expired. Send /owner CODE again.",
		"owner_done":     "✅ You now manage <b>%s</b>.\nNew orders will arrive in this chat with buttons to accept them and to message the customer.",
		"owner_unlinked": "You no longer manage a shop.",
		"not_owner":      "This order belongs to another shop.",
		"help":           "ShopBot commands:\n/start: main menu\n/map: open the map\n/search: find a product in all shops\n/orders: my orders\n/owner CODE: manage a demo shop\n/lang: switch language",

		"new_order":          "🛍 <b>New order #%d</b> · %s",
		"customer":           "👤 %s",
		"phone":              "📞 %s",
		"account":            "🔐 Shop account: %s",
		"total":              "💰 <b>Total: %s</b>",
		"paid_kaspi":         "✅ Paid via Kaspi",
		"pending_kaspi":      "⏳ Waiting for Kaspi payment",
		"pickup":             "🏃 Pickup at the shop",
		"delivery":           "🚚 Delivery: %s",
		"comment":            "💬 «%s»",
		"source":             "📍 Came from: %s",
		"src_map":            "Map",
		"src_search":         "Product search",
		"src_direct":         "Direct link",
		"status_line":        "Status: <b>%s</b>",
		"st_pending_payment": "⏳ Waiting for payment",
		"st_paid":            "🆕 Paid, waiting for you",
		"st_accepted":        "👨‍🍳 Accepted",
		"st_ready":           "📦 Ready",
		"st_delivered":       "✅ Delivered",
		"st_cancelled":       "❌ Cancelled",
		"btn_msg":            "💬 Message customer",
		"btn_reply":          "✉️ Reply via ShopBot",
		"btn_accept":         "✅ Accept",
		"btn_ready":          "📦 Ready",
		"btn_delivered":      "🚚 Delivered",
		"btn_cancel":         "❌ Cancel",
		"btn_markpaid":       "💰 Mark as paid",
		"status_set":         "Status: %s",
		"reply_prompt":       "✍️ Write a message to the customer of order #%d. ShopBot will send it for you.",
		"reply_sent":         "✅ Sent to the customer.",
		"reply_failed":       "⚠️ Could not deliver the message: the customer has not started the bot.",
		"from_shop":          "💬 <b>%s</b> about order #%d:\n\n%s",

		"cust_paid":           "✅ <b>Order #%d is paid</b>\n%s · %s\n\nThe shop will confirm it soon. We will write here when the status changes.",
		"cust_accepted":       "👨‍🍳 <b>%s</b> accepted order #%d and is preparing it.",
		"cust_ready_pickup":   "📦 Order #%d is ready! Pick it up at <b>%s</b>, %s.",
		"cust_ready_delivery": "🚚 Order #%d is ready and is on the way to you.",
		"cust_delivered":      "🎉 Order #%d is complete. Thank you for shopping with %s!\n\nHow was ShopBot? Your rating helps us improve.",
		"cust_cancelled":      "❌ %s cancelled order #%d. Kaspi will return the money.",
		"btn_track":           "📦 Track order",
		"btn_rate":            "⭐ Rate ShopBot",
	},
	"ru": {
		"welcome": "👋 <b>Добро пожаловать в ShopBot, %s!</b>\n\n" +
			"Находите магазины рядом в Астане, смотрите каталоги и заказывайте в один тап прямо в Telegram.\n\n" +
			"🗺 <b>Карта</b>: торговые центры, пекарни, одежда и другое рядом с вами\n" +
			"🔍 <b>Найти товар</b>: сравните один товар во многих магазинах\n" +
			"💳 Оплата через Kaspi, статус заказа в этом чате",
		"no_url":        "\n\n⚠️ Адрес Mini App ещё не готов (туннель запускается). Отправьте /start через несколько секунд.",
		"btn_map":       "🗺 Открыть карту",
		"btn_clothes":   "👗 Одежда",
		"btn_bakery":    "🥐 Пекарни",
		"btn_mall":      "🏬 ТРЦ",
		"btn_grocery":   "🛒 Продукты",
		"btn_pharmacy":  "💊 Аптеки",
		"btn_electro":   "📱 Электроника",
		"btn_flowers":   "💐 Цветы",
		"btn_search":    "🔍 Найти товар",
		"btn_cart":      "🧺 Корзина",
		"btn_orders":    "📦 Мои заказы",
		"btn_accounts":  "🔐 Мои аккаунты магазинов",
		"btn_lang":      "🌐 English",
		"btn_owners":    "🏪 Для владельцев магазинов",
		"btn_dashboard": "📊 Открыть панель",
		"btn_unlink":    "🔓 Перестать управлять магазином",
		"btn_back":      "⬅️ Меню",
		"menu_button":   "Открыть ShopBot",
		"lang_set":      "Язык: русский",
		"owner_intro": "🏪 <b>ShopBot для бизнеса</b>\n\n" +
			"Откройте магазин в Telegram за один день: каталог, оплата Kaspi, заказы в этом чате и панель продаж. Без сайта и без IT-команды.\n\n" +
			"<b>Демо:</b> отправьте <code>/owner КОД</code>, чтобы управлять одним из демо-магазинов. Его заказы будут приходить сюда.",
		"owner_linked":   "🏪 Вы управляете магазином <b>%s</b>.\nНовые заказы приходят в этот чат. Статистика и каталог в панели.",
		"owner_pick":     "Выберите демо-магазин, которым хотите управлять:",
		"owner_badcode":  "❌ Неверный код. Спросите демо-код у команды ShopBot.",
		"owner_expired":  "Код устарел. Отправьте /owner КОД ещё раз.",
		"owner_done":     "✅ Теперь вы управляете магазином <b>%s</b>.\nНовые заказы будут приходить в этот чат с кнопками «принять» и «написать клиенту».",
		"owner_unlinked": "Вы больше не управляете магазином.",
		"not_owner":      "Этот заказ принадлежит другому магазину.",
		"help":           "Команды ShopBot:\n/start: главное меню\n/map: открыть карту\n/search: найти товар во всех магазинах\n/orders: мои заказы\n/owner КОД: управлять демо-магазином\n/lang: сменить язык",

		"new_order":          "🛍 <b>Новый заказ #%d</b> · %s",
		"customer":           "👤 %s",
		"phone":              "📞 %s",
		"account":            "🔐 Аккаунт в магазине: %s",
		"total":              "💰 <b>Итого: %s</b>",
		"paid_kaspi":         "✅ Оплачено через Kaspi",
		"pending_kaspi":      "⏳ Ожидает оплаты Kaspi",
		"pickup":             "🏃 Самовывоз из магазина",
		"delivery":           "🚚 Доставка: %s",
		"comment":            "💬 «%s»",
		"source":             "📍 Источник: %s",
		"src_map":            "Карта",
		"src_search":         "Поиск товара",
		"src_direct":         "Прямая ссылка",
		"status_line":        "Статус: <b>%s</b>",
		"st_pending_payment": "⏳ Ожидает оплаты",
		"st_paid":            "🆕 Оплачен, ждёт вас",
		"st_accepted":        "👨‍🍳 Принят",
		"st_ready":           "📦 Готов",
		"st_delivered":       "✅ Выдан",
		"st_cancelled":       "❌ Отменён",
		"btn_msg":            "💬 Написать клиенту",
		"btn_reply":          "✉️ Ответить через ShopBot",
		"btn_accept":         "✅ Принять",
		"btn_ready":          "📦 Готов",
		"btn_delivered":      "🚚 Выдан",
		"btn_cancel":         "❌ Отменить",
		"btn_markpaid":       "💰 Отметить оплату",
		"status_set":         "Статус: %s",
		"reply_prompt":       "✍️ Напишите сообщение клиенту заказа #%d. ShopBot отправит его за вас.",
		"reply_sent":         "✅ Отправлено клиенту.",
		"reply_failed":       "⚠️ Не удалось отправить: клиент не запускал бота.",
		"from_shop":          "💬 <b>%s</b> по заказу #%d:\n\n%s",

		"cust_paid":           "✅ <b>Заказ #%d оплачен</b>\n%s · %s\n\nМагазин скоро подтвердит его. Мы напишем сюда, когда статус изменится.",
		"cust_accepted":       "👨‍🍳 <b>%s</b> принял заказ #%d и готовит его.",
		"cust_ready_pickup":   "📦 Заказ #%d готов! Заберите его в <b>%s</b>, %s.",
		"cust_ready_delivery": "🚚 Заказ #%d готов и уже едет к вам.",
		"cust_delivered":      "🎉 Заказ #%d выполнен. Спасибо за покупку в %s!\n\nКак вам ShopBot? Ваша оценка помогает нам стать лучше.",
		"cust_cancelled":      "❌ %s отменил заказ #%d. Kaspi вернёт деньги.",
		"btn_track":           "📦 Отследить заказ",
		"btn_rate":            "⭐ Оценить ShopBot",
	},
}

func t(lang, key string, args ...any) string {
	s, ok := texts[lang][key]
	if !ok {
		s = texts["en"][key]
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

// money formats tenge like "12 900 ₸".
func money(n int) string {
	s := strconv.Itoa(n)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteRune(' ')
		}
		b.WriteRune(c)
	}
	return b.String() + " ₸"
}
