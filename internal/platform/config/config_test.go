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
