package media

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"net/http"
	"strings"

	"github.com/disintegration/imaging"
)

const (
	// MaxBytes é o tamanho máximo aceito para uma imagem.
	MaxBytes = 10 << 20
	// MaxPhotosPerEntry é a quantidade máxima de fotos por registro.
	MaxPhotosPerEntry = 4
	// MaxPixels limita a área da imagem para evitar consumo excessivo de memória.
	MaxPixels = 40_000_000
	// MaxDimension é o tamanho máximo do maior lado de uma imagem armazenada.
	MaxDimension = 1024
)

// ValidationError descreve uma imagem inválida informada pela pessoa.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func invalid(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}

// Prepared contém a imagem normalizada pronta para armazenamento.
type Prepared struct {
	Data        []byte
	ContentType string
	Extension   string
	Width       int
	Height      int
}

// Prepare valida a imagem, aplica a orientação EXIF, reduz o maior lado para
// MaxDimension e remove metadados pessoais, como localização e modelo do
// aparelho. GIFs são preservados para manter a animação.
func Prepare(r io.Reader) (*Prepared, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}
	if len(data) == 0 {
		return nil, invalid("photos", "Selecione uma imagem válida.")
	}
	if len(data) > MaxBytes {
		return nil, invalid("photos", "Cada imagem deve ter no máximo 10 MB.")
	}

	detected := http.DetectContentType(data)
	if !strings.HasPrefix(detected, "image/") {
		return nil, invalid("photos", "O arquivo enviado não é uma imagem.")
	}

	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, invalid("photos", "Formato de imagem não suportado. Use JPEG, PNG ou GIF.")
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > MaxPixels {
		return nil, invalid("photos", "A imagem excede o limite de 40 megapixels.")
	}

	switch format {
	case "jpeg":
		return reencode(data, imaging.JPEG, "image/jpeg", ".jpg")
	case "png":
		return reencode(data, imaging.PNG, "image/png", ".png")
	case "gif":
		// GIFs são mantidos como estão para preservar a animação.
		return &Prepared{
			Data:        data,
			ContentType: "image/gif",
			Extension:   ".gif",
			Width:       config.Width,
			Height:      config.Height,
		}, nil
	default:
		return nil, invalid("photos", "Formato de imagem não suportado. Use JPEG, PNG ou GIF.")
	}
}

func reencode(data []byte, format imaging.Format, contentType, extension string) (*Prepared, error) {
	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, invalid("photos", "Não foi possível processar a imagem.")
	}

	// Reduz proporcionalmente para que o maior lado tenha no máximo
	// MaxDimension. O Fit não amplia imagens que já cabem no limite.
	img = imaging.Fit(img, MaxDimension, MaxDimension, imaging.Lanczos)

	var buffer bytes.Buffer
	if err := imaging.Encode(&buffer, img, format, imaging.JPEGQuality(85)); err != nil {
		return nil, fmt.Errorf("encode image: %w", err)
	}

	bounds := img.Bounds()

	return &Prepared{
		Data:        buffer.Bytes(),
		ContentType: contentType,
		Extension:   extension,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
	}, nil
}
