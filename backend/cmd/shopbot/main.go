// ShopBot MVP: one binary with the HTTP API, the Mini App files and the Telegram bot.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"shopbot/internal/analytics"
	"shopbot/internal/api"
	"shopbot/internal/auth"
	"shopbot/internal/bot"
	"shopbot/internal/config"
	"shopbot/internal/db"
	"shopbot/internal/orders"
	"shopbot/internal/osm"
	"shopbot/internal/payment"
	"shopbot/internal/seed"
	"shopbot/internal/store"
	"shopbot/internal/vault"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer conn.Close()
	if err := seed.Run(conn); err != nil {
		log.Fatalf("seed: %v", err)
	}

	st := store.New(conn)
	svc := orders.New(st)
	v, err := vault.New(cfg.VaultKey)
	if err != nil {
		log.Fatalf("vault: %v", err)
	}
	srv := &api.Server{
		Cfg:       cfg,
		Store:     st,
		Orders:    svc,
		Analytics: &analytics.Service{DB: conn},
		Vault:     v,
		Payment:   payment.New(cfg.PaymentMode),
		OSM:       osm.New(st),
		Sessions:  auth.Sessions{Secret: cfg.SessionSecret},
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var tgBot *bot.Bot
	if cfg.BotToken != "" {
		tgBot, err = bot.New(cfg, st, svc)
		if err != nil {
			log.Fatalf("telegram bot: %v (check BOT_TOKEN)", err)
		}
		svc.SetNotifier(tgBot)
		srv.BotUsername = tgBot.Username
		go tgBot.Start(ctx)
		go tgBot.SetupMenu(ctx)
	} else {
		log.Print("BOT_TOKEN is empty: running without the Telegram bot (browser mode)")
	}

	if cfg.PublicURL() == "" && cfg.TunnelMetricsURL != "" {
		go discoverTunnel(ctx, cfg, func() {
			if tgBot != nil {
				tgBot.SetupMenu(ctx)
			}
		})
	}

	httpSrv := &http.Server{Addr: cfg.Addr, Handler: srv.Router(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdown)
	}()

	log.Printf("http: listening on %s (payment mode: %s, dev auth: %v)", cfg.Addr, cfg.PaymentMode, cfg.DevAuth)
	if u := cfg.PublicURL(); u != "" {
		log.Printf("Mini App: %s", u)
	}
	if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("http: %v", err)
	}
}

// discoverTunnel asks cloudflared for its quick-tunnel address. Every start
// of a quick tunnel gets a new random https://*.trycloudflare.com address.
func discoverTunnel(ctx context.Context, cfg *config.Config, onFound func()) {
	client := &http.Client{Timeout: 3 * time.Second}
	for i := 0; i < 90; i++ {
		resp, err := client.Get(cfg.TunnelMetricsURL + "/quicktunnel")
		if err == nil {
			var body struct {
				Hostname string `json:"hostname"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&body)
			resp.Body.Close()
			if body.Hostname != "" {
				cfg.SetPublicURL("https://" + body.Hostname)
				log.Printf("Mini App: %s (from cloudflared)", cfg.PublicURL())
				onFound()
				return
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
	log.Print("tunnel: could not get the public URL from cloudflared; set PUBLIC_URL by hand")
}
