package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptionDecryption(t *testing.T) {
	// 1. Generate Key
	key, b64, err := GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	if len(key) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key))
	}

	// 2. Decode Key (Simulating reading from URL)
	decodedKey, err := DecodeKey(b64)
	if err != nil {
		t.Fatalf("Failed to decode key: %v", err)
	}
	if !bytes.Equal(key, decodedKey) {
		t.Errorf("Decoded key does not match original key")
	}

	// 3. Encrypt Data
	plaintext := []byte("Top Secret CodeDrop File Data")
	ciphertext, err := Encrypt(decodedKey, plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// 4. Verify it's actually encrypted (not just returning plaintext)
	if bytes.Equal(plaintext, ciphertext) {
		t.Errorf("Ciphertext is identical to plaintext!")
	}

	// 5. Decrypt Data
	decrypted, err := Decrypt(decodedKey, ciphertext)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	// 6. Verify Original
	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("Decrypted text does not match original plaintext. Got %s", string(decrypted))
	}
}

func TestTamperedDataFails(t *testing.T) {
	key, _, _ := GenerateKey()
	plaintext := []byte("Sensitive Info")
	
	ciphertext, _ := Encrypt(key, plaintext)

	// Simulate a network error or hacker flipping a single byte
	ciphertext[len(ciphertext)-1] ^= 0xFF 

	_, err := Decrypt(key, ciphertext)
	if err == nil {
		t.Errorf("Expected decryption to fail on tampered data, but it succeeded!")
	}
}