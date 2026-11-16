// Package worker executa o processamento assíncrono da aplicação.
package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DeliveryProcessor entrega atividades federadas pendentes.
type DeliveryProcessor interface {
	ProcessDeliveries(ctx context.Context) (int, error)
}

// Deps reúne as dependências do trabalhador assíncrono.
type Deps struct {
	DB           *pgxpool.Pool
	Logger       *slog.Logger
	PollInterval time.Duration
	Deliveries   DeliveryProcessor
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

			if deps.Deliveries == nil {
				continue
			}

			delivered, err := deps.Deliveries.ProcessDeliveries(ctx)
			if err != nil {
				deps.Logger.Warn("worker federation delivery failed", "error", err)
				continue
			}
			if delivered > 0 {
				deps.Logger.Info("worker delivered federation activities", "count", delivered)
			}
		}
	}
}
