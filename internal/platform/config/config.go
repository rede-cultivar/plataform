package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

// Config reúne as configurações da aplicação, carregadas de variáveis de ambiente.
type Config struct {
	Env                    string
	HTTPAddr               string
	LocalDomain            string
	InstanceName           string
	LogLevel               string
	DatabaseURL            string
	SecretKey              string
	ShutdownTimeout        time.Duration
	WorkerPollInterval     time.Duration
	SessionIdleTimeout     time.Duration
	SessionAbsoluteTimeout time.Duration
	CookieSecure           bool
	S3                     S3
}

// S3 reúne as configurações de armazenamento de objetos compatível com S3.
type S3 struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

// Load carrega a configuração a partir das variáveis de ambiente, aplicando
// valores padrão adequados ao ambiente de desenvolvimento local.
func Load() (*Config, error) {
	cfg := &Config{
		Env:                    strings.ToLower(env("CULTIVAR_ENV", EnvDevelopment)),
		HTTPAddr:               env("CULTIVAR_HTTP_ADDR", ":8080"),
		LocalDomain:            env("CULTIVAR_LOCAL_DOMAIN", "localhost:8080"),
		InstanceName:           env("CULTIVAR_INSTANCE_NAME", "Cultivar"),
		LogLevel:               strings.ToLower(env("CULTIVAR_LOG_LEVEL", "info")),
		DatabaseURL:            env("CULTIVAR_DATABASE_URL", "postgres://cultivar:cultivar@localhost:5432/cultivar?sslmode=disable"),
		SecretKey:              env("CULTIVAR_SECRET_KEY", ""),
		ShutdownTimeout:        duration("CULTIVAR_SHUTDOWN_TIMEOUT", 15*time.Second),
		WorkerPollInterval:     duration("CULTIVAR_WORKER_POLL_INTERVAL", 5*time.Second),
		SessionIdleTimeout:     duration("CULTIVAR_SESSION_IDLE_TIMEOUT", 12*time.Hour),
		SessionAbsoluteTimeout: duration("CULTIVAR_SESSION_ABSOLUTE_TIMEOUT", 30*24*time.Hour),
		CookieSecure:           boolean("CULTIVAR_COOKIE_SECURE", false),
		S3: S3{
			Endpoint:  env("CULTIVAR_S3_ENDPOINT", "localhost:9000"),
			Region:    env("CULTIVAR_S3_REGION", "us-east-1"),
			Bucket:    env("CULTIVAR_S3_BUCKET", "cultivar"),
			AccessKey: env("CULTIVAR_S3_ACCESS_KEY", "cultivar"),
			SecretKey: env("CULTIVAR_S3_SECRET_KEY", "cultivar123"),
			UseSSL:    boolean("CULTIVAR_S3_USE_SSL", false),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// IsProduction informa se a aplicação está configurada para produção.
func (c *Config) IsProduction() bool {
	return c.Env == EnvProduction
}

// SecureCookies informa se os cookies devem ser enviados apenas por HTTPS.
func (c *Config) SecureCookies() bool {
	return c.CookieSecure || c.IsProduction()
}

func (c *Config) validate() error {
	switch c.Env {
	case EnvDevelopment, EnvProduction:
	default:
		return fmt.Errorf("CULTIVAR_ENV must be %q or %q", EnvDevelopment, EnvProduction)
	}

	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("CULTIVAR_LOG_LEVEL must be one of debug, info, warn or error")
	}

	if strings.TrimSpace(c.LocalDomain) == "" {
		return fmt.Errorf("CULTIVAR_LOCAL_DOMAIN is required")
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("CULTIVAR_DATABASE_URL is required")
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("CULTIVAR_SHUTDOWN_TIMEOUT must be greater than zero")
	}
	if c.WorkerPollInterval <= 0 {
		return fmt.Errorf("CULTIVAR_WORKER_POLL_INTERVAL must be greater than zero")
	}
	if c.SessionIdleTimeout <= 0 {
		return fmt.Errorf("CULTIVAR_SESSION_IDLE_TIMEOUT must be greater than zero")
	}
	if c.SessionAbsoluteTimeout <= 0 {
		return fmt.Errorf("CULTIVAR_SESSION_ABSOLUTE_TIMEOUT must be greater than zero")
	}
	if c.SessionIdleTimeout >= c.SessionAbsoluteTimeout {
		return fmt.Errorf("CULTIVAR_SESSION_IDLE_TIMEOUT must be less than CULTIVAR_SESSION_ABSOLUTE_TIMEOUT")
	}
	if c.S3.Endpoint == "" || c.S3.Bucket == "" {
		return fmt.Errorf("CULTIVAR_S3_ENDPOINT and CULTIVAR_S3_BUCKET are required")
	}
	if c.IsProduction() && len(c.SecretKey) < 32 {
		return fmt.Errorf("CULTIVAR_SECRET_KEY must have at least 32 characters in production")
	}

	return nil
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func duration(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func boolean(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
