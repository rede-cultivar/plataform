# ADR 0007 — Stack da aplicação

**Status:** Aceito  
**Data:** 2026-11-10

## Contexto

As decisões anteriores estabeleceram que o projeto será uma aplicação própria, com domínio independente, integrada ao Fediverso por meio do ActivityPub (ADR 0001), organizada em contextos funcionais (ADR 0002), com persistência relacional e armazenamento de objetos (ADR 0005) e com processamento assíncrono para a federação (ADR 0004).

Para iniciar a implementação, é necessário escolher tecnologias concretas. Essa escolha deve considerar os princípios do projeto:

- Software livre e ausência de dependência de serviços proprietários.
- Facilidade de instalação e operação por comunidades com diferentes capacidades técnicas.
- Baixo custo operacional e simplicidade de implantação.
- Longevidade e maturidade das tecnologias.
- Possibilidade de participação de colaboradores com diferentes níveis de experiência.
- Adequação à implementação de protocolos abertos, especialmente o ActivityPub.

Este ADR define a stack inicial. As escolhas poderão ser revistas por meio de novas decisões, desde que haja justificativa técnica e alinhamento com os princípios do projeto.

## Decisão

A aplicação adotará uma stack baseada em **Go, PostgreSQL, armazenamento compatível com S3 e interface renderizada no servidor**.

### 1. Linguagem e backend

A aplicação será desenvolvida em **Go**, utilizando a versão estável mais recente no início da implementação e mantendo uma política explícita de atualização.

Componentes iniciais:

- **HTTP:** biblioteca padrão `net/http`, com o roteador leve `chi`.
- **Acesso a dados:** `pgx` como driver PostgreSQL e `sqlc` para geração de código tipado a partir de consultas SQL explícitas.
- **Migrações:** `goose`, com migrações SQL versionadas e embutidas no binário.
- **Configuração:** variáveis de ambiente, seguindo os princípios de Twelve-Factor App.
- **Logs:** `log/slog`, com saída estruturada em JSON.
- **Testes:** biblioteca padrão `testing`, com `testcontainers-go` para testes de integração com PostgreSQL e armazenamento de objetos.
- **Qualidade:** `gofmt`, `go vet` e `golangci-lint`.

O backend será distribuído como um único binário executável, com os arquivos estáticos embutidos por meio de `go:embed`.

### 2. Banco de dados

O banco relacional será o **PostgreSQL 16 ou superior**.

Diretrizes:

- O modelo de domínio será representado por tabelas e relações explícitas.
- Campos `JSONB` poderão ser utilizados para dados semiestruturados, como payloads do ActivityStreams e metadados de extensão, mas não substituirão o modelamento relacional das entidades de domínio.
- A busca textual inicial utilizará os recursos nativos do PostgreSQL. Um mecanismo especializado poderá ser introduzido posteriormente, se necessário.
- Consultas e índices deverão ser definidos de forma explícita e versionada.

### 3. Armazenamento de objetos

Imagens e arquivos serão armazenados em serviço compatível com a **API S3**.

Diretrizes:

- O ambiente de desenvolvimento e as instâncias auto-hospedadas poderão utilizar **MinIO**.
- Provedores externos compatíveis com S3 poderão ser utilizados, sem que a aplicação dependa de um fornecedor específico.
- O banco de dados armazenará apenas metadados, referências e informações de controle.
- Conteúdo binário não será armazenado diretamente no banco relacional.

### 4. Interface de usuário

A interface inicial será **renderizada no servidor**, utilizando:

- `html/template` da biblioteca padrão para templates.
- **HTMX** para interações pontuais sem necessidade de uma aplicação SPA.
- JavaScript mínimo e progressivamente aprimorado.
- Um sistema de estilos próprio e simples, com opção de uso de Tailwind CSS por meio de etapa de build opcional.

Não será adotada uma aplicação SPA na primeira versão.

### 5. Processamento assíncrono

O processamento assíncrono inicial utilizará o próprio PostgreSQL, por meio do padrão **Transactional Outbox**, sem a introdução imediata de um broker externo.

Diretrizes:

- Atividades pendentes serão registradas em tabela própria na mesma transação da operação de domínio.
- Um processo trabalhador, executado pelo mesmo binário por meio de um comando dedicado, consumirá a fila.
- O consumo utilizará `FOR UPDATE SKIP LOCKED` para permitir múltiplos trabalhadores com segurança.
- Retentativas, agendamento e controle de falhas serão persistidos no banco.
- Um broker dedicado, como Redis, NATS ou RabbitMQ, somente será introduzido se o volume ou os requisitos operacionais justificarem.

### 6. Organização do repositório

O projeto será mantido em **monorepo**, com um único módulo Go.

