# ADR 0011 — Diário de cultivo e biblioteca de práticas

**Status:** Proposto
**Data:** 2026-11-14

## Contexto

A plataforma pretende apoiar pessoas que cultivam, compartilham experiências e aprendem umas com as outras. O ADR 0002 definiu os contextos iniciais de identidade, grupos, organizações, projetos, publicações, recursos e atividades. Esses contextos cobrem a organização social e a comunicação, mas não contemplam dois conceitos centrais para a dimensão educativa do produto:

- O registro estruturado e cronológico de um ciclo de cultivo, incluindo observações, imagens, medidas, intervenções e resultados.
- O compartilhamento de conhecimentos práticos de cultivo, como adubos caseiros, construção de vasos e recipientes, preparo de substrato, compostagem, controle de pragas e conservação de sementes.

Sem esses conceitos, o aprendizado dependeria apenas de publicações genéricas em um feed. Publicações são adequadas para comunicação, mas não para comparar experiências, filtrar resultados por espécie, ambiente ou técnica, nem acompanhar a evolução de um cultivo ao longo do tempo.

O documento [`FICHA_PLANTIO_SEMENTES.md`](../FICHA_PLANTIO_SEMENTES.md) já registra a necessidade de acompanhar identificação, substrato, condições ambientais, germinação, manejo, ocorrências, colheita e avaliação final. Essa ficha evidencia que o registro de cultivo é parte do domínio do produto, não apenas um formato de publicação.

O público espera:

- Aprender a partir de experiências reais, e não apenas de conteúdo genérico.
- Encontrar cultivos semelhantes ao seu por espécie, clima, espaço, técnica ou problema.
- Registrar o próprio cultivo sem precisar de uma ferramenta de gestão agrícola completa.
- Descobrir e adaptar práticas de baixo custo para o seu contexto.
- Compartilhar resultados e receber perguntas, sugestões e relatos de outras pessoas.

A decisão precisa preservar a privacidade, evitar prescrições técnicas inadequadas e manter a interoperabilidade com o Fediverso, conforme os ADRs 0001, 0002, 0004 e 0009.

## Decisão

Serão incorporados dois novos contextos de domínio, distintos dos contextos existentes:

1. **Diário de cultivo:** registros estruturados de ciclos de cultivo.
2. **Conhecimento e práticas:** biblioteca de práticas de cultivo e faça você mesmo.

Esses contextos não substituem Publicações, Projetos ou Recursos. Eles representam o conhecimento e a experiência de cultivo de forma estruturada e podem se relacionar com os demais contextos.

### 1. Diário de cultivo

Um **ciclo de cultivo** representa o acompanhamento de um cultivo desde o início até sua conclusão ou arquivamento.

Um ciclo poderá conter:

- Identificação: nome do ciclo, espécie, variedade ou cultivar e categoria.
- Contexto: ambiente (interno, externo, estufa ou viveiro), localização opcional, substrato, recipiente e técnica utilizada.
- Período: data de início, estágio atual e data de encerramento.
- Responsável: pessoa autora e, opcionalmente, projeto associado.
- Visibilidade: privada, restrita a seguidores ou pública.
- Resultado: colheita, aprendizado, dificuldades e avaliação final.

Um **registro de cultivo** representa um acontecimento datado dentro do ciclo. Poderá conter:

- Data e hora.
- Texto, imagens e anexos.
- Estágio observado (germinação, muda, vegetativo, floração, frutificação, colheita ou encerrado).
- Medidas observadas, como altura, temperatura, umidade, pH ou quantidade colhida.
- Intervenções realizadas, como rega, adubação, poda, desbaste, transplante ou controle de pragas.
- Ocorrências, como praga, doença, deficiência nutricional, evento climático ou falha de equipamento.
- Vínculo opcional com uma prática utilizada.
- Visibilidade herdada do ciclo ou definida individualmente.

Diretrizes:

- O diário deverá priorizar o registro simples e rápido, sem transformar a pessoa em operadora de um sistema de gestão agrícola.
- Os campos estruturados deverão permitir filtros e comparações; o texto livre continuará disponível para observações.
- As medidas e categorias deverão ser extensíveis, pois variam conforme espécie, clima e método de cultivo.
- A edição de registros deverá preservar a data original e indicar a última atualização.
- O sistema não deverá emitir diagnóstico, prescrição ou recomendação agronômica automatizada.
- Insights deverão ser descritivos e baseados em dados registrados, como contagens, prazos observados, taxas de germinação e resultados de colheita, sempre respeitando a visibilidade dos dados.

