package testenv

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"hash/fnv"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
)

// FakeML stands in for the inference service of matching-engine section 9 with deterministic vectors: identical image
// bytes embed identically, texts embed as hashed bags of words so shared words raise the cosine
type FakeML struct {
	*httptest.Server
	mu       sync.Mutex
	status   int
	text     string
	analysis map[string]any
	revision string
	Calls    map[string]int
}

var words = regexp.MustCompile(`[\p{L}\p{N}]+`)

func NewFakeML(t testing.TB) *FakeML {
	f := &FakeML{revision: "r1", Calls: map[string]int{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/embed/image", f.embedImage)
	mux.HandleFunc("POST /v1/embed/text", f.embedText)
	mux.HandleFunc("POST /v1/analyze/image", f.analyze)
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		status := f.status
		f.Calls[r.URL.Path]++
		f.mu.Unlock()
		if status != 0 {
			w.WriteHeader(status)
			return
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(f.Close)
	return f
}

// every request answers with this status until called with 0
func (f *FakeML) Fail(status int) { f.mu.Lock(); f.status = status; f.mu.Unlock() }

// the text OCR finds on every photo
func (f *FakeML) ShowText(s string) { f.mu.Lock(); f.text = s; f.mu.Unlock() }

// a different revision is a different model version
func (f *FakeML) Revision(r string) { f.mu.Lock(); f.revision = r; f.mu.Unlock() }

// replaces the categories, colours and brands that /v1/analyze/image answers with
func (f *FakeML) Analysis(a map[string]any) { f.mu.Lock(); f.analysis = a; f.mu.Unlock() }

func (f *FakeML) state() (string, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.text, f.revision
}

func unit(v []float32) []float32 {
	var n float64
	for _, x := range v {
		n += float64(x) * float64(x)
	}
	n = math.Sqrt(n)
	if n == 0 {
		v[0], n = 1, 1
	}
	for i := range v {
		v[i] = float32(float64(v[i]) / n)
	}
	return v
}

func bytesVector(b []byte, dim int) []float32 {
	sum := sha256.Sum256(b)
	r := rand.New(rand.NewPCG( //nolint:gosec // vectors must be reproducible, not secret
		binary.LittleEndian.Uint64(sum[:8]), binary.LittleEndian.Uint64(sum[8:16])))
	v := make([]float32, dim)
	for i := range v {
		v[i] = float32(r.NormFloat64())
	}
	return unit(v)
}

func textVector(s string, dim int) []float32 {
	v := make([]float32, dim)
	for _, w := range words.FindAllString(strings.ToLower(s), -1) {
		if w == "query" {
			continue
		}
		h := fnv.New32a()
		_, _ = h.Write([]byte(w))
		v[int(h.Sum32())%dim]++
	}
	return unit(v)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

const maxUpload = 32 << 20

func (f *FakeML) embedImage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
	if err := r.ParseMultipartForm(maxUpload); err != nil { //nolint:gosec // the body is capped by MaxBytesReader above
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, rev := f.state()
	var out [][]float32
	for _, fh := range r.MultipartForm.File["files"] {
		file, err := fh.Open()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		b, _ := io.ReadAll(file)
		_ = file.Close()
		out = append(out, bytesVector(b, 512))
	}
	writeJSON(w, map[string]any{"component": "image_encoder", "model": "test/clip", "revision": rev, "embeddings": out})
}

func (f *FakeML) embedText(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Texts []string `json:"texts"`
		Kinds []string `json:"kinds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, rev := f.state()
	out := map[string]any{}
	for _, k := range in.Kinds {
		component, model, dim := "text_encoder", "test/e5", 384
		if k == "clip_text" {
			component, model, dim = "xmodal_text_encoder", "test/clip", 512
		}
		var vs [][]float32
		for _, t := range in.Texts {
			vs = append(vs, textVector(t, dim))
		}
		out[k] = map[string]any{"component": component, "model": model, "revision": rev, "embeddings": vs}
	}
	writeJSON(w, out)
}

func (f *FakeML) analyze(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
	if err := r.ParseMultipartForm(maxUpload); err != nil { //nolint:gosec // the body is capped by MaxBytesReader above
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	text, _ := f.state()
	f.mu.Lock()
	extra := f.analysis
	f.mu.Unlock()
	out := map[string]any{
		"categories": []any{}, "colours": []any{}, "brands": []any{},
		"text":     map[string]any{"value": text, "masked": false},
		"versions": map[string]string{"ocr": "test"},
	}
	for k, v := range extra {
		out[k] = v
	}
	writeJSON(w, out)
}
