// Package worker executa o processamento assíncrono da aplicação.
//
// A primeira versão apenas mantém o processo ativo e verifica a conectividade
// com o banco. O processamento do outbox federado será adicionado na etapa de
// implementação da federação.
package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps reúne as dependências do trabalhador assíncrono.
type Deps struct {
	DB           *pgxpool.Pool
	Logger       *slog.Logger
	PollInterval time.Duration
}

// Run mantém o trabalhador em execução até o cancelamento do contexto.
func Run(ctx context.Context, deps Deps) error {
	if deps.PollInterval <= 0 {
		deps.PollInterval = 5 * time.Second
	}

	deps.Logger.Info("worker started", "poll_interval", deps.PollInterval)

	ticker := time.NewTicker(deps.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			deps.Logger.Info("worker stopped")
			return nil
		case <-ticker.C:
			if err := deps.DB.Ping(ctx); err != nil {
				deps.Logger.Warn("worker database check failed", "error", err)
				continue
			}
			deps.Logger.Debug("worker tick; outbox processing is not implemented yet")
		}
	}
}
