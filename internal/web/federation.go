package web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/rede-cultivar/plataform/internal/federation"
	"github.com/rede-cultivar/plataform/internal/identity"
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
		federation.NewNodeInfo(s.cfg.InstanceName, version.Version, stats),
	)
}

// actor publica a representação ActivityPub de uma pessoa.
func (s *server) actor(w http.ResponseWriter, r *http.Request) {
	person, ok := s.findActorPerson(w, r)
	if !ok {
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSONType(w, http.StatusOK, federation.ActivityContentType, federation.NewPersonActor(s.federationURLs(), *person))
}

// actorOutbox publica a caixa de saída do Actor.
func (s *server) actorOutbox(w http.ResponseWriter, r *http.Request) {
	person, ok := s.findActorPerson(w, r)
	if !ok {
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

// inbox responde às entregas federadas. O recebimento será implementado junto
// com a validação de assinaturas HTTP e o processamento assíncrono.
func (s *server) inbox(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.findActorPerson(w, r); !ok {
		return
	}

	writeJSON(w, http.StatusNotImplemented, map[string]string{
		"error": "recebimento federado ainda não implementado",
	})
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
