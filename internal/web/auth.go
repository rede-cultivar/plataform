package web

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rede-cultivar/plataform/internal/identity"
)

const (
	sessionCookieName = "cultivar_session"
	csrfCookieName    = "cultivar_csrf"
	csrfTokenLength   = 32
	csrfMaxAge        = 24 * time.Hour
	maxFormBytes      = 64 << 10
)

type contextKey string

const accountContextKey contextKey = "account"

func accountFrom(ctx context.Context) *identity.Account {
	account, _ := ctx.Value(accountContextKey).(*identity.Account)
	return account
}

// loadSession identifica a conta autenticada a partir do cookie de sessão.
func (s *server) loadSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.identity == nil || isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(sessionCookieName)
		if err == nil {
			if account, err := s.identity.AuthenticateSession(r.Context(), cookie.Value); err == nil {
				r = r.WithContext(context.WithValue(r.Context(), accountContextKey, account))
			}
		}

		next.ServeHTTP(w, r)
	})
}

// requireAccount redireciona visitantes para a página de entrada.
func (s *server) requireAccount(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if accountFrom(r.Context()) == nil {
			target := "/entrar"
			if r.Method == http.MethodGet {
				target += "?next=" + url.QueryEscape(r.URL.RequestURI())
			}
			http.Redirect(w, r, target, http.StatusSeeOther)
			return
		}

		next(w, r)
	}
}

func isPublicPath(path string) bool {
	return strings.HasPrefix(path, "/static/") ||
		strings.HasPrefix(path, "/.well-known/") ||
		strings.HasPrefix(path, "/nodeinfo/") ||
		strings.HasPrefix(path, "/actors/") ||
		strings.HasPrefix(path, "/notas/") ||
		strings.HasPrefix(path, "/activities/") ||
		path == "/healthz" ||
		path == "/readyz"
}

func (s *server) registerForm(w http.ResponseWriter, r *http.Request) {
	if accountFrom(r.Context()) != nil {
		http.Redirect(w, r, "/conta", http.StatusSeeOther)
		return
	}

	s.render(w, http.StatusOK, "register.html", s.newPage(w, r))
}

func (s *server) register(w http.ResponseWriter, r *http.Request) {
	if s.identity == nil {
		http.Error(w, "cadastro indisponível", http.StatusServiceUnavailable)
		return
	}
	settings, err := s.currentSiteSettings(r.Context())
	if err != nil {
		s.logger.Error("load registration settings", "error", err)
		http.Error(w, "cadastro indisponível", http.StatusServiceUnavailable)
		return
	}
	if !settings.Open() {
		s.renderRegister(w, r, http.StatusForbidden, "", "", "", "")
		return
	}
	if !s.parseForm(w, r) {
		return
	}
	if !s.validateCSRF(r) {
		s.renderRegister(w, r, http.StatusForbidden, "Sessão do formulário expirada. Tente novamente.", "", "", "")
		return
	}

	ip := clientIP(r)
	if !s.authLimiter.allow("register:" + ip) {
		s.renderRegister(w, r, http.StatusTooManyRequests, "Muitas tentativas. Aguarde alguns minutos e tente novamente.", "", "", "")
		return
	}

	email := identity.NormalizeEmail(r.PostFormValue("email"))
	handle := identity.NormalizeHandle(r.PostFormValue("handle"))
	displayName := strings.TrimSpace(r.PostFormValue("display_name"))
	password := r.PostFormValue("password")
	passwordConfirm := r.PostFormValue("password_confirm")

	if password != passwordConfirm {
		s.renderRegister(w, r, http.StatusBadRequest, "As senhas não coincidem.", email, handle, displayName)
		return
	}

	status := identity.StatusActive
	if settings.RequiresApproval() {
		status = identity.StatusPending
	}
	account, err := s.identity.Register(r.Context(), identity.RegisterInput{
		Email:       email,
		Password:    password,
		Handle:      handle,
		DisplayName: displayName,
		Status:      status,
	})
	if err != nil {
		if message, ok := registerErrorMessage(err); ok {
			s.renderRegister(w, r, http.StatusBadRequest, message, email, handle, displayName)
			return
		}

		s.logger.Error("register account", "error", err)
		s.renderRegister(w, r, http.StatusInternalServerError, "Não foi possível criar a conta. Tente novamente.", email, handle, displayName)
		return
	}
	if account.Status == identity.StatusPending {
		data := s.newPage(w, r)
		data.RegistrationPending = true
		data.Notice = "Cadastro enviado. A conta ficará disponível depois da aprovação da administração."
		s.render(w, http.StatusCreated, "register.html", data)
		return
	}

	token, err := s.identity.StartSession(r.Context(), account.ID, ip, r.UserAgent())
	if err != nil {
		s.logger.Error("start session after registration", "error", err)
		s.renderRegister(w, r, http.StatusInternalServerError, "Conta criada, mas não foi possível iniciar a sessão. Tente entrar.", email, handle, displayName)
		return
	}

	s.setSessionCookie(w, token)
	s.rotateCSRF(w)
	s.authLimiter.reset("register:" + ip)
	http.Redirect(w, r, "/conta", http.StatusSeeOther)
}

