# ADR 0005 — Persistência e armazenamento

**Status:** Aceito  
**Data:** 2026-11-05

## Contexto

A plataforma possui diferentes necessidades de persistência:

- Dados estruturados de pessoas, grupos, organizações e projetos.
- Relacionamentos entre participantes e entidades.
- Publicações, comentários e interações.
- Atividades recebidas e enviadas pelo ActivityPub.
- Informações operacionais da federação.
- Imagens, documentos e outros arquivos enviados pelos participantes.

Essas informações possuem características distintas de acesso, consistência, volume e ciclo de vida.

A estratégia de persistência deverá atender às necessidades iniciais do produto sem introduzir complexidade operacional prematura, mantendo a possibilidade de evolução conforme o crescimento da comunidade.

## Decisão

A aplicação adotará inicialmente uma estratégia de persistência baseada em:

1. Banco de dados relacional para os dados estruturados e transacionais.
2. Armazenamento de objetos para imagens e arquivos.
3. Referências e metadados no banco de dados para relacionar arquivos às entidades da aplicação.
4. Persistência das atividades federadas e informações operacionais necessárias à integração.
5. Separação lógica entre os dados de domínio e os dados utilizados exclusivamente pela infraestrutura de federação.

### 1. Dados de domínio

O banco de dados principal será responsável por armazenar informações estruturadas relacionadas aos contextos funcionais da aplicação.

Entre elas:

- Contas e perfis.
- Pessoas e relacionamentos.
- Grupos e participantes.
- Organizações.
- Projetos e iniciativas.
- Publicações e comentários.
- Ofertas e pedidos.
- Atividades e eventos.
- Configurações e permissões.

A persistência deverá preservar a integridade das relações e permitir transações nas operações que exigem consistência.

O modelo de dados deverá refletir as fronteiras conceituais definidas no ADR 0002, evitando dependências desnecessárias entre contextos.

### 2. Arquivos e conteúdo multimídia

Imagens, fotografias e documentos não deverão ser armazenados diretamente como conteúdo binário no banco de dados principal.

Será utilizado um mecanismo de armazenamento de objetos.

O banco de dados manterá os metadados necessários para identificar e relacionar os arquivos às entidades correspondentes.

Esses metadados poderão incluir:

- Identificador do arquivo.
- Nome original e nome interno.
- Tipo de conteúdo.
- Tamanho.
- Data de envio.
- Pessoa responsável pelo envio.
- Referência à entidade associada.
- Informações de processamento e disponibilidade.

A estratégia deverá considerar:

- Limites de tamanho e tipos de arquivo.
- Validação do conteúdo recebido.
- Geração de miniaturas e versões otimizadas.
- Controle de acesso.
- Remoção de arquivos sem referências válidas.
- Possibilidade de distribuição por CDN.

Arquivos privados não deverão ser disponibilizados publicamente apenas por estarem armazenados em um serviço acessível pela internet.

### 3. Atividades federadas

As atividades relacionadas ao ActivityPub deverão possuir persistência adequada ao seu ciclo de vida.

Deverão ser armazenados os dados necessários para:

- Identificar atividades recebidas e enviadas.
- Evitar processamento duplicado.
- Controlar entregas pendentes.
- Registrar tentativas e resultados.
- Permitir retentativas e recuperação.
- Manter referências aos objetos federados.

A persistência dessas informações deverá ser compatível com a estratégia assíncrona definida no ADR 0004.

O armazenamento das atividades federadas não deverá obrigar o domínio da aplicação a adotar o formato ActivityStreams como modelo interno de todas as entidades.

### 4. Consistência e transações

As operações que alteram dados relacionados deverão utilizar transações locais quando necessário.

A aplicação deverá evitar depender de transações distribuídas entre banco de dados, filas e serviços externos.

Para operações que envolvem persistência de domínio e distribuição federada, deverá ser considerado o padrão Transactional Outbox.

A consistência entre a aplicação e as demais instâncias será eventual, conforme estabelecido no ADR 0004.

