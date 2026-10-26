# ShopBot MVP — features and user paths

This file supports Steps 5–7 of Assignment 5 (user path map, prioritized functions, MVP scope).

## User path map

### Shopper

| Step | What the shopper does | What ShopBot shows | Event we record |
|---|---|---|---|
| 1. Start | Opens the bot, sends /start | Menu with Map, categories, search, cart, orders | app_open |
| 2. Find | Taps Map or a category (e.g. Bakeries) | Map of Astana with drop pins; real OSM shops in grey | map_view |
| 3. Choose | Taps a pin | Preview: name, rating, hours, distance | pin_tap |
| 4. Browse | Opens the catalogue (or a mall, then a shop) | Categories and all items | shop_view |
| 5. Compare (alt.) | Searches "hoodie" | Offers from many shops, filters by price, size, material | search |
| 6. Decide | Opens a product, picks size and color | Product sheet | product_view |
| 7. Sign in | Adds to cart; the shop asks for an account | Login sheet, or automatic sign-in with a saved account | — |
| 8. Buy | Checkout, pickup or delivery | Order summary | add_to_cart, checkout |
| 9. Pay | Pay with Kaspi | Kaspi Pay page (demo) | paid |
| 10. Wait | Gets messages in the chat | Paid → Accepted → Ready → Delivered | — |
| 11. Return | Rates ShopBot; next time ShopBot signs in by itself | Feedback form; saved accounts | feedback |

### Shop owner

| Step | What the owner does | What ShopBot shows |
|---|---|---|
| 1. Join | Sends /owner CODE, picks the shop (later: self-service sign-up) | "You now manage Nan & Co" |
| 2. New order | Gets a Telegram message | Items, total, delivery, customer; Message customer, Accept, Cancel |
| 3. Fulfil | Taps Accept → Ready → Delivered | The message updates; the customer is told each time |
| 4. Talk | Taps Message customer | Opens the chat with the customer (or replies through the bot) |
| 5. Analyse | Opens the dashboard | KPIs, revenue, best sellers, peak hours, funnel, sources |
| 6. Manage | Edits prices, stock, products; sets the Kaspi link | Catalogue and settings pages |

## Functions by priority (MoSCoW)

| # | Function | Priority | Why | MVP |
|---|---|---|---|---|
| 1 | Bot menu that opens the Mini App | Must | Entry point, no app install | ✅ |
| 2 | Map with shops and category filters | Must | Main way to discover shops | ✅ |
| 3 | Shop catalogue (categories, items, variants) | Must | Shoppers must see what they can buy | ✅ |
| 4 | Cart and checkout | Must | Core value: order in Telegram | ✅ |
| 5 | Kaspi payment | Must | Trusted payment method in Kazakhstan (G5) | ✅ demo / link |
| 6 | New order message to the shop + status buttons | Must | Pain P1–P2: lost and slow orders | ✅ |
| 7 | "Message customer" button | Must | Owner can clarify the order at once | ✅ |
| 8 | Owner dashboard with analytics | Must | Gain G3: clear data on what sells | ✅ |
| 9 | Shared product search with filters | Should | Main difference from other bot builders (G1) | ✅ |
| 10 | Shop sign-in when needed | Should | Shops keep their own customers (P5) | ✅ |
| 11 | Saved shop accounts (auto sign-in) | Should | No need to remember many passwords | ✅ |
| 12 | Real shops from OpenStreetMap + invite | Should | Real data; lead channel for B2B sales | ✅ |
| 13 | Live order feed in the dashboard | Should | Owner reacts fast | ✅ |
| 14 | Catalogue editing by the owner | Should | No IT staff needed (P4) | ✅ basic |
| 15 | English / Russian | Should | Market language | ✅ |
| 16 | Founder metrics and feedback | Should | Data for HADI and beta testing | ✅ |
| 17 | Real Kaspi API (merchant contract) | Could | Needs a Kaspi merchant agreement | ❌ later |
| 18 | Halyk Bank payments, delivery services | Could | More choice | ❌ later |
| 19 | Loyalty points, broadcasts | Could | Gain G4: loyal customers | ❌ later |
| 20 | Self-service shop sign-up and photo upload | Could | Scale to many shops | ❌ later |
| 21 | Kazakh language | Could | Wider audience | ❌ later |
| 22 | Own marketplace / delivery fleet | Won't | Not our model (B2B SaaS) | — |

## MVP scope

- **In:** functions 1–16, for 14 demo partner shops in Astana with 156 products.
- **Out (next versions):** functions 17–21. Kaspi works in demo mode or through the shop's own payment link.
- **Real data:** OpenStreetMap shops near the user; all events, orders, invites and ratings from testers (see `/admin`).
- **Demo data:** shop catalogues and 180 days of order history, generated with a fixed seed.
