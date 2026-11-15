package media

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func testPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 12, 8))
	for x := 0; x < 12; x++ {
		for y := 0; y < 8; y++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 20), G: uint8(y * 30), B: 80, A: 255})
		}
	}

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatalf("encode test png: %v", err)
	}
	return buffer.Bytes()
}

func TestPreparePNG(t *testing.T) {
	prepared, err := Prepare(bytes.NewReader(testPNG(t)))
	if err != nil {
		t.Fatalf("prepare png: %v", err)
	}

	if prepared.ContentType != "image/png" {
		t.Fatalf("content type = %q, want image/png", prepared.ContentType)
	}
	if prepared.Extension != ".png" {
		t.Fatalf("extension = %q, want .png", prepared.Extension)
	}
	if prepared.Width != 12 || prepared.Height != 8 {
		t.Fatalf("dimensions = %dx%d, want 12x8", prepared.Width, prepared.Height)
	}
	if len(prepared.Data) == 0 {
		t.Fatal("prepared data is empty")
	}

	if _, _, err := image.Decode(bytes.NewReader(prepared.Data)); err != nil {
		t.Fatalf("decode prepared png: %v", err)
	}
}

func TestPrepareRejectsNonImage(t *testing.T) {
	_, err := Prepare(strings.NewReader("não é uma imagem"))
	if err == nil {
		t.Fatal("expected validation error")
	}

	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %T, want *ValidationError", err)
	}
	if validation.Field != "photos" {
		t.Fatalf("field = %q, want photos", validation.Field)
	}
}

func TestSafeName(t *testing.T) {
	if got := safeName("../../etc/passwd"); got != "passwd" {
		t.Fatalf("safeName = %q, want passwd", got)
	}
	if got := safeName("   "); got != "imagem" {
		t.Fatalf("safeName = %q, want imagem", got)
	}
}