### 5. Evolução e escalabilidade

A estratégia deverá permitir:

- Crescimento gradual do volume de dados.
- Indexação adequada às consultas mais frequentes.
- Paginação e carregamento incremental de conteúdos.
- Separação entre armazenamento de dados e arquivos.
- Backup e recuperação.
- Migrações controladas de esquema.
- Evolução independente dos mecanismos de persistência.

A introdução de mecanismos adicionais, como cache, busca especializada ou bancos específicos para determinados tipos de dados, deverá ocorrer somente quando houver necessidade demonstrável.

## Alternativas consideradas

### 1. Banco de dados relacional e armazenamento de objetos

Utilizar banco relacional para dados estruturados e armazenamento de objetos para arquivos.

**Vantagens:**
- Integridade referencial e suporte a transações.
- Boa adequação aos relacionamentos do domínio.
- Separação entre dados estruturados e arquivos.
- Ampla disponibilidade de ferramentas de administração e backup.
- Possibilidade de utilizar serviços gerenciados ou infraestrutura própria.

**Desvantagens:**
- Necessidade de administrar dois mecanismos de persistência.
- Consistência entre metadados e arquivos exige cuidados.
- Algumas consultas especializadas podem demandar soluções adicionais.

### 2. Banco de documentos como persistência principal

Utilizar um banco orientado a documentos para armazenar os dados de domínio.

**Vantagens:**
- Flexibilidade de estrutura.
- Facilidade para representar documentos e estruturas aninhadas.
- Evolução de determinados formatos com menor dependência de esquemas relacionais.

**Desvantagens:**
- Relacionamentos complexos podem exigir maior cuidado.
- Integridade referencial não é normalmente garantida da mesma forma que em bancos relacionais.
- Pode incentivar duplicação de informações.
- Nem todas as características do domínio se beneficiam de um modelo documental.

### 3. Banco relacional e armazenamento de arquivos no próprio banco

Armazenar dados estruturados e arquivos no mesmo mecanismo de persistência.

**Vantagens:**
- Centralização da infraestrutura.
- Possibilidade de manter metadados e arquivos na mesma transação.
- Menor quantidade de serviços externos.

**Desvantagens:**
- Crescimento significativo do volume do banco.
- Maior custo de backup e recuperação.
- Dificuldade de distribuição e entrega eficiente de arquivos.
- Mistura de necessidades operacionais distintas.

## Consequências

A decisão estabelece:

- Banco relacional como estratégia inicial para persistência do domínio.
- Armazenamento de objetos para conteúdo multimídia e arquivos.
- Separação entre dados de domínio e informações operacionais da federação.
- Uso de transações locais para operações que exigem consistência.
- Consistência eventual para operações federadas.
- Necessidade de políticas de backup, recuperação e retenção.
- Possibilidade de introduzir cache e mecanismos especializados futuramente.

A decisão não implica a adoção de microserviços nem exige bancos independentes para cada contexto funcional.

## Escopo e decisões futuras

Este ADR não determina ainda:

- Qual banco relacional será utilizado.
- Qual serviço ou tecnologia de armazenamento de objetos será adotado.
- Como será implementado o versionamento do esquema.
- Qual será a estratégia de backup e recuperação.
- Como serão definidos os períodos de retenção.
- Qual mecanismo de cache será utilizado.
- Se haverá um mecanismo especializado de busca.
- Como será feita a distribuição de imagens e arquivos por CDN.
- Qual será a estratégia de replicação e alta disponibilidade.
- Como serão tratados dados pessoais e solicitações de exclusão.

Esses aspectos deverão ser detalhados conforme os requisitos técnicos e operacionais forem estabelecidos.

## Referências

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Amazon S3 Documentation](https://docs.aws.amazon.com/s3/)
- [Transactional Outbox Pattern](https://microservices.io/patterns/data/transactional-outbox.html)
- [Domain-Driven Design — Eric Evans](https://www.domainlanguage.com/ddd/)