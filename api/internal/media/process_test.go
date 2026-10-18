package media

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
)

func picture(t *testing.T, w, h int, enc string) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: uint8(x ^ y), A: 255})
		}
	}
	var buf bytes.Buffer
	var err error
	if enc == "png" {
		err = png.Encode(&buf, img)
	} else {
		err = jpeg.Encode(&buf, img, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func withExif(b []byte) []byte {
	payload := append([]byte("Exif\x00\x00"), []byte("MM\x00\x2a\x00\x00\x00\x08GPSLatitude")...)
	seg := append([]byte{0xFF, 0xE1, 0, byte(len(payload) + 2)}, payload...)
	out := append([]byte{}, b[:2]...)
	out = append(out, seg...)
	return append(out, b[2:]...)
}

func code(err error) string {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return ""
}

func TestProcessProducesVariantsWithoutMetadata(t *testing.T) {
	p, err := Process(withExif(picture(t, 3000, 2000, "jpeg")))
	if err != nil {
		t.Fatal(err)
	}
	if p.Mime != "image/jpeg" || p.Width != 3000 || len(p.SHA256) != 32 || p.PHash == 0 {
		t.Fatalf("metadata: %+v", p)
	}
	want := map[string]int{"original": 2560, "display": 1280, "thumb": 320, "thumb_blur": 320}
	for _, v := range p.Variants {
		img, _, err := image.Decode(bytes.NewReader(v.Data))
		if err != nil {
			t.Fatalf("%s does not decode: %v", v.Name, err)
		}
		if got := img.Bounds().Dx(); got != want[v.Name] {
			t.Errorf("%s width %d, want %d", v.Name, got, want[v.Name])
		}
		if bytes.Contains(v.Data, []byte("GPSLatitude")) {
			t.Errorf("%s still carries EXIF", v.Name)
		}
	}
	if len(p.Variants) != 4 {
		t.Fatalf("variants %d", len(p.Variants))
	}
}

func TestProcessKeepsSmallImagesAtTheirSize(t *testing.T) {
	p, err := Process(picture(t, 640, 480, "png"))
	if err != nil {
		t.Fatal(err)
	}
	if p.Mime != "image/png" {
		t.Fatalf("mime %s", p.Mime)
	}
	for _, v := range p.Variants {
		img, _, _ := image.Decode(bytes.NewReader(v.Data))
		if v.Name == "original" && img.Bounds().Dx() != 640 {
			t.Fatalf("original upscaled or shrunk: %d", img.Bounds().Dx())
		}
	}
}

func TestProcessRejects(t *testing.T) {
	if _, err := Process([]byte("GIF89a not really")); code(err) != "media_invalid_type" {
		t.Fatalf("gif: %v", err)
	}
	if _, err := Process(picture(t, 400, 300, "jpeg")); code(err) != "media_too_small" {
		t.Fatalf("short side 300: %v", err)
	}
	if _, err := Process(bytes.Repeat([]byte{0xFF}, MaxBytes+1)); code(err) != "payload_too_large" {
		t.Fatalf("too large: %v", err)
	}
}
