# Desenvolvimento

Este documento descreve como preparar o ambiente local, executar a aplicação e contribuir com o código.

## Requisitos

- **Go 1.27** ou superior.
- **Docker** com **Docker Compose** (para PostgreSQL e armazenamento S3 local).
- **Make** (opcional, mas recomendado).
- **golangci-lint** (opcional, para o alvo `make lint`).

## Início rápido

```sh
cp .env.example .env
make deps      # sobe PostgreSQL e o armazenamento S3 local
make migrate   # aplica as migrações
make run       # inicia o servidor em http://localhost:8080
```

Em outro terminal, para iniciar o trabalhador assíncrono:

```sh
make worker
```

Verificações úteis:

```sh
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

O ambiente local usa o [Adobe S3Mock](https://github.com/adobe/S3Mock), um serviço
leve compatível com a API S3. O bucket `cultivar` é criado automaticamente. O
endpoint fica em <http://localhost:9000> e aceita as credenciais configuradas em
`.env` (por padrão, `cultivar` / `cultivar123`).

Qualquer outro serviço compatível com S3, como MinIO ou um provedor externo, pode
ser utilizado ajustando as variáveis `CULTIVAR_S3_*`.

## Contas e sessões

A primeira entrega funcional implementa contas locais com e-mail e senha:

| Rota | Descrição |
| --- | --- |
| `GET /cadastro` | Formulário de criação de conta |
| `POST /cadastro` | Cria a pessoa, a conta e inicia a sessão |
| `GET /entrar` | Formulário de entrada |
| `POST /entrar` | Autentica e inicia a sessão |
| `POST /sair` | Revoga a sessão atual |
| `GET /conta` | Área autenticada com os dados da conta |
| `GET /perfil/{identificador}` | Perfil público da pessoa |

As senhas são armazenadas com **Argon2id**. As sessões usam tokens opacos de alta
entropia; apenas o hash SHA-256 do token é persistido. Os cookies são `HttpOnly`,
`SameSite=Lax` e `Secure` em produção. Formulários são protegidos por CSRF com
token de dupla submissão, e as rotas de cadastro e entrada têm limitação de
tentativas por endereço.

A verificação de e-mail e a recuperação de senha ainda não foram implementadas.
Até lá, contas novas ficam com `email_verified_at` vazio.

## Perfil e identidade federada

O perfil pode ser editado em `GET /conta/perfil`, com nome de exibição e
biografia de até 500 caracteres. A mesma identidade é publicada para o
Fediverso:

| Rota | Descrição |
| --- | --- |
| `GET /.well-known/webfinger?resource=acct:...` | Descoberta WebFinger |
| `GET /.well-known/nodeinfo` | Descoberta do NodeInfo 2.0 |
| `GET /nodeinfo/2.0` | Informações e estatísticas da instância |
| `GET /actors/{id}` | Actor ActivityPub (`Person`) |
| `GET /actors/{id}/outbox` | Caixa de saída (coleção vazia por enquanto) |
| `GET /actors/{id}/followers` | Seguidores (coleção vazia por enquanto) |
| `GET /actors/{id}/following` | Pessoas seguidas (coleção vazia por enquanto) |
| `POST /actors/{id}/inbox` | Recebimento federado (retorna 501 por enquanto) |

Os identificadores de Actor são opacos e estáveis, no formato
`{URL pública}/actors/{uuid}`. O identificador público é exibido como
`@nome@domínio` e resolvido por WebFinger. Exemplos:

```sh
curl 'http://localhost:8080/.well-known/webfinger?resource=acct:ana@localhost:8080'
curl -H 'Accept: application/activity+json' http://localhost:8080/actors/<uuid>
curl http://localhost:8080/nodeinfo/2.0
```

As coleções e o recebimento de atividades ainda são placeholders. A entrega e a
validação de assinaturas HTTP serão implementadas junto com as publicações e o
outbox transacional.

## Diário de cultivo e práticas

O diário registra a experiência de cultivo ao longo do tempo. Cada ciclo reúne
contexto, registros datados e um resultado final. A biblioteca de práticas reúne
conhecimento reproduzível, como adubos caseiros e construção de recipientes.

| Rota | Descrição |
| --- | --- |
| `GET /diario` | Descoberta de ciclos públicos, com filtros |
| `GET /diario/meu` | Ciclos da pessoa autenticada, incluindo privados |
| `GET /diario/novo` | Formulário de criação de ciclo |
| `POST /diario` | Cria um ciclo privado por padrão |
| `GET /diario/{id}` | Ciclo e registros visíveis |
| `GET /diario/{id}/editar` | Edição do ciclo pela pessoa responsável |
| `POST /diario/{id}/editar` | Atualiza contexto, situação e resultado |
| `GET /diario/{id}/registros/novo` | Formulário de novo registro |
| `POST /diario/{id}/registros` | Cria um registro datado |
| `GET /praticas` | Biblioteca pública de práticas |
| `GET /praticas/minhas` | Práticas da pessoa autenticada, incluindo rascunhos |
| `GET /praticas/nova` | Formulário de criação de prática |
| `POST /praticas` | Cria rascunho ou publica |
| `GET /praticas/{id}` | Detalhe da prática visível |
| `GET /praticas/{id}/editar` | Edição pela pessoa autora |
| `POST /praticas/{id}/editar` | Atualiza e registra nova versão |
| `POST /praticas/{id}/publicar` | Publica uma prática validada |
| `POST /praticas/{id}/arquivar` | Arquiva uma prática |

Ciclos são privados por padrão; a publicação é explícita. Registros podem herdar
a visibilidade do ciclo ou restringi-la, nunca ampliá-la. Práticas publicadas
exigem pelo menos uma etapa e uma licença definida. O vínculo entre um registro e
uma prática pública materializa o ciclo **prática utilizada → aplicação →
resultado observado**.

As tabelas `cultivation_cycles`, `cultivation_entries`, `practices`,
`practice_versions`, `media_assets`, `cultivation_entry_media` e `practice_media`
são criadas pela migração `00003_cultivation.sql`.

Limitações atuais:

- O upload de mídia ainda não está disponível; o esquema já reserva as tabelas.
- A visibilidade `followers` é aceita no modelo, mas ainda não é aplicada na
  leitura: apenas a pessoa responsável e o público geral têm acesso garantido.
- A federação de ciclos e práticas ainda não publica atividades no outbox.

## Stack completa em containers

Para executar a aplicação, o trabalhador, o banco e o armazenamento juntos:

```sh
make up
make logs
make down
```

O serviço `migrate` executa as migrações antes de iniciar a aplicação e o
trabalhador. Para remover também os volumes de dados:

```sh
make reset   # destrutivo
```

## Comandos

| Comando | Descrição |
| --- | --- |
| `make build` | Compila o binário em `bin/cultivar` |
| `make run` | Inicia o servidor HTTP local |
| `make worker` | Inicia o trabalhador assíncrono |
| `make migrate` | Aplica as migrações pendentes |
| `make migrate-down` | Reverte a última migração |
| `make migrate-status` | Exibe o status das migrações |
| `make test` | Executa os testes |
| `make vet` | Executa o `go vet` |
| `make fmt` | Formata o código |
| `make lint` | Executa o `golangci-lint` |
| `make ci` | Executa `vet`, testes e build |
| `make deps` | Sobe PostgreSQL e o armazenamento S3 local |
| `make up` / `make down` | Sobe e derruba a stack completa |
| `make reset` | Remove containers e volumes |
| `make clean` | Remove artefatos de build |

## Configuração

A configuração é carregada de variáveis de ambiente. O arquivo `.env.example`
documenta os valores usados no desenvolvimento local.

| Variável | Padrão | Descrição |
| --- | --- | --- |
| `CULTIVAR_ENV` | `development` | `development` ou `production` |
| `CULTIVAR_HTTP_ADDR` | `:8080` | Endereço de escuta do servidor HTTP |
| `CULTIVAR_LOCAL_DOMAIN` | `localhost:8080` | Domínio canônico da instância |
| `CULTIVAR_INSTANCE_NAME` | `Cultivar` | Nome exibido da instância |
| `CULTIVAR_LOG_LEVEL` | `info` | `debug`, `info`, `warn` ou `error` |
| `CULTIVAR_DATABASE_URL` | `postgres://...` | Conexão com o PostgreSQL |
| `CULTIVAR_SECRET_KEY` | vazio | Segredo de sessões/chaves; obrigatório em produção (32+ caracteres) |
| `CULTIVAR_SHUTDOWN_TIMEOUT` | `15s` | Tempo máximo de encerramento gracioso |
| `CULTIVAR_WORKER_POLL_INTERVAL` | `5s` | Intervalo de verificação periódica do trabalhador |
| `CULTIVAR_SESSION_IDLE_TIMEOUT` | `12h` | Prazo de inatividade da sessão |
| `CULTIVAR_SESSION_ABSOLUTE_TIMEOUT` | `720h` | Prazo absoluto da sessão (30 dias) |
| `CULTIVAR_COOKIE_SECURE` | `false` | Envia cookies apenas por HTTPS; sempre ativo em produção |
| `CULTIVAR_S3_ENDPOINT` | `localhost:9000` | Endpoint S3 (host:porta) |
| `CULTIVAR_S3_REGION` | `us-east-1` | Região S3 |
| `CULTIVAR_S3_BUCKET` | `cultivar` | Bucket de mídia |
| `CULTIVAR_S3_ACCESS_KEY` | `cultivar` | Credencial S3 |
| `CULTIVAR_S3_SECRET_KEY` | `cultivar123` | Segredo S3 |
| `CULTIVAR_S3_USE_SSL` | `false` | Usa TLS na conexão com o S3 |

