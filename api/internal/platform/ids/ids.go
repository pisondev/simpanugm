package ids

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"

	"github.com/google/uuid"
)

func New() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func Token() (plain string, hash []byte) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	plain = base64.RawURLEncoding.EncodeToString(b)
	return plain, HashToken(plain)
}

func HashToken(plain string) []byte {
	h := sha256.Sum256([]byte(plain))
	return h[:]
}
