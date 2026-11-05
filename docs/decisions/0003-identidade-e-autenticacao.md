# ADR 0003 — Identidade e autenticação

**Status:** Proposto  
**Data:** 2026-11-05

## Contexto

A plataforma será uma aplicação independente, integrada ao Fediverso por meio do ActivityPub.

Essa característica exige distinguir conceitos que, em aplicações tradicionais, muitas vezes são tratados como uma única coisa:

- A pessoa que participa da plataforma.
- A conta utilizada para acessar a aplicação.
- A identidade pública utilizada para interagir na rede.
- A identidade federada utilizada na comunicação com outras instâncias.

Também será necessário permitir que grupos e organizações tenham presença própria na plataforma, sem que isso implique necessariamente a criação de contas de acesso individuais para cada entidade.

A arquitetura deverá contemplar esses conceitos sem introduzir complexidade desnecessária na primeira versão.

## Decisão proposta

A plataforma adotará inicialmente um modelo de identidade baseado em contas locais, com separação conceitual entre autenticação, perfil e identidade federada.

### 1. Conta de acesso

Representa as credenciais e os mecanismos utilizados por uma pessoa para acessar a plataforma.

Responsabilidades:

- Autenticar o participante.
- Manter informações necessárias à segurança da conta.
- Permitir recuperação e alteração de credenciais.
- Controlar sessões e mecanismos de acesso.
- Manter o estado da conta, como ativa, suspensa ou encerrada.

Uma conta de acesso não deverá conter diretamente todas as informações do perfil público ou da representação federada.

### 2. Pessoa e perfil

A pessoa representa o participante no domínio da aplicação.

O perfil contém suas informações de apresentação e participação, como:

- Nome de exibição.
- Identificador público.
- Biografia.
- Imagem de perfil.
- Interesses.
- Informações públicas de participação.

A pessoa será associada a uma conta de acesso, mas os conceitos permanecerão separados.

### 3. Identidade federada

Cada participante local poderá possuir uma identidade pública compatível com ActivityPub.

Essa identidade será representada por um Actor, com identificador URI próprio e recursos necessários à comunicação federada.

Responsabilidades:

- Identificar o participante na rede.
- Disponibilizar informações públicas.
- Expor Inbox e Outbox.
- Permitir descoberta por mecanismos como WebFinger.
- Assinar atividades federadas.
- Receber e distribuir atividades conforme as regras de visibilidade.

A identidade federada deverá ser independente da implementação interna da conta de acesso.

### 4. Identidades externas

A arquitetura deverá permitir que uma pessoa associe futuramente identidades externas à sua conta local.

Essa associação poderá permitir, por exemplo, acompanhar contas externas e administrar relacionamentos com participantes de outras instâncias.

Entretanto, a associação de uma identidade externa não será obrigatória para utilizar a plataforma.

A autenticação na aplicação e a autenticação em outras instâncias permanecerão independentes.

## Identidades coletivas

Grupos e organizações possuem identidade própria no domínio da aplicação.

A arquitetura deverá permitir que essas entidades tenham representação pública e, futuramente, identidade federada própria.

Uma identidade coletiva não deverá ser confundida com uma conta de acesso.

O acesso administrativo a um grupo ou organização será realizado por pessoas autenticadas, conforme suas respectivas permissões.

A representação federada dessas entidades será detalhada em uma decisão específica, considerando as diferenças de suporte entre implementações do ActivityPub.

## Princípios de segurança

A implementação deverá observar os seguintes princípios:

- Credenciais não serão armazenadas em texto puro.
- Informações privadas não serão expostas como parte do perfil público.
- A autenticação não deverá depender exclusivamente da identidade federada.
- A autorização deverá ser verificada nas operações que alteram dados ou representam ações de terceiros.
- Chaves privadas utilizadas na federação deverão ser protegidas.
- A associação de identidades externas deverá exigir verificação de controle sobre a identidade associada.
- Suspensão ou encerramento de contas deverá considerar seus efeitos sobre as identidades federadas.

## Alternativas consideradas

### 1. Utilizar exclusivamente identidades externas

A plataforma permitiria acesso apenas por meio de contas existentes no Fediverso.

**Vantagens:**
- Menor responsabilidade inicial sobre autenticação.
- Aproveitamento de identidades já existentes.
- Menor necessidade de gerenciamento de credenciais locais.

**Desvantagens:**
- Dependência de provedores externos.
- Experiência de entrada mais complexa para novos participantes.
- Menor controle sobre o ciclo de vida das contas.
- Dificuldade de estabelecer uma experiência independente.

### 2. Utilizar exclusivamente contas locais, sem identidade federada

A plataforma manteria suas próprias contas e perfis, sem representar participantes como Actors.

**Vantagens:**
- Implementação mais simples.
- Controle integral sobre identidade e autenticação.

**Desvantagens:**
- Impossibilidade de participação federada nativa.
- Isolamento em relação ao Fediverso.

### 3. Separar conta local, perfil e identidade federada

A aplicação manteria contas próprias, perfis de pessoas e identidades federadas independentes.

**Vantagens:**
- Autonomia sobre autenticação.
- Integração nativa com o Fediverso.
- Separação entre segurança, apresentação e interoperabilidade.
- Possibilidade de associar identidades externas no futuro.
- Suporte à representação de entidades coletivas.

**Desvantagens:**
- Maior quantidade de conceitos e relações.
- Necessidade de sincronizar informações públicas entre domínio e federação.
- Maior responsabilidade sobre segurança e ciclo de vida das identidades.

## Consequências

A decisão estabelece:

- Contas locais como mecanismo inicial de acesso.
- Separação entre conta, pessoa, perfil e identidade federada.
- Identidades públicas compatíveis com ActivityPub para participantes locais.
- Possibilidade futura de associação de identidades externas.
- Administração de grupos e organizações por pessoas autenticadas.
- Independência entre autenticação local e comunicação federada.

## Escopo e decisões futuras

Este ADR não determina ainda:

- O mecanismo específico de autenticação.
- O uso de sessões, tokens ou outros mecanismos.
- O provedor ou serviço de autenticação.
- A política de recuperação de contas.
- O modelo detalhado de permissões.
- A representação federada de grupos e organizações.
- A possibilidade de múltiplas identidades públicas associadas à mesma pessoa.
- As regras de migração, exclusão e transferência de identidades.

Essas decisões deverão ser registradas separadamente.

## Referências

- [ActivityPub — W3C Recommendation](https://www.w3.org/TR/activitypub/)
- [WebFinger — RFC 7033](https://www.rfc-editor.org/rfc/rfc7033)
- [HTTP Signatures — RFC 9421](https://www.rfc-editor.org/rfc/rfc9421)