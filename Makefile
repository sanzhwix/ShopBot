.PHONY: env up down logs local dev test build reset bot-profile

# Create .env from the example (does not overwrite).
env:
	@test -f .env || (cp .env.example .env && echo "Created .env — put your BOT_TOKEN in it")

# Everything in Docker: app + HTTPS tunnel. Open the bot in Telegram and send /start.
up: env
	docker compose up --build -d
	@echo "Waiting for the tunnel address..."
	@sleep 8; docker compose logs app | grep -E "Mini App|bot:" | tail -3 || true

down:
	docker compose down

# Fill in the bot's Telegram profile once: name, descriptions, commands (EN + RU) and the avatar.
bot-profile: env
	set -a; . ./.env; set +a; cd backend && go run ./cmd/botsetup -photo ../assets/bot/avatar.jpg

logs:
	docker compose logs -f app

# App with Go on this machine + tunnel in Docker (needs Go 1.26 and Node 22).
local: env
	./scripts/local.sh

# Development without Telegram: Go API on :8080 and Vite on http://localhost:5173 (fake user).
dev:
	@echo "Open http://localhost:5173 — owner dashboard: http://localhost:5173/owner"
	(cd backend && DEV_AUTH=true DB_PATH=../data/shopbot.db go run ./cmd/shopbot) & \
	(cd webapp && npm run dev); kill %1

test:
	cd backend && go vet ./... && go test ./...
	cd webapp && npm run typecheck

build:
	cd webapp && npm run build
	cd backend && CGO_ENABLED=0 go build -o ../shopbot ./cmd/shopbot

# Delete the database (keys are kept). The next start seeds fresh demo data.
reset:
	rm -f data/shopbot.db data/shopbot.db-wal data/shopbot.db-shm
