package models

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// encryptionKey should be loaded from env/config in production.
// A 32-byte key for AES-256-GCM.
var encryptionKey = []byte("eino-ops-agent-32byte-secret!!!!") // 32 bytes

// SetEncryptionKey replaces the default encryption key.
func SetEncryptionKey(key []byte) {
	if len(key) == 32 {
		encryptionKey = make([]byte, 32)
		copy(encryptionKey, key)
	}
}

// EncryptCredential encrypts a credential string with AES-256-GCM.
// Returns base64-encoded ciphertext.
func EncryptCredential(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	block, err := aes.NewCipher(encryptionKey)
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
	ciphertext := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptCredential decrypts a credential that was encrypted with EncryptCredential.
// Falls back to returning the plaintext for legacy unencrypted credentials.
func DecryptCredential(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		// Legacy: credential was stored in plaintext, return as-is
		return encoded, nil
	}
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("credential: ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("credential: decryption failed")
	}
	return string(plain), nil
}
