package vault

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	v, err := New(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	a, _ := v.Encrypt("demo123")
	b, _ := v.Encrypt("demo123")
	if bytes.Equal(a, b) {
		t.Fatal("same password must give different ciphertext (random nonce)")
	}
	if bytes.Contains(a, []byte("demo123")) {
		t.Fatal("ciphertext contains the plain password")
	}
	got, err := v.Decrypt(a)
	if err != nil || got != "demo123" {
		t.Fatalf("Decrypt = %q, %v", got, err)
	}

	other, _ := New(bytes.Repeat([]byte{8}, 32))
	if _, err := other.Decrypt(a); err == nil {
		t.Fatal("a different key must not decrypt")
	}
	a[len(a)-1] ^= 1
	if _, err := v.Decrypt(a); err == nil {
		t.Fatal("changed ciphertext must fail")
	}
}