## Estrutura do repositório

```text
cmd/cultivar/          ponto de entrada (serve, worker, migrate, version)
internal/
  identity/            contas, pessoas, perfis e sessões
  cultivation/         diário de cultivo: ciclos, registros e visibilidade
  practices/           conhecimento e práticas: versões, publicação e arquivo
  publications/        publicações e interações (a implementar)
  federation/          ActivityStreams, WebFinger e NodeInfo
  platform/
    config/            carregamento e validação de configuração
    database/          pool PostgreSQL
    logging/           logs estruturados
    storage/           cliente S3
  web/                 roteador, handlers, templates e ativos
  worker/              processamento assíncrono
migrations/            migrações SQL versionadas
docs/                  visão, arquitetura, roadmap e ADRs
```

## Migrações

As migrações usam o formato do [goose](https://github.com/pressly/goose) e são
embutidas no binário.

Para criar uma nova migração, adicione um arquivo em `migrations/` seguindo a
numeração sequencial:

```sql
-- +goose Up
CREATE TABLE exemplo (
    id BIGSERIAL PRIMARY KEY
);

-- +goose Down
DROP TABLE exemplo;
```

Depois execute:

```sh
make migrate
make migrate-status
```

As migrações devem seguir o padrão expandir/contrair para permitir atualizações
sem indisponibilidade, conforme o ADR 0010.

## Federação

A identidade federada já publica WebFinger, Actors `Person` e NodeInfo. A
tradução entre o domínio e o ActivityPub fica em `internal/federation`, conforme
os ADRs 0004 e 0009.

A entrega de atividades, a validação de assinaturas HTTP e o recebimento no inbox
ainda não estão implementados. Ciclos públicos e práticas publicadas serão
traduzidos para `Note`/`Article` em uma entrega futura; até lá, o outbox permanece
uma coleção vazia.

## Solução de problemas

- **Porta ocupada:** ajuste `CULTIVAR_HTTP_ADDR` ou libere as portas `8080`,
  `5432`, `9000` e `9001`.
- **`/readyz` retorna erro de banco:** confirme que o PostgreSQL está saudável
  com `docker compose ps` e que `CULTIVAR_DATABASE_URL` está correta.
- **Armazenamento S3 indisponível:** o servidor inicia mesmo assim, mas uploads
  falharão até que o bucket esteja acessível. Verifique `make deps` e o serviço
  `s3mock` com `docker compose ps`.
- **Dados inconsistentes no ambiente local:** use `make reset` para recriar os
  volumes (isso apaga os dados locais).