func (s *server) loginForm(w http.ResponseWriter, r *http.Request) {
	if accountFrom(r.Context()) != nil {
		http.Redirect(w, r, "/conta", http.StatusSeeOther)
		return
	}

	data := s.newPage(w, r)
	data.Next = safeNext(r.URL.Query().Get("next"))

	s.render(w, http.StatusOK, "login.html", data)
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	if s.identity == nil {
		http.Error(w, "entrada indisponível", http.StatusServiceUnavailable)
		return
	}
	if !s.parseForm(w, r) {
		return
	}
	if !s.validateCSRF(r) {
		s.renderLogin(w, r, http.StatusForbidden, "Sessão do formulário expirada. Tente novamente.", "", "")
		return
	}

	ip := clientIP(r)
	if !s.authLimiter.allow("login:" + ip) {
		s.renderLogin(w, r, http.StatusTooManyRequests, "Muitas tentativas. Aguarde alguns minutos e tente novamente.", "", "")
		return
	}

	email := identity.NormalizeEmail(r.PostFormValue("email"))
	next := safeNext(r.PostFormValue("next"))

	account, err := s.identity.Authenticate(r.Context(), email, r.PostFormValue("password"))
	if err != nil {
		if errors.Is(err, identity.ErrAccountPending) {
			s.renderLogin(w, r, http.StatusForbidden, "Seu cadastro aguarda aprovação da administração.", email, next)
			return
		}
		if errors.Is(err, identity.ErrInvalidCredentials) || errors.Is(err, identity.ErrAccountDisabled) {
			s.renderLogin(w, r, http.StatusUnauthorized, "E-mail ou senha inválidos.", email, next)
			return
		}

		s.logger.Error("authenticate account", "error", err)
		s.renderLogin(w, r, http.StatusInternalServerError, "Não foi possível entrar. Tente novamente.", email, next)
		return
	}

	token, err := s.identity.StartSession(r.Context(), account.ID, ip, r.UserAgent())
	if err != nil {
		s.logger.Error("start session", "error", err)
		s.renderLogin(w, r, http.StatusInternalServerError, "Não foi possível iniciar a sessão. Tente novamente.", email, next)
		return
	}

	s.setSessionCookie(w, token)
	s.rotateCSRF(w)
	s.authLimiter.reset("login:" + ip)
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	if !s.parseForm(w, r) {
		return
	}
	if !s.validateCSRF(r) {
		http.Error(w, "sessão do formulário expirada", http.StatusForbidden)
		return
	}

	if s.identity != nil {
		if cookie, err := r.Cookie(sessionCookieName); err == nil {
			if err := s.identity.EndSession(r.Context(), cookie.Value); err != nil {
				s.logger.Warn("end session", "error", err)
			}
		}
	}

	s.clearSessionCookie(w)
	s.rotateCSRF(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// profileEditForm exibe o formulário de edição do perfil da conta atual.
func (s *server) profileEditForm(w http.ResponseWriter, r *http.Request) {
	account := accountFrom(r.Context())
	if account == nil {
		http.Error(w, "conta não encontrada", http.StatusUnauthorized)
		return
	}

	data := s.newPage(w, r)
	data.DisplayName = account.Person.DisplayName
	data.Bio = account.Person.Bio

	if r.URL.Query().Has("atualizado") {
		data.Notice = "Perfil atualizado."
	}

	s.render(w, http.StatusOK, "profile_edit.html", data)
}

// profileEdit valida e salva o nome de exibição e a biografia.
func (s *server) profileEdit(w http.ResponseWriter, r *http.Request) {
	account := accountFrom(r.Context())
	if account == nil {
		http.Error(w, "conta não encontrada", http.StatusUnauthorized)
		return
	}
	if s.identity == nil {
		http.Error(w, "edição de perfil indisponível", http.StatusServiceUnavailable)
		return
	}
	if !s.parseForm(w, r) {
		return
	}
	if !s.validateCSRF(r) {
		s.renderProfileEdit(w, r, http.StatusForbidden, "Sessão do formulário expirada. Tente novamente.", account.Person.DisplayName, account.Person.Bio)
		return
	}

	displayName := strings.TrimSpace(r.PostFormValue("display_name"))
	bio := strings.TrimSpace(r.PostFormValue("bio"))

	if _, err := s.identity.UpdateProfile(r.Context(), account.Person.ID, displayName, bio); err != nil {
		if message, ok := profileErrorMessage(err); ok {
			s.renderProfileEdit(w, r, http.StatusBadRequest, message, displayName, bio)
			return
		}

		s.logger.Error("update profile", "error", err)
		s.renderProfileEdit(w, r, http.StatusInternalServerError, "Não foi possível atualizar o perfil. Tente novamente.", displayName, bio)
		return
	}

	http.Redirect(w, r, "/conta/perfil?atualizado=1", http.StatusSeeOther)
}

func (s *server) renderProfileEdit(w http.ResponseWriter, r *http.Request, status int, message, displayName, bio string) {
	data := s.newPage(w, r)
	data.Error = message
	data.DisplayName = displayName
	data.Bio = bio

	s.render(w, status, "profile_edit.html", data)
}

func profileErrorMessage(err error) (string, bool) {
	switch {
	case errors.Is(err, identity.ErrInvalidDisplayName):
		return "Informe um nome de exibição com até 80 caracteres.", true
	case errors.Is(err, identity.ErrInvalidBio):
		return "A biografia deve ter no máximo 500 caracteres.", true
	default:
		return "", false
	}
}

func (s *server) renderRegister(w http.ResponseWriter, r *http.Request, status int, message, email, handle, displayName string) {
	data := s.newPage(w, r)
	data.Error = message
	data.Email = email
	data.Handle = handle
	data.DisplayName = displayName

	s.render(w, status, "register.html", data)
}

func (s *server) renderLogin(w http.ResponseWriter, r *http.Request, status int, message, email, next string) {
	data := s.newPage(w, r)
	data.Error = message
	data.Email = email
	data.Next = next

	s.render(w, status, "login.html", data)
}

func registerErrorMessage(err error) (string, bool) {
	switch {
	case errors.Is(err, identity.ErrEmailTaken):
		return "Este e-mail já está cadastrado.", true
	case errors.Is(err, identity.ErrHandleTaken):
		return "Este identificador já está em uso.", true
	case errors.Is(err, identity.ErrInvalidEmail):
		return "Informe um e-mail válido.", true
	case errors.Is(err, identity.ErrInvalidHandle):
		return "O identificador deve ter de 3 a 30 caracteres, começar com uma letra e conter apenas letras minúsculas, números e sublinhado.", true
	case errors.Is(err, identity.ErrInvalidDisplayName):
		return "Informe um nome de exibição com até 80 caracteres.", true
	case errors.Is(err, identity.ErrWeakPassword):
		return "A senha deve ter entre 12 e 128 caracteres.", true
	default:
		return "", false
	}
}

func (s *server) parseForm(w http.ResponseWriter, r *http.Request) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulário inválido", http.StatusBadRequest)
		return false
	}
	return true
}

