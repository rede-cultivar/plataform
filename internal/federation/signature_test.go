package federation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newRemoteActorServer publica um Actor remoto de teste e devolve o servidor,
// o identificador do Actor e o par de chaves usado para assinar requisições.
func newRemoteActorServer(t *testing.T) (*httptest.Server, string, *KeyPair) {
	t.Helper()

	keys, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	var actorID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor := RemoteActor{
			ID:                actorID,
			Type:              "Person",
			PreferredUsername: "bob",
			Inbox:             actorID + "/inbox",
		}
		actor.PublicKey.ID = actorID + "#main-key"
		actor.PublicKey.Owner = actorID
		actor.PublicKey.PublicKeyPEM = keys.PublicPEM

		w.Header().Set("Content-Type", ActivityContentType)
		if err := json.NewEncoder(w).Encode(actor); err != nil {
			t.Errorf("encode remote actor: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	actorID = server.URL + "/users/bob"
	return server, actorID, keys
}

func signedRequest(t *testing.T, target string, body []byte, keyID string, keys *KeyPair) *http.Request {
	t.Helper()

	private, err := ParsePrivateKey(keys.PrivatePEM)
	if err != nil {
		t.Fatalf("ParsePrivateKey() error = %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	if err := SignRequest(req, body, keyID, private); err != nil {
		t.Fatalf("SignRequest() error = %v", err)
	}
	return req
}

func TestSignRequestAndVerifyRequest(t *testing.T) {
	server, actorID, keys := newRemoteActorServer(t)
	body := []byte(`{"id":"https://remote.example/follows/1","type":"Follow"}`)

	req := signedRequest(t, "https://cultivar.example/actors/ana/inbox", body, actorID+"#main-key", keys)

	if got := req.Header.Get("Digest"); !strings.HasPrefix(got, "SHA-256=") {
		t.Errorf("Digest = %q, want SHA-256 value", got)
	}
	if got := req.Header.Get("Signature"); !strings.Contains(got, `keyId="`+actorID+`#main-key"`) {
		t.Errorf("Signature = %q, want keyId", got)
	}

	verifier := NewVerifier(server.Client())
	actor, err := verifier.VerifyRequest(context.Background(), req, body)
	if err != nil {
		t.Fatalf("VerifyRequest() error = %v", err)
	}
	if actor.ID != actorID {
		t.Errorf("actor.ID = %q, want %q", actor.ID, actorID)
	}
	if actor.Inbox != actorID+"/inbox" {
		t.Errorf("actor.Inbox = %q", actor.Inbox)
	}
}

func TestVerifyRequestRejectsTamperedBody(t *testing.T) {
	body := []byte(`{"type":"Follow"}`)
	keys, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	req := signedRequest(t, "https://cultivar.example/actors/ana/inbox", body, "https://remote.example/users/bob#main-key", keys)

	_, err = NewVerifier(nil).VerifyRequest(context.Background(), req, []byte(`{"type":"Undo"}`))
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("VerifyRequest() error = %v, want ErrInvalidSignature", err)
	}
}

func TestVerifyRequestRejectsStaleDate(t *testing.T) {
	body := []byte(`{"type":"Follow"}`)
	keys, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	req := signedRequest(t, "https://cultivar.example/actors/ana/inbox", body, "https://remote.example/users/bob#main-key", keys)
	req.Header.Set("Date", time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat))

	_, err = NewVerifier(nil).VerifyRequest(context.Background(), req, body)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("VerifyRequest() error = %v, want ErrInvalidSignature", err)
	}
	if !strings.Contains(err.Error(), "data fora") {
		t.Errorf("error = %q, want date window message", err)
	}
}

func TestVerifyRequestRejectsMissingSignature(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://cultivar.example/actors/ana/inbox", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	_, err = NewVerifier(nil).VerifyRequest(context.Background(), req, nil)
	if !errors.Is(err, ErrMissingSignature) {
		t.Fatalf("VerifyRequest() error = %v, want ErrMissingSignature", err)
	}
}

func TestVerifyRequestRejectsMissingSignedHeader(t *testing.T) {
	body := []byte(`{"type":"Follow"}`)
	keys, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	req := signedRequest(t, "https://cultivar.example/actors/ana/inbox", body, "https://remote.example/users/bob#main-key", keys)
	req.Header.Del("Date")

	_, err = NewVerifier(nil).VerifyRequest(context.Background(), req, body)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("VerifyRequest() error = %v, want ErrInvalidSignature", err)
	}
}

func TestVerifyRequestRequiresSignedTargetHostAndDate(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://cultivar.example/actors/ana/inbox", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	req.Header.Set("Signature", `keyId="https://remote.example/users/bob#main-key",algorithm="rsa-sha256",headers="date",signature="abc"`)

	_, err = NewVerifier(nil).VerifyRequest(context.Background(), req, nil)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("VerifyRequest() error = %v, want ErrInvalidSignature", err)
	}
	if !strings.Contains(err.Error(), "(request-target)") {
		t.Errorf("error = %q, want missing request-target", err)
	}
}

