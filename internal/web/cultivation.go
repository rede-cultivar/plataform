package web

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/rede-cultivar/plataform/internal/cultivation"
	"github.com/rede-cultivar/plataform/internal/federation"
	"github.com/rede-cultivar/plataform/internal/identity"
	"github.com/rede-cultivar/plataform/internal/media"
	"github.com/rede-cultivar/plataform/internal/practices"
)

// replyErrorQuery identifica o motivo de uma resposta não publicada.
const replyErrorQuery = "erro_resposta"

// replyErrorMessage traduz o código da URL em uma mensagem para a pessoa.
func replyErrorMessage(code string) string {
	switch code {
	case "vazia":
		return "Escreva uma resposta antes de enviar."
	case "longa":
		return "A resposta deve ter no máximo 2000 caracteres."
	case "origem":
		return "A mensagem original não está mais disponível."
	case "publico":
		return "Só é possível responder a registros públicos."
	case "indisponivel":
		return "Não foi possível publicar a resposta. Tente novamente."
	default:
		return ""
	}
}

// entryAnchor devolve o endereço do registro, opcionalmente com um aviso.
func entryAnchor(cycleID, entryID, errorCode string) string {
	target := "/diario/" + cycleID
	if errorCode != "" {
		target += "?" + replyErrorQuery + "=" + errorCode
	}
	return target + "#registro-" + entryID
}

