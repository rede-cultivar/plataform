# ADR 0012 — Modos de cadastro e administração da instância

**Status:** Proposto
**Data:** 2026-11-16

## Contexto

O ADR 0008 definiu os papéis de instância — administrador, moderador e membro —
e estabeleceu que a autorização é verificada na camada de aplicação. A
implementação inicial permitia cadastro aberto e criava sessão imediatamente
após a criação da conta.

A instância pública entrou em operação e passou a precisar de controle sobre
novos cadastros. O objetivo é permitir que a administração decida, sem
reimplantar a aplicação, se o cadastro está aberto, sujeito a aprovação ou
fechado. O modelo de referência é o do Mastodon, tanto para os modos quanto para
a experiência de gestão.

Restrições consideradas:

- A configuração deve poder ser alterada por uma pessoa administradora.
- O bloqueio precisa valer no servidor, não apenas esconder o formulário.
- A federação deve refletir o estado real do cadastro no NodeInfo.
- O primeiro administrador não pode ser promovido pelo cadastro público.
- Comentários, documentação e interface seguem em português.

## Decisão

### 1. Modos de cadastro

A instância terá uma configuração persistida com três modos, equivalentes aos
do Mastodon:

| Modo | Comportamento |
| --- | --- |
| `open` | Qualquer pessoa pode criar conta e entra imediatamente. |
| `approved` | O cadastro é aceito, mas a conta fica `pending` até aprovação administrativa. |
| `none` | Novos cadastros estão fechados. |

No modo `none`, `GET /cadastro` exibe a página de cadastros fechados com uma
mensagem configurável, `POST /cadastro` responde `403`, os links de cadastro
deixam de aparecer e o NodeInfo publica `openRegistrations=false`.

A mensagem de cadastros fechados tem limite de 500 caracteres e um texto padrão
quando não configurada.

### 2. Persistência e operação

As configurações ficam na tabela `site_settings`, com uma única linha, e são
alteradas pelo painel administrativo. Não são necessárias variáveis de ambiente
novas nem reinício da aplicação.

A migração inicial semeia o modo `open` para preservar o comportamento
existente. O fechamento efetivo é uma decisão operacional tomada no painel.

### 3. Papéis e promoção do primeiro administrador

Os papéis seguem o ADR 0008: `admin`, `moderator` e `member`. O cadastro público
sempre cria contas com papel `member`, inclusive no modo `approved`.

O primeiro administrador é promovido por comando explícito no servidor:

```sh
cultivar admin promote <identificador>
```

Os comandos `cultivar admin demote` e `cultivar admin list` completam a operação.

### 4. Gestão de contas pendentes

O painel fica em `/admin/settings` e exige o papel `admin`. Além dos modos de
cadastro, ele lista as contas pendentes e oferece as ações:

| Rota | Ação |
| --- | --- |
| `POST /admin/accounts/{accountID}/approve` | Ativa a conta pendente. |
| `POST /admin/accounts/{accountID}/reject` | Exclui a conta e a pessoa associada. |

A rejeição libera o e-mail e o identificador para um novo cadastro. Aprovações,
rejeições e alterações de configuração são registradas no log com a identificação
da pessoa administradora.

### 5. Experiência de cadastro e entrada

No modo `approved`, a página de cadastro informa que a conta depende de
aprovação. Após o envio, a pessoa vê uma confirmação e não recebe sessão. Ao
tentar entrar antes da aprovação, recebe uma mensagem específica de cadastro
aguardando aprovação, sem revelar detalhes além do necessário.

## Alternativas consideradas

### 1. Variável de ambiente

**Vantagens:**
- Implementação imediata, sem migração nem painel.

**Desvantagens:**
- Exigiria reiniciar ou reimplantar a aplicação para cada mudança.
- Não permitiria aprovação individual de contas.
- Concentraria decisões de produto na operação do servidor.

### 2. Modo single-user permanente

**Vantagens:**
- Simples e alinhado a uma instância estritamente pessoal.

**Desvantagens:**
- Não atende à possibilidade de reabrir o cadastro no futuro.
- Não oferece o fluxo de aprovação nem a gestão de pendentes.
- Distancia a instância do modelo de gestão do Fediverso.

### 3. Aprovação por e-mail

**Vantagens:**
- Não exige intervenção administrativa para cada conta.

**Desvantagens:**
- A verificação de e-mail ainda não está implementada.
- Não substitui a decisão editorial sobre quem entra na instância.

## Consequências

- A administração passa a controlar o cadastro pelo painel, sem alterar o
  ambiente de execução.
- Contas pendentes não criam sessão e não acessam a área autenticada.
- O NodeInfo e as páginas públicas refletem o modo configurado.
- A migração adiciona `accounts.role`, o status `pending` e a tabela
  `site_settings`.
- A promoção do primeiro administrador é um passo operacional documentado em
  `docs/DEVELOPMENT.md`.
- O painel cresce como ponto de partida para a moderação e a administração de
  contas previstas no ADR 0008.

## Escopo e decisões futuras

Este ADR não define:

- Verificação de e-mail, recuperação de senha ou convites.
- Moderação de conteúdo, denúncias ou suspensão de contas.
- Papéis em grupos e organizações.
- Registro de auditoria consultável pela interface.
- Visibilidade de publicações restrita a seguidores.

## Referências

- [ADR 0008 — Autenticação e autorização](0008-autenticacao-e-autorizacao.md)
- [Documentação de desenvolvimento](../DEVELOPMENT.md)
- [Mastodon — Configuração de cadastro](https://docs.joinmastodon.org/admin/config/#registrations)
