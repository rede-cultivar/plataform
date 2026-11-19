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

O Docker Compose também carrega automaticamente as variáveis de `.env`. Para
usar um domínio público, ajuste `CULTIVAR_LOCAL_DOMAIN` (por exemplo,
`https://delorme.dev.br`) e mantenha `CULTIVAR_SECRET_KEY` fora do controle de
versão. O Compose usa `CULTIVAR_DOCKER_DATABASE_URL` e
`CULTIVAR_DOCKER_S3_ENDPOINT` para acessar os serviços pelos nomes internos
`postgres` e `s3mock`; `CULTIVAR_DATABASE_URL` e `CULTIVAR_S3_ENDPOINT` continuam
servindo à execução local pelo `make`.

As portas publicadas do app, PostgreSQL e S3Mock ficam vinculadas a `127.0.0.1`
por padrão. O Nginx no mesmo host pode encaminhar para `localhost:8080`, sem
expor diretamente o banco ou o S3Mock à internet.

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

### Persistência local para testes

O S3Mock mantém os objetos em memória e perde as mídias quando o container é
recriado. Para testes locais que precisam manter as imagens entre reinícios, use
o backend de arquivos:

```sh
# .env
CULTIVAR_STORAGE_DRIVER=filesystem
# Opcional fora do Docker; o padrão é ./tmp/media.
# CULTIVAR_STORAGE_PATH=./tmp/media
```

No Compose, o caminho padrão é `/data/media`, montado no volume nomeado
`media-data`. Assim as mídias sobrevivem a `docker compose restart`, `down` e
novos `up`. O S3Mock continua disponível e volta a ser usado quando
`CULTIVAR_STORAGE_DRIVER=s3`, que é o padrão.

O diretório `tmp/` já está no `.gitignore`. Para descartar as mídias locais,
remova `tmp/media` ou o volume com `make reset`.

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
| `GET /conta/seguidores` | Seguidores remotos da conta autenticada |
| `GET /perfil/{identificador}` | Perfil público da pessoa |

As senhas são armazenadas com **Argon2id**. As sessões usam tokens opacos de alta
entropia; apenas o hash SHA-256 do token é persistido. Os cookies são `HttpOnly`,
`SameSite=Lax` e `Secure` em produção. Formulários são protegidos por CSRF com
token de dupla submissão, e as rotas de cadastro e entrada têm limitação de
tentativas por endereço.

A verificação de e-mail e a recuperação de senha ainda não foram implementadas.
Até lá, contas novas ficam com `email_verified_at` vazio.

A administração da instância é acessada em `/admin/settings` por contas com o
papel `admin`. Para promover a primeira conta, use o comando administrativo
após aplicar as migrações, por exemplo:

```sh
cultivar admin promote identificador
```

O painel permite deixar os cadastros abertos, exigir aprovação administrativa
ou fechá-los com uma mensagem configurável. Contas pendentes podem ser
aprovadas ou rejeitadas pela mesma página. O modo fechado também bloqueia o
`POST /cadastro`, não apenas o formulário, e é refletido no NodeInfo. A decisão
está registrada no
[ADR 0012](decisions/0012-modos-de-cadastro-e-administracao-da-instancia.md).

## Perfil e identidade federada

O perfil pode ser editado em `GET /conta/perfil`, com nome de exibição e
biografia de até 500 caracteres. A mesma identidade é publicada para o
Fediverso:

| Rota | Descrição |
| --- | --- |
| `GET /.well-known/webfinger?resource=acct:...` | Descoberta WebFinger |
| `GET /.well-known/nodeinfo` | Descoberta do NodeInfo 2.0 |
| `GET /nodeinfo/2.0` | Informações e estatísticas da instância |
| `GET /actors/{id}` | Actor ActivityPub (`Person`), incluindo a chave pública |
| `GET /actors/{id}/outbox` | Publicações públicas de saída (apenas `Create`, sem versões removidas, restritas ou duplicadas) |
| `GET /actors/{id}/followers` | Seguidores remotos aceitos |
| `GET /actors/{id}/following` | Pessoas seguidas na instância (contas remotas pendentes) |
| `POST /actors/{id}/inbox` | Recebe atividades com assinatura HTTP válida |
| `GET /notas/{id}` | Representação `Note` de um registro federado (token para `followers`) |
| `GET /activities/{id}` | Atividade federada de saída já publicada |

