package federation

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	// followerProfileTTL define por quanto tempo os dados públicos de um perfil
	// remoto ficam em cache na memória do processo.
	followerProfileTTL = time.Hour
	// followerFetchConcurrency limita quantas buscas de perfil acontecem ao
	// mesmo tempo para não sobrecarregar instâncias remotas.
	followerFetchConcurrency = 8
)

// Follower representa um seguidor remoto com os dados públicos exibíveis.
//
// Handle, DisplayName e AvatarURL podem ficar vazios quando a instância remota
// não responde; nesse caso a interface mostra apenas o identificador do Actor.
type Follower struct {
	ActorID     string
	Handle      string
	Domain      string
	DisplayName string
	ProfileURL  string
	AvatarURL   string
}

type cachedFollower struct {
	follower Follower
	expires  time.Time
}

// FollowerProfiles lista os seguidores aceitos e complementa cada um com os
// dados públicos do Actor remoto.
//
// A busca é feita em paralelo e usa um cache em memória. Falhas individuais não
// interrompem a listagem: o seguidor aparece apenas com o identificador.
func (s *Service) FollowerProfiles(ctx context.Context, personID string) ([]Follower, error) {
	follows, err := s.store.ListFollowers(ctx, personID)
	if err != nil {
		return nil, err
	}

	followers := make([]Follower, len(follows))
	for i, follow := range follows {
		followers[i] = followerFromActorID(follow.ActorID)
	}

	workers := followerFetchConcurrency
	if len(follows) < workers {
		workers = len(follows)
	}

	work := make(chan int)
	go func() {
		defer close(work)
		for i := range follows {
			select {
			case work <- i:
			case <-ctx.Done():
				return
			}
		}
	}()

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range work {
				actorID := follows[index].ActorID

				if profile, ok := s.cachedFollower(actorID); ok {
					followers[index] = profile
					continue
				}

				actor, err := fetchActorDocument(ctx, s.client, actorID)
				if err != nil {
					if ctx.Err() == nil {
						s.logger.Warn("fetch follower profile", "actor", actorID, "error", err)
					}
					continue
				}

				profile := followerFromActor(actor)
				s.cacheFollower(actorID, profile)
				followers[index] = profile
			}
		}()
	}

	wg.Wait()

	return followers, nil
}

// cachedFollower devolve o perfil guardado, quando ainda válido.
func (s *Service) cachedFollower(actorID string) (Follower, bool) {
	s.profileMu.Lock()
	defer s.profileMu.Unlock()

	entry, ok := s.profiles[actorID]
	if !ok || s.now().After(entry.expires) {
		return Follower{}, false
	}

	return entry.follower, true
}

// cacheFollower guarda os dados públicos de um perfil remoto.
func (s *Service) cacheFollower(actorID string, follower Follower) {
	s.profileMu.Lock()
	defer s.profileMu.Unlock()

	if s.profiles == nil {
		s.profiles = make(map[string]cachedFollower)
	}

	s.profiles[actorID] = cachedFollower{
		follower: follower,
		expires:  s.now().Add(followerProfileTTL),
	}
}

// followerFromActorID monta o seguidor apenas com os dados do identificador.
func followerFromActorID(actorID string) Follower {
	follower := Follower{
		ActorID:    actorID,
		ProfileURL: actorID,
	}

	if parsed, err := url.Parse(actorID); err == nil {
		follower.Domain = parsed.Host
	}

	return follower
}

// followerFromActor extrai os campos exibíveis do documento Actor remoto.
func followerFromActor(actor *RemoteActor) Follower {
	follower := followerFromActorID(actor.ID)
	follower.Handle = strings.TrimSpace(actor.PreferredUsername)
	follower.DisplayName = strings.TrimSpace(actor.Name)

	if profileURL := safeRemoteURL(actor.URL); profileURL != "" {
		follower.ProfileURL = profileURL
	}

	if avatarURL := safeRemoteURL(actor.Icon.URL); avatarURL != "" {
		follower.AvatarURL = avatarURL
	} else if avatarURL := safeRemoteURL(actor.Image.URL); avatarURL != "" {
		follower.AvatarURL = avatarURL
	}

	return follower
}

// safeRemoteURL aceita apenas endereços HTTP(S) completos, evitando esquemas
// inesperados vindos de instâncias remotas.
func safeRemoteURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	if parsed.Host == "" {
		return ""
	}

	return parsed.String()
}
