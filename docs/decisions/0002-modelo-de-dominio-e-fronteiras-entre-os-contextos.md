# ADR 0002 — Modelo de domínio e fronteiras entre os contextos

**Status:** Aceito  
**Data:** 2026-11-05

## Contexto

A plataforma pretende atender pessoas, grupos, organizações e iniciativas relacionadas ao cultivo, à agroecologia, à biodiversidade e à mobilização socioambiental.

Esses participantes possuem necessidades e formas de organização diferentes.

Uma pessoa possui identidade individual e participa da rede por meio de seu perfil. Um grupo representa um espaço de participação e relacionamento entre pessoas. Uma organização representa uma entidade coletiva com identidade própria, que pode existir independentemente da plataforma. Um projeto representa uma iniciativa concreta, que pode envolver pessoas, grupos e organizações.

Embora esses conceitos estejam relacionados, possuem responsabilidades, ciclos de vida e regras de negócio diferentes.

Um modelo excessivamente genérico dificultaria a evolução dessas regras e poderia tornar ambíguas as relações entre os participantes da plataforma.

## Decisão

O domínio será organizado em contextos funcionais com responsabilidades próprias.

As entidades centrais serão:

### 1. Identidade e participação

Responsável pelas identidades individuais e pelas relações entre participantes.

Conceitos principais:

- **Pessoa:** identidade individual de um participante.
- **Perfil:** informações públicas, apresentação e interesses de uma pessoa.
- **Relacionamento:** vínculos entre participantes, como seguir e acompanhar publicações.

A identidade de uma pessoa poderá futuramente estar associada a uma identidade federada externa.

### 2. Grupos e comunidades

Responsável pelos espaços de participação e organização coletiva.

Um grupo representa uma comunidade de pessoas que compartilham interesses, objetivos ou atividades.

Responsabilidades:

- Criar e configurar grupos.
- Administrar participantes.
- Definir regras básicas de participação e visibilidade.
- Publicar conteúdos em nome do grupo.
- Associar projetos e atividades.
- Preservar a autonomia de comunidades já existentes.

Um grupo não representa necessariamente uma organização formal ou jurídica.

### 3. Organizações

Responsável pela representação de entidades coletivas com identidade própria.

Uma organização pode representar uma associação, coletivo, cooperativa, ONG, iniciativa comunitária ou outra entidade.

Responsabilidades:

- Manter identidade e apresentação institucional.
- Identificar pessoas responsáveis por sua administração.
- Divulgar sua atuação.
- Associar projetos e iniciativas.
- Participar de grupos e atividades.

Uma organização pode existir fora da plataforma e utilizar a rede para estabelecer sua presença digital.

### 4. Projetos e iniciativas

Responsável pela representação de iniciativas concretas.

Um projeto pode representar, por exemplo:

- Uma horta comunitária.
- Um jardim coletivo.
- Uma iniciativa de recuperação ambiental.
- Uma coleção ou viveiro de plantas.
- Uma ação de cultivo individual.
- Uma iniciativa de preservação da biodiversidade.

Responsabilidades:

- Manter informações e objetivos da iniciativa.
- Registrar participantes e responsáveis.
- Publicar atualizações.
- Associar grupos e organizações envolvidos.
- Registrar resultados e evolução.

Um projeto não precisa pertencer exclusivamente a um grupo ou organização.

### 5. Publicações e comunicação

Responsável pela produção e circulação de conteúdos.

Publicações podem ser realizadas por pessoas e, quando autorizado, em nome de grupos ou organizações.

Responsabilidades:

- Criar e consultar publicações.
- Anexar imagens e outros conteúdos.
- Permitir comentários e reações.
- Organizar a apresentação dos conteúdos em feeds.
- Controlar visibilidade e permissões.
- Integrar publicações à federação.

As publicações serão uma forma de comunicação, não o mecanismo principal de representação de projetos ou organizações.

### 6. Compartilhamento de recursos

Responsável pela divulgação de ofertas e necessidades de recursos.

