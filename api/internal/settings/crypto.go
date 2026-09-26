package settings

// Credential encryption.
//
// Integration credentials (M-Pesa keys, WhatsApp tokens, S3 keys, ...) are
// supplied by school staff through the Settings screen, so they must not sit in
// the database in plaintext. They are sealed with AES-256-GCM, which gives
// confidentiality plus tamper detection: a modified ciphertext fails to open.
//
// The key is derived once at startup:
//
//	key = SHA-256("shule360-tenant-settings-v1:" + SETTINGS_ENCRYPTION_KEY)
//
// SETTINGS_ENCRYPTION_KEY is a dedicated secret. When it is not set we fall
// back to JWT_SECRET so the feature still works out of the box, but operators
// should set a dedicated value: rotating JWT_SECRET then also requires
// re-entering every school credential (and SETTINGS_ENCRYPTION_KEY can be
// rotated independently of the sessions secret).
//
// Ciphertext format: "v1:" + base64(nonce || ciphertext+tag). The version
// prefix leaves room to rotate the scheme later.

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	cipherVersion = "v1"
	keyContext    = "shule360-tenant-settings-v1:"
)

var ErrNoEncryptionKey = errors.New("no encryption key configured")

// Sealer seals and opens integration secrets.
type Sealer struct {
	aead cipher.AEAD
}

// NewSealer derives the sealing key from the configured secret.
func NewSealer(secret string) (*Sealer, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, ErrNoEncryptionKey
	}
	sum := sha256.Sum256([]byte(keyContext + secret))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, fmt.Errorf("derive cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	return &Sealer{aead: aead}, nil
}

// Seal encrypts plaintext, returning "v1:<base64>".
func (s *Sealer) Seal(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	// Seal appends the authentication tag to the ciphertext.
	sealed := s.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return cipherVersion + ":" + base64.StdEncoding.EncodeToString(sealed), nil
}

// Open decrypts a value produced by Seal. A tampered or foreign ciphertext
// returns an error rather than garbage.
func (s *Sealer) Open(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	version, payload, ok := strings.Cut(value, ":")
	if !ok {
		return "", fmt.Errorf("unrecognised ciphertext format")
	}
	if version != cipherVersion {
		return "", fmt.Errorf("unsupported ciphertext version %q", version)
	}
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	nonceSize := s.aead.NonceSize()
	if len(raw) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	plaintext, err := s.aead.Open(nil, raw[:nonceSize], raw[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}
