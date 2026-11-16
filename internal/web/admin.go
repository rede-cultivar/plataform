package web

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/rede-cultivar/plataform/internal/identity"
	"github.com/rede-cultivar/plataform/internal/sitesettings"
)

func (s *server) currentSiteSettings(ctx context.Context) (*sitesettings.Settings, error) {
	if s.siteSettings == nil {
		return sitesettings.Default(), nil
	}
	return s.siteSettings.Get(ctx)
}

func (s *server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		account := accountFrom(r.Context())
		if account == nil {
			http.Redirect(w, r, "/entrar", http.StatusSeeOther)
			return
		}
		if !account.IsAdmin() {
			http.Error(w, "operação não permitida", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func (s *server) adminSettingsForm(w http.ResponseWriter, r *http.Request) {
	if s.siteSettings == nil {
		http.Error(w, "configurações administrativas indisponíveis", http.StatusServiceUnavailable)
		return
	}
	s.renderAdminSettings(w, r, http.StatusOK, "", nil)
}

func (s *server) adminSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	if s.siteSettings == nil {
		http.Error(w, "configurações administrativas indisponíveis", http.StatusServiceUnavailable)
		return
	}
	if !s.parseForm(w, r) {
		return
	}
	if !s.validateCSRF(r) {
		s.renderAdminSettings(w, r, http.StatusForbidden, "Sessão do formulário expirada. Tente novamente.", nil)
		return
	}

	input := sitesettings.UpdateInput{
		RegistrationsMode:          r.PostFormValue("registrations_mode"),
		ClosedRegistrationsMessage: r.PostFormValue("closed_registrations_message"),
	}
	if _, err := s.siteSettings.Update(r.Context(), input); err != nil {
		switch {
		case errors.Is(err, sitesettings.ErrInvalidMode):
			s.renderAdminSettings(w, r, http.StatusBadRequest, "Selecione um modo de cadastro válido.", &input)
		case errors.Is(err, sitesettings.ErrClosedMessageTooLong):
			s.renderAdminSettings(w, r, http.StatusBadRequest, "A mensagem pode ter no máximo 500 caracteres.", &input)
		default:
			s.logger.Error("update site settings", "error", err)
			s.renderAdminSettings(w, r, http.StatusInternalServerError, "Não foi possível salvar as configurações.", &input)
		}
		return
	}

	http.Redirect(w, r, "/admin/settings?updated=1", http.StatusSeeOther)
}

func (s *server) adminApproveAccount(w http.ResponseWriter, r *http.Request) {
	s.updatePendingAccount(w, r, true)
}

func (s *server) adminRejectAccount(w http.ResponseWriter, r *http.Request) {
	s.updatePendingAccount(w, r, false)
}

func (s *server) updatePendingAccount(w http.ResponseWriter, r *http.Request, approve bool) {
	if s.identity == nil {
		http.Error(w, "gestão de contas indisponível", http.StatusServiceUnavailable)
		return
	}
	if !s.parseForm(w, r) {
		return
	}
	if !s.validateCSRF(r) {
		http.Error(w, "sessão do formulário expirada", http.StatusForbidden)
		return
	}

	accountID := chi.URLParam(r, "accountID")
	var err error
	if approve {
		err = s.identity.ApproveAccount(r.Context(), accountID)
	} else {
		err = s.identity.RejectAccount(r.Context(), accountID)
	}
	if errors.Is(err, identity.ErrNotFound) {
		http.Error(w, "cadastro pendente não encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("update pending account", "account", accountID, "approve", approve, "error", err)
		http.Error(w, "não foi possível atualizar o cadastro", http.StatusInternalServerError)
		return
	}

	if approve {
		http.Redirect(w, r, "/admin/settings?approved=1", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/settings?rejected=1", http.StatusSeeOther)
}

func (s *server) renderAdminSettings(w http.ResponseWriter, r *http.Request, status int, message string, input *sitesettings.UpdateInput) {
	settings, err := s.siteSettings.Get(r.Context())
	if err != nil {
		s.logger.Error("load site settings", "error", err)
		http.Error(w, "não foi possível carregar as configurações", http.StatusInternalServerError)
		return
	}

	data := s.newPage(w, r)
	data.Error = message
	data.RegistrationsOpen = settings.Open()
	data.RegistrationsMode = settings.RegistrationsMode
	data.ClosedRegistrationsMessage = settings.ClosedRegistrationsMessage
	if input != nil {
		data.RegistrationsMode = input.RegistrationsMode
		data.ClosedRegistrationsMessage = input.ClosedRegistrationsMessage
	}
	switch {
	case r.URL.Query().Get("updated") == "1":
		data.Notice = "Configurações atualizadas."
	case r.URL.Query().Get("approved") == "1":
		data.Notice = "Cadastro aprovado."
	case r.URL.Query().Get("rejected") == "1":
		data.Notice = "Cadastro rejeitado."
	}
	if s.identity != nil {
		pending, err := s.identity.ListPendingAccounts(r.Context())
		if err != nil {
			s.logger.Error("list pending accounts", "error", err)
			http.Error(w, "não foi possível carregar os cadastros pendentes", http.StatusInternalServerError)
			return
		}
		data.PendingAccounts = pending
	}
	s.render(w, status, "admin_settings.html", data)
}