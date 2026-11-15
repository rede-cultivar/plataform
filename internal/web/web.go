package web

import (
	"context"
	"embed"
	"encoding/json"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rede-cultivar/plataform/internal/platform/config"
	"github.com/rede-cultivar/plataform/internal/platform/storage"
	"github.com/rede-cultivar/plataform/internal/version"
)

//go:embed templates/*.html static/*
var assets embed.FS

// Deps reúne as dependências necessárias para montar o roteador HTTP.
type Deps struct {
	Config  *config.Config
	DB      *pgxpool.Pool
	Storage *storage.Client
	Logger  *slog.Logger
}

type server struct {
	cfg       *config.Config
	db        *pgxpool.Pool
	storage   *storage.Client
	logger    *slog.Logger
	templates *template.Template
}

// NewRouter monta o roteador HTTP da aplicação.
func NewRouter(deps Deps) http.Handler {
	templates := template.Must(template.ParseFS(assets, "templates/*.html"))

	s := &server{
		cfg:       deps.Config,
		db:        deps.DB,
		storage:   deps.Storage,
		logger:    deps.Logger,
		templates: templates,
	}

	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(s.securityHeaders)
	router.Use(s.logRequests)

	router.Get("/healthz", s.health)
	router.Get("/readyz", s.ready)
	router.Get("/", s.home)

	router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})

	return router
}

func (s *server) home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := map[string]any{
		"InstanceName": s.cfg.InstanceName,
		"Env":          s.cfg.Env,
		"Version":      version.Version,
		"LocalDomain":  s.cfg.LocalDomain,
		"Year":         time.Now().Year(),
	}

	if err := s.templates.ExecuteTemplate(w, "base", data); err != nil {
		s.logger.Error("render home page", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": version.Version,
	})
}

func (s *server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	checks := map[string]string{
		"database": "ok",
		"storage":  "ok",
	}
	status := http.StatusOK
	overall := "ok"

	if err := s.db.Ping(ctx); err != nil {
		checks["database"] = "error"
		status = http.StatusServiceUnavailable
		overall = "degraded"
		s.logger.Warn("readiness check failed", "check", "database", "error", err)
	}

	if s.storage != nil {
		if err := s.storage.Ping(ctx); err != nil {
			checks["storage"] = "error"
			s.logger.Warn("readiness check failed", "check", "storage", "error", err)
		}
	} else {
		checks["storage"] = "disabled"
	}

	writeJSON(w, status, map[string]any{
		"status":  overall,
		"version": version.Version,
		"checks":  checks,
	})
}

func (s *server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self'; base-uri 'self'; form-action 'self'")

		if s.cfg.IsProduction() {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}

func (s *server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(wrapped, r)

		s.logger.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.Status(),
			"bytes", wrapped.BytesWritten(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", middleware.GetReqID(r.Context()),
			"remote_addr", r.RemoteAddr,
		)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
