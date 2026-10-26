// Package api is the HTTP API of the Mini App and the owner dashboard.
// It also serves the built Mini App files.
package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"shopbot/internal/analytics"
	"shopbot/internal/auth"
	"shopbot/internal/config"
	"shopbot/internal/orders"
	"shopbot/internal/osm"
	"shopbot/internal/payment"
	"shopbot/internal/store"
	"shopbot/internal/vault"
)

type Server struct {
	Cfg       *config.Config
	Store     *store.Store
	Orders    *orders.Service
	Analytics *analytics.Service
	Vault     *vault.Vault
	Payment   payment.Provider
	OSM       *osm.Client
	Sessions  auth.Sessions
	// BotUsername is set when the bot starts; the Mini App uses it for "back to bot" links.
	BotUsername func() string
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer, logRequests)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
		r.Get("/config", s.handleConfig)

		r.Group(func(r chi.Router) {
			r.Use(s.requireUser)
			r.Get("/me", s.handleMe)
			r.Patch("/me", s.handleUpdateMe)
			r.Post("/events", s.handleEvent)
			r.Post("/feedback", s.handleFeedback)

			r.Get("/map", s.handleMap)
			r.Get("/osm/nearby", s.handleOSMNearby)
			r.Post("/invites", s.handleInvite)
			r.Get("/malls/{id}", s.handleMall)
			r.Get("/shops/{id}", s.handleShop)
			r.Get("/products/{id}", s.handleProduct)
			r.Get("/search", s.handleSearch)

			r.Post("/shops/{id}/login", s.handleShopLogin)
			r.Post("/shops/{id}/register", s.handleShopRegister)
			r.Post("/shops/{id}/login/telegram", s.handleShopTelegramLogin)
			r.Post("/shops/{id}/login/vault", s.handleShopVaultLogin)
			r.Post("/shops/{id}/logout", s.handleShopLogout)
			r.Get("/vault", s.handleVaultList)
			r.Delete("/vault/{id}", s.handleVaultDelete)

			r.Get("/cart", s.handleCart)
			r.Post("/cart", s.handleAddToCart)
			r.Patch("/cart/{id}", s.handleCartQty)
			r.Post("/checkout", s.handleCheckout)
			r.Get("/orders", s.handleOrders)
			r.Get("/orders/{id}", s.handleOrder)
			r.Get("/payments/{id}", s.handlePayment)
			r.Post("/payments/{id}/confirm", s.handlePaymentConfirm)
		})

		r.Post("/owner/login", s.handleOwnerLogin)
		r.Post("/owner/logout", s.handleOwnerLogout)
		r.Group(func(r chi.Router) {
			r.Use(s.requireOwner)
			r.Get("/owner/me", s.handleOwnerMe)
			r.Get("/owner/analytics", s.handleOwnerAnalytics)
			r.Get("/owner/orders", s.handleOwnerOrders)
			r.Patch("/owner/orders/{id}", s.handleOwnerOrderStatus)
			r.Get("/owner/stream", s.handleOwnerStream)
			r.Get("/owner/catalogue", s.handleOwnerCatalogue)
			r.Post("/owner/categories", s.handleOwnerAddCategory)
			r.Post("/owner/products", s.handleOwnerAddProduct)
			r.Patch("/owner/products/{id}", s.handleOwnerUpdateProduct)
			r.Post("/owner/products/{id}/photo", s.handleOwnerProductPhoto)
			r.Patch("/owner/shop", s.handleOwnerUpdateShop)
		})

		r.Post("/admin/login", s.handleAdminLogin)
		r.With(s.requireAdmin).Get("/admin/metrics", s.handleAdminMetrics)
	})

	r.Get("/uploads/*", s.serveUpload)
	r.NotFound(s.serveWebApp)
	return r
}

func (s *Server) handleConfig(w http.ResponseWriter, _ *http.Request) {
	username := ""
	if s.BotUsername != nil {
		username = s.BotUsername()
	}
	writeJSON(w, 200, map[string]any{
		"bot_username":  username,
		"payment_mode":  s.Cfg.PaymentMode,
		"demo_login":    "demo@shopbot.kz",
		"demo_password": "demo123",
	})
}

// serveWebApp serves the built Mini App. Unknown paths get index.html,
// because the app has its own routes (/map, /shop/3, /owner ...).
func (s *Server) serveWebApp(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeError(w, 404, "not found")
		return
	}
	path := filepath.Join(s.Cfg.WebDir, filepath.Clean("/"+r.URL.Path))
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		http.ServeFile(w, r, path)
		return
	}
	index := filepath.Join(s.Cfg.WebDir, "index.html")
	if _, err := os.Stat(index); err != nil {
		http.Error(w, "Mini App is not built. Run: cd webapp && npm run build", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, index)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/owner/stream" {
			log.Printf("%s %s %d %s", r.Method, r.URL.Path, ww.Status(), time.Since(start).Round(time.Millisecond))
		}
	})
}

// --- helpers ---

var errNotYours = errors.New("belongs to another shop")

type apiError struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, apiError{Error: msg})
}

// writeStoreError maps store errors to HTTP codes. Unknown errors are logged, not shown.
func writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, 404, "not found")
		return
	}
	log.Printf("internal error: %v", err)
	writeError(w, 500, "internal error")
}

func readJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	return dec.Decode(v)
}

func idParam(r *http.Request, name string) int64 {
	id, _ := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	return id
}

func intQuery(r *http.Request, name string) int {
	n, _ := strconv.Atoi(r.URL.Query().Get(name))
	return n
}

func floatQuery(r *http.Request, name string) float64 {
	f, _ := strconv.ParseFloat(r.URL.Query().Get(name), 64)
	return f
}
