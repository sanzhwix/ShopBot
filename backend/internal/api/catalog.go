package api

import (
	"log"
	"net/http"
	"strings"

	"shopbot/internal/osm"
	"shopbot/internal/store"
)

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	cartCount, _ := s.Store.CartCount(u.ID)
	owner, err := s.Store.OwnerByTelegram(u.ID)
	var ownerShop any
	if err == nil {
		ownerShop = owner.ShopID
	}
	writeJSON(w, 200, map[string]any{"user": u, "cart_count": cartCount, "owner_shop_id": ownerShop})
}

func (s *Server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Lang  *string `json:"lang"`
		Phone *string `json:"phone"`
	}
	if err := readJSON(r, &in); err != nil {
		writeError(w, 400, "bad JSON")
		return
	}
	u := userFrom(r)
	if in.Lang != nil {
		_ = s.Store.SetUserLang(u.ID, *in.Lang)
	}
	if in.Phone != nil {
		_ = s.Store.SetUserPhone(u.ID, strings.TrimSpace(*in.Phone))
	}
	u, err := s.Store.GetUser(u.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"user": u})
}

func (s *Server) handleEvent(w http.ResponseWriter, r *http.Request) {
	var e store.Event
	if err := readJSON(r, &e); err != nil || !store.EventTypes[e.Type] {
		writeError(w, 400, "unknown event")
		return
	}
	if len(e.Meta) > 200 {
		e.Meta = e.Meta[:200]
	}
	if err := s.Store.AddEvent(userFrom(r).ID, e); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleFeedback(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}
	if err := readJSON(r, &in); err != nil || in.Rating < 1 || in.Rating > 5 {
		writeError(w, 400, "rating must be 1–5")
		return
	}
	if len(in.Comment) > 1000 {
		in.Comment = in.Comment[:1000]
	}
	if err := s.Store.AddFeedback(userFrom(r).ID, in.Rating, strings.TrimSpace(in.Comment)); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, 201, map[string]bool{"ok": true})
}

// handleMap returns partner malls and shops. ?cat=bakery shows only that category;
// ?cat=mall shows only malls.
func (s *Server) handleMap(w http.ResponseWriter, r *http.Request) {
	cat := r.URL.Query().Get("cat")
	l := lang(r)
	malls := []store.Mall{}
	shops := []store.Shop{}
	var err error
	if cat == "" || cat == "mall" {
		if malls, err = s.Store.ListMalls(l); err != nil {
			writeStoreError(w, err)
			return
		}
	}
	if cat != "mall" {
		if shops, err = s.Store.ListShops(l, cat, 0); err != nil {
			writeStoreError(w, err)
			return
		}
	}
	writeJSON(w, 200, map[string]any{"malls": malls, "shops": shops})
}

type osmPlace struct {
	osm.Place
	Invites int `json:"invites"`
}

// handleOSMNearby returns real OpenStreetMap shops near the user, minus places
// that are already ShopBot partners or partner malls (closer than 80 m).
func (s *Server) handleOSMNearby(w http.ResponseWriter, r *http.Request) {
	lat, lng := floatQuery(r, "lat"), floatQuery(r, "lng")
	if lat == 0 || lng == 0 {
		writeError(w, 400, "lat and lng are required")
		return
	}
	places, source, err := s.OSM.Nearby(r.Context(), lat, lng)
	if err != nil {
		log.Printf("osm: %v", err)
		writeJSON(w, 200, map[string]any{"places": []osmPlace{}, "error": "OpenStreetMap is not available right now"})
		return
	}
	malls, _ := s.Store.ListMalls("en")
	shops, _ := s.Store.ListShops("en", "", 0)
	var points [][2]float64
	for _, m := range malls {
		points = append(points, [2]float64{m.Lat, m.Lng})
	}
	for _, sh := range shops {
		points = append(points, [2]float64{sh.Lat, sh.Lng})
	}

	out := []osmPlace{}
	for _, p := range places {
		isPartner := false
		for _, pt := range points {
			if osm.DistanceM(p.Lat, p.Lng, pt[0], pt[1]) < 80 {
				isPartner = true
				break
			}
		}
		if !isPartner {
			n, _ := s.Store.InviteCount(p.ID)
			out = append(out, osmPlace{Place: p, Invites: n})
		}
	}
	writeJSON(w, 200, map[string]any{"places": out, "source": source})
}

func (s *Server) handleInvite(w http.ResponseWriter, r *http.Request) {
	var in store.Invite
	if err := readJSON(r, &in); err != nil || in.OsmID == "" || in.Name == "" {
		writeError(w, 400, "osm_id and name are required")
		return
	}
	u := userFrom(r)
	if err := s.Store.AddInvite(u.ID, in); err != nil {
		writeStoreError(w, err)
		return
	}
	_ = s.Store.AddEvent(u.ID, store.Event{Type: "invite", Meta: in.Name})
	n, _ := s.Store.InviteCount(in.OsmID)
	writeJSON(w, 201, map[string]int{"invites": n})
}

func (s *Server) handleMall(w http.ResponseWriter, r *http.Request) {
	l := lang(r)
	mall, err := s.Store.GetMall(idParam(r, "id"), l)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	shops, err := s.Store.ListShops(l, "", mall.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"mall": mall, "shops": shops})
}

type shopSession struct {
	SignedIn   bool   `json:"signed_in"`
	Login      string `json:"login"`
	HasSaved   bool   `json:"has_saved_account"`
	SavedLogin string `json:"saved_login"`
	ShopID     int64  `json:"shop_id"`
	ShopName   string `json:"shop_name"`
}

func (s *Server) shopSessionInfo(userID int64, shop store.Shop) shopSession {
	info := shopSession{ShopID: shop.ID, ShopName: shop.Name}
	if acc, err := s.Store.ShopSession(userID, shop.ID); err == nil {
		info.SignedIn, info.Login = true, acc.Login
	}
	if v, err := s.Store.GetVaultEntry(userID, shop.ID); err == nil {
		info.HasSaved, info.SavedLogin = true, v.Login
	}
	return info
}

func (s *Server) handleShop(w http.ResponseWriter, r *http.Request) {
	l := lang(r)
	shop, err := s.Store.GetShop(idParam(r, "id"), l)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	cats, err := s.Store.ShopCategories(shop.ID, l)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	products, err := s.Store.ShopProducts(shop.ID, l)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{
		"shop":       shop,
		"categories": cats,
		"products":   products,
		"session":    s.shopSessionInfo(userFrom(r).ID, shop),
	})
}

func (s *Server) handleProduct(w http.ResponseWriter, r *http.Request) {
	l := lang(r)
	p, err := s.Store.GetProduct(idParam(r, "id"), l)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	shop, err := s.Store.GetShop(p.ShopID, l)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"product": p, "shop": shop})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := store.SearchParams{
		Query:    q.Get("q"),
		Category: q.Get("cat"),
		Size:     q.Get("size"),
		Material: q.Get("material"),
		MinPrice: intQuery(r, "min"),
		MaxPrice: intQuery(r, "max"),
	}
	hits, err := s.Store.Search(lang(r), params)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if params.Query != "" {
		_ = s.Store.AddEvent(userFrom(r).ID, store.Event{Type: "search", Meta: strings.TrimSpace(params.Query)})
	}
	writeJSON(w, 200, map[string]any{"results": hits})
}
