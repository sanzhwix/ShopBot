package auth

import (
	"net/url"
	"strconv"
	"testing"
	"time"
)

const token = "123456:TEST-TOKEN"

func initData(authDate time.Time) url.Values {
	return url.Values{
		"auth_date": {strconv.FormatInt(authDate.Unix(), 10)},
		"query_id":  {"AAHdF6IQAAAAAN0XohDhrOrc"},
		"user":      {`{"id":42,"first_name":"Aigerim","username":"aigerim_k","language_code":"ru"}`},
	}
}

func TestValidateInitData(t *testing.T) {
	now := time.Now()
	signed := SignInitData(initData(now), token)

	u, err := ValidateInitData(signed, token, time.Hour, now)
	if err != nil {
		t.Fatalf("valid initData rejected: %v", err)
	}
	if u.ID != 42 || u.Username != "aigerim_k" || u.LanguageCode != "ru" {
		t.Fatalf("wrong user: %+v", u)
	}

	if _, err := ValidateInitData(signed, "other:TOKEN", time.Hour, now); err != ErrBadHash {
		t.Fatalf("wrong bot token: got %v, want ErrBadHash", err)
	}

	tampered, _ := url.ParseQuery(signed)
	tampered.Set("user", `{"id":43,"first_name":"Hacker"}`)
	if _, err := ValidateInitData(tampered.Encode(), token, time.Hour, now); err != ErrBadHash {
		t.Fatalf("changed user: got %v, want ErrBadHash", err)
	}

	old := SignInitData(initData(now.Add(-48*time.Hour)), token)
	if _, err := ValidateInitData(old, token, 24*time.Hour, now); err != ErrExpired {
		t.Fatalf("old auth_date: got %v, want ErrExpired", err)
	}

	if _, err := ValidateInitData("user=%7B%7D", token, time.Hour, now); err != ErrNoHash {
		t.Fatalf("no hash: got %v, want ErrNoHash", err)
	}
}

func TestSessions(t *testing.T) {
	s := Sessions{Secret: []byte("0123456789abcdef0123456789abcdef")}
	cookie := s.Sign("owner", 7, time.Hour)

	if id, err := s.Verify(cookie, "owner"); err != nil || id != 7 {
		t.Fatalf("Verify = %d, %v; want 7, nil", id, err)
	}
	if _, err := s.Verify(cookie, "admin"); err == nil {
		t.Fatal("an owner cookie must not work as admin")
	}
	if _, err := s.Verify(cookie+"x", "owner"); err == nil {
		t.Fatal("a changed signature must fail")
	}
	if _, err := s.Verify(s.Sign("owner", 7, -time.Minute), "owner"); err == nil {
		t.Fatal("an expired cookie must fail")
	}
}
