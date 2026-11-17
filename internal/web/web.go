package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rede-cultivar/plataform/internal/cultivation"
	"github.com/rede-cultivar/plataform/internal/federation"
	"github.com/rede-cultivar/plataform/internal/identity"
	"github.com/rede-cultivar/plataform/internal/media"
	"github.com/rede-cultivar/plataform/internal/platform/config"
	"github.com/rede-cultivar/plataform/internal/platform/storage"
	"github.com/rede-cultivar/plataform/internal/practices"
	"github.com/rede-cultivar/plataform/internal/sitesettings"
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
	FindPersonByID(ctx context.Context, id string) (*identity.Person, error)
	UpdateProfile(ctx context.Context, personID, displayName, bio string) (*identity.Person, error)
	Stats(ctx context.Context) (identity.Stats, error)
	ListPendingAccounts(ctx context.Context) ([]identity.Account, error)
	ApproveAccount(ctx context.Context, accountID string) error
	RejectAccount(ctx context.Context, accountID string) error
}

// SiteSettingsService reúne as configurações administrativas da instância.
type SiteSettingsService interface {
	Get(ctx context.Context) (*sitesettings.Settings, error)
	Update(ctx context.Context, input sitesettings.UpdateInput) (*sitesettings.Settings, error)
}

// CultivationService reúne as operações do diário de cultivo usadas pela interface.
type CultivationService interface {
	CreateCycle(ctx context.Context, personID string, input cultivation.CycleInput) (*cultivation.Cycle, error)
	UpdateCycle(ctx context.Context, personID, cycleID string, input cultivation.CycleUpdateInput) (*cultivation.Cycle, error)
	FindCycle(ctx context.Context, viewerID, cycleID string) (*cultivation.Cycle, []cultivation.Entry, error)
	ListPersonCycles(ctx context.Context, personID string) ([]cultivation.Cycle, error)
	ListPublicCycles(ctx context.Context, filter cultivation.CycleFilter) ([]cultivation.Cycle, error)
	CreateEntry(ctx context.Context, personID, cycleID string, input cultivation.EntryInput) (*cultivation.Entry, error)
	UpdateEntry(ctx context.Context, personID, cycleID, entryID string, input cultivation.EntryInput) (*cultivation.Entry, error)
	FindEntry(ctx context.Context, entryID string) (*cultivation.Entry, error)
}

// PracticeService reúne as operações de conhecimento e práticas usadas pela interface.
type PracticeService interface {
	CreatePractice(ctx context.Context, authorID string, input practices.Input) (*practices.Practice, error)
	UpdatePractice(ctx context.Context, authorID, practiceID string, input practices.Input) (*practices.Practice, error)
	FindPractice(ctx context.Context, viewerID, practiceID string) (*practices.Practice, error)
	ListPublicPractices(ctx context.Context, filter practices.Filter) ([]practices.Practice, error)
	ListAuthorPractices(ctx context.Context, authorID string) ([]practices.Practice, error)
	PublishPractice(ctx context.Context, authorID, practiceID string) (*practices.Practice, error)
	ArchivePractice(ctx context.Context, authorID, practiceID string) (*practices.Practice, error)
}

// MediaService reúne as operações de imagens usadas pela interface web.
type MediaService interface {
	Upload(ctx context.Context, ownerID, filename string, r io.Reader) (*media.Asset, error)
	AttachToEntry(ctx context.Context, entryID, mediaID string, position int) error
	ListByEntryIDs(ctx context.Context, entryIDs []string) (map[string][]media.Asset, error)
	Open(ctx context.Context, id string) (*media.Asset, io.ReadCloser, error)
	CanView(ctx context.Context, id, viewerID string) (bool, error)
	Delete(ctx context.Context, ownerID, id string) error
}

