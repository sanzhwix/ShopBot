package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"shopbot/internal/store"
)

// Each partner shop has its own customer accounts ("the shop's website").
// Adding to cart needs a session at that shop. ShopBot can save the login
// in the user's vault and sign in again automatically next time.

type shopLoginInput struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Save     bool   `json:"save"`
}

func (s *Server) shopForAuth(w http.ResponseWriter, r *http.Request) (store.Shop, bool) {
	shop, err := s.Store.GetShop(idParam(r, "id"), lang(r))
	if err != nil {
		writeStoreError(w, err)
		return shop, false
	}
	return shop, true
}

// signIn starts the session and, if asked, saves the login in the vault.
func (s *Server) signIn(w http.ResponseWriter, userID int64, shop store.Shop, acc store.ShopAccount, password string, save bool) {
	if err := s.Store.StartShopSession(userID, shop.ID, acc.ID); err != nil {
		writeStoreError(w, err)
		return
	}
	if save {
		secret, err := s.Vault.Encrypt(password)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if err := s.Store.SaveVaultEntry(userID, shop.ID, acc.Login, secret); err != nil {
			writeStoreError(w, err)
			return
		}
	}
	writeJSON(w, 200, s.shopSessionInfo(userID, shop))
}

func (s *Server) handleShopLogin(w http.ResponseWriter, r *http.Request) {
	shop, ok := s.shopForAuth(w, r)
	if !ok {
		return
	}
	var in shopLoginInput
	if err := readJSON(r, &in); err != nil {
		writeError(w, 400, "bad JSON")
		return
	}
	in.Login = strings.ToLower(strings.TrimSpace(in.Login))
	acc, err := s.Store.FindShopAccount(shop.ID, in.Login)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(in.Password)) != nil {
		writeJSON(w, 401, apiError{Error: "wrong login or password", Code: "BAD_CREDENTIALS"})
		return
	}
	s.signIn(w, userFrom(r).ID, shop, acc, in.Password, in.Save)
}

func (s *Server) handleShopRegister(w http.ResponseWriter, r *http.Request) {
	shop, ok := s.shopForAuth(w, r)
	if !ok {
		return
	}
	var in shopLoginInput
	if err := readJSON(r, &in); err != nil {
		writeError(w, 400, "bad JSON")
		return
	}
	in.Login = strings.ToLower(strings.TrimSpace(in.Login))
	if !strings.Contains(in.Login, "@") && !strings.HasPrefix(in.Login, "+") {
		writeJSON(w, 400, apiError{Error: "use an email or a phone number", Code: "BAD_LOGIN"})
		return
	}
	if len(in.Password) < 6 {
		writeJSON(w, 400, apiError{Error: "password must have at least 6 characters", Code: "WEAK_PASSWORD"})
		return
	}
	if _, err := s.Store.FindShopAccount(shop.ID, in.Login); err == nil {
		writeJSON(w, 409, apiError{Error: "this account already exists, please sign in", Code: "ACCOUNT_EXISTS"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	u := userFrom(r)
	acc, err := s.Store.CreateShopAccount(shop.ID, in.Login, string(hash), strings.TrimSpace(in.Name+" "), nil)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	s.signIn(w, u.ID, shop, acc, in.Password, in.Save)
}

// handleShopTelegramLogin is "Continue with Telegram": one tap creates (or reuses)
// a shop account tied to the Telegram id. It gets a random password that is
// stored in the vault, so later sign-ins go through the same vault path.
func (s *Server) handleShopTelegramLogin(w http.ResponseWriter, r *http.Request) {
	shop, ok := s.shopForAuth(w, r)
	if !ok {
		return
	}
	u := userFrom(r)
	password := randomPassword()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	acc, err := s.Store.FindShopAccountByTelegram(shop.ID, u.ID)
	if errors.Is(err, store.ErrNotFound) {
		login := fmt.Sprintf("tg:%d", u.ID)
		if u.Username != "" {
			login = "@" + strings.ToLower(u.Username)
		}
		tgID := u.ID
		acc, err = s.Store.CreateShopAccount(shop.ID, login, string(hash), strings.TrimSpace(u.FirstName+" "+u.LastName), &tgID)
	} else if err == nil {
		err = s.Store.SetShopAccountPassword(acc.ID, string(hash))
	}
	if err != nil {
		writeStoreError(w, err)
		return
	}
	s.signIn(w, u.ID, shop, acc, password, true)
}

// handleShopVaultLogin signs in with the saved login, exactly like a password
// manager: decrypt, then check the password against the shop account.
func (s *Server) handleShopVaultLogin(w http.ResponseWriter, r *http.Request) {
	shop, ok := s.shopForAuth(w, r)
	if !ok {
		return
	}
	u := userFrom(r)
	entry, err := s.Store.GetVaultEntry(u.ID, shop.ID)
	if err != nil {
		writeJSON(w, 404, apiError{Error: "no saved account for this shop", Code: "NO_SAVED_ACCOUNT"})
		return
	}
	password, err := s.Vault.Decrypt(entry.Secret)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	acc, err := s.Store.FindShopAccount(shop.ID, entry.Login)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(password)) != nil {
		writeJSON(w, 401, apiError{Error: "the saved password does not work any more", Code: "VAULT_STALE"})
		return
	}
	_ = s.Store.TouchVaultEntry(entry.ID)
	s.signIn(w, u.ID, shop, acc, password, false)
}

func (s *Server) handleShopLogout(w http.ResponseWriter, r *http.Request) {
	shop, ok := s.shopForAuth(w, r)
	if !ok {
		return
	}
	u := userFrom(r)
	if err := s.Store.EndShopSession(u.ID, shop.ID); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, 200, s.shopSessionInfo(u.ID, shop))
}

func (s *Server) handleVaultList(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListVault(userFrom(r).ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"accounts": list})
}

func (s *Server) handleVaultDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.DeleteVaultEntry(userFrom(r).ID, idParam(r, "id")); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func randomPassword() string {
	b := make([]byte, 18)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
