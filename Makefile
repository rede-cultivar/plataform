SHELL := /bin/sh

BINARY     := bin/cultivar
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X github.com/rede-cultivar/plataform/internal/version.Version=$(VERSION) \
	-X github.com/rede-cultivar/plataform/internal/version.Commit=$(COMMIT) \
	-X github.com/rede-cultivar/plataform/internal/version.Date=$(BUILD_DATE)

# Carrega o arquivo .env, quando existir, e exporta as variáveis para os comandos.
-include .env
export

.DEFAULT_GOAL := help

.PHONY: help
help: ## Exibe esta ajuda
	@printf "\nUso: make <alvo>\n\n"
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@printf "\n"

.PHONY: build
build: ## Compila o binário em bin/cultivar
	@mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/cultivar

.PHONY: run
run: ## Inicia o servidor HTTP local
	go run ./cmd/cultivar serve

.PHONY: worker
worker: ## Inicia o trabalhador assíncrono local
	go run ./cmd/cultivar worker

.PHONY: migrate
migrate: ## Aplica as migrações pendentes
	go run ./cmd/cultivar migrate up

.PHONY: migrate-down
migrate-down: ## Reverte a última migração
	go run ./cmd/cultivar migrate down

.PHONY: migrate-status
migrate-status: ## Exibe o status das migrações
	go run ./cmd/cultivar migrate status

.PHONY: test
test: ## Executa os testes
	go test ./...

.PHONY: vet
vet: ## Executa o go vet
	go vet ./...

.PHONY: fmt
fmt: ## Formata o código Go
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

.PHONY: lint
lint: ## Executa o golangci-lint (requer instalação local)
	golangci-lint run

.PHONY: ci
ci: vet test build ## Executa as verificações usadas na integração contínua

.PHONY: deps
deps: ## Sobe PostgreSQL e o armazenamento S3 local
	docker compose up -d postgres s3mock

.PHONY: up
up: ## Sobe a stack completa em containers
	docker compose up -d --build

.PHONY: down
down: ## Derruba os containers
	docker compose down

.PHONY: logs
logs: ## Acompanha os logs dos containers
	docker compose logs -f

.PHONY: reset
reset: ## Derruba os containers e remove os volumes (destrutivo)
	docker compose down -v

.PHONY: clean
clean: ## Remove artefatos de build
	rm -rf bin coverage.out
