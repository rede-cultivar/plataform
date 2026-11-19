# Roadmap do Projeto

**Versão:** 0.1 — Proposta inicial  
**Status:** Documento em construção  
**Nome provisório:** Cultivar

## 1. Objetivo

Este documento apresenta uma proposta de evolução do projeto, organizando suas principais etapas de desenvolvimento e estabelecendo prioridades iniciais.

O roadmap deve orientar o planejamento sem se transformar em um compromisso rígido de entregas.

Sua evolução deverá considerar:

- Os objetivos definidos no Documento de Visão.
- Os princípios estabelecidos na Carta de Fundação.
- As necessidades identificadas pela comunidade.
- A capacidade disponível de desenvolvimento e manutenção.
- Os resultados obtidos com a utilização das versões anteriores.

## 2. Estratégia de desenvolvimento

O projeto será desenvolvido incrementalmente, priorizando a validação de sua proposta de valor antes da ampliação de funcionalidades.

A estratégia considera três objetivos principais:

1. Construir uma infraestrutura funcional, segura e interoperável.
2. Permitir que pessoas e comunidades utilizem a plataforma para compartilhar experiências e desenvolver iniciativas.
3. Ampliar progressivamente os recursos de colaboração e mobilização.

Cada etapa deverá produzir uma versão utilizável e permitir aprendizado a partir da experiência real dos participantes.

## 3. Etapa 0 — Fundação do projeto

<badge color="success">Em andamento</badge>

Estabelecer as bases institucionais, organizacionais e técnicas necessárias para o desenvolvimento coletivo.

### Objetivos

- Definir a identidade e os princípios do projeto.
- Organizar a documentação inicial.
- Estabelecer o modelo de contribuição e governança.
- Preparar os repositórios e a infraestrutura de desenvolvimento.
- Definir as primeiras decisões arquiteturais.

### Entregas previstas

- Carta de Fundação.
- Política de Marca e Identidade.
- Modelo de Governança.
- Código de Conduta.
- Guia de Contribuição.
- Documento de Visão do Produto.
- Roadmap.
- Arquitetura inicial.
- Repositório de código.
- Ambiente inicial de desenvolvimento.

### Situação atual

As primeiras decisões arquiteturais (ADRs 0001–0011) estão registradas. O ambiente de desenvolvimento já está disponível, com PostgreSQL, armazenamento compatível com S3, migrações, servidor HTTP e trabalhador assíncrono. As instruções de uso estão em [`docs/DEVELOPMENT.md`](DEVELOPMENT.md).

### Critério de conclusão

A comunidade dispõe de documentação suficiente para compreender o propósito do projeto, participar de sua construção e iniciar o desenvolvimento de maneira organizada.

## 4. Etapa 1 — Fundação técnica e MVP

<badge color="info">Em andamento</badge>

Construir a primeira versão funcional da plataforma, priorizando identidade, publicação, interação e uma primeira experiência de federação.

### Situação atual

Contas locais com e-mail e senha, sessões no servidor, proteção CSRF, perfis públicos editáveis e a primeira camada de identidade federada (WebFinger, Actor, NodeInfo, assinaturas HTTP, entrega assíncrona e inbox) já estão implementados. As próximas entregas são o relacionamento entre participantes, o feed de publicações e a validação contínua com instâncias reais.

### Objetivos

- Disponibilizar uma plataforma utilizável.
- Permitir a criação de identidades e relacionamentos.
- Oferecer recursos básicos de publicação e interação.
- Validar a integração com o Fediverso.
- Estabelecer os mecanismos básicos de segurança e moderação.

### Funcionalidades previstas

#### Identidade e participação

- Cadastro e autenticação.
- Perfil de participante.
- Gerenciamento de interesses.
- Consulta de perfis.
- Relacionamento entre participantes.
- Seguir e deixar de seguir pessoas. *(Vínculo local e leitura de conteúdo
  `followers` implementados; federação desse conteúdo pendente.)*

#### Publicações e interações

- Criar e consultar publicações.
- Publicar imagens.
- Feed de publicações.
- Comentar publicações.
- Curtir ou reagir.
- Compartilhar conteúdo.

#### Federação

- Identidade federada para participantes. *(WebFinger, Actor, NodeInfo, assinaturas HTTP, entrega e inbox implementados; descoberta e acompanhamento de contas externas pendentes.)*
- Descoberta de contas externas.
- Seguir participantes de outras instâncias.
- Distribuir publicações para instâncias compatíveis.
- Receber publicações e interações externas.
- Tratamento inicial de bloqueios e restrições.

