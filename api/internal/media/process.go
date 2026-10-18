package media

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"net/http"

	"github.com/corona10/goimagehash"
	"github.com/disintegration/imaging"
	"github.com/gen2brain/webp"

	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
)

const (
	MaxBytes       = 10 << 20
	minShortSide   = 320
	maxPixels      = 40_000_000
	originalSide   = 2560
	displaySide    = 1280
	thumbSide      = 320
	blurSigma      = 8
	jpegQuality    = 90
	displayQuality = 80
	thumbQuality   = 75
)

type Variant struct {
	Name        string
	Ext         string
	ContentType string
	Data        []byte
}

type Processed struct {
	Mime     string
	Width    int
	Height   int
	Bytes    int
	SHA256   []byte
	PHash    int64
	Variants []Variant
}

var allowed = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}

func errType() error {
	return apperr.New(http.StatusUnsupportedMediaType, "media_invalid_type", "Format tidak didukung", "Gunakan foto JPEG, PNG, atau WebP.")
}

// full re-encode so no EXIF or GPS survives, FR-MEDIA-04
func Process(data []byte) (Processed, error) {
	if len(data) > MaxBytes {
		return Processed{}, apperr.New(http.StatusRequestEntityTooLarge, "payload_too_large", "Foto terlalu besar", "Ukuran foto maksimal 10 MB.")
	}
	mime := http.DetectContentType(data)
	if !allowed[mime] {
		return Processed{}, errType()
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Processed{}, errType()
	}
	if cfg.Width*cfg.Height > maxPixels {
		return Processed{}, apperr.Unprocessable("media_too_large", "Resolusi terlalu besar", "Resolusi foto terlalu besar.")
	}
	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return Processed{}, errType()
	}
	b := img.Bounds()
	if min(b.Dx(), b.Dy()) < minShortSide {
		return Processed{}, apperr.Unprocessable("media_too_small", "Foto terlalu kecil", "Sisi terpendek foto minimal 320 piksel.")
	}
	original := fit(img, originalSide)
	display := fit(img, displaySide)
	thumb := fit(display, thumbSide)
	blurred := imaging.Blur(thumb, blurSigma)

	var out Processed
	out.Mime, out.Width, out.Height, out.Bytes = mime, b.Dx(), b.Dy(), len(data)
	sum := sha256.Sum256(data)
	out.SHA256 = sum[:]
	h, err := goimagehash.PerceptionHash(display)
	if err != nil {
		return Processed{}, fmt.Errorf("media: perceptual hash: %w", err)
	}
	out.PHash = int64(h.GetHash())

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, original, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return Processed{}, fmt.Errorf("media: encode original: %w", err)
	}
	out.Variants = append(out.Variants, Variant{Name: "original", Ext: "jpg", ContentType: "image/jpeg", Data: bytes.Clone(buf.Bytes())})
	for _, v := range []struct {
		name    string
		img     image.Image
		quality int
	}{{"display", display, displayQuality}, {"thumb", thumb, thumbQuality}, {"thumb_blur", blurred, thumbQuality}} {
		buf.Reset()
		if err := webp.Encode(&buf, v.img, webp.Options{Quality: v.quality}); err != nil {
			return Processed{}, fmt.Errorf("media: encode %s: %w", v.name, err)
		}
		out.Variants = append(out.Variants, Variant{Name: v.name, Ext: "webp", ContentType: "image/webp", Data: bytes.Clone(buf.Bytes())})
	}
	return out, nil
}

func fit(img image.Image, side int) image.Image {
	b := img.Bounds()
	if b.Dx() <= side && b.Dy() <= side {
		return img
	}
	return imaging.Fit(img, side, side, imaging.Lanczos)
}