Exemplos:

- Oferta de sementes e mudas.
- Pedido de ferramentas.
- Doação de materiais.
- Compartilhamento de composto ou insumos.
- Disponibilização de conhecimentos e serviços comunitários.

O contexto deverá permitir registrar ofertas, pedidos e manifestações de interesse, sem constituir inicialmente um marketplace comercial.

### 7. Atividades e mobilização

Responsável pela organização e divulgação de ações coletivas.

Exemplos:

- Mutirões.
- Encontros comunitários.
- Campanhas socioambientais.
- Ações de plantio.
- Atividades de recuperação e preservação ambiental.

A mobilização deverá ser compreendida como uma capacidade transversal da plataforma, podendo envolver pessoas, grupos, organizações e projetos.

## Relações entre os conceitos

As entidades deverão se relacionar sem perder sua identidade e responsabilidade próprias.

Relações conceituais iniciais:

- Uma pessoa pode participar de vários grupos.
- Uma pessoa pode administrar uma organização.
- Uma pessoa pode criar e participar de projetos.
- Um grupo pode reunir pessoas com interesses comuns.
- Um grupo pode estar associado a vários projetos.
- Uma organização pode manter vários projetos.
- Um projeto pode envolver pessoas, grupos e organizações diferentes.
- Publicações podem ser realizadas por pessoas, grupos ou organizações.
- Atividades podem ser organizadas por pessoas, grupos ou organizações.
- Ofertas e pedidos podem estar associados a pessoas, grupos ou organizações.

Essas relações não implicam propriedade exclusiva. A participação em um projeto, por exemplo, não significa que ele pertença exclusivamente ao grupo ou à organização que o divulga.

## Fronteiras e responsabilidades

Cada contexto deverá ser responsável pelas regras de negócio relacionadas aos seus conceitos.

A comunicação entre contextos deverá ocorrer por meio de interfaces explícitas, evitando dependências diretas entre suas implementações internas.

Algumas informações poderão ser referenciadas por diferentes contextos, mas sua manutenção deverá permanecer sob responsabilidade de um único contexto.

Por exemplo:

- A identidade de uma pessoa pertence ao contexto de Identidade e Participação.
- A associação de uma pessoa a um grupo pertence ao contexto de Grupos e Comunidades.
- A identidade institucional pertence ao contexto de Organizações.
- A evolução de uma iniciativa pertence ao contexto de Projetos.
- A distribuição de uma publicação federada pertence à camada de Federação.

A arquitetura não deverá impor que cada contexto corresponda necessariamente a um serviço ou aplicação independente. A separação inicial será conceitual e lógica.

## Consequências

A decisão estabelece:

- Um modelo de domínio explícito, orientado aos conceitos reais do produto.
- Separação entre identidade individual, organização coletiva, espaço comunitário e iniciativa concreta.
- Possibilidade de evolução independente das regras de cada contexto.
- Menor acoplamento entre funcionalidades.
- Maior clareza para definir permissões e responsabilidades.
- Possibilidade de integrar os conceitos à federação sem tornar o ActivityPub responsável pelas regras de negócio internas.

Também implica maior cuidado na definição das relações e na consistência dos dados entre contextos.

## Escopo e decisões futuras

Este ADR estabelece as fronteiras conceituais iniciais, mas não determina:

- O modelo físico de dados.
- A estratégia de persistência.
- A estrutura de módulos ou pacotes.
- O mecanismo de autorização e permissões.
- O ciclo de vida detalhado de grupos e organizações.
- A representação federada de cada entidade.
- A possibilidade de uma organização administrar múltiplos grupos ou vice-versa.

Esses aspectos deverão ser detalhados conforme a evolução do projeto.

## Referências

- [Domain-Driven Design — Eric Evans](https://www.domainlanguage.com/ddd/)
- [Bounded Context — Martin Fowler](https://martinfowler.com/bliki/BoundedContext.html)
- [ActivityPub — W3C Recommendation](https://www.w3.org/TR/activitypub/)