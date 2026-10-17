package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"

	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/reqctx"
)

const ProblemBase = "https://simpanugm.my.id/problems/"

type problemBody struct {
	Type              string        `json:"type"`
	Title             string        `json:"title"`
	Status            int           `json:"status"`
	Code              string        `json:"code"`
	Detail            string        `json:"detail,omitempty"`
	Instance          string        `json:"instance,omitempty"`
	RequestID         string        `json:"request_id,omitempty"`
	RetryAfterSeconds int           `json:"retry_after_seconds,omitempty"`
	Errors            []fieldErrDTO `json:"errors,omitempty"`
}

type fieldErrDTO struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apperr.Error
	if !errors.As(err, &ae) {
		slog.ErrorContext(r.Context(), "unhandled error", "error", err.Error(), "path", r.URL.Path)
		ae = apperr.New(http.StatusInternalServerError, "internal", "Terjadi kesalahan", "Terjadi kesalahan di sisi kami.")
	} else if ae.Status >= 500 {
		slog.ErrorContext(r.Context(), "server error", "error", err.Error(), "error_code", ae.Code)
	}
	WriteProblem(w, r, ae)
}

func WriteProblem(w http.ResponseWriter, r *http.Request, ae *apperr.Error) {
	body := problemBody{
		Type:      ProblemBase + ae.Code,
		Title:     ae.Title,
		Status:    ae.Status,
		Code:      ae.Code,
		Detail:    ae.Detail,
		Instance:  r.URL.Path,
		RequestID: reqctx.From(r.Context()).RequestID,
	}
	for _, f := range ae.Fields {
		body.Errors = append(body.Errors, fieldErrDTO(f))
	}
	if ae.RetryAfter > 0 {
		secs := int(math.Ceil(ae.RetryAfter.Seconds()))
		body.RetryAfterSeconds = secs
		w.Header().Set("Retry-After", strconv.Itoa(secs))
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(ae.Status)
	_ = json.NewEncoder(w).Encode(body)
}