// FederationService reúne as operações de federação usadas pela interface web.
type FederationService interface {
	ActorKey(ctx context.Context, personID string) (*federation.KeyPair, error)
	PublishEntry(ctx context.Context, person identity.Person, cycle cultivation.Cycle, entry cultivation.Entry, assets []media.Asset) error
	Outbox(ctx context.Context, personID string) (federation.OrderedCollection, error)
	Followers(ctx context.Context, personID string) (federation.OrderedCollection, error)
	FollowerProfiles(ctx context.Context, personID string) ([]federation.Follower, error)
	HandleInbox(ctx context.Context, personID string, r *http.Request) error
	FindActivity(ctx context.Context, activityID string) (json.RawMessage, error)
}

// Deps reúne as dependências necessárias para montar o roteador HTTP.
type Deps struct {
	Config       *config.Config
	DB           *pgxpool.Pool
	Storage      *storage.Client
	Identity     IdentityService
	Cultivation  CultivationService
	Practices    PracticeService
	Media        MediaService
	Federation   FederationService
	SiteSettings SiteSettingsService
	Logger       *slog.Logger
}

type server struct {
	cfg          *config.Config
	db           *pgxpool.Pool
	storage      *storage.Client
	identity     IdentityService
	cultivation  CultivationService
	practices    PracticeService
	media        MediaService
	federation   FederationService
	siteSettings SiteSettingsService
	logger       *slog.Logger
	templates    map[string]*template.Template
	authLimiter  *rateLimiter
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

	IsAdmin                    bool
	RegistrationsOpen          bool
	RegistrationPending        bool
	RegistrationsMode          string
	ClosedRegistrationsMessage string
	ClosedRegistrationsNotice  string
	PendingAccounts            []identity.Account

	Email       string
	Handle      string
	DisplayName string
	Bio         string
	Next        string

	Cycles         []cultivation.Cycle
	Cycle          *cultivation.Cycle
	Entries        []cultivation.Entry
	EntryMedia     map[string][]media.Asset
	CycleFilter    cultivation.CycleFilter
	FormCycle      cultivation.Cycle
	FormEntry      cultivation.Entry
	FormPractice   practices.Practice
	PracticeList   []practices.Practice
	Practice       *practices.Practice
	PracticeFilter practices.Filter
	Practices      []practices.Practice

	Followers []federation.Follower

	MeasurementsText  string
	InterventionsText string
	OccurrencesText   string
	MaterialsText     string
	StepsText         string

	CanEdit    bool
	IsNew      bool
	FormAction string
}

