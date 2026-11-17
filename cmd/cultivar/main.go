package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pressly/goose/v3"

	"github.com/rede-cultivar/plataform/internal/cultivation"
	"github.com/rede-cultivar/plataform/internal/federation"
	"github.com/rede-cultivar/plataform/internal/identity"
	"github.com/rede-cultivar/plataform/internal/media"
	"github.com/rede-cultivar/plataform/internal/platform/config"
	"github.com/rede-cultivar/plataform/internal/platform/database"
	"github.com/rede-cultivar/plataform/internal/platform/logging"
	"github.com/rede-cultivar/plataform/internal/platform/storage"
	"github.com/rede-cultivar/plataform/internal/practices"
	"github.com/rede-cultivar/plataform/internal/sitesettings"
	"github.com/rede-cultivar/plataform/internal/version"
	"github.com/rede-cultivar/plataform/internal/web"
	"github.com/rede-cultivar/plataform/internal/worker"
	"github.com/rede-cultivar/plataform/migrations"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "cultivar: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		args = []string{"serve"}
	}

	switch args[0] {
	case "serve":
		return runServe()
	case "worker":
		return runWorker()
	case "migrate":
		return runMigrate(args[1:])
	case "admin":
		return runAdmin(args[1:])
	case "version":
		fmt.Println(version.String())
		return nil
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		return fmt.Errorf("unknown command %q (try \"cultivar help\")", args[0])
	}
}

func usage() {
	fmt.Print(`Cultivar — infraestrutura digital comunitária

Uso:
  cultivar [comando]

Comandos:
  serve              Inicia o servidor HTTP (padrão)
  worker             Inicia o processamento assíncrono
  migrate <comando>  Executa migrações: up, down, status ou version
  admin <comando>    Administra papéis: promote, demote ou list
  version            Exibe a versão em execução
  help               Exibe esta ajuda

A configuração é carregada de variáveis de ambiente. Consulte .env.example.
`)
}

func runServe() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(cfg.LogLevel, cfg.Env)
	ctx, stop := signalContext()
	defer stop()

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	store, err := storage.New(cfg.S3)
	if err != nil {
		return err
	}

	setupCtx, cancelSetup := context.WithTimeout(ctx, 5*time.Second)
	if err := store.EnsureBucket(setupCtx); err != nil {
		logger.Warn("object storage is not ready; uploads will fail until it is available", "error", err)
	} else {
		logger.Info("object storage ready", "bucket", store.Bucket())
	}
	cancelSetup()

	federationService := federation.NewService(
		db,
		federation.NewURLs(cfg.BaseURL()),
		logger,
		!cfg.IsProduction(),
	)

	handler := web.NewRouter(web.Deps{
		Config:       cfg,
		DB:           db,
		Storage:      store,
		Identity:     identity.NewService(db, cfg.SessionIdleTimeout, cfg.SessionAbsoluteTimeout),
		Cultivation:  cultivation.NewService(db),
		Practices:    practices.NewService(db),
		Media:        media.NewService(db, store),
		Federation:   federationService,
		SiteSettings: sitesettings.NewService(db),
		Logger:       logger,
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("http server listening",
			"addr", cfg.HTTPAddr,
			"domain", cfg.LocalDomain,
			"env", cfg.Env,
			"version", version.Version,
		)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
		logger.Info("shutting down http server")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	return nil
}

func runWorker() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(cfg.LogLevel, cfg.Env)
	ctx, stop := signalContext()
	defer stop()

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	federationService := federation.NewService(
		db,
		federation.NewURLs(cfg.BaseURL()),
		logger,
		!cfg.IsProduction(),
	)

	return worker.Run(ctx, worker.Deps{
		DB:           db,
		Logger:       logger,
		PollInterval: cfg.WorkerPollInterval,
		Deliveries:   federationService,
	})
}

func runMigrate(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: cultivar migrate <up|down|status|version>")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signalContext()
	defer stop()

	db, err := database.OpenSQL(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set migration dialect: %w", err)
	}

	var migrateErr error
	switch args[0] {
	case "up":
		migrateErr = goose.UpContext(ctx, db, ".")
	case "down":
		migrateErr = goose.DownContext(ctx, db, ".")
	case "status":
		migrateErr = goose.StatusContext(ctx, db, ".")
	case "version":
		migrateErr = goose.VersionContext(ctx, db, ".")
	default:
		return fmt.Errorf("unknown migrate command %q", args[0])
	}

	if migrateErr != nil {
		return fmt.Errorf("migrate %s: %w", args[0], migrateErr)
	}

	return nil
}

func runAdmin(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: cultivar admin <promote|demote|list> [identificador]")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signalContext()
	defer stop()

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	service := identity.NewService(db, cfg.SessionIdleTimeout, cfg.SessionAbsoluteTimeout)

	switch args[0] {
	case "promote":
		if len(args) != 2 {
			return errors.New("usage: cultivar admin promote <identificador>")
		}
		handle := identity.NormalizeHandle(args[1])
		if err := service.SetRoleByHandle(ctx, handle, identity.RoleAdmin); err != nil {
			return fmt.Errorf("promover @%s: %w", handle, err)
		}
		fmt.Printf("@%s agora é administrador da instância.\n", handle)
	case "demote":
		if len(args) != 2 {
			return errors.New("usage: cultivar admin demote <identificador>")
		}
		handle := identity.NormalizeHandle(args[1])
		if err := service.SetRoleByHandle(ctx, handle, identity.RoleMember); err != nil {
			return fmt.Errorf("remover administração de @%s: %w", handle, err)
		}
		fmt.Printf("@%s agora é membro da instância.\n", handle)
	case "list":
		accounts, err := service.ListAccountsWithRoles(ctx)
		if err != nil {
			return fmt.Errorf("listar contas: %w", err)
		}
		for _, account := range accounts {
			fmt.Printf("@%s\t%s\t%s\t%s\n", account.Person.Handle, account.Role, account.Status, account.Email)
		}
	default:
		return fmt.Errorf("comando administrativo desconhecido %q", args[0])
	}

	return nil
}

func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
