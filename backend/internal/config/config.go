// Package config reads all settings from environment variables.
package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Config struct {
	Addr             string // HTTP listen address, e.g. ":8080"
	DBPath           string // SQLite file
	WebDir           string // built Mini App (webapp/dist)
	UploadDir        string // product photos uploaded by owners
	BotToken         string // from @BotFather; empty = run without the bot
	TunnelMetricsURL string // cloudflared metrics server, used to discover the public URL
	OwnerCode        string // code for /owner, so testers cannot take over shops
	AdminPassword    string // founder metrics page
	PaymentMode      string // "demo" or "link"
	DevAuth          bool   // allow a fake user when the app is opened outside Telegram
	VaultKey         []byte // 32 bytes, AES-256 key for saved shop passwords
	SessionSecret    []byte // HMAC key for owner/admin cookies

	publicURL   string
	publicURLMu sync.RWMutex
}

func Load() (*Config, error) {
	c := &Config{
		Addr:             env("ADDR", ":8080"),
		DBPath:           env("DB_PATH", "../data/shopbot.db"),
		WebDir:           env("WEB_DIR", "../webapp/dist"),
		BotToken:         os.Getenv("BOT_TOKEN"),
		TunnelMetricsURL: os.Getenv("TUNNEL_METRICS_URL"),
		OwnerCode:        env("OWNER_CODE", "demo2026"),
		AdminPassword:    env("ADMIN_PASSWORD", "admin2026"),
		PaymentMode:      env("PAYMENT_MODE", "demo"),
		DevAuth:          os.Getenv("DEV_AUTH") == "true",
	}
	c.SetPublicURL(os.Getenv("PUBLIC_URL"))

	dataDir := filepath.Dir(c.DBPath)
	c.UploadDir = filepath.Join(dataDir, "uploads")
	if err := os.MkdirAll(c.UploadDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	var err error
	if c.VaultKey, err = loadKey("VAULT_KEY", filepath.Join(dataDir, "vault.key")); err != nil {
		return nil, err
	}
	if c.SessionSecret, err = loadKey("SESSION_SECRET", filepath.Join(dataDir, "session.key")); err != nil {
		return nil, err
	}
	return c, nil
}

// PublicURL is the HTTPS address of the Mini App. It can change at runtime,
// because a quick tunnel gets a new address on every start.
func (c *Config) PublicURL() string {
	c.publicURLMu.RLock()
	defer c.publicURLMu.RUnlock()
	return c.publicURL
}

func (c *Config) SetPublicURL(u string) {
	c.publicURLMu.Lock()
	c.publicURL = strings.TrimRight(u, "/")
	c.publicURLMu.Unlock()
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// loadKey reads a base64 32-byte key from the environment. If it is not set,
// the key is read from (or created in) a file next to the database, so it
// survives restarts.
func loadKey(envName, path string) ([]byte, error) {
	if v := os.Getenv(envName); v != "" {
		key, err := base64.StdEncoding.DecodeString(v)
		if err != nil || len(key) != 32 {
			return nil, fmt.Errorf("%s must be 32 bytes in base64", envName)
		}
		return key, nil
	}
	if b, err := os.ReadFile(path); err == nil {
		key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
		if err == nil && len(key) == 32 {
			return key, nil
		}
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(key)), 0o600); err != nil {
		return nil, fmt.Errorf("save %s: %w", path, err)
	}
	return key, nil
}
