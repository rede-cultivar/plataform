package web

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/rede-cultivar/plataform/internal/cultivation"
	"github.com/rede-cultivar/plataform/internal/media"
)

// maxEntryUploadBytes limita o corpo do formulário de registro com fotos.
const maxEntryUploadBytes = maxFormBytes + media.MaxPhotosPerEntry*media.MaxBytes + (1 << 20)

// parseMultipartForm lê o formulário aceitando arquivos, com limite próprio.
//
// Formulários sem arquivos continuam aceitos no formato urlencoded.
func (s *server) parseMultipartForm(w http.ResponseWriter, r *http.Request, limit int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, limit)

	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "formulário inválido", http.StatusBadRequest)
			return false
		}
		return true
	}

	if err := r.ParseMultipartForm(maxFormBytes); err != nil {
		http.Error(w, "formulário inválido", http.StatusBadRequest)
		return false
	}
	return true
}

// photoFiles devolve as fotos efetivamente enviadas no formulário.
func photoFiles(r *http.Request) []*multipart.FileHeader {
	if r.MultipartForm == nil {
		return nil
	}

	headers := r.MultipartForm.File["photos"]
	files := make([]*multipart.FileHeader, 0, len(headers))
	for _, header := range headers {
		if header.Filename == "" {
			continue
		}
		files = append(files, header)
	}
	return files
}

// mediaMessage traduz um erro de validação de imagem para a interface.
func mediaMessage(err error) (string, bool) {
	var validation *media.ValidationError
	if errors.As(err, &validation) {
		return validation.Message, true
	}
	return "", false
}

// mediaAsset serve o conteúdo de uma imagem autorizada.
//
// A verificação de visibilidade acontece no serviço de mídia, que considera a
// pessoa responsável e os vínculos com registros e práticas públicas.
func (s *server) mediaAsset(w http.ResponseWriter, r *http.Request) {
	if s.media == nil {
		http.Error(w, "imagem não encontrada", http.StatusNotFound)
		return
	}

	viewerID := ""
	if account := accountFrom(r.Context()); account != nil {
		viewerID = account.Person.ID
	}

	id := chi.URLParam(r, "id")

	allowed, err := s.media.CanView(r.Context(), id, viewerID)
	if err != nil {
		s.logger.Error("check media visibility", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "imagem não encontrada", http.StatusNotFound)
		return
	}

	asset, reader, err := s.media.Open(r.Context(), id)
	if errors.Is(err, media.ErrNotFound) {
		http.Error(w, "imagem não encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("open media", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", asset.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(asset.ByteSize, 10))
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Cache-Control", "private, max-age=300")

	if _, err := io.Copy(w, reader); err != nil {
		s.logger.Warn("stream media", "error", err)
	}
}

// entryPhotoDelete remove uma foto vinculada a um registro.
//
// A rota exige sessão, CSRF e confirma que a pessoa é responsável pelo ciclo.
// A mídia precisa estar vinculada ao registro informado, o que evita apagar
// uma imagem alheia apenas informando outro identificador.
func (s *server) entryPhotoDelete(w http.ResponseWriter, r *http.Request) {
	account := accountFrom(r.Context())
	if account == nil {
		http.Error(w, "conta não encontrada", http.StatusUnauthorized)
		return
	}
	if s.cultivation == nil || s.media == nil {
		http.Error(w, "diário indisponível", http.StatusServiceUnavailable)
		return
	}
	if !s.parseMultipartForm(w, r, maxFormBytes) {
		return
	}
	if !s.validateCSRF(r) {
		http.Error(w, "sessão expirada", http.StatusForbidden)
		return
	}

	cycleID := chi.URLParam(r, "cycleID")
	entryID := chi.URLParam(r, "entryID")
	mediaID := chi.URLParam(r, "mediaID")

	cycle, _, err := s.cultivation.FindCycle(r.Context(), account.Person.ID, cycleID)
	if errors.Is(err, cultivation.ErrNotFound) {
		http.Error(w, "página não encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("find cycle for photo delete", "error", err)
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
		s.logger.Error("find entry for photo delete", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if entry.CycleID != cycle.ID {
		http.Error(w, "operação não permitida", http.StatusForbidden)
		return
	}

	attached, err := s.media.ListByEntryIDs(r.Context(), []string{entryID})
	if err != nil {
		s.logger.Error("list entry media for photo delete", "entry", entryID, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	found := false
	for _, asset := range attached[entryID] {
		if asset.ID == mediaID {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "foto não encontrada", http.StatusNotFound)
		return
	}

	if err := s.media.Delete(r.Context(), account.Person.ID, mediaID); err != nil {
		if errors.Is(err, media.ErrNotFound) {
			http.Error(w, "foto não encontrada", http.StatusNotFound)
			return
		}
		s.logger.Error("delete entry photo", "entry", entryID, "media", mediaID, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/diario/"+cycleID+"#registro-"+entryID, http.StatusSeeOther)
}
