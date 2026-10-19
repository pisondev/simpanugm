// Package mlclient calls the internal inference service of matching-engine section 9
package mlclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

const (
	timeoutImage   = 10 * time.Second
	timeoutText    = 5 * time.Second
	timeoutAnalyze = 8 * time.Second
	timeoutModels  = 2 * time.Second
	maxBody        = 8 << 20
)

// the service lacks the endpoint or answers that it never will, retrying cannot help until a deploy
var ErrUnavailable = errors.New("mlclient: endpoint unavailable")

// a timeout, a refused connection or a 5xx, River retries these with backoff
type TemporaryError struct{ Err error }

func (e TemporaryError) Error() string { return "mlclient: temporary: " + e.Err.Error() }
func (e TemporaryError) Unwrap() error { return e.Err }

func IsTemporary(err error) bool {
	var t TemporaryError
	return errors.As(err, &t)
}

type Observer interface {
	ObserveMLRequest(endpoint string, status int, d time.Duration)
}

type Client struct {
	base     string
	token    string
	http     *http.Client
	observer Observer
}

// nil when no base URL is configured, every caller then treats inference as missing
func New(base, token string) *Client {
	if strings.TrimSpace(base) == "" {
		return nil
	}
	return &Client{base: strings.TrimRight(base, "/"), token: token, http: &http.Client{}}
}

func (c *Client) WithObserver(obs Observer) *Client {
	if c != nil {
		c.observer = obs
	}
	return c
}

type Version struct {
	Component string `json:"component"`
	Model     string `json:"model"`
	Revision  string `json:"revision"`
}

type ModelInfo struct {
	Component string  `json:"component"`
	ModelID   string  `json:"model_id"`
	Revision  string  `json:"revision"`
	Dimension *int    `json:"dimension"`
	LoadedAt  *string `json:"loaded_at"`
}

func (c *Client) Models(ctx context.Context) ([]ModelInfo, error) {
	var out struct {
		Items []ModelInfo `json:"items"`
	}
	err := c.do(ctx, timeoutModels, http.MethodGet, "/v1/models", nil, "", &out)
	return out.Items, err
}

type ImageEmbeddings struct {
	Version
	Embeddings [][]float32 `json:"embeddings"`
}

func (c *Client) EmbedImage(ctx context.Context, images [][]byte) (ImageEmbeddings, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for i, img := range images {
		fw, err := w.CreateFormFile("files", fmt.Sprintf("image-%d.webp", i))
		if err != nil {
			return ImageEmbeddings{}, err
		}
		if _, err := fw.Write(img); err != nil {
			return ImageEmbeddings{}, err
		}
	}
	if err := w.Close(); err != nil {
		return ImageEmbeddings{}, err
	}
	var out ImageEmbeddings
	err := c.do(ctx, timeoutImage, http.MethodPost, "/v1/embed/image", &body, w.FormDataContentType(), &out)
	if err == nil && len(out.Embeddings) != len(images) {
		err = fmt.Errorf("mlclient: %d embeddings for %d images", len(out.Embeddings), len(images))
	}
	return out, err
}

type TextEmbeddings struct {
	Version
	Embeddings [][]float32 `json:"embeddings"`
}

// kinds are "e5" and "clip_text", the answer carries one block per requested kind
func (c *Client) EmbedText(ctx context.Context, texts []string, kinds []string) (map[string]TextEmbeddings, error) {
	payload, err := json.Marshal(map[string]any{"texts": texts, "kinds": kinds})
	if err != nil {
		return nil, err
	}
	out := map[string]TextEmbeddings{}
	if err := c.do(ctx, timeoutText, http.MethodPost, "/v1/embed/text", bytes.NewReader(payload), "application/json", &out); err != nil {
		return nil, err
	}
	for _, k := range kinds {
		if len(out[k].Embeddings) != len(texts) {
			return nil, fmt.Errorf("mlclient: %s answered %d embeddings for %d texts", k, len(out[k].Embeddings), len(texts))
		}
	}
	return out, nil
}

type CategoryPrompts struct {
	Slug    string   `json:"slug"`
	Prompts []string `json:"prompts"`
}

type Analysis struct {
	Categories []struct {
		Slug  string  `json:"slug"`
		Score float64 `json:"score"`
	} `json:"categories"`
	Colours []struct {
		Name  string  `json:"name"`
		Hex   string  `json:"hex"`
		Share float64 `json:"share"`
	} `json:"colours"`
	Text struct {
		Value  string `json:"value"`
		Masked bool   `json:"masked"`
	} `json:"text"`
	Brands []struct {
		Name  string  `json:"name"`
		Score float64 `json:"score"`
	} `json:"brands"`
	Versions map[string]string `json:"versions"`
}

func (c *Client) AnalyzeImage(ctx context.Context, image []byte, categories []CategoryPrompts, promptSetHash string) (Analysis, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, err := w.CreateFormFile("file", "image.webp")
	if err != nil {
		return Analysis{}, err
	}
	if _, err := fw.Write(image); err != nil {
		return Analysis{}, err
	}
	cats, err := json.Marshal(categories)
	if err != nil {
		return Analysis{}, err
	}
	if err := w.WriteField("categories", string(cats)); err != nil {
		return Analysis{}, err
	}
	if err := w.WriteField("prompt_set_hash", promptSetHash); err != nil {
		return Analysis{}, err
	}
	if err := w.Close(); err != nil {
		return Analysis{}, err
	}
	var out Analysis
	err = c.do(ctx, timeoutAnalyze, http.MethodPost, "/v1/analyze/image", &body, w.FormDataContentType(), &out)
	return out, err
}

func (c *Client) do(ctx context.Context, timeout time.Duration, method, path string, body io.Reader, contentType string, out any) error {
	if c == nil {
		return ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	start := time.Now()
	res, err := c.http.Do(req)
	dur := time.Since(start)
	if err != nil {
		if c.observer != nil {
			c.observer.ObserveMLRequest(path, 0, dur)
		}
		return TemporaryError{Err: err}
	}
	if c.observer != nil {
		c.observer.ObserveMLRequest(path, res.StatusCode, dur)
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(res.Body, maxBody))
	if err != nil {
		return TemporaryError{Err: err}
	}
	switch {
	case res.StatusCode == http.StatusNotFound || res.StatusCode == http.StatusNotImplemented:
		return fmt.Errorf("%w: %s %s answered %d", ErrUnavailable, method, path, res.StatusCode)
	case res.StatusCode >= 500 || res.StatusCode == http.StatusTooManyRequests:
		return TemporaryError{Err: fmt.Errorf("%s %s answered %d", method, path, res.StatusCode)}
	case res.StatusCode >= 400:
		return fmt.Errorf("mlclient: %s %s answered %d: %s", method, path, res.StatusCode, truncate(raw))
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("mlclient: decode %s: %w", path, err)
	}
	return nil
}

func truncate(b []byte) string {
	if len(b) > 300 {
		b = b[:300]
	}
	return string(b)
}
