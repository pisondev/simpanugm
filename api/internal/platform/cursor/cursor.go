package cursor

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
)

type Key struct {
	At time.Time `json:"t"`
	ID uuid.UUID `json:"id"`
}

type offset struct {
	Offset int `json:"o"`
}

func Encode(at time.Time, id uuid.UUID) *string {
	b, _ := json.Marshal(Key{At: at, ID: id})
	s := base64.RawURLEncoding.EncodeToString(b)
	return &s
}

func Decode(s *string) (*Key, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(*s)
	if err != nil {
		return nil, apperr.BadRequest("Cursor tidak valid.")
	}
	var k Key
	if err := json.Unmarshal(b, &k); err != nil || k.ID == uuid.Nil {
		return nil, apperr.BadRequest("Cursor tidak valid.")
	}
	return &k, nil
}

func EncodeOffset(n int) *string {
	b, _ := json.Marshal(offset{Offset: n})
	s := base64.RawURLEncoding.EncodeToString(b)
	return &s
}

func DecodeOffset(s *string) (int, error) {
	if s == nil || *s == "" {
		return 0, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(*s)
	if err != nil {
		return 0, apperr.BadRequest("Cursor tidak valid.")
	}
	var o offset
	if err := json.Unmarshal(b, &o); err != nil || o.Offset < 0 || o.Offset > 10000 {
		return 0, apperr.BadRequest("Cursor tidak valid.")
	}
	return o.Offset, nil
}

func Limit(l *int) int {
	if l == nil || *l < 1 {
		return 20
	}
	if *l > 100 {
		return 100
	}
	return *l
}
