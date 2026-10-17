package auth

import (
	"bufio"
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// NFR-SEC-03, the OWASP minimum for argon2id
const (
	argonMemory  = 19 * 1024
	argonTime    = 2
	argonThreads = 1
	argonKeyLen  = 32
	argonSaltLen = 16
)

//go:embed commonpasswords.txt
var commonPasswordsRaw string

var commonPasswords = func() map[string]struct{} {
	m := make(map[string]struct{}, 10000)
	sc := bufio.NewScanner(strings.NewReader(commonPasswordsRaw))
	for sc.Scan() {
		if w := strings.TrimSpace(sc.Text()); w != "" {
			m[strings.ToLower(w)] = struct{}{}
		}
	}
	return m
}()

func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

// rehash is true when the stored hash used older parameters
func VerifyPassword(password, encoded string) (ok bool, rehash bool, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, false, errors.New("auth: unsupported hash format")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, false, fmt.Errorf("auth: hash version: %w", err)
	}
	var memory uint32
	var iterations uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &threads); err != nil {
		return false, false, fmt.Errorf("auth: hash params: %w", err)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, false, fmt.Errorf("auth: hash salt: %w", err)
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, false, fmt.Errorf("auth: hash key: %w", err)
	}
	got := argon2.IDKey([]byte(password), salt, iterations, memory, threads, uint32(len(want)))
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return false, false, nil
	}
	rehash = memory != argonMemory || iterations != argonTime || threads != argonThreads || version != argon2.Version
	return true, rehash, nil
}

// spends the same time as a real verification so unknown emails cannot be told apart
var dummyHash, _ = HashPassword("simpanugm-dummy-password")

func burnVerify(password string) { _, _, _ = VerifyPassword(password, dummyHash) }

// FR-AUTH-02, an empty code means the password is acceptable
func PasswordProblem(password, email string) (code, message string) {
	n := utf8.RuneCountInString(password)
	switch {
	case n < 10:
		return "too_short", "Kata sandi minimal 10 karakter"
	case n > 128:
		return "too_long", "Kata sandi maksimal 128 karakter"
	case strings.EqualFold(password, email):
		return "equals_email", "Kata sandi tidak boleh sama dengan email"
	}
	if _, common := commonPasswords[strings.ToLower(password)]; common {
		return "too_common", "Kata sandi ini terlalu umum, pilih yang lain"
	}
	return "", ""
}
