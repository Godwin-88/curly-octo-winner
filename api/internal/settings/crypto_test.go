package settings

// Credentials entered by school staff must never be readable from the database
// or from an API response. These tests pin that property at the crypto layer.

import (
	"errors"
	"strings"
	"testing"
)

func TestSealerRoundTrip(t *testing.T) {
	sealer, err := NewSealer("test-secret")
	if err != nil {
		t.Fatalf("NewSealer: %v", err)
	}

	plaintext := `{"consumer_key":"abc","consumer_secret":"shhh"}`
	sealed, err := sealer.Seal(plaintext)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if !strings.HasPrefix(sealed, cipherVersion+":") {
		t.Errorf("sealed value %q is missing the %q prefix", sealed, cipherVersion)
	}
	if strings.Contains(sealed, "shhh") || strings.Contains(sealed, "abc") {
		t.Fatalf("sealed value leaks the plaintext: %q", sealed)
	}

	opened, err := sealer.Open(sealed)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if opened != plaintext {
		t.Errorf("Open() = %q, want %q", opened, plaintext)
	}
}

// Two seals of the same value must differ (fresh nonce) so the ciphertext does
// not reveal that two schools share a credential.
func TestSealUsesFreshNonce(t *testing.T) {
	sealer, _ := NewSealer("test-secret")
	first, err := sealer.Seal("same-value")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	second, err := sealer.Seal("same-value")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if first == second {
		t.Error("two seals of the same value are identical — the nonce is not random")
	}
}

func TestOpenRejectsTamperedCiphertext(t *testing.T) {
	sealer, _ := NewSealer("test-secret")
	sealed, err := sealer.Seal("sensitive")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	// Flip one character of the base64 payload.
	body := strings.SplitN(sealed, ":", 2)[1]
	tampered := body[:len(body)-1] + flipLast(body[len(body)-1])
	if _, err := sealer.Open(cipherVersion + ":" + tampered); err == nil {
		t.Fatal("expected a tampered ciphertext to fail authentication")
	}
}

func flipLast(c byte) string {
	if c == 'A' {
		return "B"
	}
	return "A"
}

func TestOpenRejectsUnknownVersion(t *testing.T) {
	sealer, _ := NewSealer("test-secret")
	if _, err := sealer.Open("v9:abc"); err == nil {
		t.Fatal("expected an unsupported version to be rejected")
	}
	if _, err := sealer.Open("no-colon"); err == nil {
		t.Fatal("expected a malformed value to be rejected")
	}
}

// A different key must not be able to open the value: this is what stops a
// leaked JWT_SECRET (used as the fallback key) from silently reading a value
// sealed with a rotated key.
func TestOpenFailsWithDifferentKey(t *testing.T) {
	writer, _ := NewSealer("key-one")
	reader, _ := NewSealer("key-two")
	sealed, err := writer.Seal("sensitive")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if _, err := reader.Open(sealed); err == nil {
		t.Fatal("expected a value sealed with another key to be rejected")
	}
}

func TestSealerRequiresSecret(t *testing.T) {
	if _, err := NewSealer("   "); !errors.Is(err, ErrNoEncryptionKey) {
		t.Errorf("NewSealer(blank) error = %v, want ErrNoEncryptionKey", err)
	}
}

func TestSealEmptyStringIsEmpty(t *testing.T) {
	sealer, _ := NewSealer("test-secret")
	sealed, err := sealer.Seal("")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if sealed != "" {
		t.Errorf("Seal(\"\") = %q, want empty (means: leave the stored value alone)", sealed)
	}
}

// Placeholder detection keeps a principal from being shown a scary failure for
// credentials that were never filled in.
func TestIsPlaceholder(t *testing.T) {
	placeholders := []string{"", "  ", "your_api_key", "changeme", "YOUR_KEY_HERE", "xxx", "<token>", "TODO"}
	for _, v := range placeholders {
		if !isPlaceholder(v) {
			t.Errorf("isPlaceholder(%q) = false, want true", v)
		}
	}
	// Made-up values shaped like real ones. A test never carries a real
	// credential, a real host, or any part of either.
	real := []string{"atsk_0123456789abcde", "example-instance.upstash.io", "gQAAAAAAexampleToken"}
	for _, v := range real {
		if isPlaceholder(v) {
			t.Errorf("isPlaceholder(%q) = true, want false", v)
		}
	}
}