Os identificadores de Actor são opacos e estáveis, no formato
`{URL pública}/actors/{uuid}`. O identificador público é exibido como
`@nome@domínio` e resolvido por WebFinger. Exemplos:

```sh
curl 'http://localhost:8080/.well-known/webfinger?resource=acct:ana@localhost:8080'
curl -H 'Accept: application/activity+json' http://localhost:8080/actors/<uuid>
curl -H 'Accept: application/activity+json' http://localhost:8080/notas/<id>
curl http://localhost:8080/nodeinfo/2.0
```

O recebimento valida `HTTP Signature` (RSA-SHA256), `Digest` e uma janela de
relógio de cinco minutos antes de processar a atividade. `Follow` é aceito
automaticamente e gera uma atividade `Accept`; `Undo` remove o seguidor. As
demais atividades são registradas para garantir idempotência.

A página autenticada `GET /conta/seguidores` mostra os seguidores aceitos com
nome, avatar e link do perfil remoto. Os dados públicos de cada Actor são
buscados sob demanda e mantidos em cache na memória do processo por uma hora;
quando a instância remota não responde, a listagem degrada para o identificador
do Actor. As imagens de avatar são carregadas diretamente da instância remota
pelo navegador; por isso a política de segurança de conteúdo permite imagens
`https:` de qualquer origem.

### Seguir pessoas e visibilidade `followers`

Qualquer pessoa autenticada pode seguir outra pessoa da instância pelo botão no
perfil (`POST /perfil/{handle}/seguir`) e deixar de seguir
(`POST /perfil/{handle}/deixar`). O vínculo é imediato, sem pedido de aprovação,
e fica em `person_follows`. O perfil mostra as listas de seguidores e de pessoas
seguidas, além dos ciclos visíveis para quem consulta.

A visibilidade `followers` é aplicada na leitura: ciclos e registros com essa
visibilidade ficam acessíveis à pessoa responsável e a quem a segue. A regra
vale também para `GET /media/{id}`, que verifica o vínculo antes de servir as
imagens. Visitantes e pessoas que não seguem recebem `404` ao tentar abrir o
ciclo ou o registro.

Registros de ciclos `followers` também são federados. A `Note` usa apenas a
coleção de seguidores em `To`, sem incluir a coleção `Public`, e é entregue
somente aos seguidores remotos aceitos. O acesso remoto usa URLs de capacidade:
o identificador da `Note` e as URLs das imagens recebem um parâmetro `token`
derivado por HMAC do `CULTIVAR_SECRET_KEY`. Quem recebeu a publicação consegue
abrir `/notas/{id}` e `/media/{id}` mesmo sem sessão ou assinatura HTTP; quem não
recebeu o link vê `404`. O mesmo token não é individual por seguidor, então quem
guardou o link continua com acesso enquanto o registro estiver federado. Ao
deixar de seguir, a pessoa deixa de receber novas publicações, mas mantém o que
já tinha recebido.

O link **Ver registro completo** do cartão federado aponta para a página do
ciclo com um `token` derivado do identificador do ciclo. A página aberta por
esse token equivale à visão de seguidor: mostra registros públicos e `followers`,
nunca privados, e inclui o token nas URLs das fotos. Ela é somente leitura;
editar, responder e remover continuam exigindo uma conta com permissão. Sem
token, com token inválido ou com token de outro ciclo, a página responde `404`.

Quando a visibilidade efetiva muda entre `public` e `followers`, o objeto
federado muda de identificador: a versão anterior recebe `Delete` e a nova é
publicada como `Create`. Isso evita reutilizar um identificador já removido e
mantém os destinatários corretos em cada versão.

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
| `GET /diario/{id}` | Ciclo e registros visíveis (token de capacidade para seguidores) |
| `GET /diario/{id}/editar` | Edição do ciclo pela pessoa responsável |
| `POST /diario/{id}/editar` | Atualiza contexto, situação e resultado |
| `GET /diario/{id}/registros/novo` | Formulário de novo registro |
| `POST /diario/{id}/registros` | Cria um registro datado, com até 4 fotos |
| `POST /diario/{cycleID}/registros/{entryID}/fotos/{mediaID}/apagar` | Remove uma foto vinculada ao registro |
| `GET /media/{id}` | Serve a imagem quando a pessoa tem permissão |
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

