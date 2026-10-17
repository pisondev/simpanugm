package apperr

import (
	"fmt"
	"net/http"
	"time"
)

type FieldError struct {
	Field   string
	Code    string
	Message string
}

type Error struct {
	Status     int
	Code       string
	Title      string
	Detail     string
	Fields     []FieldError
	RetryAfter time.Duration
	cause      error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Detail, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Detail)
}

func (e *Error) Unwrap() error { return e.cause }

func (e *Error) WithCause(err error) *Error {
	c := *e
	c.cause = err
	return &c
}

func New(status int, code, title, detail string) *Error {
	return &Error{Status: status, Code: code, Title: title, Detail: detail}
}

func BadRequest(detail string) *Error {
	return New(http.StatusBadRequest, "bad_request", "Permintaan tidak valid", detail)
}

func Unauthenticated() *Error {
	return New(http.StatusUnauthorized, "unauthenticated", "Belum masuk", "Silakan masuk terlebih dahulu.")
}

func Forbidden() *Error {
	return New(http.StatusForbidden, "forbidden", "Akses ditolak", "Kamu tidak memiliki akses untuk tindakan ini.")
}

func AccountUnverified() *Error {
	return New(http.StatusForbidden, "account_unverified", "Email belum diverifikasi", "Verifikasi emailmu terlebih dahulu.")
}

func NotFound() *Error {
	return New(http.StatusNotFound, "not_found", "Tidak ditemukan", "Data tidak ditemukan.")
}

func StaleVersion() *Error {
	return New(http.StatusConflict, "stale_version", "Data sudah berubah", "Data sudah diubah oleh orang lain. Muat ulang lalu coba lagi.")
}

func InvalidTransition(current string) *Error {
	return New(http.StatusConflict, "invalid_transition", "Tindakan tidak dapat dilakukan", "Tindakan ini tidak tersedia pada status "+current+".")
}

func Unprocessable(code, title, detail string) *Error {
	return New(http.StatusUnprocessableEntity, code, title, detail)
}

func RateLimited(retryAfter time.Duration) *Error {
	e := New(http.StatusTooManyRequests, "rate_limited", "Terlalu banyak percobaan", "Terlalu banyak percobaan. Coba lagi nanti.")
	e.RetryAfter = retryAfter
	return e
}

func Validation(fields ...FieldError) *Error {
	e := New(http.StatusUnprocessableEntity, "validation_failed", "Data tidak valid", "Periksa kembali isian yang ditandai.")
	e.Fields = fields
	return e
}

type Collector struct {
	fields []FieldError
}

func (c *Collector) Add(field, code, message string) {
	c.fields = append(c.fields, FieldError{Field: field, Code: code, Message: message})
}

func (c *Collector) Check(ok bool, field, code, message string) {
	if !ok {
		c.Add(field, code, message)
	}
}

func (c *Collector) Has(field string) bool {
	for _, f := range c.fields {
		if f.Field == field {
			return true
		}
	}
	return false
}

func (c *Collector) Err() error {
	if len(c.fields) == 0 {
		return nil
	}
	return Validation(c.fields...)
}
