package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rede-cultivar/plataform/internal/platform/config"
)

func testRouter(t *testing.T) http.Handler {
	t.Helper()

	return NewRouter(Deps{
		Config: &config.Config{
			Env:          config.EnvDevelopment,
			InstanceName: "Cultivar Test",
			LocalDomain:  "localhost:8080",
			LogLevel:     "error",
		},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
}

func TestHealthEndpoint(t *testing.T) {
	router := testRouter(t)

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestHomeEndpoint(t *testing.T) {
	router := testRouter(t)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html", contentType)
	}
}

func TestNotFound(t *testing.T) {
	router := testRouter(t)

	request := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}
