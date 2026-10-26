package store

import (
	"database/sql"
	"time"
)

// ShopAccount is a customer's account at one shop ("the shop's website").
type ShopAccount struct {
	ID           int64  `json:"id"`
	ShopID       int64  `json:"shop_id"`
	Login        string `json:"login"`
	Name         string `json:"name"`
	PasswordHash string `json:"-"`
}

const ShopSessionTTL = 30 * 24 * time.Hour

func (s *Store) FindShopAccount(shopID int64, login string) (ShopAccount, error) {
	var a ShopAccount
	err := s.DB.QueryRow(`SELECT id, shop_id, login, name, password_hash FROM shop_accounts WHERE shop_id = ? AND login = ?`,
		shopID, login).Scan(&a.ID, &a.ShopID, &a.Login, &a.Name, &a.PasswordHash)
	return a, notFound(err)
}

func (s *Store) FindShopAccountByTelegram(shopID, telegramID int64) (ShopAccount, error) {
	var a ShopAccount
	err := s.DB.QueryRow(`SELECT id, shop_id, login, name, password_hash FROM shop_accounts WHERE shop_id = ? AND telegram_id = ?`,
		shopID, telegramID).Scan(&a.ID, &a.ShopID, &a.Login, &a.Name, &a.PasswordHash)
	return a, notFound(err)
}