// diary exibe a descoberta pública de ciclos de cultivo.
func (s *server) diary(w http.ResponseWriter, r *http.Request) {
	if s.cultivation == nil {
		http.Error(w, "diário indisponível", http.StatusServiceUnavailable)
		return
	}

	filter := cultivation.NormalizeFilter(cultivation.CycleFilter{
		Species:     r.URL.Query().Get("species"),
		Environment: cultivation.Environment(r.URL.Query().Get("environment")),
		Stage:       cultivation.Stage(r.URL.Query().Get("stage")),
		Status:      cultivation.Status(r.URL.Query().Get("status")),
	})

	cycles, err := s.cultivation.ListPublicCycles(r.Context(), filter)
	if err != nil {
		s.logger.Error("list public cycles", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data := s.newPage(w, r)
	data.Cycles = cycles
	data.CycleFilter = filter

	s.render(w, http.StatusOK, "diary.html", data)
}

// myDiary exibe os ciclos da pessoa autenticada.
func (s *server) myDiary(w http.ResponseWriter, r *http.Request) {
	account := accountFrom(r.Context())
	if account == nil {
		http.Error(w, "conta não encontrada", http.StatusUnauthorized)
		return
	}
	if s.cultivation == nil {
		http.Error(w, "diário indisponível", http.StatusServiceUnavailable)
		return
	}

	cycles, err := s.cultivation.ListPersonCycles(r.Context(), account.Person.ID)
	if err != nil {
		s.logger.Error("list person cycles", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data := s.newPage(w, r)
	data.Cycles = cycles

	s.render(w, http.StatusOK, "diary_mine.html", data)
}

// cycleNewForm exibe o formulário de criação de um ciclo.
func (s *server) cycleNewForm(w http.ResponseWriter, r *http.Request) {
	form := cultivation.Cycle{
		Environment: cultivation.EnvironmentOutdoor,
		Stage:       cultivation.StageGermination,
		Visibility:  cultivation.VisibilityPrivate,
		StartedOn:   time.Now(),
	}
	s.renderCycleNew(w, r, http.StatusOK, "", form)
}

// cycleCreate valida e cria um ciclo.
func (s *server) cycleCreate(w http.ResponseWriter, r *http.Request) {
	account := accountFrom(r.Context())
	if account == nil {
		http.Error(w, "conta não encontrada", http.StatusUnauthorized)
		return
	}
	if s.cultivation == nil {
		http.Error(w, "diário indisponível", http.StatusServiceUnavailable)
		return
	}
	if !s.parseForm(w, r) {
		return
	}
	if !s.validateCSRF(r) {
		s.renderCycleNew(w, r, http.StatusForbidden, "Sessão do formulário expirada. Tente novamente.", cultivation.Cycle{})
		return
	}

	input, form, err := cycleInputFromForm(r)
	if err != nil {
		s.renderCycleNew(w, r, http.StatusBadRequest, "Informe uma data de início válida.", form)
		return
	}

	cycle, err := s.cultivation.CreateCycle(r.Context(), account.Person.ID, input)
	if err != nil {
		if message, ok := validationMessage(err); ok {
			s.renderCycleNew(w, r, http.StatusBadRequest, message, form)
			return
		}
		s.logger.Error("create cycle", "error", err)
		s.renderCycleNew(w, r, http.StatusInternalServerError, "Não foi possível criar o ciclo. Tente novamente.", form)
		return
	}

	http.Redirect(w, r, "/diario/"+cycle.ID, http.StatusSeeOther)
}

// cycleShow exibe um ciclo e seus registros visíveis.
func (s *server) cycleShow(w http.ResponseWriter, r *http.Request) {
	if s.cultivation == nil {
		http.Error(w, "diário indisponível", http.StatusServiceUnavailable)
		return
	}

	viewerID := ""
	if account := accountFrom(r.Context()); account != nil {
		viewerID = account.Person.ID
	}

	cycle, entries, err := s.cultivation.FindCycle(r.Context(), viewerID, chi.URLParam(r, "id"))
	if errors.Is(err, cultivation.ErrNotFound) {
		http.Error(w, "página não encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("find cycle", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data := s.newPage(w, r)
	data.Cycle = cycle
	data.Entries = entries
	data.CanEdit = cycle.IsOwner(viewerID)
	data.Error = replyErrorMessage(r.URL.Query().Get(replyErrorQuery))

	if s.media != nil && len(entries) > 0 {
		ids := make([]string, len(entries))
		for i, entry := range entries {
			ids[i] = entry.ID
		}

		assets, err := s.media.ListByEntryIDs(r.Context(), ids)
		if err != nil {
			s.logger.Warn("list entry media", "error", err)
		} else {
			data.EntryMedia = assets
		}
	}

	if s.federation != nil && len(entries) > 0 {
		ids := make([]string, len(entries))
		for i, entry := range entries {
			ids[i] = entry.ID
		}

		interactions, err := s.federation.EntryInteractions(r.Context(), ids)
		if err != nil {
			s.logger.Warn("list entry interactions", "error", err)
		} else {
			data.Interactions = interactions
		}
	}

	s.render(w, http.StatusOK, "cycle.html", data)
}

// cycleEditForm exibe o formulário de edição de um ciclo.
func (s *server) cycleEditForm(w http.ResponseWriter, r *http.Request) {
	account := accountFrom(r.Context())
	if account == nil {
		http.Error(w, "conta não encontrada", http.StatusUnauthorized)
		return
	}
	if s.cultivation == nil {
		http.Error(w, "diário indisponível", http.StatusServiceUnavailable)
		return
	}

	cycle, _, err := s.cultivation.FindCycle(r.Context(), account.Person.ID, chi.URLParam(r, "id"))
	if errors.Is(err, cultivation.ErrNotFound) {
		http.Error(w, "página não encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("find cycle for edit", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !cycle.IsOwner(account.Person.ID) {
		http.Error(w, "operação não permitida", http.StatusForbidden)
		return
	}

	s.renderCycleEdit(w, r, http.StatusOK, "", *cycle)
}

// cycleEdit valida e atualiza um ciclo.
func (s *server) cycleEdit(w http.ResponseWriter, r *http.Request) {
	account := accountFrom(r.Context())
	if account == nil {
		http.Error(w, "conta não encontrada", http.StatusUnauthorized)
		return
	}
	if s.cultivation == nil {
		http.Error(w, "diário indisponível", http.StatusServiceUnavailable)
		return
	}
	if !s.parseForm(w, r) {
		return
	}

	cycleID := chi.URLParam(r, "id")
	cycle, entries, err := s.cultivation.FindCycle(r.Context(), account.Person.ID, cycleID)
	if errors.Is(err, cultivation.ErrNotFound) {
		http.Error(w, "página não encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("find cycle for update", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !cycle.IsOwner(account.Person.ID) {
		http.Error(w, "operação não permitida", http.StatusForbidden)
		return
	}

	if !s.validateCSRF(r) {
		s.renderCycleEdit(w, r, http.StatusForbidden, "Sessão do formulário expirada. Tente novamente.", *cycle)
		return
	}

	input, form, err := cycleUpdateInputFromForm(r)
	if err != nil {
		form.ID = cycle.ID
		form.PersonHandle = cycle.PersonHandle
		s.renderCycleEdit(w, r, http.StatusBadRequest, "Informe datas válidas.", form)
		return
	}

	form.ID = cycle.ID
	form.PersonHandle = cycle.PersonHandle

	before := *cycle

	updated, err := s.cultivation.UpdateCycle(r.Context(), account.Person.ID, cycleID, input)
	if err != nil {
		if message, ok := validationMessage(err); ok {
			s.renderCycleEdit(w, r, http.StatusBadRequest, message, form)
			return
		}
		s.logger.Error("update cycle", "error", err)
		s.renderCycleEdit(w, r, http.StatusInternalServerError, "Não foi possível atualizar o ciclo. Tente novamente.", form)
		return
	}

	s.reconcileCycleFederation(r.Context(), account.Person, before, *updated, entries)

	http.Redirect(w, r, "/diario/"+updated.ID, http.StatusSeeOther)
}

// reconcileCycleFederation mantém as publicações alinhadas após editar um ciclo.
//
// Registros que deixaram de ser públicos recebem Delete. Quando o ciclo passa a
// ser público ou quando muda um campo usado no cartão, as publicações são
// redistribuídas com Update; registros ainda não federados são publicados pela
// primeira vez.
func (s *server) reconcileCycleFederation(ctx context.Context, person identity.Person, before, after cultivation.Cycle, entries []cultivation.Entry) {
	if s.federation == nil {
		return
	}

	cardChanged := cycleCardChanged(before, after)
	becamePublic := before.Visibility != cultivation.VisibilityPublic &&
		after.Visibility == cultivation.VisibilityPublic

	for _, entry := range entries {
		if entry.EffectiveVisibility(&after) == cultivation.VisibilityPublic {
			if cardChanged || becamePublic {
				s.publishEntryUpdate(ctx, person, after, entry)
			}
			continue
		}
		if entry.EffectiveVisibility(&before) == cultivation.VisibilityPublic {
			s.publishEntryDelete(ctx, person, after, entry)
		}
	}
}

// cycleCardChanged informa se a edição alterou campos usados no cartão federado.
//
// Estágio, status, substrato e demais campos internos não entram na publicação,
// então não justificam reenviar todas as Notes do ciclo.
func cycleCardChanged(before, after cultivation.Cycle) bool {
	return before.Name != after.Name ||
		before.Species != after.Species ||
		before.Variety != after.Variety ||
		before.Environment != after.Environment ||
		before.Location != after.Location ||
		before.StartedOn.Format(dateLayout) != after.StartedOn.Format(dateLayout)
}

// entryNewForm exibe o formulário de criação de um registro.
func (s *server) entryNewForm(w http.ResponseWriter, r *http.Request) {
	account := accountFrom(r.Context())
	if account == nil {
		http.Error(w, "conta não encontrada", http.StatusUnauthorized)
		return
	}
	if s.cultivation == nil {
		http.Error(w, "diário indisponível", http.StatusServiceUnavailable)
		return
	}

	cycle, _, err := s.cultivation.FindCycle(r.Context(), account.Person.ID, chi.URLParam(r, "id"))
	if errors.Is(err, cultivation.ErrNotFound) {
		http.Error(w, "página não encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("find cycle for entry", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !cycle.IsOwner(account.Person.ID) {
		http.Error(w, "operação não permitida", http.StatusForbidden)
		return
	}

	form := cultivation.Entry{
		OccurredAt: time.Now(),
		Visibility: cultivation.EntryVisibilityInherit,
	}

	s.renderEntryNew(w, r, http.StatusOK, "", cycle, form)
}

// entryCreate valida e cria um registro em um ciclo.
func (s *server) entryCreate(w http.ResponseWriter, r *http.Request) {
	account := accountFrom(r.Context())
	if account == nil {
		http.Error(w, "conta não encontrada", http.StatusUnauthorized)
		return
	}
	if s.cultivation == nil {
		http.Error(w, "diário indisponível", http.StatusServiceUnavailable)
		return
	}
	if !s.parseMultipartForm(w, r, maxEntryUploadBytes) {
		return
	}

	cycleID := chi.URLParam(r, "id")
	cycle, _, err := s.cultivation.FindCycle(r.Context(), account.Person.ID, cycleID)
	if errors.Is(err, cultivation.ErrNotFound) {
		http.Error(w, "página não encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("find cycle for new entry", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !cycle.IsOwner(account.Person.ID) {
		http.Error(w, "operação não permitida", http.StatusForbidden)
		return
	}

	if !s.validateCSRF(r) {
		form := cultivation.Entry{OccurredAt: time.Now(), Visibility: cultivation.EntryVisibilityInherit}
		s.renderEntryNew(w, r, http.StatusForbidden, "Sessão do formulário expirada. Tente novamente.", cycle, form)
		return
	}

	input, form, err := entryInputFromForm(r)
	if err != nil {
		s.renderEntryNew(w, r, http.StatusBadRequest, "Informe uma data e uma quantidade válidas.", cycle, form)
		return
	}

	files := photoFiles(r)
	if len(files) > media.MaxPhotosPerEntry {
		s.renderEntryNew(w, r, http.StatusBadRequest, "Envie no máximo 4 fotos por registro.", cycle, form)
		return
	}
	if len(files) > 0 && s.media == nil {
		s.renderEntryNew(w, r, http.StatusServiceUnavailable, "O envio de fotos está indisponível.", cycle, form)
		return
	}

	uploaded := make([]*media.Asset, 0, len(files))
	cleanup := func() {
		for _, asset := range uploaded {
			if err := s.media.Delete(r.Context(), account.Person.ID, asset.ID); err != nil {
				s.logger.Warn("cleanup entry photo", "media", asset.ID, "error", err)
			}
		}
	}

	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			cleanup()
			s.renderEntryNew(w, r, http.StatusBadRequest, "Não foi possível ler a imagem enviada.", cycle, form)
			return
		}

		asset, uploadErr := s.media.Upload(r.Context(), account.Person.ID, header.Filename, file)
		file.Close()
		if uploadErr != nil {
			cleanup()
			if message, ok := mediaMessage(uploadErr); ok {
				s.renderEntryNew(w, r, http.StatusBadRequest, message, cycle, form)
				return
			}
			s.logger.Error("upload entry photo", "error", uploadErr)
			s.renderEntryNew(w, r, http.StatusInternalServerError, "Não foi possível salvar a foto. Tente novamente.", cycle, form)
			return
		}

		uploaded = append(uploaded, asset)
	}

	entry, err := s.cultivation.CreateEntry(r.Context(), account.Person.ID, cycleID, input)
	if err != nil {
		cleanup()
		if message, ok := validationMessage(err); ok {
			s.renderEntryNew(w, r, http.StatusBadRequest, message, cycle, form)
			return
		}
		s.logger.Error("create entry", "error", err)
		s.renderEntryNew(w, r, http.StatusInternalServerError, "Não foi possível salvar o registro. Tente novamente.", cycle, form)
		return
	}

	attached := make([]media.Asset, 0, len(uploaded))
	for position, asset := range uploaded {
		if err := s.media.AttachToEntry(r.Context(), entry.ID, asset.ID, position); err != nil {
			s.logger.Error("attach entry photo", "entry", entry.ID, "media", asset.ID, "error", err)
			continue
		}
		attached = append(attached, *asset)
	}

	if s.federation != nil && entry.EffectiveVisibility(cycle) == cultivation.VisibilityPublic {
		if err := s.federation.PublishEntry(r.Context(), account.Person, *cycle, *entry, attached); err != nil {
			s.logger.Warn("publish cultivation entry", "entry", entry.ID, "error", err)
		}
	}

	http.Redirect(w, r, "/diario/"+cycleID+"#registro-"+entry.ID, http.StatusSeeOther)
}

func (s *server) entryEditForm(w http.ResponseWriter, r *http.Request) {
	account := accountFrom(r.Context())
	if account == nil {
		http.Error(w, "conta não encontrada", http.StatusUnauthorized)
		return
	}
	if s.cultivation == nil {
		http.Error(w, "diário indisponível", http.StatusServiceUnavailable)
		return
	}

	cycleID := chi.URLParam(r, "cycleID")
	cycle, _, err := s.cultivation.FindCycle(r.Context(), account.Person.ID, cycleID)
	if errors.Is(err, cultivation.ErrNotFound) {
		http.Error(w, "página não encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("find cycle for entry edit", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !cycle.IsOwner(account.Person.ID) {
		http.Error(w, "operação não permitida", http.StatusForbidden)
		return
	}

	entryID := chi.URLParam(r, "entryID")
	entry, err := s.cultivation.FindEntry(r.Context(), entryID)
	if errors.Is(err, cultivation.ErrNotFound) {
		http.Error(w, "registro não encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("find entry for edit", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if entry.CycleID != cycle.ID {
		http.Error(w, "operação não permitida", http.StatusForbidden)
		return
	}

	s.renderEntryEdit(w, r, http.StatusOK, "", cycle, *entry)
}

func (s *server) entryEdit(w http.ResponseWriter, r *http.Request) {
	account := accountFrom(r.Context())
	if account == nil {
		http.Error(w, "conta não encontrada", http.StatusUnauthorized)
		return
	}
	if s.cultivation == nil {
		http.Error(w, "diário indisponível", http.StatusServiceUnavailable)
		return
	}
	if !s.parseMultipartForm(w, r, maxEntryUploadBytes) {
		return
	}
	if !s.validateCSRF(r) {
		cycleID := chi.URLParam(r, "cycleID")
		cycle, _, err := s.cultivation.FindCycle(r.Context(), account.Person.ID, cycleID)
		if err == nil {
			entry, _ := s.cultivation.FindEntry(r.Context(), chi.URLParam(r, "entryID"))
			if entry != nil {
				s.renderEntryEdit(w, r, http.StatusForbidden, "Sessão do formulário expirada. Tente novamente.", cycle, *entry)
				return
			}
		}
		http.Error(w, "sessão expirada", http.StatusForbidden)
		return
	}

	cycleID := chi.URLParam(r, "cycleID")
	entryID := chi.URLParam(r, "entryID")
	cycle, _, err := s.cultivation.FindCycle(r.Context(), account.Person.ID, cycleID)
	if errors.Is(err, cultivation.ErrNotFound) {
		http.Error(w, "página não encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("find cycle for entry update", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !cycle.IsOwner(account.Person.ID) {
		http.Error(w, "operação não permitida", http.StatusForbidden)
		return
	}

	entry, err := s.cultivation.FindEntry(r.Context(), entryID)
	if errors.Is(err, cultivation.ErrNotFound) {
		http.Error(w, "registro não encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("find entry for update", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if entry.CycleID != cycle.ID {
		http.Error(w, "operação não permitida", http.StatusForbidden)
		return
	}

	input, form, err := entryInputFromForm(r)
	if err != nil {
		s.renderEntryEdit(w, r, http.StatusBadRequest, "Informe uma data e uma quantidade válidas.", cycle, *entry)
		return
	}

	updated, err := s.cultivation.UpdateEntry(r.Context(), account.Person.ID, cycleID, entryID, input)
	if err != nil {
		if message, ok := validationMessage(err); ok {
			s.renderEntryEdit(w, r, http.StatusBadRequest, message, cycle, form)
			return
		}
		s.logger.Error("update entry", "error", err)
		s.renderEntryEdit(w, r, http.StatusInternalServerError, "Não foi possível salvar as alterações. Tente novamente.", cycle, form)
		return
	}

	if s.media != nil {
		currentIDs := make(map[string]struct{})
		deletedCount := 0
		if existing, err := s.media.ListByEntryIDs(r.Context(), []string{entryID}); err == nil {
			for _, assets := range existing {
				for _, asset := range assets {
					currentIDs[asset.ID] = struct{}{}
				}
			}
		} else {
			s.logger.Warn("load entry media before delete", "entry", entryID, "error", err)
		}

		for _, deleteID := range r.Form["delete_photo"] {
			if deleteID == "" {
				continue
			}
			if _, ok := currentIDs[deleteID]; !ok {
				continue
			}
			deletedCount++
			if err := s.media.Delete(r.Context(), account.Person.ID, deleteID); err != nil {
				s.logger.Warn("delete entry photo", "entry", entryID, "media", deleteID, "error", err)
			}
		}

		files := photoFiles(r)
		if len(currentIDs)-deletedCount+len(files) > media.MaxPhotosPerEntry {
			s.renderEntryEdit(w, r, http.StatusBadRequest, "Envie no máximo 4 fotos por registro.", cycle, *updated)
			return
		}
		if len(files) > 0 {
			uploaded := make([]*media.Asset, 0, len(files))
			cleanup := func() {
				for _, asset := range uploaded {
					if err := s.media.Delete(r.Context(), account.Person.ID, asset.ID); err != nil {
						s.logger.Warn("cleanup updated entry photo", "media", asset.ID, "error", err)
					}
				}
			}
			for _, header := range files {
				file, err := header.Open()
				if err != nil {
					cleanup()
					s.renderEntryEdit(w, r, http.StatusBadRequest, "Não foi possível ler a imagem enviada.", cycle, *updated)
					return
				}
				asset, uploadErr := s.media.Upload(r.Context(), account.Person.ID, header.Filename, file)
				file.Close()
				if uploadErr != nil {
					cleanup()
					if message, ok := mediaMessage(uploadErr); ok {
						s.renderEntryEdit(w, r, http.StatusBadRequest, message, cycle, *updated)
						return
					}
					s.logger.Error("upload updated entry photo", "error", uploadErr)
					s.renderEntryEdit(w, r, http.StatusInternalServerError, "Não foi possível salvar a foto. Tente novamente.", cycle, *updated)
					return
				}
				uploaded = append(uploaded, asset)
			}

			for position, asset := range uploaded {
				if err := s.media.AttachToEntry(r.Context(), entryID, asset.ID, position); err != nil {
					s.logger.Error("attach updated entry photo", "entry", entryID, "media", asset.ID, "error", err)
				}
			}
		}
	}

	if updated.EffectiveVisibility(cycle) == cultivation.VisibilityPublic {
		s.publishEntryUpdate(r.Context(), account.Person, *cycle, *updated)
	} else if entry.EffectiveVisibility(cycle) == cultivation.VisibilityPublic {
		s.publishEntryDelete(r.Context(), account.Person, *cycle, *updated)
	}

	http.Redirect(w, r, "/diario/"+cycleID+"#registro-"+updated.ID, http.StatusSeeOther)
}

// entryReply publica uma resposta a uma mensagem recebida no Fediverso.
//
// Só o dono do ciclo pode responder e apenas em registros ainda públicos. A
// resposta é persistida e entregue de forma assíncrona pelo worker.
func (s *server) entryReply(w http.ResponseWriter, r *http.Request) {
	account := accountFrom(r.Context())
	if account == nil {
		http.Error(w, "conta não encontrada", http.StatusUnauthorized)
		return
	}
	if s.cultivation == nil || s.federation == nil {
		http.Error(w, "respostas indisponíveis", http.StatusServiceUnavailable)
		return
	}
	if !s.parseForm(w, r) {
		return
	}

	cycleID := chi.URLParam(r, "cycleID")
	entryID := chi.URLParam(r, "entryID")

	cycle, _, err := s.cultivation.FindCycle(r.Context(), account.Person.ID, cycleID)
	if errors.Is(err, cultivation.ErrNotFound) {
		http.Error(w, "página não encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("find cycle for reply", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !cycle.IsOwner(account.Person.ID) {
		http.Error(w, "operação não permitida", http.StatusForbidden)
		return
	}
	if !s.validateCSRF(r) {
		http.Error(w, "Sessão do formulário expirada. Tente novamente.", http.StatusForbidden)
		return
	}

	entry, err := s.cultivation.FindEntry(r.Context(), entryID)
	if errors.Is(err, cultivation.ErrNotFound) {
		http.Error(w, "registro não encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("find entry for reply", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if entry.CycleID != cycle.ID {
		http.Error(w, "operação não permitida", http.StatusForbidden)
		return
	}
	if entry.EffectiveVisibility(cycle) != cultivation.VisibilityPublic {
		http.Redirect(w, r, entryAnchor(cycleID, entryID, "publico"), http.StatusSeeOther)
		return
	}

	content := strings.TrimSpace(r.PostFormValue("content"))
	if content == "" {
		http.Redirect(w, r, entryAnchor(cycleID, entryID, "vazia"), http.StatusSeeOther)
		return
	}
	if utf8.RuneCountInString(content) > federation.MaxReplyContent {
		http.Redirect(w, r, entryAnchor(cycleID, entryID, "longa"), http.StatusSeeOther)
		return
	}

	replyID := strings.TrimSpace(r.PostFormValue("reply_id"))
	_, err = s.federation.ReplyToInteraction(
		r.Context(), account.Person, *cycle, *entry, replyID, content,
	)
	switch {
	case errors.Is(err, federation.ErrNotFound):
		http.Redirect(w, r, entryAnchor(cycleID, entryID, "origem"), http.StatusSeeOther)
		return
	case errors.Is(err, federation.ErrInvalidReply):
		http.Redirect(w, r, entryAnchor(cycleID, entryID, "vazia"), http.StatusSeeOther)
		return
	case err != nil:
		s.logger.Warn("publish entry reply", "entry", entryID, "error", err)
		http.Redirect(w, r, entryAnchor(cycleID, entryID, "indisponivel"), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, entryAnchor(cycleID, entryID, ""), http.StatusSeeOther)
}

// publishEntryUpdate redistribui uma publicação após alterações no registro.
//
// A atividade Update só é enviada quando o registro é público. As fotos são
// recarregadas para refletir inclusões e remoções feitas no formulário.
func (s *server) publishEntryUpdate(ctx context.Context, person identity.Person, cycle cultivation.Cycle, entry cultivation.Entry) {
	if s.federation == nil || entry.EffectiveVisibility(&cycle) != cultivation.VisibilityPublic {
		return
	}

	var assets []media.Asset
	if s.media != nil {
		values, err := s.media.ListByEntryIDs(ctx, []string{entry.ID})
		if err != nil {
			s.logger.Warn("load entry media for federation update", "entry", entry.ID, "error", err)
		} else {
			assets = values[entry.ID]
		}
	}

	if err := s.federation.PublishEntryUpdate(ctx, person, cycle, entry, assets); err != nil {
		s.logger.Warn("publish cultivation entry update", "entry", entry.ID, "error", err)
	}
}

// publishEntryDelete remove do Fediverso uma publicação que deixou de ser
// pública, evitando que o registro continue visível em instâncias remotas.
func (s *server) publishEntryDelete(ctx context.Context, person identity.Person, cycle cultivation.Cycle, entry cultivation.Entry) {
	if s.federation == nil {
		return
	}

	if err := s.federation.PublishEntryDelete(ctx, person, cycle, entry); err != nil {
		s.logger.Warn("unpublish cultivation entry", "entry", entry.ID, "error", err)
	}
}

func (s *server) renderCycleNew(w http.ResponseWriter, r *http.Request, status int, message string, form cultivation.Cycle) {
	if form.StartedOn.IsZero() {
		form.StartedOn = time.Now()
	}
	if form.Environment == "" {
		form.Environment = cultivation.EnvironmentOutdoor
	}
	if form.Stage == "" {
		form.Stage = cultivation.StageGermination
	}
	if form.Visibility == "" {
		form.Visibility = cultivation.VisibilityPrivate
	}

	data := s.newPage(w, r)
	data.Error = message
	data.FormCycle = form
	data.IsNew = true
	s.render(w, status, "cycle_new.html", data)
}

func (s *server) renderCycleEdit(w http.ResponseWriter, r *http.Request, status int, message string, form cultivation.Cycle) {
	data := s.newPage(w, r)
	data.Error = message
	data.FormCycle = form
	data.Cycle = &form
	data.CanEdit = true
	s.render(w, status, "cycle_edit.html", data)
}

func (s *server) renderEntryNew(w http.ResponseWriter, r *http.Request, status int, message string, cycle *cultivation.Cycle, form cultivation.Entry) {
	data := s.newPage(w, r)
	data.Error = message
	data.Cycle = cycle
	data.FormEntry = form
	data.MeasurementsText = measurementsText(form.Measurements)
	data.InterventionsText = listText(form.Interventions)
	data.OccurrencesText = listText(form.Occurrences)
	data.Practices = s.publicPractices(r)
	s.render(w, status, "entry_new.html", data)
}

func (s *server) renderEntryEdit(w http.ResponseWriter, r *http.Request, status int, message string, cycle *cultivation.Cycle, form cultivation.Entry) {
	data := s.newPage(w, r)
	data.Error = message
	data.Cycle = cycle
	data.FormEntry = form
	data.MeasurementsText = measurementsText(form.Measurements)
	data.InterventionsText = listText(form.Interventions)
	data.OccurrencesText = listText(form.Occurrences)
	data.Practices = s.publicPractices(r)
	if s.media != nil {
		mediaByEntry, err := s.media.ListByEntryIDs(r.Context(), []string{form.ID})
		if err != nil {
			s.logger.Warn("list entry media for edit form", "entry", form.ID, "error", err)
		} else {
			data.EntryMedia = mediaByEntry
		}
	}
	s.render(w, status, "entry_edit.html", data)
}

func (s *server) publicPractices(r *http.Request) []practices.Practice {
	if s.practices == nil {
		return nil
	}

	list, err := s.practices.ListPublicPractices(r.Context(), practices.Filter{Limit: 100})
	if err != nil {
		s.logger.Warn("list public practices", "error", err)
		return nil
	}
	return list
}

func cycleInputFromForm(r *http.Request) (cultivation.CycleInput, cultivation.Cycle, error) {
	startedOn, err := parseDate(r.PostFormValue("started_on"))
	if err != nil {
		return cultivation.CycleInput{}, cultivation.Cycle{}, err
	}

	input := cultivation.CycleInput{
		Name:        r.PostFormValue("name"),
		Species:     r.PostFormValue("species"),
		Variety:     r.PostFormValue("variety"),
		Category:    r.PostFormValue("category"),
		Environment: cultivation.Environment(r.PostFormValue("environment")),
		Location:    r.PostFormValue("location"),
		Substrate:   r.PostFormValue("substrate"),
		Container:   r.PostFormValue("container"),
		Technique:   r.PostFormValue("technique"),
		Stage:       cultivation.Stage(r.PostFormValue("stage")),
		Visibility:  cultivation.Visibility(r.PostFormValue("visibility")),
		StartedOn:   startedOn,
	}

	form := cultivation.Cycle{
		Name:        input.Name,
		Species:     input.Species,
		Variety:     input.Variety,
		Category:    input.Category,
		Environment: input.Environment,
		Location:    input.Location,
		Substrate:   input.Substrate,
		Container:   input.Container,
		Technique:   input.Technique,
		Stage:       input.Stage,
		Visibility:  input.Visibility,
		StartedOn:   startedOn,
	}

	return input, form, nil
}

func cycleUpdateInputFromForm(r *http.Request) (cultivation.CycleUpdateInput, cultivation.Cycle, error) {
	startedOn, err := parseDate(r.PostFormValue("started_on"))
	if err != nil {
		return cultivation.CycleUpdateInput{}, cultivation.Cycle{}, err
	}
	finishedOn, err := parseOptionalDate(r.PostFormValue("finished_on"))
	if err != nil {
		return cultivation.CycleUpdateInput{}, cultivation.Cycle{}, err
	}

	input := cultivation.CycleUpdateInput{
		Name:        r.PostFormValue("name"),
		Species:     r.PostFormValue("species"),
		Variety:     r.PostFormValue("variety"),
		Category:    r.PostFormValue("category"),
		Environment: cultivation.Environment(r.PostFormValue("environment")),
		Location:    r.PostFormValue("location"),
		Substrate:   r.PostFormValue("substrate"),
		Container:   r.PostFormValue("container"),
		Technique:   r.PostFormValue("technique"),
		Stage:       cultivation.Stage(r.PostFormValue("stage")),
		Status:      cultivation.Status(r.PostFormValue("status")),
		Visibility:  cultivation.Visibility(r.PostFormValue("visibility")),
		StartedOn:   startedOn,
		FinishedOn:  finishedOn,
		Outcome:     r.PostFormValue("outcome"),
	}

	form := cultivation.Cycle{
		Name:        input.Name,
		Species:     input.Species,
		Variety:     input.Variety,
		Category:    input.Category,
		Environment: input.Environment,
		Location:    input.Location,
		Substrate:   input.Substrate,
		Container:   input.Container,
		Technique:   input.Technique,
		Stage:       input.Stage,
		Status:      input.Status,
		Visibility:  input.Visibility,
		StartedOn:   input.StartedOn,
		FinishedOn:  input.FinishedOn,
		Outcome:     input.Outcome,
	}

	return input, form, nil
}

func entryInputFromForm(r *http.Request) (cultivation.EntryInput, cultivation.Entry, error) {
	occurredAt, err := parseDateTime(r.PostFormValue("occurred_at"))
	if err != nil {
		return cultivation.EntryInput{}, cultivation.Entry{}, err
	}
	quantity, err := parseOptionalFloat(r.PostFormValue("harvest_quantity"))
	if err != nil {
		return cultivation.EntryInput{}, cultivation.Entry{}, err
	}

	input := cultivation.EntryInput{
		OccurredAt:      occurredAt,
		Body:            r.PostFormValue("body"),
		Stage:           parseOptionalStage(r.PostFormValue("stage")),
		Measurements:    parseMeasurements(r.PostFormValue("measurements")),
		Interventions:   parseList(r.PostFormValue("interventions")),
		Occurrences:     parseList(r.PostFormValue("occurrences")),
		HarvestQuantity: quantity,
		HarvestUnit:     r.PostFormValue("harvest_unit"),
		PracticeID:      r.PostFormValue("practice_id"),
		Visibility:      cultivation.EntryVisibility(r.PostFormValue("visibility")),
	}

	form := cultivation.Entry{
		OccurredAt:      input.OccurredAt,
		Body:            input.Body,
		Stage:           input.Stage,
		Measurements:    input.Measurements,
		Interventions:   input.Interventions,
		Occurrences:     input.Occurrences,
		HarvestQuantity: input.HarvestQuantity,
		HarvestUnit:     input.HarvestUnit,
		PracticeID:      input.PracticeID,
		Visibility:      input.Visibility,
	}

	return input, form, nil
}
