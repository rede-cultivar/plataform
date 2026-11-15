package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rede-cultivar/plataform/internal/identity"
	"github.com/rede-cultivar/plataform/internal/platform/config"
	"github.com/rede-cultivar/plataform/internal/platform/storage"
	"github.com/rede-cultivar/plataform/internal/version"
)

//go:embed templates/*.html static/*
var assets embed.FS

// IdentityService reúne as operações de identidade usadas pela interface web.
type IdentityService interface {
	Register(ctx context.Context, input identity.RegisterInput) (*identity.Account, error)
	Authenticate(ctx context.Context, email, password string) (*identity.Account, error)
	StartSession(ctx context.Context, accountID, ipAddress, userAgent string) (string, error)
	AuthenticateSession(ctx context.Context, token string) (*identity.Account, error)
	EndSession(ctx context.Context, token string) error
	FindPersonByHandle(ctx context.Context, handle string) (*identity.Person, error)
	UpdateProfile(ctx context.Context, personID, displayName, bio string) (*identity.Person, error)
}

// Deps reúne as dependências necessárias para montar o roteador HTTP.
type Deps struct {
	Config   *config.Config
	DB       *pgxpool.Pool
	Storage  *storage.Client
	Identity IdentityService
	Logger   *slog.Logger
}

type server struct {
	cfg         *config.Config
	db          *pgxpool.Pool
	storage     *storage.Client
	identity    IdentityService
	logger      *slog.Logger
	templates   map[string]*template.Template
	authLimiter *rateLimiter
}

// pageData reúne os dados comuns às páginas renderizadas.
type pageData struct {
	InstanceName string
	Env          string
	Version      string
	LocalDomain  string
	Year         int

	Account   *identity.Account
	Person    *identity.Person
	CSRFToken string
	Error     string
	Notice    string

	Email       string
	Handle      string
	DisplayName string
	Bio         string
	Next        string
}

var pageFiles = []string{
	"home.html",
	"register.html",
	"login.html",
	"profile.html",
	"profile_edit.html",
	"account.html",
}

// NewRouter monta o roteador HTTP da aplicação.
func NewRouter(deps Deps) http.Handler {
	templates := make(map[string]*template.Template, len(pageFiles))
	for _, page := range pageFiles {
		templates[page] = template.Must(
			template.New("base.html").ParseFS(assets, "templates/base.html", "templates/"+page),
		)
	}

	s := &server{
		cfg:         deps.Config,
		db:          deps.DB,
		storage:     deps.Storage,
		identity:    deps.Identity,
		logger:      deps.Logger,
		templates:   templates,
		authLimiter: newRateLimiter(10, 15*time.Minute),
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
	router.Use(s.loadSession)

	router.Get("/healthz", s.health)
	router.Get("/readyz", s.ready)
	router.Get("/", s.home)

	router.Get("/cadastro", s.registerForm)
	router.Post("/cadastro", s.register)
	router.Get("/entrar", s.loginForm)
	router.Post("/entrar", s.login)
	router.Post("/sair", s.logout)

	router.Get("/perfil/{handle}", s.profile)
	router.Get("/conta", s.requireAccount(s.account))
	router.Get("/conta/perfil", s.requireAccount(s.profileEditForm))
	router.Post("/conta/perfil", s.requireAccount(s.profileEdit))

	router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "página não encontrada", http.StatusNotFound)
	})

	return router
}

func (s *server) newPage(w http.ResponseWriter, r *http.Request) pageData {
	return pageData{
		InstanceName: s.cfg.InstanceName,
		Env:          s.cfg.Env,
		Version:      version.Version,
		LocalDomain:  s.cfg.LocalDomain,
		Year:         time.Now().Year(),
		Account:      accountFrom(r.Context()),
		CSRFToken:    s.csrfToken(w, r),
	}
}

func (s *server) render(w http.ResponseWriter, status int, page string, data pageData) {
	tmpl, ok := s.templates[page]
	if !ok {
		s.logger.Error("template not found", "page", page)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		s.logger.Error("render page", "page", page, "error", err)
	}
}

func (s *server) home(w http.ResponseWriter, r *http.Request) {
	s.render(w, http.StatusOK, "home.html", s.newPage(w, r))
}

func (s *server) account(w http.ResponseWriter, r *http.Request) {
	s.render(w, http.StatusOK, "account.html", s.newPage(w, r))
}

func (s *server) profile(w http.ResponseWriter, r *http.Request) {
	if s.identity == nil {
		http.Error(w, "página não encontrada", http.StatusNotFound)
		return
	}

	person, err := s.identity.FindPersonByHandle(r.Context(), chi.URLParam(r, "handle"))
	if errors.Is(err, identity.ErrNotFound) {
		http.Error(w, "página não encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("find person by handle", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data := s.newPage(w, r)
	data.Person = person

	s.render(w, http.StatusOK, "profile.html", data)
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

	if s.db == nil {
		checks["database"] = "disabled"
	} else if err := s.db.Ping(ctx); err != nil {
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
