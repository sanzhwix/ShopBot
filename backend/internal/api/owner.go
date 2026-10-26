package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"shopbot/internal/orders"
	"shopbot/internal/store"
)

func (s *Server) handleOwnerLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &in); err != nil {
		writeError(w, 400, "bad JSON")
		return
	}
	owner, err := s.Store.OwnerByEmail(strings.ToLower(strings.TrimSpace(in.Email)))
	if err != nil || bcrypt.CompareHashAndPassword([]byte(owner.PasswordHash), []byte(in.Password)) != nil {
		writeJSON(w, 401, apiError{Error: "wrong email or password", Code: "BAD_CREDENTIALS"})
		return
	}
	s.setCookie(w, r, ownerCookie, s.Sessions.Sign("owner", owner.ID, sessionTTL), int(sessionTTL.Seconds()))
	writeJSON(w, 200, map[string]any{"owner": owner})
}

func (s *Server) handleOwnerLogout(w http.ResponseWriter, r *http.Request) {
	s.setCookie(w, r, ownerCookie, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleOwnerMe(w http.ResponseWriter, r *http.Request) {
	owner := ownerFrom(r)
	shop, err := s.Store.GetShop(owner.ShopID, lang(r))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"owner": owner, "shop": shop, "telegram_linked": shop.NotifyChatID != nil})
}

func (s *Server) handleOwnerAnalytics(w http.ResponseWriter, r *http.Request) {
	days := intQuery(r, "range")
	if days != 7 && days != 30 && days != 90 {
		days = 30
	}
	stats, err := s.Analytics.OwnerStats(ownerFrom(r).ShopID, days, lang(r), time.Now())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, 200, stats)
}

func (s *Server) handleOwnerOrders(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListShopOrders(ownerFrom(r).ShopID, r.URL.Query().Get("status"), 60, lang(r))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"orders": list})
}

func (s *Server) handleOwnerOrderStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status string `json:"status"`
	}
	if err := readJSON(r, &in); err != nil {
		writeError(w, 400, "bad JSON")
		return
	}
	order, err := s.Store.GetOrder(idParam(r, "id"), lang(r))
	if err == nil && order.ShopID != ownerFrom(r).ShopID {
		err = store.ErrNotFound
	}
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if _, err := s.Orders.SetStatus(r.Context(), order.ID, in.Status); err != nil {
		if errors.Is(err, orders.ErrBadStatus) {
			writeError(w, 400, err.Error())
			return
		}
		writeStoreError(w, err)
		return
	}
	order, _ = s.Store.GetOrder(order.ID, lang(r))
	writeJSON(w, 200, map[string]any{"order": order})
}

// handleOwnerStream pushes new and changed orders to the dashboard (Server-Sent Events).
func (s *Server) handleOwnerStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, 500, "streaming is not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	events, stop := s.Orders.Hub.Subscribe(ownerFrom(r).ShopID)
	defer stop()

	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()
	ping := time.NewTicker(20 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case e := <-events:
			b, _ := json.Marshal(e)
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) handleOwnerCatalogue(w http.ResponseWriter, r *http.Request) {
	shopID := ownerFrom(r).ShopID
	cats, err := s.Store.ShopCategories(shopID, lang(r))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	products, err := s.Store.ShopProducts(shopID, lang(r))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"categories": cats, "products": products})
}

func (s *Server) handleOwnerAddCategory(w http.ResponseWriter, r *http.Request) {
	var in struct {
		NameEn string `json:"name_en"`
		NameRu string `json:"name_ru"`
	}
	if err := readJSON(r, &in); err != nil || strings.TrimSpace(in.NameEn) == "" {
		writeError(w, 400, "name_en is required")
		return
	}
	if strings.TrimSpace(in.NameRu) == "" {
		in.NameRu = in.NameEn
	}
	id, err := s.Store.AddCategory(ownerFrom(r).ShopID, strings.TrimSpace(in.NameEn), strings.TrimSpace(in.NameRu))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, 201, map[string]int64{"id": id})
}

func (s *Server) handleOwnerAddProduct(w http.ResponseWriter, r *http.Request) {
	var in store.ProductInput
	if err := readJSON(r, &in); err != nil || strings.TrimSpace(in.NameEn) == "" || in.Price <= 0 {
		writeError(w, 400, "name_en and a positive price are required")
		return
	}
	if strings.TrimSpace(in.NameRu) == "" {
		in.NameRu = in.NameEn
	}
	if in.Image == "" {
		in.Image = "📦"
	}
	id, err := s.Store.AddProduct(ownerFrom(r).ShopID, in)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, 201, map[string]int64{"id": id})
}

func (s *Server) handleOwnerUpdateProduct(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Price   int  `json:"price"`
		InStock bool `json:"in_stock"`
	}
	if err := readJSON(r, &in); err != nil || in.Price <= 0 {
		writeError(w, 400, "price must be positive")
		return
	}
	if err := s.Store.UpdateProduct(ownerFrom(r).ShopID, idParam(r, "id"), in.Price, in.InStock); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleOwnerUpdateShop(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Hours       string `json:"hours"`
		KaspiPayURL string `json:"kaspi_pay_url"`
	}
	if err := readJSON(r, &in); err != nil {
		writeError(w, 400, "bad JSON")
		return
	}
	in.KaspiPayURL = strings.TrimSpace(in.KaspiPayURL)
	if in.KaspiPayURL != "" && !strings.HasPrefix(in.KaspiPayURL, "https://") {
		writeError(w, 400, "the Kaspi link must start with https://")
		return
	}
	if err := s.Store.UpdateShopSettings(ownerFrom(r).ShopID, strings.TrimSpace(in.Hours), in.KaspiPayURL); err != nil {
		writeStoreError(w, err)
		return
	}
	s.handleOwnerMe(w, r)
}

func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Password string `json:"password"`
	}
	if err := readJSON(r, &in); err != nil || in.Password != s.Cfg.AdminPassword {
		writeJSON(w, 401, apiError{Error: "wrong password", Code: "BAD_CREDENTIALS"})
		return
	}
	s.setCookie(w, r, adminCookie, s.Sessions.Sign("admin", 1, sessionTTL), int(sessionTTL.Seconds()))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminMetrics(w http.ResponseWriter, _ *http.Request) {
	stats, err := s.Analytics.FounderStats(time.Now())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, 200, stats)
}
