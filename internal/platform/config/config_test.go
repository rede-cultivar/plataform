package config

import "testing"

func TestLoadDevelopmentDefaults(t *testing.T) {
	t.Setenv("CULTIVAR_ENV", EnvDevelopment)
	t.Setenv("CULTIVAR_SECRET_KEY", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":8080")
	}
	if cfg.LocalDomain != "localhost:8080" {
		t.Errorf("LocalDomain = %q, want %q", cfg.LocalDomain, "localhost:8080")
	}
	if cfg.S3.Bucket != "cultivar" {
		t.Errorf("S3.Bucket = %q, want %q", cfg.S3.Bucket, "cultivar")
	}
}

func TestLoadProductionRequiresSecret(t *testing.T) {
	t.Setenv("CULTIVAR_ENV", EnvProduction)
	t.Setenv("CULTIVAR_SECRET_KEY", "short")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error for short production secret")
	}
}

func TestLoadRejectsUnknownEnvironment(t *testing.T) {
	t.Setenv("CULTIVAR_ENV", "staging")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error for unknown environment")
	}
}

func TestLoadRejectsInvalidSessionWindow(t *testing.T) {
	t.Setenv("CULTIVAR_SESSION_IDLE_TIMEOUT", "48h")
	t.Setenv("CULTIVAR_SESSION_ABSOLUTE_TIMEOUT", "24h")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error when idle timeout exceeds absolute timeout")
	}
}

func TestBaseURL(t *testing.T) {
	development := &Config{Env: EnvDevelopment, LocalDomain: "localhost:8080"}
	if got := development.BaseURL(); got != "http://localhost:8080" {
		t.Errorf("BaseURL() = %q, want %q", got, "http://localhost:8080")
	}

	production := &Config{Env: EnvProduction, LocalDomain: "cultivar.example"}
	if got := production.BaseURL(); got != "https://cultivar.example" {
		t.Errorf("BaseURL() = %q, want %q", got, "https://cultivar.example")
	}

	explicit := &Config{Env: EnvDevelopment, LocalDomain: "https://cultivar.example/"}
	if got := explicit.BaseURL(); got != "https://cultivar.example" {
		t.Errorf("BaseURL() = %q, want %q", got, "https://cultivar.example")
	}
}