#### Administração e segurança

- Administração básica da instância.
- Controles de visibilidade das publicações.
- Denúncia e moderação básica.
- Proteção de credenciais e dados pessoais.
- Registro de eventos relevantes.
- Mecanismos de tratamento de falhas na comunicação federada.

### Critério de conclusão

Uma pessoa consegue criar uma conta, configurar seu perfil, publicar experiências, interagir com outras pessoas e estabelecer relacionamentos com participantes de pelo menos uma implementação externa do Fediverso.

A federação deve ser validada na prática, não apenas por testes isolados de protocolo.

## 5. Etapa 2 — Diário de cultivo e conhecimento prático

Introduzir a dimensão educativa do produto, permitindo registrar experiências de cultivo e transformá-las em conhecimento compartilhado.

### Objetivos

- Permitir que pessoas registrem e acompanhem seus ciclos de cultivo.
- Facilitar a descoberta de experiências semelhantes por espécie, ambiente, técnica e resultado.
- Organizar práticas de cultivo e faça você mesmo de forma reproduzível.
- Conectar prática utilizada, aplicação e resultado observado.
- Publicar experiências e práticas de forma federada, quando houver escolha explícita.

### Funcionalidades previstas

#### Diário de cultivo

- Criar, editar, encerrar e arquivar ciclos de cultivo.
- Identificar espécie, variedade, ambiente, substrato, recipiente e técnica.
- Registrar observações, imagens, medidas e intervenções ao longo do tempo.
- Registrar ocorrências, como pragas, doenças, deficiências e eventos climáticos.
- Registrar colheita, avaliação final e aprendizados.
- Definir visibilidade por ciclo e por registro.
- Acompanhar ciclos públicos de outras pessoas.
- Comentar e fazer perguntas sobre registros.

#### Filtros e indicadores

- Filtrar ciclos e registros por espécie, ambiente, técnica, estágio, problema e resultado.
- Comparar experiências semelhantes.
- Apresentar indicadores descritivos, como prazos observados, taxas de germinação e resultados de colheita.
- Exibir relatos de uso que relacionem práticas, aplicações e resultados.

#### Conhecimento e práticas

- Criar, publicar e editar práticas de cultivo e faça você mesmo.
- Descrever materiais, ferramentas, etapas, riscos e adaptações.
- Categorizar práticas por aplicabilidade, dificuldade, tempo e custo.
- Preservar autoria, licença e histórico de versões.
- Comentar, perguntar e relatar adaptações.
- Vincular práticas a ciclos e resultados observados.
- Denunciar e moderar conteúdo inadequado ou perigoso.

#### Federação

- Publicar registros e práticas públicas no outbox do Actor responsável. *(Registros públicos já são publicados como `Note`; práticas e atualizações pendentes.)*
- Representar registros como `Note` ou `Article` e práticas como `Article`.
- Receber comentários e reações de outras instâncias.
- Manter campos estruturados como extensões opcionais para consumidores externos.

### Critério de conclusão

Uma pessoa consegue registrar um ciclo de cultivo completo, com observações, intervenções e resultado. Outra pessoa consegue encontrar esse ciclo por filtros, compreender o que foi feito e relacioná-lo a uma prática. Práticas publicadas podem ser adaptadas e ter seus resultados registrados. Conteúdo público pode ser publicado e receber interações do Fediverso.

## 6. Etapa 3 — Projetos e comunidades

Ampliar a plataforma para permitir que pessoas se organizem em torno de iniciativas concretas.

### Objetivos

- Introduzir os conceitos de projetos e grupos.
- Permitir que comunidades existentes utilizem a plataforma.
- Facilitar a divulgação e o acompanhamento de iniciativas.
- Estabelecer identidades coletivas básicas.

### Funcionalidades previstas

#### Projetos

- Criar e editar projetos.
- Apresentar objetivos e descrição.
- Associar pessoas participantes.
- Publicar atualizações.
- Consultar histórico de atividades.
- Associar localização opcional.
- Divulgar resultados e experiências.

#### Grupos

- Criar grupos.
- Definir identidade e descrição.
- Configurar visibilidade.
- Administrar participantes.
- Publicar em nome do grupo.
- Associar projetos.
- Gerenciar administradores.

#### Organizações

- Criar perfis institucionais.
- Apresentar informações e identidade.
- Associar projetos.
- Divulgar iniciativas.
- Estabelecer relacionamentos com grupos e participantes.

### Critério de conclusão

