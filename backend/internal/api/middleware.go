package api

import (
	"context"
	"net/http"
	"time"

	"shopbot/internal/auth"
	"shopbot/internal/store"
)

type ctxKey int

const (
	userKey ctxKey = iota
	ownerKey
)

const (
	ownerCookie = "sb_owner"
	adminCookie = "sb_admin"
	sessionTTL  = 7 * 24 * time.Hour
)

// devUserID is the fake shopper used when DEV_AUTH=true and the app runs outside Telegram.
const devUserID = 1000001

func userFrom(r *http.Request) store.User { return r.Context().Value(userKey).(store.User) }

func ownerFrom(r *http.Request) store.Owner { return r.Context().Value(ownerKey).(store.Owner) }

// lang is the language for this request: the X-Lang header (the Mini App's
// current choice), or the user's saved language.
func lang(r *http.Request) string {
	if l := r.Header.Get("X-Lang"); l != "" {
		return store.Lang(l)
	}
	if u, ok := r.Context().Value(userKey).(store.User); ok {
		return u.Lang
	}
	return "en"
}

func langFromTelegram(code string) string {
	if code == "ru" || code == "kk" {
		return "ru"
	}
	return "en"
}

// telegramUser reads and checks the Telegram initData of the request.
// EventSource cannot send headers, so the owner stream passes it as ?init=.
func (s *Server) telegramUser(r *http.Request) (auth.TelegramUser, bool) {
	initData := r.Header.Get("X-Telegram-Init-Data")
	if initData == "" {
		initData = r.URL.Query().Get("init")
	}
	if initData == "" || s.Cfg.BotToken == "" {
		return auth.TelegramUser{}, false
	}
	u, err := auth.ValidateInitData(initData, s.Cfg.BotToken, 24*time.Hour, time.Now())
	return u, err == nil
}

func (s *Server) requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tu, ok := s.telegramUser(r)
		if !ok {
			// The demo user is only for a plain browser. A request that carries
			// Telegram data with a bad signature is always refused.
			hasInitData := r.Header.Get("X-Telegram-Init-Data") != "" || r.URL.Query().Get("init") != ""
			if !s.Cfg.DevAuth || hasInitData {
				writeJSON(w, 401, apiError{Error: "open ShopBot from Telegram", Code: "TELEGRAM_REQUIRED"})
				return
			}
			tu = auth.TelegramUser{ID: devUserID, FirstName: "Demo", LastName: "Shopper", Username: "demo_shopper", LanguageCode: r.Header.Get("X-Lang")}
		}
		u, err := s.Store.UpsertUser(store.User{ID: tu.ID, FirstName: tu.FirstName, LastName: tu.LastName, Username: tu.Username, Lang: langFromTelegram(tu.LanguageCode)})
		if err != nil {
			writeStoreError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, u)))
	})
}

// requireOwner accepts the dashboard cookie (browser login) or Telegram
// initData of a user who linked a shop with /owner.
func (s *Server) requireOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var owner store.Owner
		var err error = store.ErrNotFound
		if c, cerr := r.Cookie(ownerCookie); cerr == nil {
			if id, verr := s.Sessions.Verify(c.Value, "owner"); verr == nil {
				owner, err = s.Store.OwnerByID(id)
			}
		}
		if err != nil {
			if tu, ok := s.telegramUser(r); ok {
				owner, err = s.Store.OwnerByTelegram(tu.ID)
			}
		}
		if err != nil {
			writeJSON(w, 401, apiError{Error: "owner login required", Code: "OWNER_LOGIN_REQUIRED"})
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ownerKey, owner)))
	})
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(adminCookie)
		if err != nil {
			writeJSON(w, 401, apiError{Error: "admin login required", Code: "ADMIN_LOGIN_REQUIRED"})
			return
		}
		if _, err := s.Sessions.Verify(c.Value, "admin"); err != nil {
			writeJSON(w, 401, apiError{Error: "admin login required", Code: "ADMIN_LOGIN_REQUIRED"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) setCookie(w http.ResponseWriter, r *http.Request, name, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		SameSite: http.SameSiteLaxMode,
	})
}
