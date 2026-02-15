package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// GenerateKey creates a cryptographically secure random 32-byte (256-bit) key.
// It returns the raw bytes and a Base64 encoded string to put in the URL.
func GenerateKey() ([]byte, string, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, "", fmt.Errorf("failed to generate random key: %w", err)
	}
	
	// Base64 is used so we can safely copy-paste the key in a URL fragment
	// We use URLEncoding so there are no weird symbols like '+' or '/'
	encodedKey := base64.URLEncoding.EncodeToString(key)
	return key, encodedKey, nil
}

// DecodeKey converts the Base64 URL fragment back into raw bytes
func DecodeKey(encodedKey string) ([]byte, error) {
	key, err := base64.URLEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid key length: expected 32 bytes, got %d", len(key))
	}
	return key, nil
}

// Encrypt takes a 256-bit key and plaintext, and returns AES-GCM ciphertext.
func Encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// GCM requires a unique "nonce" (number used once) for every encryption.
	// Standard GCM nonce size is 12 bytes.
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Seal encrypts and authenticates the plaintext.
	// We append the ciphertext to the nonce so we can extract the nonce later during decryption.
	// Format: [12 bytes Nonce][... Ciphertext ...]
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt takes a 256-bit key and ciphertext, and returns the original plaintext.
func Decrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract the nonce and the actual encrypted data
	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Open decrypts and authenticates. If the data was tampered with, this will throw an error!
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong key or corrupted data): %w", err)
	}

	return plaintext, nil
}