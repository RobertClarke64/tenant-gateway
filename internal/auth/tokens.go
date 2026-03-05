package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	TokenLength = 32 // 32 bytes = 256 bits of entropy
	PrefixLength = 8
)

// GenerateToken creates a new random token and returns both the plaintext and hash
func GenerateToken(cost int) (plaintext string, hash string, prefix string, err error) {
	// Generate random bytes
	bytes := make([]byte, TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", "", fmt.Errorf("generating random bytes: %w", err)
	}

	// Encode to base64 for the plaintext token
	plaintext = base64.URLEncoding.EncodeToString(bytes)

	// Extract prefix for quick lookups
	prefix = plaintext[:PrefixLength]

	// Hash the token for storage
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(plaintext), cost)
	if err != nil {
		return "", "", "", fmt.Errorf("hashing token: %w", err)
	}
	hash = string(hashBytes)

	return plaintext, hash, prefix, nil
}

// VerifyToken checks if a plaintext token matches a hash
func VerifyToken(plaintext, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext))
	return err == nil
}

// GetTokenPrefix returns the prefix of a token for database lookups
func GetTokenPrefix(token string) string {
	if len(token) < PrefixLength {
		return token
	}
	return token[:PrefixLength]
}