As fotos do diário passam por validação e normalização antes de serem gravadas
no armazenamento de objetos. Cada registro aceita até 4 imagens de 10 MB, nos
formatos JPEG, PNG e GIF. Imagens JPEG e PNG são recodificadas, o que corrige a
orientação EXIF e remove metadados pessoais, como localização e modelo do
aparelho. O maior lado é reduzido proporcionalmente para no máximo 1024 pixels;
imagens menores não são ampliadas. GIFs são preservados como enviados para
manter a animação. O banco guarda apenas os metadados e a chave do objeto.

A pessoa responsável pelo ciclo pode remover fotos pela linha do tempo, com o
botão **Remover foto** exibido em cada imagem, ou marcando a foto no formulário
de edição do registro. As duas formas exigem sessão, CSRF e vínculo entre a foto
e o registro.

O acesso a `GET /media/{id}` é verificado no servidor: a pessoa responsável vê
todas as suas imagens, enquanto as demais só acessam imagens vinculadas a
registros de ciclos públicos, a registros de visibilidade `followers` de quem
elas seguem, ou a práticas publicadas. Imagens de ciclos privados continuam
protegidas pela mesma regra.

Limitações atuais:

- Práticas ainda não são federadas; apenas registros do diário geram `Note`.
- A coleção `following` do Actor lista apenas as pessoas seguidas nesta
  instância; o acompanhamento de contas remotas ainda não está implementado.
- O token de capacidade é derivado do identificador e do segredo da instância,
  não de cada seguidor. Não há revogação individual: um link já entregue
  continua válido enquanto o conteúdo permanecer federado.
- Uma publicação removida com `Delete` não pode ser recriada com o mesmo
  identificador: o Mastodon guarda o `Tombstone` e ignora um novo `Create` para
  o mesmo objeto. A troca de visibilidade contorna isso usando um identificador
  novo para a versão restrita.
- Chaves privadas de Actor ficam no banco nesta etapa; proteção, rotação e
  recuperação de chaves serão tratadas antes da operação em produção.

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

O binário também expõe comandos operacionais:

| Comando | Descrição |
| --- | --- |
| `cultivar migrate <up\|down\|status\|version>` | Executa migrações |
| `cultivar admin <promote\|demote\|list>` | Administra papéis |
| `cultivar federation refresh <registro>` | Reconstrói e redistribui um registro (ou o remove, se não for mais federado) |

## Configuração

A configuração é carregada de variáveis de ambiente. O arquivo `.env.example`
documenta os valores usados no desenvolvimento local.

| Variável | Padrão | Descrição |
| --- | --- | --- |
| `CULTIVAR_ENV` | `development` | `development` ou `production` |
| `CULTIVAR_HTTP_ADDR` | `:8080` | Endereço de escuta do servidor HTTP |
| `CULTIVAR_LOCAL_DOMAIN` | `localhost:8080` | Domínio ou URL pública canônica da instância |
| `CULTIVAR_INSTANCE_NAME` | `Cultivar` | Nome exibido da instância |
| `CULTIVAR_LOG_LEVEL` | `info` | `debug`, `info`, `warn` ou `error` |
| `CULTIVAR_DATABASE_URL` | `postgres://...` | Conexão com o PostgreSQL |
| `CULTIVAR_SECRET_KEY` | vazio | Segredo de sessões/chaves; obrigatório em produção (32+ caracteres) |
| `CULTIVAR_SHUTDOWN_TIMEOUT` | `15s` | Tempo máximo de encerramento gracioso |
| `CULTIVAR_WORKER_POLL_INTERVAL` | `5s` | Intervalo de verificação periódica do trabalhador |
| `CULTIVAR_SESSION_IDLE_TIMEOUT` | `12h` | Prazo de inatividade da sessão |
| `CULTIVAR_SESSION_ABSOLUTE_TIMEOUT` | `720h` | Prazo absoluto da sessão (30 dias) |
| `CULTIVAR_COOKIE_SECURE` | `false` | Envia cookies apenas por HTTPS; sempre ativo em produção |
| `CULTIVAR_STORAGE_DRIVER` | `s3` | Backend de mídia: `s3` ou `filesystem` (teste local) |
| `CULTIVAR_STORAGE_PATH` | `./tmp/media` | Diretório do backend `filesystem` (no Compose, `/data/media`) |
| `CULTIVAR_S3_ENDPOINT` | `localhost:9000` | Endpoint S3 (host:porta) |
| `CULTIVAR_S3_REGION` | `us-east-1` | Região S3 |
| `CULTIVAR_S3_BUCKET` | `cultivar` | Bucket de mídia |
| `CULTIVAR_S3_ACCESS_KEY` | `cultivar` | Credencial S3 |
| `CULTIVAR_S3_SECRET_KEY` | `cultivar123` | Segredo S3 |
| `CULTIVAR_S3_USE_SSL` | `false` | Usa TLS na conexão com o S3 |

