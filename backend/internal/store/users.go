package store

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Lang      string `json:"lang"`
	Phone     string `json:"phone"`
}

// UpsertUser creates the user on first visit and refreshes the profile later.
// The language is only set on insert, so a user's own choice is kept.
func (s *Store) UpsertUser(u User) (User, error) {
	t := now()
	_, err := s.DB.Exec(`
		INSERT INTO users (id, first_name, last_name, username, lang, created_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			first_name = excluded.first_name,
			last_name = excluded.last_name,
			username = excluded.username,
			last_seen_at = excluded.last_seen_at`,
		u.ID, u.FirstName, u.LastName, u.Username, Lang(u.Lang), t, t)
	if err != nil {
		return User{}, err
	}
	return s.GetUser(u.ID)
}

func (s *Store) GetUser(id int64) (User, error) {
	var u User
	err := s.DB.QueryRow(`SELECT id, first_name, last_name, username, lang, phone FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.FirstName, &u.LastName, &u.Username, &u.Lang, &u.Phone)
	return u, notFound(err)
}

func (s *Store) SetUserLang(id int64, lang string) error {
	_, err := s.DB.Exec(`UPDATE users SET lang = ? WHERE id = ?`, Lang(lang), id)
	return err
}

func (s *Store) SetUserPhone(id int64, phone string) error {
	_, err := s.DB.Exec(`UPDATE users SET phone = ? WHERE id = ?`, phone, id)
	return err
}
