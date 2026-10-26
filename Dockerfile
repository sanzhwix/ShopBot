# 1. Build the Telegram Mini App (React).
FROM node:22-alpine AS web
WORKDIR /web
COPY webapp/package.json webapp/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY webapp/ ./
RUN npm run build

# 2. Build the Go server (API + bot). Pure Go SQLite, so no C compiler is needed.
FROM golang:1.26-alpine AS api
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /shopbot ./cmd/shopbot

# 3. Small runtime image.
FROM alpine:3.22
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=api /shopbot /app/shopbot
COPY --from=web /web/dist /app/web
ENV ADDR=:8080 DB_PATH=/app/data/shopbot.db WEB_DIR=/app/web
VOLUME /app/data
EXPOSE 8080
CMD ["/app/shopbot"]
