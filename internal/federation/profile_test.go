package federation

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testProfileService() *Service {
	return &Service{
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		now:      time.Now,
		profiles: make(map[string]cachedFollower),
	}
}

func TestFollowerFromActorExtractsProfile(t *testing.T) {
	actor := &RemoteActor{
		ID:                "https://remote.example/users/bob",
		PreferredUsername: "bob",
		Name:              "Bob Remoto",
		URL:               "https://remote.example/@bob",
	}
	actor.Icon.URL = "https://remote.example/avatars/bob.png"

	follower := followerFromActor(actor)

	if follower.ActorID != actor.ID {
		t.Errorf("ActorID = %q", follower.ActorID)
	}
	if follower.Handle != "bob" {
		t.Errorf("Handle = %q", follower.Handle)
	}
	if follower.Domain != "remote.example" {
		t.Errorf("Domain = %q", follower.Domain)
	}
	if follower.DisplayName != "Bob Remoto" {
		t.Errorf("DisplayName = %q", follower.DisplayName)
	}
	if follower.ProfileURL != "https://remote.example/@bob" {
		t.Errorf("ProfileURL = %q", follower.ProfileURL)
	}
	if follower.AvatarURL != "https://remote.example/avatars/bob.png" {
		t.Errorf("AvatarURL = %q", follower.AvatarURL)
	}
}

func TestFollowerFromActorFallsBackToActorID(t *testing.T) {
	actor := &RemoteActor{ID: "https://remote.example/users/bob"}

	follower := followerFromActor(actor)

	if follower.ProfileURL != actor.ID {
		t.Errorf("ProfileURL = %q, want actor ID", follower.ProfileURL)
	}
	if follower.AvatarURL != "" {
		t.Errorf("AvatarURL = %q, want empty", follower.AvatarURL)
	}
	if follower.Handle != "" {
		t.Errorf("Handle = %q, want empty", follower.Handle)
	}
}

func TestFollowerFromActorRejectsUnsafeURLs(t *testing.T) {
	actor := &RemoteActor{
		ID:  "https://remote.example/users/bob",
		URL: "javascript:alert(1)",
	}
	actor.Icon.URL = "data:image/png;base64,AAAA"

	follower := followerFromActor(actor)

	if follower.ProfileURL != actor.ID {
		t.Errorf("ProfileURL = %q, want actor ID", follower.ProfileURL)
	}
	if follower.AvatarURL != "" {
		t.Errorf("AvatarURL = %q, want empty", follower.AvatarURL)
	}
}

func TestFetchActorDocumentReadsProfileFields(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if accept := r.Header.Get("Accept"); !strings.Contains(accept, ActivityContentType) {
			t.Errorf("Accept = %q, want ActivityPub content type", accept)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":                server.URL + "/users/bob",
			"type":              "Person",
			"preferredUsername": "bob",
			"name":              "Bob Remoto",
			"url":               "https://remote.example/@bob",
			"icon":              map[string]any{"type": "Image", "url": "https://remote.example/avatars/bob.png"},
		})
	}))
	defer server.Close()

	actor, err := fetchActorDocument(context.Background(), NewHTTPClient(true), server.URL+"/users/bob")
	if err != nil {
		t.Fatalf("fetchActorDocument() error = %v", err)
	}

	if actor.ID != server.URL+"/users/bob" {
		t.Errorf("ID = %q", actor.ID)
	}
	if actor.PreferredUsername != "bob" || actor.Name != "Bob Remoto" {
		t.Errorf("actor = %+v", actor)
	}
	if actor.Icon.URL != "https://remote.example/avatars/bob.png" {
		t.Errorf("Icon.URL = %q", actor.Icon.URL)
	}
}

func TestFetchActorDocumentAcceptsStringIcon(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"id":"`+server.URL+`/users/bob","icon":"https://remote.example/avatars/bob.png"}`)
	}))
	defer server.Close()

	actor, err := fetchActorDocument(context.Background(), NewHTTPClient(true), server.URL+"/users/bob")
	if err != nil {
		t.Fatalf("fetchActorDocument() error = %v", err)
	}
	if actor.Icon.URL != "https://remote.example/avatars/bob.png" {
		t.Errorf("Icon.URL = %q", actor.Icon.URL)
	}
}

func TestFollowerProfileCacheExpires(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	service := testProfileService()
	service.now = func() time.Time { return now }

	follower := Follower{ActorID: "https://remote.example/users/bob", Handle: "bob"}
	service.cacheFollower(follower.ActorID, follower)

	cached, ok := service.cachedFollower(follower.ActorID)
	if !ok {
		t.Fatal("cachedFollower() = false, want cached entry")
	}
	if cached.Handle != "bob" {
		t.Errorf("Handle = %q", cached.Handle)
	}

	now = now.Add(followerProfileTTL + time.Second)
	if _, ok := service.cachedFollower(follower.ActorID); ok {
		t.Error("cachedFollower() = true after TTL, want expired")
	}
}

func TestFollowerProfileCacheStartsEmpty(t *testing.T) {
	service := testProfileService()

	if _, ok := service.cachedFollower("https://remote.example/users/bob"); ok {
		t.Error("cachedFollower() = true on empty cache")
	}
}
