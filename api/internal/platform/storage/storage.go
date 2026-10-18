package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var ErrNotFound = errors.New("storage: object not found")

type Info struct {
	Size        int64
	ContentType string
	ModTime     time.Time
}

type Store interface {
	Put(ctx context.Context, key string, r io.Reader) error
	Get(ctx context.Context, key string) (io.ReadCloser, Info, error)
	Delete(ctx context.Context, key string) error
}

var keyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9/_.-]*$`)

func ValidKey(key string) error {
	if !keyPattern.MatchString(key) || strings.Contains(key, "..") || strings.HasSuffix(key, "/") || len(key) > 200 {
		return fmt.Errorf("storage: invalid key %q", key)
	}
	return nil
}

type FS struct {
	root string
}

func NewFS(root string) (*FS, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("storage: create root: %w", err)
	}
	return &FS{root: abs}, nil
}

func (s *FS) path(key string) (string, error) {
	if err := ValidKey(key); err != nil {
		return "", err
	}
	return filepath.Join(s.root, filepath.FromSlash(key)), nil
}

func (s *FS) Put(_ context.Context, key string, r io.Reader) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return fmt.Errorf("storage: mkdir: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".upload-*")
	if err != nil {
		return fmt.Errorf("storage: temp file: %w", err)
	}
	if _, err := io.Copy(tmp, r); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("storage: write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("storage: close: %w", err)
	}
	if err := os.Rename(tmp.Name(), p); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("storage: rename: %w", err)
	}
	return nil
}

func (s *FS) Get(_ context.Context, key string) (io.ReadCloser, Info, error) {
	p, err := s.path(key)
	if err != nil {
		return nil, Info{}, err
	}
	f, err := os.Open(p) //nolint:gosec // p is built from a key that ValidKey has checked
	if errors.Is(err, os.ErrNotExist) {
		return nil, Info{}, ErrNotFound
	}
	if err != nil {
		return nil, Info{}, fmt.Errorf("storage: open: %w", err)
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, Info{}, fmt.Errorf("storage: stat: %w", err)
	}
	return f, Info{Size: st.Size(), ContentType: mime.TypeByExtension(path.Ext(key)), ModTime: st.ModTime()}, nil
}

func (s *FS) Delete(_ context.Context, key string) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("storage: delete: %w", err)
	}
	return nil
}