Pessoas e comunidades conseguem representar suas iniciativas na plataforma, manter espaços de participação e divulgar projetos de maneira organizada.

## 7. Etapa 4 — Compartilhamento e colaboração

Introduzir recursos que facilitem a circulação de materiais, conhecimentos e recursos entre participantes.

### Objetivos

- Facilitar trocas e doações.
- Conectar necessidades e disponibilidades.
- Estimular a colaboração entre pessoas, grupos e organizações.

### Funcionalidades previstas

- Publicar ofertas de recursos.
- Publicar pedidos e necessidades.
- Categorizar recursos.
- Consultar ofertas e pedidos.
- Manifestar interesse.
- Estabelecer contato entre participantes.
- Registrar trocas e doações.
- Consultar histórico de compartilhamentos.

Os recursos poderão incluir sementes, mudas, ferramentas, composto, materiais e outros itens relacionados às iniciativas da comunidade.

O foco inicial será o compartilhamento comunitário, não a criação de um marketplace comercial.

### Critério de conclusão

Participantes conseguem divulgar necessidades e disponibilidades, estabelecer contato e realizar trocas ou doações por meio das conexões criadas na plataforma.

## 8. Etapa 5 — Mobilização e ações coletivas

Ampliar os recursos de organização e articulação de atividades presenciais e iniciativas coletivas.

### Objetivos

- Apoiar a realização de ações concretas.
- Facilitar encontros e mutirões.
- Fortalecer campanhas e mobilizações.
- Registrar resultados e aprendizados.

### Funcionalidades previstas

- Criar atividades e eventos.
- Divulgar encontros e mutirões.
- Informar local, data e descrição.
- Manifestar interesse ou participação.
- Associar atividades a grupos e projetos.
- Criar e divulgar campanhas.
- Compartilhar resultados.
- Manter histórico de ações realizadas.

A mobilização deverá permanecer integrada aos demais domínios da plataforma.

### Critério de conclusão

Grupos e organizações conseguem utilizar a plataforma para divulgar, organizar e registrar atividades coletivas, conectando a comunicação digital à realização de ações presenciais.

## 9. Evoluções futuras

Após a consolidação das etapas anteriores, poderão ser avaliadas novas possibilidades.

Entre elas:

- Representação federada mais completa de grupos e organizações.
- Integração com outras aplicações do Fediverso.
- Mecanismos avançados de descoberta geográfica.
- Ferramentas de apoio à organização comunitária.
- Recursos ampliados de acessibilidade.
- Indicadores de impacto socioambiental.
- Integrações com ferramentas abertas de mapeamento.
- Aplicações móveis.
- Importação e exportação de diários de cultivo.
- Análises comparativas avançadas e indicadores de aprendizagem.
- Mecanismos de interoperabilidade com outras iniciativas e plataformas comunitárias.

A inclusão dessas funcionalidades dependerá de necessidades reais identificadas pela comunidade.

## 10. Itens fora do escopo inicial

Para preservar o foco e a capacidade de desenvolvimento, os seguintes itens não integram as prioridades iniciais:

- Marketplace e comércio eletrônico.
- Sistemas completos de gestão agrícola.
- Gestão financeira de propriedades ou projetos.
- Prescrição agronômica e acompanhamento técnico profissional.
- Automação de irrigação e monitoramento por sensores.
- Cursos e trilhas formais de aprendizagem.
- Sistemas complexos de gamificação.
- Inteligência artificial para recomendação ou diagnóstico agrícola.
- Integrações proprietárias que criem dependência tecnológica desnecessária.

Esses itens poderão ser reavaliados futuramente, desde que exista justificativa alinhada à visão do produto.

## 11. Priorização e acompanhamento

As funcionalidades devem ser priorizadas considerando:

- Alinhamento com a visão do produto.
- Benefício para os participantes.
- Impacto sobre a colaboração comunitária.
- Dependências técnicas.
- Complexidade de implementação e manutenção.
- Riscos de segurança e privacidade.
- Capacidade disponível de desenvolvimento.

O progresso deverá ser acompanhado por entregas funcionais e critérios verificáveis de conclusão.

A conclusão de uma etapa não depende apenas da implementação de funcionalidades, mas também de sua validação e documentação.

## 12. Revisão do roadmap

Este roadmap deverá ser revisado periodicamente pela comunidade.

Mudanças de prioridade, inclusão de novas etapas e alterações relevantes de escopo devem ser discutidas e registradas de maneira transparente.

O roadmap representa uma direção compartilhada para o desenvolvimento, não uma promessa de entrega em datas predeterminadas.