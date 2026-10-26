# ShopBot — MVP

ShopBot is a Telegram commerce platform for small and medium shops in Kazakhstan.
A shopper opens the bot, finds shops on a map of Astana, looks through a shop's
catalogue, signs in to the shop, adds items to the cart and pays with Kaspi.
The shop owner gets the order in Telegram and sees sales analytics in a dashboard.

Team: Aknyssan Itegul, Sanzhar Serzhanov · Technological Entrepreneurship · Assignment 5 (MVP)

## What works

**Shopper (Telegram Mini App)**
- Bot menu with buttons: Map, Clothes, Bakeries, Malls, Groceries, Pharmacy, Electronics, Flowers, Find a product, Cart, Orders, My shop accounts, language, For shop owners.
- Map of Astana with water-drop pins (icon in a white circle). A category button opens the map with only that category.
- Real shops near the user from OpenStreetMap as grey pins, with an "Invite this shop" button.
- Mall page (shops inside), shop catalogue with categories and all items, product sheet with size, color and quantity.
- Shared product search: one search over all shops, filters by price, size and material, sort by price or distance.
- Shop sign-in when it is needed: adding to the cart asks for an account at that shop (password, new account, or "Continue with Telegram").
- **My shop accounts**: ShopBot saves shop logins (AES-256-GCM encrypted). Next time a shop asks for an account, ShopBot signs in automatically.
- Cart per shop, checkout (pickup or delivery, phone from Telegram), **Kaspi payment** (demo page, or the shop's own Kaspi link).
- Order tracking with status updates in the chat. English and Russian.

**Shop owner**
- New order message in Telegram with all details and buttons: 💬 Message customer, ✅ Accept, 📦 Ready, 🚚 Delivered, ❌ Cancel. The customer is told about each change.
- Dashboard: revenue, orders, average order, new customers, conversion (with change against the previous period), revenue per day, best sellers, sales by category, orders by weekday and hour, customer funnel, order sources, new and returning customers, latest orders. A new paid order appears live.
- Orders page with status actions, catalogue editor (price, stock, new products, **photo upload** from the phone or laptop), settings (hours, Kaspi link).

**Founders** (`/admin`): real tester data only — users, daily activity, funnel, top searches, most invited shops, feedback. This is the data for the HADI cycle and beta testing.

## Run it

Short version with all commands: **[start.md](start.md)**.

### With Telegram (the real demo)

Our bot is **[@ShopBotMVPbot](https://t.me/ShopBotMVPbot)**.

1. In Telegram, open **@BotFather**, send `/newbot` and copy the token. (Done for @ShopBotMVPbot.)
2. `make env`, then put the token in `.env` as `BOT_TOKEN=...` (done).
3. `make bot-profile` — once: sets the avatar (`assets/bot/avatar.jpg`), the name, the description,
   the short "About" text and the command list, in English and Russian (done for @ShopBotMVPbot).
4. `make up` — builds and starts the app and a free Cloudflare HTTPS tunnel (Docker needed).
   After about 10 seconds the log shows `Mini App: https://….trycloudflare.com`. The bot then sets its menu button by itself.
5. Open the bot in Telegram and press **Start** (or send `/start`).
6. To receive orders as a shop owner: send `/owner demo2026` to the bot and pick a shop.
   Use a second Telegram account (your teammate) for the owner, so customer and owner are two phones.

`make logs` shows the server log, `make down` stops everything.
Without Docker for the app: `make local` (needs Go 1.26 and Node 22; the tunnel still runs in Docker).

**Optional, in @BotFather** (the Bot API cannot do these):
- `/setdescriptionpic` → choose the bot → send `assets/bot/description.png`. It is the picture above the description in an empty chat.
- `/setjoingroups` → Disable. ShopBot works in private chats only.

**If something looks wrong**
- The menu buttons do nothing: the tunnel address changed after a restart. Send `/start` again for fresh buttons.
- `409 Conflict` in the log: the bot runs twice (for example `make up` and `make local`). Stop one of them.
- "Open ShopBot from Telegram" in the app: the page was opened outside Telegram while `DEV_AUTH=false`.

### Bot commands

| Command | What it does |
|---|---|
| `/start` | Main menu: Map, categories, Find a product, Cart, My orders, My shop accounts, language, For shop owners |
| `/map` | Opens the map |
| `/search` | Opens the product search over all shops |
| `/orders` | Opens my orders |
| `/owner demo2026` | Links this Telegram account to a demo shop; its orders then come to this chat |
| `/owner` | Shows the linked shop, the dashboard button and "stop managing" |
| `/lang` | Switches English / Russian |

The **ShopBot** button next to the message field always opens the map.

### In a browser (no Telegram)

```bash
docker compose up --build app      # or: make dev  (Vite on http://localhost:5173)
```

| Page | Address | Login |
|---|---|---|
| Mini App | http://localhost:8080/map | none (`DEV_AUTH=true` gives a demo user) |
| Owner dashboard | http://localhost:8080/owner | `owner@nan-and-co.kz` / `demo123` (any shop: `owner@<shop-slug>.kz`) |
| Founder metrics | http://localhost:8080/admin | password `admin2026` |

Demo account in every shop: `demo@shopbot.kz` / `demo123`.

### Tests

```bash
make test        # Go vet + tests, TypeScript check
```

The Go tests check: Telegram initData signatures, cookie sessions, the password vault, the whole
shopping flow through the API (sign-in gate → saved account → cart → checkout → Kaspi → owner),
the analytics numbers against SQL, the OpenStreetMap fallback, and the bot messages against a fake
Telegram server (order card, "Message customer" link, privacy fallback, status edits).

## Demo script (about 4 minutes)

1. `/start` in the bot → the menu. Tap **🥐 Bakeries** → map with bakeries only. Tap a pin → **Open catalogue**.
2. Tap a croissant → **Add to cart** → the shop asks to sign in → **Fill in** the demo account, keep "Save to My shop accounts" on.
3. Profile → **My shop accounts** shows the saved login. Sign out in the shop and add again: ShopBot signs in by itself.
4. **Find a product** → "hoodie" → 8 offers from 4 shops, filter by material.
5. Cart → Checkout → **Pay with Kaspi** → Kaspi Pay (demo) → Pay.
6. The owner's phone gets the order with **Message customer** and **Accept**. Tap Accept → the customer gets "accepted".
7. On the projector: owner dashboard — the new order appears live; walk through the charts.
8. Map → a grey pin (real shop from OpenStreetMap) → **Invite this shop**. Show `/admin` with real tester data.

## How it is built

```
Telegram ──(bot updates, long polling)──► Go server ◄──HTTPS tunnel── Mini App (React in Telegram)
                                           │  API (chi) · bot · SSE live orders
                                           │  SQLite (orders, catalogue, events, vault)
                                           └─► OpenStreetMap Overpass (real shops nearby)
```

- `backend/` — Go 1.26: `cmd/shopbot` starts everything. `internal/api` (HTTP handlers), `bot` (Telegram),
  `orders` (order rules shared by API and bot), `store` (all SQL), `analytics`, `auth` (initData, cookies),
  `vault` (AES-GCM), `payment` (Kaspi demo / link), `osm`, `seed` (demo data).
- `webapp/` — React 19 + Vite + TypeScript + Tailwind. Leaflet map, Recharts.
- One binary serves the API and the built Mini App. Data lives in `data/` (database and keys).

## Data

- **Imaginary:** 14 partner shops at real Astana addresses (Khan Shatyr, Mega Silk Way, Keruen, Abu Dhabi Plaza and streets), 156 products in English and Russian, and 180 days of orders and funnel numbers. They are generated with a fixed random seed, so every demo shows the same numbers.
- **Product photos:** real, openly licensed photos (Flickr, Wikimedia Commons via [Openverse](https://openverse.org)), cropped to 4:3. Each photo shows its author and license in the product card; the full list is in [docs/photo-credits.md](docs/photo-credits.md). Owners can replace any photo; uploads are shrunk in the browser (no GPS data is kept) and stored in `data/uploads`.
- **Real:** shops from OpenStreetMap near the user, and every event, order, invite and rating from real testers.
  OpenStreetMap's free API is sometimes overloaded, so the server tries several mirrors, then a cached answer, then a saved snapshot of 980 real shops in central Astana.
  Map data © OpenStreetMap contributors (ODbL).

## Security notes

- Every Mini App request carries Telegram `initData`; the server checks its HMAC signature with the bot token.
- Owner and admin pages use signed HttpOnly cookies. An owner only sees their own shop.
- Shop passwords are stored as bcrypt hashes. Saved logins in the vault are encrypted with AES-256-GCM; the key is outside the database (`VAULT_KEY` or `data/vault.key`).
- The vault must keep passwords that can be decrypted, because auto sign-in needs them. In production, partner shops should accept "Sign in with ShopBot" (OAuth) instead of stored passwords.

## Known limits

- Real Kaspi e-commerce payments need a merchant contract, so payment runs in **demo** mode (a page marked DEMO) or **link** mode (the shop's own Kaspi link; the owner confirms in Telegram).
- The shops are imaginary, so their "websites" (customer accounts) are hosted inside ShopBot.
- The quick-tunnel address changes on every start. The bot updates its menu button automatically, but buttons in old chat messages need `/start` again.