### 2. Conhecimento e práticas

Uma **prática** representa um guia prático, reproduzível e adaptável, como:

- Adubo caseiro ou compostagem doméstica.
- Construção de vasos, jardineiras ou recipientes reaproveitados.
- Preparo de substrato e transplante.
- Controle de pragas com métodos de baixo impacto.
- Conservação e troca de sementes.
- Irrigação de baixo custo e aproveitamento de água.
- Estruturas de apoio, tutoramento e proteção.

Uma prática poderá conter:

- Título, resumo e categoria.
- Aplicabilidade: espécies, clima, espaço, materiais e nível de dificuldade.
- Tempo estimado, custo aproximado e ferramentas necessárias.
- Materiais e passo a passo.
- Avisos de segurança e riscos.
- Variações, adaptações e resultados esperados.
- Referências e autoria.
- Licença de compartilhamento.
- Histórico de versões.

Diretrizes:

- A prática deverá ter autoria identificada e licença aberta definida no cadastro.
- A pessoa autora poderá editar sua prática; a comunidade poderá comentar, perguntar e relatar adaptações.
- A edição colaborativa e a curadoria comunitária poderão ser implementadas em etapa posterior.
- Práticas com risco à saúde, segurança ou meio ambiente deverão exigir avisos explícitos.
- A plataforma deverá oferecer denúncia e moderação de conteúdo inadequado.
- A biblioteca não será um curso formal, uma trilha de aprendizagem ou uma fonte de prescrição técnica profissional.

### 3. Relação entre diário e práticas

Os dois contextos deverão se relacionar sem se confundir:

- Um registro de cultivo poderá indicar qual prática foi utilizada.
- Uma prática poderá exibir relatos públicos de uso, vinculados a ciclos e resultados.
- A combinação "prática utilizada → aplicação → resultado observado" é o principal mecanismo de aprendizagem do produto.
- Relatos de uso somente serão exibidos quando a visibilidade do ciclo permitir.
- Dados agregados de várias pessoas poderão gerar indicadores descritivos, desde que não exponham informações privadas.

### 4. Permissões e privacidade

- Ciclos e registros serão privados por padrão; a publicação deverá ser uma escolha explícita.
- A pessoa responsável controlará visibilidade, edição e exclusão.
- Ciclos associados a projetos poderão ter autoria compartilhada conforme as permissões do projeto.
- Práticas serão publicadas explicitamente; rascunhos permanecerão privados.
- A localização será opcional e deverá usar granularidade aproximada por padrão, evitando expor coordenadas exatas.
- A exclusão local não garante a remoção de cópias já entregues a outras instâncias, conforme os ADRs 0008 e 0009.
- Dados pessoais sensíveis e informações financeiras não fazem parte do escopo do diário.

### 5. Federação

- Registros públicos poderão ser publicados no outbox do Actor responsável como `Note` ou `Article`, conforme o tamanho e a estrutura do conteúdo.
- Práticas públicas poderão ser publicadas como `Article`.
- Campos estruturados do produto deverão usar namespace próprio e permanecer opcionais para consumidores externos.
- A representação federada deverá conter uma versão textual ou HTML legível sem depender das extensões do Cultivar.
- A visibilidade deverá ser refletida em `to` e `cc`, conforme o ADR 0009.
- Interações externas, como comentários e reações, deverão retornar pelo inbox.
- A importação e sincronização de diários de outras instâncias não fazem parte do escopo inicial.

### 6. Fora de escopo

A decisão não inclui:

- Gestão operacional completa de propriedades, estoques, tarefas ou finanças.
- Diagnóstico, prescrição ou recomendação agronômica automatizada.
- Cursos, certificações, trilhas formais ou avaliações de aprendizagem.
- Monitoramento por sensores, automação de irrigação ou Internet das Coisas.
- Substituição de assistência técnica, extensão rural ou responsabilidade profissional.

## Alternativas consideradas

### 1. Usar apenas publicações genéricas

