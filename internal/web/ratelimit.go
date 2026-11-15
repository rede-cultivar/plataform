package web

import (
	"sync"
	"time"
)

// rateLimiter aplica um limite simples de tentativas por chave em uma janela
// deslizante. É adequado para uma instância única e para as rotas de entrada.
type rateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
	now      func() time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		attempts: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
		now:      time.Now,
	}
}

func (l *rateLimiter) allow(key string) bool {
	now := l.now()
	cutoff := now.Add(-l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	entries := l.attempts[key]
	kept := entries[:0]
	for _, attempt := range entries {
		if attempt.After(cutoff) {
			kept = append(kept, attempt)
		}
	}

	if len(kept) >= l.limit {
		l.attempts[key] = kept
		return false
	}

	l.attempts[key] = append(kept, now)
	l.cleanup(cutoff)

	return true
}

// reset remove o histórico de uma chave após uma operação bem-sucedida.
func (l *rateLimiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}

// cleanup remove chaves sem tentativas recentes para limitar o uso de memória.
func (l *rateLimiter) cleanup(cutoff time.Time) {
	if len(l.attempts) < 5000 {
		return
	}

	for key, entries := range l.attempts {
		if len(entries) == 0 || !entries[len(entries)-1].After(cutoff) {
			delete(l.attempts, key)
		}
	}
}
