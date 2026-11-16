package web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/rede-cultivar/plataform/internal/cultivation"
	"github.com/rede-cultivar/plataform/internal/federation"
	"github.com/rede-cultivar/plataform/internal/identity"
	"github.com/rede-cultivar/plataform/internal/media"
	"github.com/rede-cultivar/plataform/internal/version"
)

// federationURLs reúne os endereços públicos usados nos documentos federados.
func (s *server) federationURLs() federation.URLs {
	return federation.NewURLs(s.cfg.BaseURL())
}

// webfinger resolve acct:identificador@domínio para o Actor correspondente.
func (s *server) webfinger(w http.ResponseWriter, r *http.Request) {
	if s.identity == nil {
		http.Error(w, "recurso não encontrado", http.StatusNotFound)
		return
	}

	resource := strings.TrimSpace(r.URL.Query().Get("resource"))
	if resource == "" {
		http.Error(w, "parâmetro resource é obrigatório", http.StatusBadRequest)
		return
	}

	handle, host, ok := federation.ParseAcctResource(resource)
	if !ok {
		http.Error(w, "recurso inválido", http.StatusBadRequest)
		return
	}

	urls := s.federationURLs()
	if host != urls.Host() || identity.ValidateHandle(handle) != nil {
		http.Error(w, "recurso não encontrado", http.StatusNotFound)
		return
	}

	person, err := s.identity.FindPersonByHandle(r.Context(), handle)
	if errors.Is(err, identity.ErrNotFound) {
		http.Error(w, "recurso não encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("resolve webfinger", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSONType(w, http.StatusOK, federation.JRDContentType, federation.NewWebFinger(urls, urls.Host(), *person))
}

// nodeInfoIndex publica o documento de descoberta do NodeInfo.
func (s *server) nodeInfoIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	writeJSONType(w, http.StatusOK, federation.JRDContentType, federation.NewNodeInfoIndex(s.federationURLs()))
}

// nodeInfo publica as informações da instância no formato NodeInfo 2.0.
func (s *server) nodeInfo(w http.ResponseWriter, r *http.Request) {
	stats := identity.Stats{}
	settings, err := s.currentSiteSettings(r.Context())
	if err != nil {
		s.logger.Error("load site settings for nodeinfo", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if s.identity != nil {
		value, err := s.identity.Stats(r.Context())
		if err != nil {
			s.logger.Error("load nodeinfo stats", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		stats = value
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	writeJSONType(
		w,
		http.StatusOK,
		federation.NodeInfoContentType,
		federation.NewNodeInfo(s.cfg.InstanceName, version.Version, stats, settings.Open()),
	)
}

// actor publica a representação ActivityPub de uma pessoa.
func (s *server) actor(w http.ResponseWriter, r *http.Request) {
	person, ok := s.findActorPerson(w, r)
	if !ok {
		return
	}

	urls := s.federationURLs()
	actor := federation.NewPersonActor(urls, *person)

	if s.federation != nil {
		key, err := s.federation.ActorKey(r.Context(), person.ID)
		if err != nil {
			s.logger.Error("load actor key", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		actor.PublicKey = &federation.PublicKey{
			ID:           urls.Key(person.ID),
			Owner:        actor.ID,
			PublicKeyPEM: key.PublicPEM,
		}
	}

	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSONType(w, http.StatusOK, federation.ActivityContentType, actor)
}

// actorOutbox publica a caixa de saída do Actor.
func (s *server) actorOutbox(w http.ResponseWriter, r *http.Request) {
	person, ok := s.findActorPerson(w, r)
	if !ok {
		return
	}

	if s.federation != nil {
		collection, err := s.federation.Outbox(r.Context(), person.ID)
		if err != nil {
			s.logger.Error("load actor outbox", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		s.writeJSONCollection(w, collection)
		return
	}

	s.writeCollection(w, s.federationURLs().Outbox(person.ID))
}

// actorFollowers publica a coleção de seguidores do Actor.
func (s *server) actorFollowers(w http.ResponseWriter, r *http.Request) {
	person, ok := s.findActorPerson(w, r)
	if !ok {
		return
	}

	if s.federation != nil {
		collection, err := s.federation.Followers(r.Context(), person.ID)
		if err != nil {
			s.logger.Error("load actor followers", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		s.writeJSONCollection(w, collection)
		return
	}

	s.writeCollection(w, s.federationURLs().Followers(person.ID))
}

// actorFollowing publica a coleção de pessoas seguidas pelo Actor.
func (s *server) actorFollowing(w http.ResponseWriter, r *http.Request) {
	person, ok := s.findActorPerson(w, r)
	if !ok {
		return
	}

	s.writeCollection(w, s.federationURLs().Following(person.ID))
}

// accountFollowers exibe, na área autenticada, os seguidores remotos da pessoa.
func (s *server) accountFollowers(w http.ResponseWriter, r *http.Request) {
	account := accountFrom(r.Context())
	if account == nil {
		http.Error(w, "conta não encontrada", http.StatusUnauthorized)
		return
	}

	data := s.newPage(w, r)
	if s.federation == nil {
		data.Error = "A federação está indisponível."
		s.render(w, http.StatusServiceUnavailable, "account_followers.html", data)
		return
	}

	followers, err := s.federation.FollowerProfiles(r.Context(), account.Person.ID)
	if err != nil {
		s.logger.Error("load account followers", "error", err)
		data.Error = "Não foi possível carregar os seguidores."
		s.render(w, http.StatusInternalServerError, "account_followers.html", data)
		return
	}

	data.Followers = followers
	s.render(w, http.StatusOK, "account_followers.html", data)
}

// inbox valida e processa entregas federadas.
func (s *server) inbox(w http.ResponseWriter, r *http.Request) {
	person, ok := s.findActorPerson(w, r)
	if !ok {
		return
	}
	if s.federation == nil {
		http.Error(w, "federação indisponível", http.StatusServiceUnavailable)
		return
	}

	err := s.federation.HandleInbox(r.Context(), person.ID, r)
	switch {
	case errors.Is(err, federation.ErrMissingSignature), errors.Is(err, federation.ErrInvalidSignature):
		s.logger.Warn("reject inbox activity", "remote_addr", r.RemoteAddr, "error", err)
		http.Error(w, "assinatura inválida", http.StatusUnauthorized)
		return
	case errors.Is(err, federation.ErrInvalidActivity):
		http.Error(w, "atividade inválida", http.StatusBadRequest)
		return
	case err != nil:
		s.logger.Error("handle inbox activity", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// note publica a representação ActivityPub de um registro público.
func (s *server) note(w http.ResponseWriter, r *http.Request) {
	if s.cultivation == nil || s.identity == nil {
		http.Error(w, "nota não encontrada", http.StatusNotFound)
		return
	}

	entry, err := s.cultivation.FindEntry(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, cultivation.ErrNotFound) {
		http.Error(w, "nota não encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("find entry for note", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	cycle, _, err := s.cultivation.FindCycle(r.Context(), "", entry.CycleID)
	if err != nil || entry.EffectiveVisibility(cycle) != cultivation.VisibilityPublic {
		http.Error(w, "nota não encontrada", http.StatusNotFound)
		return
	}

	person, err := s.identity.FindPersonByID(r.Context(), entry.AuthorID)
	if errors.Is(err, identity.ErrNotFound) {
		http.Error(w, "nota não encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("find note author", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var assets []media.Asset
	if s.media != nil {
		values, err := s.media.ListByEntryIDs(r.Context(), []string{entry.ID})
		if err != nil {
			s.logger.Warn("load note media", "error", err)
		} else {
			assets = values[entry.ID]
		}
	}

	note := federation.NewEntryNote(s.federationURLs(), *person, *cycle, *entry, assets)

	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSONType(w, http.StatusOK, federation.ActivityContentType, note)
}

// activity devolve uma atividade pública já publicada.
func (s *server) activity(w http.ResponseWriter, r *http.Request) {
	if s.federation == nil {
		http.Error(w, "atividade não encontrada", http.StatusNotFound)
		return
	}

	activityID := s.federationURLs().Activity(chi.URLParam(r, "id"))
	payload, err := s.federation.FindActivity(r.Context(), activityID)
	if errors.Is(err, federation.ErrNotFound) {
		http.Error(w, "atividade não encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("find federation activity", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSONType(w, http.StatusOK, federation.ActivityContentType, payload)
}

// findActorPerson resolve o Actor pelo identificador opaco da rota.
func (s *server) findActorPerson(w http.ResponseWriter, r *http.Request) (*identity.Person, bool) {
	if s.identity == nil {
		http.Error(w, "ator não encontrado", http.StatusNotFound)
		return nil, false
	}

	person, err := s.identity.FindPersonByID(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, identity.ErrNotFound) {
		http.Error(w, "ator não encontrado", http.StatusNotFound)
		return nil, false
	}
	if err != nil {
		s.logger.Error("find actor", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return nil, false
	}

	return person, true
}

func (s *server) writeCollection(w http.ResponseWriter, id string) {
	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSONType(w, http.StatusOK, federation.ActivityContentType, federation.NewOrderedCollection(id))
}

func (s *server) writeJSONCollection(w http.ResponseWriter, collection federation.OrderedCollection) {
	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSONType(w, http.StatusOK, federation.ActivityContentType, collection)
}