func (s *Store) CreateShopAccount(shopID int64, login, passwordHash, name string, telegramID *int64) (ShopAccount, error) {
	res, err := s.DB.Exec(`INSERT INTO shop_accounts (shop_id, login, password_hash, name, telegram_id, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		shopID, login, passwordHash, name, telegramID, now())
	if err != nil {
		return ShopAccount{}, err
	}
	id, _ := res.LastInsertId()
	return ShopAccount{ID: id, ShopID: shopID, Login: login, Name: name, PasswordHash: passwordHash}, nil
}

func (s *Store) SetShopAccountPassword(id int64, passwordHash string) error {
	_, err := s.DB.Exec(`UPDATE shop_accounts SET password_hash = ? WHERE id = ?`, passwordHash, id)
	return err
}

// ShopSession returns the account the user is signed in with at this shop.
func (s *Store) ShopSession(userID, shopID int64) (ShopAccount, error) {
	var a ShopAccount
	err := s.DB.QueryRow(`
		SELECT a.id, a.shop_id, a.login, a.name, a.password_hash
		FROM shop_sessions ss JOIN shop_accounts a ON a.id = ss.account_id
		WHERE ss.user_id = ? AND ss.shop_id = ? AND ss.expires_at > ?`, userID, shopID, now()).
		Scan(&a.ID, &a.ShopID, &a.Login, &a.Name, &a.PasswordHash)
	return a, notFound(err)
}

func (s *Store) StartShopSession(userID, shopID, accountID int64) error {
	_, err := s.DB.Exec(`
		INSERT INTO shop_sessions (user_id, shop_id, account_id, expires_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id, shop_id) DO UPDATE SET account_id = excluded.account_id, expires_at = excluded.expires_at`,
		userID, shopID, accountID, time.Now().Add(ShopSessionTTL).Unix())
	return err
}

func (s *Store) EndShopSession(userID, shopID int64) error {
	_, err := s.DB.Exec(`DELETE FROM shop_sessions WHERE user_id = ? AND shop_id = ?`, userID, shopID)
	return err
}

// VaultEntry is one saved shop login. The secret never leaves the server.
type VaultEntry struct {
	ID         int64  `json:"id"`
	ShopID     int64  `json:"shop_id"`
	ShopName   string `json:"shop_name"`
	ShopLogo   string `json:"shop_logo"`
	Category   string `json:"category"`
	Login      string `json:"login"`
	CreatedAt  int64  `json:"created_at"`
	LastUsedAt *int64 `json:"last_used_at"`
	Signed     bool   `json:"signed_in"`
	Secret     []byte `json:"-"`
}

func (s *Store) SaveVaultEntry(userID, shopID int64, login string, secret []byte) error {
	_, err := s.DB.Exec(`
		INSERT INTO vault_entries (user_id, shop_id, login, secret, created_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(user_id, shop_id) DO UPDATE SET login = excluded.login, secret = excluded.secret`,
		userID, shopID, login, secret, now())
	return err
}

func (s *Store) GetVaultEntry(userID, shopID int64) (VaultEntry, error) {
	var v VaultEntry
	err := s.DB.QueryRow(`SELECT id, shop_id, login, secret FROM vault_entries WHERE user_id = ? AND shop_id = ?`, userID, shopID).
		Scan(&v.ID, &v.ShopID, &v.Login, &v.Secret)
	return v, notFound(err)
}

func (s *Store) TouchVaultEntry(id int64) error {
	_, err := s.DB.Exec(`UPDATE vault_entries SET last_used_at = ? WHERE id = ?`, now(), id)
	return err
}

func (s *Store) ListVault(userID int64) ([]VaultEntry, error) {
	rows, err := s.DB.Query(`
		SELECT v.id, v.shop_id, sh.name, sh.logo, sh.category, v.login, v.created_at, v.last_used_at,
		       EXISTS (SELECT 1 FROM shop_sessions ss WHERE ss.user_id = v.user_id AND ss.shop_id = v.shop_id AND ss.expires_at > ?)
		FROM vault_entries v JOIN shops sh ON sh.id = v.shop_id
		WHERE v.user_id = ? ORDER BY COALESCE(v.last_used_at, v.created_at) DESC`, now(), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []VaultEntry{}
	for rows.Next() {
		var v VaultEntry
		var last sql.NullInt64
		if err := rows.Scan(&v.ID, &v.ShopID, &v.ShopName, &v.ShopLogo, &v.Category, &v.Login, &v.CreatedAt, &last, &v.Signed); err != nil {
			return nil, err
		}
		if last.Valid {
			v.LastUsedAt = &last.Int64
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) DeleteVaultEntry(userID, id int64) error {
	res, err := s.DB.Exec(`DELETE FROM vault_entries WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Owner is a shop owner who can open the dashboard.
type Owner struct {
	ID           int64  `json:"id"`
	ShopID       int64  `json:"shop_id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	PasswordHash string `json:"-"`
	TelegramID   *int64 `json:"telegram_id"`
}

func (s *Store) ownerWhere(where string, arg any) (Owner, error) {
	var o Owner
	var tg sql.NullInt64
	err := s.DB.QueryRow(`SELECT id, shop_id, email, name, password_hash, telegram_id FROM shop_owners WHERE `+where, arg).
		Scan(&o.ID, &o.ShopID, &o.Email, &o.Name, &o.PasswordHash, &tg)
	if tg.Valid {
		o.TelegramID = &tg.Int64
	}
	return o, notFound(err)
}

func (s *Store) OwnerByEmail(email string) (Owner, error) { return s.ownerWhere("email = ?", email) }
func (s *Store) OwnerByID(id int64) (Owner, error)        { return s.ownerWhere("id = ?", id) }
func (s *Store) OwnerByTelegram(tgID int64) (Owner, error) {
	return s.ownerWhere("telegram_id = ?", tgID)
}
func (s *Store) OwnerByShop(shopID int64) (Owner, error) { return s.ownerWhere("shop_id = ?", shopID) }

// LinkOwnerTelegram makes this Telegram user the owner of the shop and sends
// the shop's orders to chatID. One Telegram user owns one demo shop at a time.
func (s *Store) LinkOwnerTelegram(shopID, telegramID, chatID int64) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE shops SET notify_chat_id = NULL WHERE id IN (SELECT shop_id FROM shop_owners WHERE telegram_id = ?)`, telegramID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE shop_owners SET telegram_id = NULL WHERE telegram_id = ?`, telegramID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE shop_owners SET telegram_id = ? WHERE shop_id = ?`, telegramID, shopID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE shops SET notify_chat_id = ? WHERE id = ?`, chatID, shopID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) UnlinkOwnerTelegram(telegramID int64) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE shops SET notify_chat_id = NULL WHERE id IN (SELECT shop_id FROM shop_owners WHERE telegram_id = ?)`, telegramID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE shop_owners SET telegram_id = NULL WHERE telegram_id = ?`, telegramID); err != nil {
		return err
	}
	return tx.Commit()
}
