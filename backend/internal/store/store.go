// Package store holds all SQL queries. Handlers and services use it; they never write SQL.
package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	DB *sql.DB
}

func New(db *sql.DB) *Store { return &Store{DB: db} }

// Lang is "en" or "ru". Anything else becomes "en".
func Lang(s string) string {
	if s == "ru" {
		return "ru"
	}
	return "en"
}

// loc returns the localized column name, e.g. loc("ru", "name") = "name_ru".
// lang is always passed through Lang first, so the result is safe to put in SQL.
func loc(lang, col string) string {
	return col + "_" + Lang(lang)
}

func now() int64 { return time.Now().Unix() }

func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func decodeList(s string) []string {
	var out []string
	_ = json.Unmarshal([]byte(s), &out)
	if out == nil {
		out = []string{}
	}
	return out
}

func encodeList(v []string) string {
	if v == nil {
		v = []string{}
	}
	b, _ := json.Marshal(v)
	return string(b)
}
