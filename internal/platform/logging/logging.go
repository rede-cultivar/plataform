package logging

import (
	"log/slog"
	"os"
	"strings"

	"github.com/rede-cultivar/plataform/internal/platform/config"
)

// New cria um logger estruturado. Em desenvolvimento usa texto legível;
// nos demais ambientes usa JSON, adequado à coleta por ferramentas externas.
func New(level, environment string) *slog.Logger {
	var parsed slog.Level
	switch strings.ToLower(level) {
	case "debug":
		parsed = slog.LevelDebug
	case "warn":
		parsed = slog.LevelWarn
	case "error":
		parsed = slog.LevelError
	default:
		parsed = slog.LevelInfo
	}

	options := &slog.HandlerOptions{Level: parsed}

	if environment == config.EnvDevelopment {
		return slog.New(slog.NewTextHandler(os.Stdout, options))
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, options))
}
