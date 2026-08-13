// Package cryptoutil provides shared AES-GCM encrypt/decrypt helpers used
// across the settings, storage, and analysis packages. Keeping the
// implementation in one place avoids the four-copy duplication that existed
// before and makes it easy to swap the scheme application-wide.
package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// Encrypt encrypts plain with the given 32-byte AES-GCM key and returns a
// base64-encoded nonce+ciphertext string.
func Encrypt(key []byte, plain string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plain), nil)), nil
}

// Decrypt reverses Encrypt. It returns an error if the key is invalid, the
// encoded value is malformed, or authentication fails.
func Decrypt(key []byte, encoded string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	raw, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("invalid encrypted value")
	}
	n := gcm.NonceSize()
	if len(raw) < n {
		return "", fmt.Errorf("invalid encrypted value")
	}
	plain, err := gcm.Open(nil, raw[:n], raw[n:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