Estrutura inicial sugerida:

```text
cmd/            comandos executáveis (servidor, worker, migrações)
internal/       código interno da aplicação
  identity/     identidade, contas e perfis
  publications/ publicações e interações
  federation/   integração com ActivityPub
  platform/     infraestrutura compartilhada (banco, storage, config)
  web/          handlers, templates e ativos
migrations/     migrações SQL
docs/           documentação
```

A separação entre contextos será lógica, com interfaces explícitas. Não serão introduzidos microserviços nesta etapa.

## Alternativas consideradas

### 1. Elixir com Phoenix

**Vantagens:**
- Forte tradição de implementações federadas, como Pleroma e Akkoma.
- Modelo de concorrência adequado a processamento assíncrono.
- Excelente suporte a tempo real.

**Desvantagens:**
- Menor quantidade de colaboradores familiarizados com a linguagem.
- Operação e diagnóstico exigem conhecimento específico da BEAM.
- Menor reaproveitamento de bibliotecas e práticas comuns em outras partes do ecossistema.

### 2. TypeScript com Node.js

**Vantagens:**
- Grande ecossistema e ampla disponibilidade de colaboradores.
- Compartilhamento de linguagem entre backend e frontend.

**Desvantagens:**
- Necessidade de runtime e processo de build adicionais em produção.
- Maior superfície de dependências e risco de manutenção.
- Tipagem e concorrência menos previsíveis que em Go para este tipo de aplicação.

### 3. Python com Django

**Vantagens:**
- Curva de aprendizado acessível.
- Ecossistema maduro para aplicações web.

**Desvantagens:**
- Desempenho e concorrência menos adequados para processamento federado intensivo.
- Maior custo operacional em comparação com um binário único.

### 4. Rust com Axum

**Vantagens:**
- Segurança de memória e alto desempenho.
- Binário único e baixo consumo de recursos.

**Desvantagens:**
- Curva de aprendizado mais íngreme.
- Iteração inicial mais lenta.
- Menor quantidade de colaboradores disponíveis.

### 5. Interface como SPA

**Vantagens:**
- Maior interatividade e separação clara entre frontend e backend.

**Desvantagens:**
- Necessidade de build e implantação adicionais.
- Maior complexidade de acessibilidade e de carregamento inicial.
- Duplicação de modelos e validações entre cliente e servidor.

### 6. SQLite como banco inicial

**Vantagens:**
- Instalação extremamente simples.
- Adequado a instâncias pequenas.

**Desvantagens:**
- Menor concorrência de escrita.
- Recursos limitados para busca, análise e operação federada.
- Migração posterior custosa.

## Consequências

A decisão estabelece:

- Um único binário Go com ativos embutidos, simplificando a implantação.
- PostgreSQL como banco relacional principal.
- Armazenamento de objetos compatível com S3.
- Interface renderizada no servidor, sem SPA na primeira versão.
- Fila inicial baseada em PostgreSQL, sem broker externo.
- Um único módulo Go e uma separação lógica entre contextos.

A stack também implica:

- Maior esforço na implementação manual dos protocolos de federação, já que não há um framework ActivityPub maduro e amplamente adotado em Go.
- Necessidade de disciplina na definição de consultas SQL e migrações.
- Limitações de interatividade da interface inicial, que poderão ser endereçadas com aprimoramento progressivo.
- Possível necessidade futura de introduzir broker, cache e mecanismo de busca dedicados.

## Escopo e decisões futuras

Este ADR não determina ainda:

- A versão exata de Go e a política de atualização de dependências.
- O sistema de estilos e o guia de design definitivo.
- A estratégia de internacionalização.
- O formato da API pública (REST/JSON, GraphQL ou outro).
- A necessidade de cache e a escolha de tecnologia.
- A introdução de mecanismo especializado de busca.
- A estratégia de distribuição de mídia por CDN.
- A topologia e o escalonamento dos trabalhadores assíncronos.
- A organização detalhada dos pacotes internos e suas interfaces.
- Aplicações móveis ou clientes alternativos.

Esses aspectos deverão ser detalhados em decisões posteriores.

## Referências

- [The Go Programming Language](https://go.dev/)
- [chi — lightweight Go router](https://github.com/go-chi/chi)
- [pgx — PostgreSQL driver for Go](https://github.com/jackc/pgx)
- [sqlc — Generate type-safe code from SQL](https://sqlc.dev/)
- [goose — Database migrations](https://github.com/pressly/goose)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [MinIO Documentation](https://min.io/docs/)
- [HTMX Documentation](https://htmx.org/docs/)
- [The Twelve-Factor App](https://12factor.net/)
- [GoToSocial — implementação federada em Go](https://gotosocial.org/)
