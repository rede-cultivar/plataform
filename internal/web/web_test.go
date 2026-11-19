package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	for _, marker := range []string{
		`rel="manifest" href="/manifest.webmanifest"`,
		`/static/app.css?v=`,
		`/static/app.js?v=`,
		`class="bottom-nav"`,
		`id="conteudo"`,
	} {
		if !strings.Contains(string(body), marker) {
			t.Errorf("home does not contain %q", marker)
		}
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

func TestSecurityHeadersAllowRemoteAvatars(t *testing.T) {
	router := testRouter(t)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	csp := response.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "img-src 'self' data: https:") {
		t.Errorf("Content-Security-Policy = %q, want https images allowed", csp)
	}
	for _, directive := range []string{"worker-src 'self'", "manifest-src 'self'", "script-src 'self'"} {
		if !strings.Contains(csp, directive) {
			t.Errorf("Content-Security-Policy = %q, want %q", csp, directive)
		}
	}
}

func TestPWARoutes(t *testing.T) {
	router := testRouter(t)

	cases := []struct {
		path        string
		contentType string
		contains    string
	}{
		{"/manifest.webmanifest", "application/manifest+json; charset=utf-8", `"short_name": "Cultivar"`},
		{"/sw.js", "text/javascript; charset=utf-8", `const CACHE_NAME = "cultivar-`},
		{"/offline", "text/html; charset=utf-8", "Sem conexão"},
		{"/static/icons/icon-192.png", "image/png", ""},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tc.path, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
			}
			if got := response.Header().Get("Content-Type"); got != tc.contentType {
				t.Errorf("Content-Type = %q, want %q", got, tc.contentType)
			}
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if !strings.Contains(string(body), tc.contains) {
				t.Errorf("body does not contain %q", tc.contains)
			}
		})
	}
}

func TestServiceWorkerUsesAssetVersion(t *testing.T) {
	router := testRouter(t)

	request := httptest.NewRequest(http.MethodGet, "/sw.js", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if strings.Contains(string(body), "__ASSET_VERSION__") {
		t.Error("service worker still contains the version placeholder")
	}
	if !strings.Contains(string(body), `const CACHE_NAME = "cultivar-`) {
		t.Error("service worker does not define the versioned cache name")
	}
}

func TestSecurityHeadersEnableHSTSInProduction(t *testing.T) {
	router := NewRouter(Deps{
		Config: &config.Config{
			Env:          config.EnvProduction,
			InstanceName: "Cultivar Test",
			LocalDomain:  "https://cultivar.example",
			LogLevel:     "error",
			SecretKey:    strings.Repeat("x", 32),
		},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if header := response.Header().Get("Strict-Transport-Security"); header == "" {
		t.Error("Strict-Transport-Security is empty in production")
	}
}