// csrfToken retorna o token do formulário, criando-o quando necessário.
func (s *server) csrfToken(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie(csrfCookieName); err == nil && len(cookie.Value) >= csrfTokenLength {
		return cookie.Value
	}

	token, err := randomToken()
	if err != nil {
		s.logger.Error("generate csrf token", "error", err)
		return ""
	}

	s.setCSRFCookie(w, token)
	return token
}

func (s *server) validateCSRF(r *http.Request) bool {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}

	submitted := r.PostFormValue("csrf_token")
	if submitted == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(submitted)) == 1
}

func (s *server) rotateCSRF(w http.ResponseWriter) {
	token, err := randomToken()
	if err != nil {
		s.logger.Error("rotate csrf token", "error", err)
		return
	}
	s.setCSRFCookie(w, token)
}

func (s *server) setCSRFCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.SecureCookies(),
		MaxAge:   int(csrfMaxAge.Seconds()),
	})
}

func (s *server) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.SecureCookies(),
		MaxAge:   int(s.cfg.SessionAbsoluteTimeout.Seconds()),
	})
}

func (s *server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.SecureCookies(),
		MaxAge:   -1,
	})
}

func randomToken() (string, error) {
	buffer := make([]byte, csrfTokenLength)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

// safeNext aceita apenas caminhos locais como destino após a autenticação.
func safeNext(next string) string {
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.HasPrefix(next, "/\\") {
		return "/conta"
	}
	return next
}
