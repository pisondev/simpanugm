package media

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
)

type Handlers struct {
	svc *Service
}

func NewHandlers(svc *Service) *Handlers { return &Handlers{svc: svc} }

func (h *Handlers) UploadMedia(ctx context.Context, req openapi.UploadMediaRequestObject) (openapi.UploadMediaResponseObject, error) {
	var purpose string
	var file []byte
	for {
		part, err := req.Body.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, apperr.BadRequest("Format unggahan tidak valid.").WithCause(err)
		}
		switch part.FormName() {
		case "purpose":
			b, _ := io.ReadAll(io.LimitReader(part, 64))
			purpose = strings.TrimSpace(string(b))
		case "file":
			if file, err = io.ReadAll(io.LimitReader(part, MaxBytes+1)); err != nil {
				return nil, apperr.BadRequest("Gagal membaca berkas.").WithCause(err)
			}
		}
		_ = part.Close()
	}
	if file == nil {
		return nil, apperr.Validation(apperr.FieldError{Field: "file", Code: "required", Message: "Pilih foto terlebih dahulu"})
	}
	m, err := h.svc.Upload(ctx, access.FromContext(ctx), purpose, bytes.NewReader(file))
	if err != nil {
		return nil, err
	}
	return openapi.UploadMedia201JSONResponse(openapi.Media{
		Id: m.ID, Purpose: openapi.MediaPurpose(m.Purpose), State: openapi.MediaState(m.State),
		Width: int(m.Width), Height: int(m.Height), CreatedAt: m.CreatedAt,
	}), nil
}

func (h *Handlers) DeleteMedia(ctx context.Context, req openapi.DeleteMediaRequestObject) (openapi.DeleteMediaResponseObject, error) {
	if err := h.svc.DeleteStaged(ctx, access.FromContext(ctx), req.Id); err != nil {
		return nil, err
	}
	return openapi.DeleteMedia204Response{}, nil
}