Representar ciclos e práticas como publicações comuns no feed.

**Vantagens:**
- Menor esforço inicial.
- Reaproveitamento do feed, comentários e federação.

**Desvantagens:**
- Ausência de estrutura para filtros, comparações e acompanhamento.
- Registros se perdem no fluxo cronológico.
- Não permite relacionar prática, aplicação e resultado.
- Não atende à proposta educativa do produto.

### 2. Integrar ferramentas externas de diário de cultivo

Permitir que pessoas mantenham diários em serviços externos e compartilhem apenas links.

**Vantagens:**
- Menor escopo de desenvolvimento.
- Aproveitamento de ferramentas especializadas existentes.

**Desvantagens:**
- Fragmentação da experiência e dependência de serviços externos.
- Identidade, privacidade e federação não integradas.
- Dados da comunidade ficam fora da plataforma.
- Dificulta filtros e insights sobre o conjunto de experiências.

### 3. Construir um sistema completo de gestão agrícola

Ampliar o escopo para estoque, tarefas, finanças, sensores e planejamento de produção.

**Vantagens:**
- Cobertura de necessidades operacionais avançadas.
- Possibilidade de atender produtores em maior escala.

**Desvantagens:**
- Complexidade incompatível com o público e com a proposta do produto.
- Desvio do foco em colaboração, aprendizagem e compartilhamento.
- Aumento significativo de manutenção e suporte.

### 4. Manter uma wiki separada para as práticas

Criar um sistema independente para a biblioteca de práticas.

**Vantagens:**
- Ferramenta adequada para conteúdo colaborativo.
- Separação clara entre rede social e base de conhecimento.

**Desvantagens:**
- Duplicação de identidade, autenticação e moderação.
- Integração pobre com diários, projetos e federação.
- Experiência fragmentada para quem cultiva e compartilha.

## Consequências

A decisão estabelece:

- Dois novos contextos de domínio, com entidades, regras e ciclos de vida próprios.
- Registros estruturados de cultivo, com filtros, acompanhamento e resultados.
- Uma biblioteca de práticas com autoria, versionamento, segurança e licença.
- Integração entre prática, aplicação e resultado observado.
- Publicação federada opcional de registros e práticas.
- Privacidade por padrão e controle explícito de visibilidade.

A decisão também implica:

- Novas tabelas, índices e consultas de filtragem.
- Interface para ciclos, registros, linha do tempo, filtros e práticas.
- Armazenamento de imagens e anexos em serviço compatível com S3.
- Regras de permissão para pessoas e projetos.
- Moderação, denúncia e avisos de segurança.
- Mapeamento federado e extensões próprias.
- Testes de domínio, integração, contrato e interoperabilidade.
- Esforço contínuo de curadoria e documentação das práticas.

## Escopo e decisões futuras

Este ADR não determina ainda:

- O modelo físico de dados, índices e estratégia de busca.
- Os valores padrão de visibilidade e a granularidade de localização.
- O conjunto padronizado de medidas e categorias por espécie.
- Os algoritmos e limiares dos insights descritivos.
- O fluxo de edição colaborativa e curadoria das práticas.
- A licença padrão para práticas e registros publicados.
- A representação federada detalhada das extensões do produto.
- A importação e exportação de diários.
- As notificações de acompanhamento de ciclos e práticas.
- As políticas de moderação de conteúdo sensível ou perigoso.

Esses aspectos deverão ser detalhados em decisões posteriores e validados com a comunidade.

## Referências

- [ADR 0001 — Aplicação própria integrada ao Fediverso](0001-arquitetura-propria-com-activitypub.md)
- [ADR 0002 — Modelo de domínio e fronteiras entre os contextos](0002-modelo-de-dominio-e-fronteiras-entre-os-contextos.md)
- [ADR 0005 — Persistência e armazenamento](0005-persistencia-e-armazenamento.md)
- [ADR 0009 — Implementação da federação](0009-implementacao-da-federacao.md)
- [`docs/VISION.md`](../VISION.md)
- [`docs/FICHA_PLANTIO_SEMENTES.md`](../FICHA_PLANTIO_SEMENTES.md)
- [ActivityStreams 2.0 — W3C Recommendation](https://www.w3.org/TR/activitystreams-core/)
