package web

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/rede-cultivar/plataform/internal/practices"
)

// practicesIndex exibe a biblioteca pública de práticas.
func (s *server) practicesIndex(w http.ResponseWriter, r *http.Request) {
	if s.practices == nil {
		http.Error(w, "biblioteca indisponível", http.StatusServiceUnavailable)
		return
	}

	filter := practices.NormalizeFilter(practices.Filter{
		Category:   r.URL.Query().Get("category"),
		Difficulty: practices.Difficulty(r.URL.Query().Get("difficulty")),
	})

	list, err := s.practices.ListPublicPractices(r.Context(), filter)
	if err != nil {
		s.logger.Error("list public practices", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data := s.newPage(w, r)
	data.PracticeList = list
	data.PracticeFilter = filter

	s.render(w, http.StatusOK, "practices.html", data)
}

// myPractices exibe as práticas da pessoa autenticada, incluindo rascunhos.
func (s *server) myPractices(w http.ResponseWriter, r *http.Request) {
	account := accountFrom(r.Context())
	if account == nil {
		http.Error(w, "conta não encontrada", http.StatusUnauthorized)
		return
	}
	if s.practices == nil {
		http.Error(w, "biblioteca indisponível", http.StatusServiceUnavailable)
		return
	}

	list, err := s.practices.ListAuthorPractices(r.Context(), account.Person.ID)
	if err != nil {
		s.logger.Error("list author practices", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data := s.newPage(w, r)
	data.PracticeList = list

	s.render(w, http.StatusOK, "practices_mine.html", data)
}

// practiceNewForm exibe o formulário de criação de uma prática.
func (s *server) practiceNewForm(w http.ResponseWriter, r *http.Request) {
	form := practices.Practice{Difficulty: practices.DifficultyBeginner}
	s.renderPracticeNew(w, r, http.StatusOK, "", form)
}

// practiceCreate valida e cria uma prática.
func (s *server) practiceCreate(w http.ResponseWriter, r *http.Request) {
	account := accountFrom(r.Context())
	if account == nil {
		http.Error(w, "conta não encontrada", http.StatusUnauthorized)
		return
	}
	if s.practices == nil {
		http.Error(w, "biblioteca indisponível", http.StatusServiceUnavailable)
		return
	}
	if !s.parseForm(w, r) {
		return
	}
	if !s.validateCSRF(r) {
		s.renderPracticeNew(w, r, http.StatusForbidden, "Sessão do formulário expirada. Tente novamente.", practices.Practice{})
		return
	}

	input := practiceInputFromForm(r)
	form := practiceFormFromInput(input, "")

	practice, err := s.practices.CreatePractice(r.Context(), account.Person.ID, input)
	if err != nil {
		if message, ok := validationMessage(err); ok {
			s.renderPracticeNew(w, r, http.StatusBadRequest, message, form)
			return
		}
		s.logger.Error("create practice", "error", err)
		s.renderPracticeNew(w, r, http.StatusInternalServerError, "Não foi possível criar a prática. Tente novamente.", form)
		return
	}

	if s.federation != nil && federatedPractice(practice) {
		if err := s.federation.PublishPractice(r.Context(), account.Person, *practice); err != nil {
			s.logger.Warn("publish practice", "practice", practice.ID, "error", err)
		}
	}

	http.Redirect(w, r, "/praticas/"+practice.ID, http.StatusSeeOther)
}

// practiceShow exibe uma prática visível.
func (s *server) practiceShow(w http.ResponseWriter, r *http.Request) {
	if s.practices == nil {
		http.Error(w, "biblioteca indisponível", http.StatusServiceUnavailable)
		return
	}

	viewerID := ""
	if account := accountFrom(r.Context()); account != nil {
		viewerID = account.Person.ID
	}

	practice, err := s.practices.FindPractice(r.Context(), viewerID, chi.URLParam(r, "id"))
	if errors.Is(err, practices.ErrNotFound) {
		http.Error(w, "página não encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("find practice", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data := s.newPage(w, r)
	data.Practice = practice
	data.CanEdit = practice.IsOwner(viewerID)

	if s.federation != nil && federatedPractice(practice) {
		interactions, err := s.federation.PracticeInteractions(r.Context(), []string{practice.ID})
		if err != nil {
			s.logger.Warn("list practice interactions", "practice", practice.ID, "error", err)
		} else {
			data.PracticeInteractions = interactions[practice.ID]
		}
	}

	s.render(w, http.StatusOK, "practice.html", data)
}

// practiceEditForm exibe o formulário de edição de uma prática.
func (s *server) practiceEditForm(w http.ResponseWriter, r *http.Request) {
	account := accountFrom(r.Context())
	if account == nil {
		http.Error(w, "conta não encontrada", http.StatusUnauthorized)
		return
	}
	if s.practices == nil {
		http.Error(w, "biblioteca indisponível", http.StatusServiceUnavailable)
		return
	}

	practice, err := s.practices.FindPractice(r.Context(), account.Person.ID, chi.URLParam(r, "id"))
	if errors.Is(err, practices.ErrNotFound) {
		http.Error(w, "página não encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("find practice for edit", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !practice.IsOwner(account.Person.ID) {
		http.Error(w, "operação não permitida", http.StatusForbidden)
		return
	}

	s.renderPracticeEdit(w, r, http.StatusOK, "", *practice)
}

// practiceEdit valida e atualiza uma prática.
func (s *server) practiceEdit(w http.ResponseWriter, r *http.Request) {
	account := accountFrom(r.Context())
	if account == nil {
		http.Error(w, "conta não encontrada", http.StatusUnauthorized)
		return
	}
	if s.practices == nil {
		http.Error(w, "biblioteca indisponível", http.StatusServiceUnavailable)
		return
	}
	if !s.parseForm(w, r) {
		return
	}

	practiceID := chi.URLParam(r, "id")
	practice, err := s.practices.FindPractice(r.Context(), account.Person.ID, practiceID)
	if errors.Is(err, practices.ErrNotFound) {
		http.Error(w, "página não encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("find practice for update", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !practice.IsOwner(account.Person.ID) {
		http.Error(w, "operação não permitida", http.StatusForbidden)
		return
	}

	if !s.validateCSRF(r) {
		s.renderPracticeEdit(w, r, http.StatusForbidden, "Sessão do formulário expirada. Tente novamente.", *practice)
		return
	}

	input := practiceInputFromForm(r)
	form := practiceFormFromInput(input, practiceID)
	form.AuthorHandle = practice.AuthorHandle
	form.Version = practice.Version
	form.Status = practice.Status
	form.Visibility = practice.Visibility

	updated, err := s.practices.UpdatePractice(r.Context(), account.Person.ID, practiceID, input)
	if err != nil {
		if message, ok := validationMessage(err); ok {
			s.renderPracticeEdit(w, r, http.StatusBadRequest, message, form)
			return
		}
		s.logger.Error("update practice", "error", err)
		s.renderPracticeEdit(w, r, http.StatusInternalServerError, "Não foi possível atualizar a prática. Tente novamente.", form)
		return
	}

	if s.federation != nil {
		switch {
		case federatedPractice(updated):
			if err := s.federation.PublishPracticeUpdate(r.Context(), account.Person, *updated); err != nil {
				s.logger.Warn("update practice federation", "practice", updated.ID, "error", err)
			}
		case federatedPractice(practice):
			if err := s.federation.PublishPracticeDelete(r.Context(), account.Person, *practice); err != nil {
				s.logger.Warn("delete practice federation", "practice", practice.ID, "error", err)
			}
		}
	}

	http.Redirect(w, r, "/praticas/"+updated.ID, http.StatusSeeOther)
}

// practicePublish publica uma prática existente.
func (s *server) practicePublish(w http.ResponseWriter, r *http.Request) {
	account := accountFrom(r.Context())
	if account == nil {
		http.Error(w, "conta não encontrada", http.StatusUnauthorized)
		return
	}
	if s.practices == nil {
		http.Error(w, "biblioteca indisponível", http.StatusServiceUnavailable)
		return
	}
	if !s.parseForm(w, r) {
		return
	}
	if !s.validateCSRF(r) {
		http.Error(w, "sessão do formulário expirada", http.StatusForbidden)
		return
	}

	practiceID := chi.URLParam(r, "id")
	practice, err := s.practices.PublishPractice(r.Context(), account.Person.ID, practiceID)
	if err != nil {
		if message, ok := validationMessage(err); ok {
			current, findErr := s.practices.FindPractice(r.Context(), account.Person.ID, practiceID)
			if findErr != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			s.renderPracticeEdit(w, r, http.StatusBadRequest, message, *current)
			return
		}
		if errors.Is(err, practices.ErrForbidden) {
			http.Error(w, "operação não permitida", http.StatusForbidden)
			return
		}
		if errors.Is(err, practices.ErrNotFound) {
			http.Error(w, "página não encontrada", http.StatusNotFound)
			return
		}
		s.logger.Error("publish practice", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if s.federation != nil {
		if err := s.federation.PublishPracticeUpdate(r.Context(), account.Person, *practice); err != nil {
			s.logger.Warn("publish practice federation", "practice", practice.ID, "error", err)
		}
	}

	http.Redirect(w, r, "/praticas/"+practice.ID, http.StatusSeeOther)
}

// practiceArchive arquiva uma prática da pessoa autenticada.
func (s *server) practiceArchive(w http.ResponseWriter, r *http.Request) {
	account := accountFrom(r.Context())
	if account == nil {
		http.Error(w, "conta não encontrada", http.StatusUnauthorized)
		return
	}
	if s.practices == nil {
		http.Error(w, "biblioteca indisponível", http.StatusServiceUnavailable)
		return
	}
	if !s.parseForm(w, r) {
		return
	}
	if !s.validateCSRF(r) {
		http.Error(w, "sessão do formulário expirada", http.StatusForbidden)
		return
	}

	practice, err := s.practices.ArchivePractice(r.Context(), account.Person.ID, chi.URLParam(r, "id"))
	if errors.Is(err, practices.ErrForbidden) {
		http.Error(w, "operação não permitida", http.StatusForbidden)
		return
	}
	if errors.Is(err, practices.ErrNotFound) {
		http.Error(w, "página não encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("archive practice", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if s.federation != nil {
		if err := s.federation.PublishPracticeDelete(r.Context(), account.Person, *practice); err != nil {
			s.logger.Warn("archive practice federation", "practice", practice.ID, "error", err)
		}
	}

	http.Redirect(w, r, "/praticas/"+practice.ID, http.StatusSeeOther)
}

func (s *server) renderPracticeNew(w http.ResponseWriter, r *http.Request, status int, message string, form practices.Practice) {
	if form.Difficulty == "" {
		form.Difficulty = practices.DifficultyBeginner
	}

	data := s.newPage(w, r)
	data.Error = message
	data.FormPractice = form
	data.MaterialsText = linesText(form.Materials)
	data.StepsText = linesText(form.Steps)
	data.IsNew = true
	data.CanEdit = true
	s.render(w, status, "practice_new.html", data)
}

func (s *server) renderPracticeEdit(w http.ResponseWriter, r *http.Request, status int, message string, form practices.Practice) {
	data := s.newPage(w, r)
	data.Error = message
	data.FormPractice = form
	data.MaterialsText = linesText(form.Materials)
	data.StepsText = linesText(form.Steps)
	data.CanEdit = true
	s.render(w, status, "practice_edit.html", data)
}

func practiceInputFromForm(r *http.Request) practices.Input {
	return practices.Input{
		Title:        r.PostFormValue("title"),
		Summary:      r.PostFormValue("summary"),
		Category:     r.PostFormValue("category"),
		Difficulty:   practices.Difficulty(r.PostFormValue("difficulty")),
		TimeEstimate: r.PostFormValue("time_estimate"),
		CostEstimate: r.PostFormValue("cost_estimate"),
		Materials:    parseLines(r.PostFormValue("materials")),
		Steps:        parseLines(r.PostFormValue("steps")),
		Safety:       r.PostFormValue("safety"),
		Adaptations:  r.PostFormValue("adaptations"),
		References:   r.PostFormValue("references"),
		License:      r.PostFormValue("license"),
		Publish:      r.PostFormValue("action") == "publish",
	}
}

// federatedPractice informa se a prática pode ser publicada no Fediverso.
func federatedPractice(practice *practices.Practice) bool {
	return practice != nil &&
		practice.Status == practices.StatusPublished &&
		practice.Visibility == practices.VisibilityPublic
}

func practiceFormFromInput(input practices.Input, id string) practices.Practice {
	return practices.Practice{
		ID:           id,
		Title:        input.Title,
		Summary:      input.Summary,
		Category:     input.Category,
		Difficulty:   input.Difficulty,
		TimeEstimate: input.TimeEstimate,
		CostEstimate: input.CostEstimate,
		Materials:    input.Materials,
		Steps:        input.Steps,
		Safety:       input.Safety,
		Adaptations:  input.Adaptations,
		References:   input.References,
		License:      input.License,
	}
}
