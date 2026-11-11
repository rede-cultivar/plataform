# ADR 0001 — Aplicação própria integrada ao Fediverso

**Status:** Aceito  
**Data:** 2026-11-05

## Contexto

O projeto pretende oferecer uma rede social descentralizada especializada em cultivo, agroecologia, biodiversidade, colaboração comunitária e mobilização socioambiental.

A interoperabilidade com o Fediverso é um requisito estratégico, permitindo que participantes da plataforma interajam com pessoas e comunidades de outras instâncias.

Entretanto, o domínio do produto possui conceitos e funcionalidades que não são integralmente contemplados pelos modelos oferecidos pelas implementações existentes do Fediverso.

Entre esses conceitos estão:

- Projetos de cultivo e iniciativas socioambientais.
- Grupos e comunidades.
- Organizações com identidade própria.
- Ofertas e necessidades de recursos.
- Atividades, encontros e mobilizações.

A aplicação precisa preservar a capacidade de desenvolver esses conceitos de acordo com suas próprias regras de negócio, sem ficar limitada ao modelo de dados ou às funcionalidades de uma implementação externa.

## Decisão

O projeto será desenvolvido como uma aplicação própria, com identidade, interface, backend e modelo de domínio independentes.

A interoperabilidade com o Fediverso será implementada por meio de protocolos e padrões abertos, tendo o ActivityPub como principal mecanismo de comunicação federada.

A arquitetura deverá manter uma separação explícita entre:

1. O domínio e as regras de negócio da aplicação.
2. Os casos de uso e serviços da aplicação.
3. A infraestrutura e os mecanismos de persistência.
4. A integração com ActivityPub e demais protocolos do Fediverso.

O modelo interno de domínio não deverá depender diretamente das representações ou estruturas específicas do ActivityPub.

## Alternativas consideradas

### 1. Customizar uma implementação existente, como Mastodon

Utilizar uma implementação existente como base e adaptá-la às necessidades do projeto.

**Vantagens:**
- Reaproveitamento de funcionalidades já desenvolvidas.
- Integração federada disponível desde o início.
- Redução do esforço inicial de implementação.

**Desvantagens:**
- Dependência do modelo de domínio e das decisões arquiteturais da implementação escolhida.
- Adaptações potencialmente complexas para representar projetos, grupos e organizações.
- Risco de manter funcionalidades e estruturas que não correspondem aos objetivos do produto.
- Menor autonomia sobre a evolução arquitetural.

### 2. Desenvolver uma aplicação independente sem integração federada

Construir uma rede social própria, sem implementar interoperabilidade com o Fediverso.

**Vantagens:**
- Liberdade completa sobre o modelo de domínio.
- Controle integral sobre arquitetura e funcionalidades.

**Desvantagens:**
- Isolamento em relação às comunidades existentes.
- Necessidade de construir uma rede de participantes independente.
- Perda dos benefícios de interoperabilidade e descentralização proporcionados pelo Fediverso.

### 3. Desenvolver uma aplicação própria integrada ao ActivityPub

Construir uma aplicação independente, com domínio próprio, implementando os protocolos necessários para interoperar com outras instâncias.

**Vantagens:**
- Autonomia sobre o modelo de domínio.
- Interoperabilidade com implementações existentes.
- Liberdade para desenvolver funcionalidades especializadas.
- Possibilidade de evolução independente da aplicação e da camada de federação.

**Desvantagens:**
- Maior esforço inicial de desenvolvimento.
- Responsabilidade pela implementação e manutenção da federação.
- Necessidade de tratar compatibilidade, segurança, entrega assíncrona e diferenças entre implementações.

## Consequências

A decisão implica:

- O projeto deverá manter seu próprio modelo de domínio.
- A integração federada será implementada como uma camada de interoperabilidade.
- A aplicação deverá possuir mecanismos próprios de autenticação, autorização e administração.
- A comunicação federada deverá considerar processamento assíncrono, idempotência, retentativas e tratamento de falhas.
- A compatibilidade deverá ser validada com implementações reais do Fediverso.
- Funcionalidades específicas do produto não deverão depender da existência de suporte equivalente em outras instâncias.

A decisão também implica maior responsabilidade técnica e operacional, especialmente na implementação e manutenção dos mecanismos de federação.

## Escopo da decisão

Este ADR estabelece a independência da aplicação e a adoção do ActivityPub como protocolo central de federação.

Não determina ainda:

- Se as identidades de pessoas serão exclusivamente nativas ou poderão ser associadas a contas externas.
- Como será feita a representação federada de grupos e organizações.
- Qual implementação do ActivityPub será utilizada como referência de compatibilidade.
- Qual linguagem, framework ou banco de dados será adotado.
- Qual será a infraestrutura de implantação.

Essas decisões deverão ser registradas separadamente.

## Referências

- [ActivityPub — W3C Recommendation](https://www.w3.org/TR/activitypub/)
- [ActivityStreams 2.0 — W3C Recommendation](https://www.w3.org/TR/activitystreams-core/)
- [Mastodon Documentation](https://docs.joinmastodon.org/)