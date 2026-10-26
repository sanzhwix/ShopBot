// Package auth checks who is calling the API.
//
// Shoppers: the Mini App sends Telegram "initData". Telegram signs it with a key
// made from our bot token, so a valid signature proves the user id is real.
// Owners and the admin: a signed cookie after password login.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNoHash  = errors.New("initData has no hash")
	ErrBadHash = errors.New("initData signature is wrong")
	ErrExpired = errors.New("initData is too old")
	ErrNoUser  = errors.New("initData has no user")
)

type TelegramUser struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

// ValidateInitData follows https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app
//
//  1. data_check_string = all fields except "hash", sorted by key, as "key=value" joined by "\n".
//  2. secret_key = HMAC_SHA256(key "WebAppData", message bot_token).
//  3. The hash must equal hex(HMAC_SHA256(key secret_key, message data_check_string)).
func ValidateInitData(initData, botToken string, maxAge time.Duration, now time.Time) (TelegramUser, error) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return TelegramUser{}, err
	}
	hash := values.Get("hash")
	if hash == "" {
		return TelegramUser{}, ErrNoHash
	}

	keys := make([]string, 0, len(values))
	for k := range values {
		if k != "hash" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	pairs := make([]string, len(keys))
	for i, k := range keys {
		pairs[i] = k + "=" + values.Get(k)
	}
	dataCheck := strings.Join(pairs, "\n")

	secret := hmacSHA256([]byte("WebAppData"), []byte(botToken))
	expected := hex.EncodeToString(hmacSHA256(secret, []byte(dataCheck)))
	if !hmac.Equal([]byte(expected), []byte(hash)) {
		return TelegramUser{}, ErrBadHash
	}

	authDate, _ := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if maxAge > 0 && now.Sub(time.Unix(authDate, 0)) > maxAge {
		return TelegramUser{}, ErrExpired
	}

	var u TelegramUser
	if err := json.Unmarshal([]byte(values.Get("user")), &u); err != nil || u.ID == 0 {
		return TelegramUser{}, ErrNoUser
	}
	return u, nil
}

// SignInitData builds a valid initData string. Tests use it.
func SignInitData(values url.Values, botToken string) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, len(keys))
	for i, k := range keys {
		pairs[i] = k + "=" + values.Get(k)
	}
	secret := hmacSHA256([]byte("WebAppData"), []byte(botToken))
	values.Set("hash", hex.EncodeToString(hmacSHA256(secret, []byte(strings.Join(pairs, "\n")))))
	return values.Encode()
}

func hmacSHA256(key, msg []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(msg)
	return m.Sum(nil)
}
