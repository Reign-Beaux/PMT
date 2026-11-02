package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

const tokenBytes = 32

// NewToken genera un token aleatorio. Solo se guarda su hash.
func NewToken() string {
	raw := make([]byte, tokenBytes)
	rand.Read(raw)
	return hex.EncodeToString(raw)
}

// HashToken devuelve el SHA-256 del token, suficiente por su entropía.
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