O Docker Compose usa variáveis adicionais para publicar portas e apontar para os
serviços internos: `CULTIVAR_HTTP_PORT`, `POSTGRES_*`, `S3MOCK_PORT`,
`CULTIVAR_DOCKER_DATABASE_URL` e `CULTIVAR_DOCKER_S3_ENDPOINT`. Elas são
documentadas em `.env.example` e não alteram a execução direta com `make run`.

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
    storage/           backends de mídia (S3 e filesystem)
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

A federação usa ActivityPub como transporte definitivo, com o Mastodon como
referência de compatibilidade, conforme os ADRs 0001, 0004 e 0009. A tradução
entre o domínio e o ActivityStreams fica em `internal/federation`.

Fluxo implementado:

- **Publicação:** ao criar um registro em um ciclo público ou `followers`, o
  registro é traduzido para uma `Note`, com as fotos como `Image` em
  `attachment`, e distribuído em uma atividade `Create` para os seguidores
  aceitos. Conteúdo público inclui a coleção `Public` e os seguidores;
  conteúdo `followers` usa apenas os seguidores e endereços com token de
  capacidade. A `Note` fica disponível em `GET /notas/{id}` e a atividade em
  `GET /activities/{id}`.
  O conteúdo funciona como uma ficha do registro: identifica o ciclo (nome,
  espécie e variedade), traz o contexto do cultivo (ambiente, local e início),
  a data e o estágio do registro, o relato, medidas, intervenções, ocorrências,
  a prática vinculada e o link para a página completa. As fotos vão como anexos.
  O outbox do Actor lista apenas a `Create` pública mais recente de cada objeto;
  versões removidas, restritas a seguidores ou substituídas não aparecem.
- **Atualização:** ao editar um registro federado ou incluir/remover fotos, a
  `Note` é reconstruída e distribuída em uma atividade `Update`, com o mesmo
  identificador. A instância remota substitui o conteúdo e os anexos, exibindo a
  marca de edição. Registros ainda não publicados são enviados como `Create`.
  Quando o registro deixa de ser federado, uma atividade `Delete` com
  `Tombstone` remove o status remoto. Se a visibilidade efetiva muda entre
  público e seguidores, a versão anterior recebe `Delete` e a nova é publicada
  como `Create`, com identificador próprio. O comando
  `cultivar federation refresh <registro>` refaz o envio adequado para corrigir
  dados, remover uma mídia com problema ou despublicar.
- **Edição do ciclo:** como o cartão usa dados do ciclo, editar o ciclo também
  reconcilia os registros. Se ele deixa de ser federado, os registros que eram
  federados recebem `Delete`; se os campos do cartão mudam (nome, espécie,
  variedade, ambiente, local ou início), os registros federados recebem
  `Update`. Ao tornar um ciclo público ou `followers`, os registros são
  publicados pela primeira vez. Alterações em estágio, status, substrato e
  demais campos internos não geram tráfego.
  Uma publicação removida com `Delete` não pode ser recriada com o mesmo
  identificador: o Mastodon guarda o `Tombstone` e ignora um novo `Create` para
  o mesmo objeto. Republicar um registro já removido exigiria um identificador
  novo, o que ainda não está implementado.
- **Entrega:** as entregas são gravadas em `federation_deliveries` na mesma
  operação e processadas pelo trabalhador (`make worker`), com recuo exponencial
  e limite de tentativas. As requisições levam `Date`, `Digest` e
  `HTTP Signature`.
- **Recebimento:** `POST /actors/{id}/inbox` valida a assinatura, o `Digest` e a
  data; `Follow` gera `Accept`, `Undo` remove o seguidor e as demais atividades
  são persistidas para garantir idempotência. Quando o documento do Actor remoto
  responde `404` ou `410`, a atividade é descartada e o inbox responde `202`, em
  vez de `500`: o Actor não existe mais e a instância de origem não deve
  retentar.
