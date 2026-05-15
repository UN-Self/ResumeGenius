package crypto

import (
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef") // 32 bytes for AES-256
	plaintext := "-----BEGIN OPENSSH PRIVATE KEY-----\ntest-key-content\n-----END OPENSSH PRIVATE KEY-----"

	encrypted, err := Encrypt([]byte(plaintext), key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if string(encrypted) == plaintext {
		t.Fatal("encrypted text should not equal plaintext")
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if string(decrypted) != plaintext {
		t.Fatalf("decrypted text %q does not match plaintext %q", string(decrypted), plaintext)
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	key1 := []byte("0123456789abcdef0123456789abcdef")
	key2 := []byte("abcdef0123456789abcdef0123456789")

	encrypted, err := Encrypt([]byte("secret"), key1)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(encrypted, key2)
	if err == nil {
		t.Fatal("Decrypt with wrong key should fail")
	}
}