func TestVerifyRequestRequiresSignedDigest(t *testing.T) {
	body := []byte(`{"type":"Follow"}`)
	req, err := http.NewRequest(http.MethodPost, "https://cultivar.example/actors/ana/inbox", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	req.Header.Set("Digest", digestValue(body))
	req.Header.Set("Signature", `keyId="https://remote.example/users/bob#main-key",algorithm="rsa-sha256",headers="(request-target) host date",signature="abc"`)

	_, err = NewVerifier(nil).VerifyRequest(context.Background(), req, body)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("VerifyRequest() error = %v, want ErrInvalidSignature", err)
	}
	if !strings.Contains(err.Error(), "digest") {
		t.Errorf("error = %q, want missing digest", err)
	}
}

func TestParseSignatureHeader(t *testing.T) {
	params, err := ParseSignatureHeader(`keyId="https://remote.example/users/bob#main-key",algorithm="rsa-sha256",headers="(request-target) Host Date Digest Content-Type",signature="abc"`)
	if err != nil {
		t.Fatalf("ParseSignatureHeader() error = %v", err)
	}

	if params.KeyID != "https://remote.example/users/bob#main-key" {
		t.Errorf("KeyID = %q", params.KeyID)
	}
	if params.Algorithm != "rsa-sha256" {
		t.Errorf("Algorithm = %q", params.Algorithm)
	}
	if params.Signature != "abc" {
		t.Errorf("Signature = %q", params.Signature)
	}
	if got := strings.Join(params.Headers, " "); got != "(request-target) host date digest content-type" {
		t.Errorf("Headers = %q, want lowercased header names", got)
	}
}

func TestParseSignatureHeaderDefaults(t *testing.T) {
	params, err := ParseSignatureHeader(`keyId="https://remote.example/users/bob#main-key",signature="abc"`)
	if err != nil {
		t.Fatalf("ParseSignatureHeader() error = %v", err)
	}

	if params.Algorithm != signatureAlgorithm {
		t.Errorf("Algorithm = %q, want %q", params.Algorithm, signatureAlgorithm)
	}
	if got := strings.Join(params.Headers, " "); got != "(request-target) host date" {
		t.Errorf("Headers = %q, want default headers", got)
	}
}

func TestParseSignatureHeaderRejectsUnsupportedAlgorithm(t *testing.T) {
	_, err := ParseSignatureHeader(`keyId="https://remote.example/users/bob#main-key",algorithm="hs2019",signature="abc"`)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("ParseSignatureHeader() error = %v, want ErrInvalidSignature", err)
	}
}

func TestFetchActorRejectsMismatchedID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys, _ := GenerateKeyPair()
		actor := RemoteActor{
			ID:    "https://outra.example/users/bob",
			Type:  "Person",
			Inbox: "https://outra.example/users/bob/inbox",
		}
		actor.PublicKey.PublicKeyPEM = keys.PublicPEM
		_ = json.NewEncoder(w).Encode(actor)
	}))
	t.Cleanup(server.Close)

	_, err := NewVerifier(server.Client()).FetchActor(context.Background(), server.URL+"/users/bob")
	if !errors.Is(err, ErrInvalidActivity) {
		t.Fatalf("FetchActor() error = %v, want ErrInvalidActivity", err)
	}
}

func TestFetchActorRejectsIncompleteDocument(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(RemoteActor{ID: "https://remote.example/users/bob"})
	}))
	t.Cleanup(server.Close)

	_, err := NewVerifier(server.Client()).FetchActor(context.Background(), server.URL+"/users/bob")
	if !errors.Is(err, ErrInvalidActivity) {
		t.Fatalf("FetchActor() error = %v, want ErrInvalidActivity", err)
	}
}

func TestKeyPairRoundTrip(t *testing.T) {
	keys, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	if !strings.Contains(keys.PublicPEM, "BEGIN PUBLIC KEY") {
		t.Errorf("PublicPEM = %q, want PUBLIC KEY block", keys.PublicPEM)
	}
	if !strings.Contains(keys.PrivatePEM, "BEGIN RSA PRIVATE KEY") {
		t.Errorf("PrivatePEM = %q, want RSA PRIVATE KEY block", keys.PrivatePEM)
	}

	if _, err := ParsePrivateKey(keys.PrivatePEM); err != nil {
		t.Errorf("ParsePrivateKey() error = %v", err)
	}
	if _, err := ParsePublicKey(keys.PublicPEM); err != nil {
		t.Errorf("ParsePublicKey() error = %v", err)
	}
}

func TestKeyPairRejectsInvalidPEM(t *testing.T) {
	if _, err := ParsePrivateKey("não é PEM"); err == nil {
		t.Error("ParsePrivateKey() error = nil, want error")
	}
	if _, err := ParsePublicKey("não é PEM"); err == nil {
		t.Error("ParsePublicKey() error = nil, want error")
	}
}

func TestDigestValue(t *testing.T) {
	digest := digestValue([]byte("cultivar"))
	if !strings.HasPrefix(digest, "SHA-256=") {
		t.Errorf("digestValue() = %q, want SHA-256 prefix", digest)
	}
	if !validDigest(digest, []byte("cultivar")) {
		t.Error("validDigest() = false for matching body")
	}
	if validDigest(digest, []byte("outro corpo")) {
		t.Error("validDigest() = true for different body")
	}
}