- **Interações:** respostas (`Create` de uma `Note` com `inReplyTo`), curtidas
  (`Like`), impulsos (`Announce`), edições de resposta (`Update`) e remoções
  (`Delete`/`Undo`) são vinculados ao registro quando a Note citada pertence a
  esta instância e o registro ainda é federado. Em conteúdo `followers`, só são
  aceitas interações de um Actor remoto confirmado como seguidor. O texto das
  respostas é extraído do HTML e guardado em texto simples, limitado a 4000
  caracteres. Cada ator tem no máximo uma curtida e um impulso por registro;
  novas atividades do mesmo tipo atualizam a existente. As interações aparecem
  na página do ciclo, junto ao registro, com contadores e respostas. `Delete` de
  uma conta remove também suas interações. O Mastodon não oferece o botão de
  impulso em publicações `followers`; o `Announce` continua aceito pelo
  protocolo, mas não pode ser disparado por essa interface.
- **Respostas enviadas:** o dono do ciclo pode responder a uma mensagem recebida
  na própria página do registro. A resposta é uma `Note` em `/respostas/{id}`,
  com `inReplyTo` apontando para a mensagem original, menção ao autor e entrega
  ao inbox de quem respondeu e aos seguidores. Em registro `followers`, a
  resposta também usa apenas a coleção de seguidores e a pessoa respondida, sem
  a coleção `Public`. Ela é persistida em `federation_replies` com
  `direction = 'outbound'` e aparece no mesmo fio, identificada como resposta
  local. A Note só é servida enquanto o registro continuar federado.
- **Outbox:** a coleção lista apenas atividades `Create` públicas. Atividades de
  controle, como `Accept`, não são publicações e não entram no `totalItems`, que
  outras instâncias usam como contagem de publicações.
- **Novos seguidores:** o Mastodon não importa itens do outbox ao abrir um
  perfil; ele mostra apenas o que recebeu. Por isso, ao aceitar um `Follow`, as
  publicações públicas e `followers` mais recentes são reenfileiradas para o
  novo seguidor. Objetos já removidos com `Delete` ficam de fora. Como o Follow
  já foi aceito, o novo seguidor está autorizado a receber o conteúdo restrito.
- **Chaves:** cada Actor recebe um par RSA 2048 em `actor_keys`; a chave pública
  é exposta em `publicKey` no documento do Actor.
- **Segurança:** o cliente remoto bloqueia endereços privados, loopback e de
  metadados (SSRF), limita redirecionamentos e o tamanho dos documentos
  remotos. Em desenvolvimento, endereços privados são liberados para permitir
  testes com instâncias locais.

O esquema está nas migrações `00004_federation.sql` (`actor_keys`,
`federation_follows`, `federation_activities` e `federation_deliveries`),
`00006_interactions.sql` (`federation_replies` e `federation_reactions`) e
`00007_reply_outbound.sql` (respostas enviadas e inbox do autor respondido).

Para validar com uma instância real, a instância precisa ser acessível
publicamente por HTTPS. Configure `CULTIVAR_LOCAL_DOMAIN` com a URL pública
(por exemplo, `https://cultivar.example`), exponha a porta local por um túnel ou
proxy reverso e siga o Actor a partir de uma conta no Mastodon. Práticas ainda
não são federadas; alterações em registros publicados são enviadas como
`Update` e a despublicação como `Delete`.

## Solução de problemas

- **Porta ocupada:** ajuste `CULTIVAR_HTTP_ADDR` ou libere as portas `8080`,
  `5432`, `9000` e `9001`.
- **`/readyz` retorna erro de banco:** confirme que o PostgreSQL está saudável
  com `docker compose ps` e que `CULTIVAR_DATABASE_URL` está correta.
- **Armazenamento S3 indisponível:** o servidor inicia mesmo assim, mas uploads
  falharão até que o bucket esteja acessível. Verifique `make deps` e o serviço
  `s3mock` com `docker compose ps`.
- **Foto não aparece no registro:** confirme que o upload retornou sucesso e que
  o serviço S3 está acessível. Imagens de ciclos privados só são servidas para a
  pessoa responsável.
- **Dados inconsistentes no ambiente local:** use `make reset` para recriar os
  volumes (isso apaga os dados locais).