var pageFiles = []string{
	"home.html",
	"register.html",
	"login.html",
	"profile.html",
	"profile_edit.html",
	"account.html",
	"account_followers.html",
	"admin_settings.html",
	"diary.html",
	"diary_mine.html",
	"cycle_new.html",
	"cycle.html",
	"cycle_edit.html",
	"entry_new.html",
	"entry_edit.html",
	"practices.html",
	"practices_mine.html",
	"practice_new.html",
	"practice.html",
	"practice_edit.html",
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
		cfg:          deps.Config,
		db:           deps.DB,
		storage:      deps.Storage,
		identity:     deps.Identity,
		cultivation:  deps.Cultivation,
		practices:    deps.Practices,
		media:        deps.Media,
		federation:   deps.Federation,
		siteSettings: deps.SiteSettings,
		logger:       deps.Logger,
		templates:    templates,
		authLimiter:  newRateLimiter(10, 15*time.Minute),
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
	router.Get("/conta/seguidores", s.requireAccount(s.accountFollowers))
	router.Get("/conta/perfil", s.requireAccount(s.profileEditForm))
	router.Post("/conta/perfil", s.requireAccount(s.profileEdit))
	router.Get("/admin", s.requireAdmin(s.adminSettingsForm))
	router.Get("/admin/settings", s.requireAdmin(s.adminSettingsForm))
	router.Post("/admin/settings", s.requireAdmin(s.adminSettingsUpdate))
	router.Post("/admin/accounts/{accountID}/approve", s.requireAdmin(s.adminApproveAccount))
	router.Post("/admin/accounts/{accountID}/reject", s.requireAdmin(s.adminRejectAccount))

	router.Get("/diario", s.diary)
	router.Get("/diario/meu", s.requireAccount(s.myDiary))
	router.Get("/diario/novo", s.requireAccount(s.cycleNewForm))
	router.Post("/diario", s.requireAccount(s.cycleCreate))
	router.Get("/diario/{id}/editar", s.requireAccount(s.cycleEditForm))
	router.Post("/diario/{id}/editar", s.requireAccount(s.cycleEdit))
	router.Get("/diario/{id}/registros/novo", s.requireAccount(s.entryNewForm))
	router.Post("/diario/{id}/registros", s.requireAccount(s.entryCreate))
	router.Get("/diario/{cycleID}/registros/{entryID}/editar", s.requireAccount(s.entryEditForm))
	router.Post("/diario/{cycleID}/registros/{entryID}/editar", s.requireAccount(s.entryEdit))
	router.Get("/diario/{id}", s.cycleShow)

	router.Get("/media/{id}", s.mediaAsset)

	router.Get("/praticas", s.practicesIndex)
	router.Get("/praticas/minhas", s.requireAccount(s.myPractices))
	router.Get("/praticas/nova", s.requireAccount(s.practiceNewForm))
	router.Post("/praticas", s.requireAccount(s.practiceCreate))
	router.Get("/praticas/{id}/editar", s.requireAccount(s.practiceEditForm))
	router.Post("/praticas/{id}/editar", s.requireAccount(s.practiceEdit))
	router.Post("/praticas/{id}/publicar", s.requireAccount(s.practicePublish))
	router.Post("/praticas/{id}/arquivar", s.requireAccount(s.practiceArchive))
	router.Get("/praticas/{id}", s.practiceShow)

	router.Get("/.well-known/webfinger", s.webfinger)
	router.Get("/.well-known/nodeinfo", s.nodeInfoIndex)
	router.Get("/nodeinfo/2.0", s.nodeInfo)

	router.Get("/actors/{id}", s.actor)
	router.Get("/actors/{id}/outbox", s.actorOutbox)
	router.Get("/actors/{id}/followers", s.actorFollowers)
	router.Get("/actors/{id}/following", s.actorFollowing)
	router.Post("/actors/{id}/inbox", s.inbox)

	router.Get("/notas/{id}", s.note)
	router.Get("/activities/{id}", s.activity)

	router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "página não encontrada", http.StatusNotFound)
	})

	return router
}

func (s *server) newPage(w http.ResponseWriter, r *http.Request) pageData {
	account := accountFrom(r.Context())
	data := pageData{
		InstanceName:              s.cfg.InstanceName,
		Env:                       s.cfg.Env,
		Version:                   version.Version,
		LocalDomain:               s.cfg.LocalDomain,
		Year:                      time.Now().Year(),
		Account:                   account,
		IsAdmin:                   account.IsAdmin(),
		RegistrationsOpen:         true,
		RegistrationsMode:         sitesettings.ModeOpen,
		ClosedRegistrationsNotice: sitesettings.DefaultClosedMessage,
		CSRFToken:                 s.csrfToken(w, r),
	}
	if s.siteSettings != nil {
		settings, err := s.siteSettings.Get(r.Context())
		if err != nil {
			s.logger.Warn("load site settings for page", "error", err)
			data.RegistrationsOpen = false
		} else {
			data.RegistrationsOpen = settings.Open()
			data.RegistrationsMode = settings.RegistrationsMode
			data.ClosedRegistrationsNotice = settings.ClosedMessage()
		}
	}
	return data
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
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: https:; style-src 'self'; base-uri 'self'; form-action 'self'")

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
	writeJSONType(w, status, "application/json; charset=utf-8", payload)
}

func writeJSONType(w http.ResponseWriter, status int, contentType string, payload any) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
