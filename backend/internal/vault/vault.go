// Package vault encrypts saved shop passwords with AES-256-GCM.
//
// Auto sign-in needs the real password, so it cannot be a one-way hash.
// The key lives outside the database (VAULT_KEY or data/vault.key), so a copy
// of the database alone does not reveal passwords.
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
)

var ErrShortCiphertext = errors.New("ciphertext too short")

type Vault struct {
	aead cipher.AEAD
}

func New(key []byte) (*Vault, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Vault{aead: aead}, nil
}

// Encrypt returns nonce + ciphertext. A new random nonce is used every time.
func (v *Vault) Encrypt(plain string) ([]byte, error) {
	nonce := make([]byte, v.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return v.aead.Seal(nonce, nonce, []byte(plain), nil), nil
}

func (v *Vault) Decrypt(data []byte) (string, error) {
	n := v.aead.NonceSize()
	if len(data) < n {
		return "", ErrShortCiphertext
	}
	plain, err := v.aead.Open(nil, data[:n], data[n:], nil)
	return string(plain), err
}
