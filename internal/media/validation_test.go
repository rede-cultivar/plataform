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

	return encodePNG(t, 12, 8)
}

// encodePNG cria uma imagem PNG de teste com as dimensões informadas.
func encodePNG(t *testing.T, width, height int) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 80, A: 255})
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

func TestPrepareResizesLandscapeImage(t *testing.T) {
	prepared, err := Prepare(bytes.NewReader(encodePNG(t, 2048, 1024)))
	if err != nil {
		t.Fatalf("prepare landscape png: %v", err)
	}

	if prepared.Width != MaxDimension || prepared.Height != MaxDimension/2 {
		t.Fatalf("dimensions = %dx%d, want %dx%d", prepared.Width, prepared.Height, MaxDimension, MaxDimension/2)
	}
	if got := decodedSize(t, prepared.Data); got != (image.Point{MaxDimension, MaxDimension / 2}) {
		t.Fatalf("encoded dimensions = %v, want %v", got, image.Point{MaxDimension, MaxDimension / 2})
	}
}

func TestPrepareResizesPortraitImage(t *testing.T) {
	prepared, err := Prepare(bytes.NewReader(encodePNG(t, 1024, 2048)))
	if err != nil {
		t.Fatalf("prepare portrait png: %v", err)
	}

	if prepared.Width != MaxDimension/2 || prepared.Height != MaxDimension {
		t.Fatalf("dimensions = %dx%d, want %dx%d", prepared.Width, prepared.Height, MaxDimension/2, MaxDimension)
	}
	if got := decodedSize(t, prepared.Data); got != (image.Point{MaxDimension / 2, MaxDimension}) {
		t.Fatalf("encoded dimensions = %v, want %v", got, image.Point{MaxDimension / 2, MaxDimension})
	}
}

func TestPrepareDoesNotUpscaleSmallImage(t *testing.T) {
	prepared, err := Prepare(bytes.NewReader(encodePNG(t, 300, 200)))
	if err != nil {
		t.Fatalf("prepare small png: %v", err)
	}

	if prepared.Width != 300 || prepared.Height != 200 {
		t.Fatalf("dimensions = %dx%d, want 300x200", prepared.Width, prepared.Height)
	}
	if got := decodedSize(t, prepared.Data); got != (image.Point{300, 200}) {
		t.Fatalf("encoded dimensions = %v, want 300x200", got)
	}
}

// decodedSize devolve as dimensões da imagem codificada.
func decodedSize(t *testing.T, data []byte) image.Point {
	t.Helper()

	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode prepared image: %v", err)
	}
	return image.Point{X: config.Width, Y: config.Height}
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
