# How to start ShopBot

Bot: **[@ShopBotMVPbot](https://t.me/ShopBotMVPbot)** · All commands run in the project folder (`ShopBot/`).

## Before the first start

You need **Docker** (with Docker Compose). The bot token is already in `.env`.

```bash
make env          # creates .env from .env.example, only if .env does not exist
```

Check that `.env` has `BOT_TOKEN=...`. Never share or commit this file.

## Start the bot

```bash
make up           # builds and starts the app, the bot and the HTTPS tunnel
make logs         # watch the log; press Ctrl+C to stop watching (the bot keeps running)
```

Wait about 10 seconds. The log should show:

```
bot: @ShopBotMVPbot started
Mini App: https://something.trycloudflare.com (from cloudflared)
```

Then open **@ShopBotMVPbot** in Telegram and send `/start`.

## Stop and restart

```bash
make down                  # stop everything
make down && make up       # restart (for example after changing .env or the code)
```

After every restart the tunnel gets a **new address**. Send `/start` in the bot again: buttons in old messages point to the old address.

## Check that it runs

```bash
docker compose ps                          # both "app" and "tunnel" must be "running"
docker compose logs app | grep "Mini App"  # the current public address
curl http://localhost:8080/api/health      # answers {"status":"ok"}
```

## Demo with two phones

| Who | What to do |
|---|---|
| Customer (phone 1) | `/start` → Map or a category → shop → add to cart → pay with Kaspi (demo) |
| Shop owner (phone 2) | send `/owner demo2026` → choose a shop (for example Nan & Co). New orders for this shop come to this chat |
| Projector (laptop) | open http://localhost:8080/owner and log in as the owner of the same shop. New orders appear live |

## Logins

| What | Address | Login |
|---|---|---|
| Mini App in a browser | http://localhost:8080/map | no login (demo user) |
| Owner dashboard | http://localhost:8080/owner | `owner@nan-and-co.kz` / `demo123` |
| Other shops | same page | `owner@<shop>.kz` / `demo123`, for example `owner@steppe-wear.kz` |
| Account inside any shop | sign-in window in the Mini App | `demo@shopbot.kz` / `demo123` |
| Founder metrics | http://localhost:8080/admin | password `admin2026` |

Shop names for logins: `nan-and-co`, `baursak-house`, `golden-crust`, `steppe-wear`, `nomad-denim`, `tumar-basics`,
`saryarka-outdoor`, `dala-market`, `fresh-corner`, `densaulyk-pharmacy`, `shipa-pharm`, `volt-electronics`, `smart-point`, `lale-flowers`.

## Bot commands

| Command | What it does |
|---|---|
| `/start` | Main menu |
| `/map` | Map of shops |
| `/search` | Find a product in all shops |
| `/orders` | My orders |
| `/owner demo2026` | Become the owner of a demo shop (orders come to this chat) |
| `/owner` | My shop: dashboard button, stop managing |
| `/lang` | English / Русский |

## Other useful commands

```bash
make reset        # delete the database; the next start creates fresh demo data
                  # (stop first: make down && make reset && make up)
make test         # run all tests (needs Go 1.26 and Node 22)
make bot-profile  # set the bot's avatar, description and commands again
                  # (only if you change them: every run uploads one more avatar)
make local        # run the app with Go on this computer, tunnel in Docker (needs Go 1.26 and Node 22)
make dev          # development without Telegram: http://localhost:5173 (needs Go and Node)
```

Without `make`, the same with Docker Compose:

```bash
docker compose up --build -d     # = make up
docker compose logs -f app       # = make logs
docker compose down              # = make down
```

## If something goes wrong

| Problem | Fix |
|---|---|
| Buttons in the bot do nothing | The tunnel address changed. Send `/start` again |
| Log says `409 Conflict` | The bot runs twice (for example `make up` and `make local`). Stop one: `make down` |
| No `Mini App:` line in the log | The tunnel is still starting. Wait 20 seconds and run `docker compose logs app \| grep "Mini App"` |
| Owner gets no order messages | On the owner phone send `/owner demo2026` and choose **the same shop** the customer buys from |
| "Open ShopBot from Telegram" in a browser | `DEV_AUTH=false` in `.env`. Set it to `true` and restart |
| Want clean demo numbers | `make down && make reset && make up` |
| Port 8080 is busy | Stop the other program, or change `"8080:8080"` in `docker-compose.yml` to `"8090:8080"` |
