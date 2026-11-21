package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

// Vigencias de una sesión y de una invitación.
const (
	SessionTTL    = 30 * 24 * time.Hour
	InvitationTTL = 7 * 24 * time.Hour
)

// apiTokenPrefix distingue un token de usuario de uno de agente, que no lleva prefijo.
const apiTokenPrefix = "pmt_"

// NewToken genera 32 bytes aleatorios en hexadecimal. Sirve para sesiones e
// invitaciones.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// HashToken devuelve el SHA-256 del token: lo único que se guarda.
func HashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// NewAPIToken genera un token de API de usuario.
func NewAPIToken() (string, error) {
	token, err := NewToken()
	if err != nil {
		return "", err
	}
	return apiTokenPrefix + token, nil
}

// IsAPIToken indica si el token tiene la forma de un token de usuario.
func IsAPIToken(token string) bool {
	return strings.HasPrefix(token, apiTokenPrefix)
}
