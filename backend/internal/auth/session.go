package auth

import (
	"crypto/hmac"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var ErrBadSession = errors.New("session is invalid or expired")

// Sessions makes and checks cookie values of the form "role:id:expires.signature".
type Sessions struct {
	Secret []byte
}

func (s Sessions) Sign(role string, id int64, ttl time.Duration) string {
	payload := fmt.Sprintf("%s:%d:%d", role, id, time.Now().Add(ttl).Unix())
	return payload + "." + base64.RawURLEncoding.EncodeToString(hmacSHA256(s.Secret, []byte(payload)))
}

// Verify returns the id stored in the cookie if the role matches and the cookie is not expired.
func (s Sessions) Verify(value, role string) (int64, error) {
	payload, sig, ok := strings.Cut(value, ".")
	if !ok {
		return 0, ErrBadSession
	}
	want := base64.RawURLEncoding.EncodeToString(hmacSHA256(s.Secret, []byte(payload)))
	if !hmac.Equal([]byte(sig), []byte(want)) {
		return 0, ErrBadSession
	}
	parts := strings.Split(payload, ":")
	if len(parts) != 3 || parts[0] != role {
		return 0, ErrBadSession
	}
	id, err1 := strconv.ParseInt(parts[1], 10, 64)
	exp, err2 := strconv.ParseInt(parts[2], 10, 64)
	if err1 != nil || err2 != nil || time.Now().Unix() > exp {
		return 0, ErrBadSession
	}
	return id, nil
}
