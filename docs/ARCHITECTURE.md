# Arquitetura do Sistema

**Versão:** 0.1 — Proposta inicial  
**Status:** Documento em construção  
**Nome provisório:** Cultivar

## 1. Objetivo

Este documento apresenta os princípios arquiteturais e a organização conceitual do sistema.

Seu objetivo é estabelecer uma base para as decisões técnicas e orientar a implementação de uma plataforma descentralizada, especializada em cultivo, colaboração comunitária e mobilização socioambiental.

A arquitetura deverá permitir a evolução independente dos diferentes domínios funcionais, mantendo interoperabilidade com o Fediverso e evitando dependências desnecessárias de tecnologias ou serviços proprietários.

Este documento não define uma implementação definitiva. Frameworks, linguagens, bancos de dados e provedores de infraestrutura poderão ser escolhidos posteriormente.

## 2. Princípios arquiteturais

### 2.1. Domínio independente dos protocolos

O modelo de domínio da aplicação não deverá depender diretamente do ActivityPub ou de qualquer protocolo externo.

A aplicação deverá possuir suas próprias entidades, regras de negócio e casos de uso.

A integração com o Fediverso será responsabilidade de uma camada específica de interoperabilidade.

### 2.2. Interoperabilidade por padrões abertos

A comunicação com outras implementações do Fediverso deverá utilizar protocolos e formatos abertos, priorizando compatibilidade com implementações existentes.

### 2.3. Modularidade

Os domínios funcionais deverão possuir responsabilidades bem definidas e interfaces explícitas.

A arquitetura deverá permitir que funcionalidades evoluam sem exigir alterações generalizadas em todo o sistema.

### 2.4. Descentralização

A arquitetura deverá permitir que diferentes instâncias operem de maneira independente, mantendo comunicação e interoperabilidade com outras instâncias.

Nenhuma instância deverá depender permanentemente de uma autoridade central para executar suas funções essenciais.

### 2.5. Segurança e privacidade

Segurança, privacidade e controle de acesso deverão ser considerados desde o início do desenvolvimento.

A distribuição de informações entre instâncias deverá respeitar as configurações de visibilidade e as limitações dos protocolos utilizados.

### 2.6. Evolução incremental

A arquitetura deverá começar com uma estrutura suficientemente simples para viabilizar o MVP, preservando fronteiras que permitam sua evolução.

Não se deve introduzir complexidade distribuída ou infraestrutura desnecessária antes que exista uma necessidade concreta.

## 3. Visão geral da arquitetura

A aplicação será organizada em camadas com responsabilidades distintas.

As principais áreas são:

1. **Interface de usuário:** interação com pessoas, grupos e organizações.
2. **Aplicação:** execução dos casos de uso e coordenação das operações.
3. **Domínio:** entidades, regras de negócio e políticas próprias do produto.
4. **Infraestrutura:** persistência, serviços externos, filas, arquivos e mecanismos operacionais.
5. **Federação:** comunicação com outras instâncias por meio dos protocolos do Fediverso.

A federação deverá ser tratada como uma fronteira de integração, não como substituta da camada de domínio.

## 4. Organização conceitual

### 4.1. Interface de usuário

Responsável pela experiência de utilização da plataforma.

Deverá oferecer interfaces para:

- Cadastro e gerenciamento de perfis.
- Publicações e interações.
- Consulta de projetos.
- Participação em grupos.
- Identidades de organizações.
- Compartilhamento de recursos.
- Atividades e mobilizações.
- Configurações de privacidade.

A interface deverá consumir os serviços da aplicação por meio de contratos definidos.

### 4.2. Camada de aplicação

Responsável por coordenar os casos de uso.

Exemplos:

- Criar publicação.
- Seguir participante.
- Criar projeto.
- Participar de grupo.
- Publicar oferta de recurso.
- Criar atividade.
- Distribuir publicação para instâncias federadas.

A camada de aplicação não deverá conter detalhes de infraestrutura ou regras específicas de comunicação HTTP do ActivityPub.

### 4.3. Camada de domínio

Contém o modelo de negócio e suas regras.

Entre os conceitos principais estão:

- Pessoa.
- Grupo.
- Organização.
- Projeto.
- Publicação.
- Recurso.
- Atividade.
- Local.
- Relacionamentos e permissões.

As regras de negócio devem permanecer independentes de frameworks, bancos de dados e protocolos de comunicação.

### 4.4. Camada de infraestrutura

Implementa os mecanismos necessários para executar as operações da aplicação.

Entre suas responsabilidades:

- Persistência de dados.
- Armazenamento de imagens e arquivos.
- Autenticação e mecanismos criptográficos.
- Filas e processamento assíncrono.
- Comunicação HTTP.
- Logs e monitoramento.
- Integração com serviços externos.

As implementações concretas deverão respeitar os contratos definidos pelas camadas superiores.

## 5. Domínios funcionais

A aplicação deverá ser organizada em torno de domínios com responsabilidades próprias.

### 5.1. Identidade e acesso

Responsável por:

- Cadastro e autenticação.
- Perfis de pessoas.
- Perfis de entidades coletivas.
- Credenciais e sessões.
- Papéis e permissões.
- Configurações de privacidade.

A identidade interna de uma pessoa deverá ser independente de sua representação federada.

### 5.2. Relacionamentos

Responsável pelas relações entre participantes e entidades.

Exemplos:

- Seguir pessoas.
- Participar de grupos.
- Administrar grupos.
- Participar de projetos.
- Associar organizações a iniciativas.

Relacionamentos sociais e vínculos de participação não devem ser confundidos com permissões administrativas.

### 5.3. Publicações e interações

Responsável por:

- Criação e edição de publicações.
- Imagens e anexos.
- Comentários.
- Reações.
- Compartilhamentos.
- Feed e distribuição de conteúdo.

Este domínio deverá oferecer mecanismos de integração com a federação.

### 5.4. Projetos

Responsável pela representação das iniciativas concretas.

Um projeto poderá conter:

- Identificação e descrição.
- Objetivos.
- Participantes.
- Organização responsável, quando aplicável.
- Grupo associado, quando aplicável.
- Localização opcional.
- Histórico de atualizações.

### 5.5. Grupos e organizações

Responsável pelas entidades coletivas e seus mecanismos de participação.

Grupo e organização são conceitos distintos:

- Grupo representa um espaço de participação e relacionamento.
- Organização representa uma entidade coletiva com identidade própria.

Uma organização poderá manter vários projetos e participar de diferentes grupos.

Um grupo poderá reunir pessoas e projetos associados a diferentes organizações.

### 5.6. Compartilhamento de recursos

Responsável por representar ofertas e necessidades de recursos.

Deverá permitir:

- Cadastrar ofertas.
- Registrar necessidades.
- Consultar disponibilidades.
- Manifestar interesse.
- Registrar trocas e doações.

O domínio não deverá pressupor operações comerciais ou transações financeiras.

### 5.7. Atividades e mobilização

Responsável pela organização de:

- Eventos.
- Encontros.
- Mutirões.
- Campanhas.
- Ações coletivas.

As atividades poderão estar associadas a pessoas, grupos, organizações e projetos.

### 5.8. Federação

Responsável pela comunicação com outras instâncias e pela tradução entre os modelos internos e os formatos utilizados no Fediverso.

Deverá manter uma fronteira explícita em relação aos demais domínios.

## 6. Arquitetura de federação

### 6.1. Protocolos e padrões

A integração deverá considerar principalmente:

- **ActivityPub:** protocolo de comunicação federada.
- **ActivityStreams 2.0:** modelo de representação de objetos e atividades.
- **WebFinger:** mecanismo de descoberta de identidades.
- **HTTP Signatures:** mecanismos de autenticação e verificação de mensagens, conforme os perfis de compatibilidade adotados.
- **HTTPS:** comunicação segura entre instâncias.
- **NodeInfo:** mecanismo complementar para descoberta de informações sobre instâncias.

A compatibilidade deverá considerar as práticas e extensões efetivamente utilizadas pelas implementações existentes.

### 6.2. Identidades federadas

Pessoas e entidades coletivas poderão possuir identidades federadas.

Cada identidade federada deverá possuir uma representação compatível com o conceito de Actor do ActivityPub.

Essa representação poderá incluir:

- Identificador único.
- Nome e apresentação pública.
- Endereço de perfil.
- Inbox.
- Outbox.
- Chaves e mecanismos de autenticação.
- Informações de descoberta.

A identidade federada será uma representação externa da identidade mantida pela aplicação, não a própria entidade de domínio.

### 6.3. Publicações e atividades

Publicações internas poderão ser representadas por objetos ActivityStreams.

Por exemplo, uma experiência compartilhada por uma pessoa poderá ser representada por um objeto `Note`.

As ações realizadas sobre publicações poderão ser representadas por atividades como:

- `Create`.
- `Update`.
- `Delete`.
- `Follow`.
- `Accept`.
- `Undo`.
- `Like`.
- `Announce`.

A utilização de tipos e atividades deverá respeitar as convenções de interoperabilidade das implementações externas.

### 6.4. Grupos e organizações federados

Grupos e organizações poderão possuir identidades federadas próprias.

Entretanto, a representação externa não deverá substituir o modelo interno de participação e administração.

O ActivityPub não define, por si só, um sistema completo de gestão de membros, permissões, projetos e atividades coletivas.

Essas funcionalidades continuarão sendo responsabilidade da aplicação.

### 6.5. Inbox e Outbox

A implementação deverá considerar os mecanismos de recebimento e distribuição definidos pelo ActivityPub.

**Inbox:**

Responsável por receber atividades de outras instâncias, validar sua origem e encaminhá-las ao processamento interno apropriado.

**Outbox:**

Responsável por disponibilizar as atividades produzidas por uma identidade local e iniciar sua distribuição para os destinatários correspondentes.

A aplicação deverá tratar corretamente atividades repetidas, inválidas ou recebidas fora de ordem.

### 6.6. Descoberta de identidades

A descoberta de participantes externos deverá permitir a resolução de identificadores federados.

O mecanismo deverá considerar:

- WebFinger.
- Resolução de URLs de Actors.
- Consulta de perfis externos.
- Armazenamento de referências federadas.
- Atualização de informações públicas.

A descoberta deverá ser protegida contra abuso, requisições excessivas e acesso indevido a recursos internos da infraestrutura.

## 7. Processamento assíncrono

A comunicação federada não deverá depender de operações síncronas durante a interação normal da pessoa com a plataforma.

A distribuição e o recebimento de atividades deverão utilizar mecanismos de processamento assíncrono.

A arquitetura deverá prever:

- Filas de processamento.
- Retentativas de entrega.
- Controle de duplicidade.
- Idempotência.
- Registro de falhas.
- Tratamento de mensagens inválidas.
- Limites de processamento.
- Monitoramento de entregas.

Falhas temporárias de comunicação com outras instâncias não deverão comprometer a disponibilidade das funcionalidades locais.

A implementação poderá começar com mecanismos simples de fila e evoluir conforme o volume e as necessidades operacionais.

## 8. Segurança e privacidade

A arquitetura deverá considerar segurança como responsabilidade transversal.

Entre os requisitos iniciais:

- Autenticação segura.
- Autorização baseada em permissões.
- Proteção de credenciais.
- Validação de entradas.
- Proteção contra abuso de APIs.
- Controle de acesso a arquivos e imagens.
- Registro de eventos relevantes.
- Proteção contra requisições maliciosas entre servidores.
- Verificação das assinaturas das mensagens federadas.
- Tratamento seguro de conteúdo recebido de instâncias externas.

### 8.1. Visibilidade e distribuição

A aplicação deverá distinguir claramente os níveis de visibilidade das publicações.

Conteúdos privados ou restritos não deverão ser distribuídos publicamente por engano.

Deve-se considerar que, uma vez entregue a outra instância, o controle técnico sobre a cópia distribuída pode ser limitado.

A exclusão ou alteração de conteúdo local não garante a eliminação de todas as cópias externas.

### 8.2. Confiança entre instâncias

A federação implica comunicação com servidores administrados por diferentes pessoas e organizações.

A arquitetura deverá prever mecanismos de:

- Bloqueio de participantes.
- Restrição ou bloqueio de instâncias.
- Denúncia de conteúdo.
- Moderação local.
- Limitação de tráfego.
- Proteção contra spam e abuso.

Cada instância deverá manter autonomia para administrar suas próprias políticas de moderação.

## 9. Persistência e armazenamento

A escolha dos mecanismos de persistência deverá considerar os requisitos de cada domínio.

O modelo deverá permitir:

- Integridade das relações entre entidades.
- Consultas eficientes.
- Evolução do esquema de dados.
- Auditoria de operações relevantes.
- Retenção e exclusão de dados conforme as políticas estabelecidas.
- Separação entre dados internos e referências externas.

Imagens e arquivos deverão ser tratados separadamente dos registros principais de domínio, utilizando armazenamento apropriado.

A escolha do banco de dados e das tecnologias de armazenamento será registrada em decisões arquiteturais específicas.

## 10. Observabilidade e operação

A aplicação deverá oferecer mecanismos para acompanhar seu funcionamento.

Entre os recursos previstos:

- Logs estruturados.
- Métricas de aplicação.
- Monitoramento de disponibilidade.
- Rastreamento de erros.
- Monitoramento de filas.
- Acompanhamento das entregas federadas.
- Alertas para falhas operacionais relevantes.

A observabilidade deverá permitir identificar problemas locais e falhas de comunicação com outras instâncias.

## 11. Estratégia de implantação

A arquitetura deverá permitir implantação e operação independentes por diferentes comunidades.

Uma instância deverá possuir condições de:

- Ser instalada e configurada de maneira independente.
- Utilizar domínio próprio.
- Administrar seus usuários.
- Gerenciar sua política de moderação.
- Controlar sua infraestrutura.
- Comunicar-se com outras instâncias compatíveis.

A instalação deverá ser documentada e, progressivamente, simplificada para facilitar a adoção por comunidades com diferentes capacidades técnicas.

O projeto oficial poderá manter uma instância de referência, sem que ela represente uma autoridade central sobre as demais.

## 12. Estratégia de testes

A qualidade deverá ser validada em diferentes níveis.

### 12.1. Testes unitários

Validar regras de domínio, casos de uso e comportamentos isolados.

### 12.2. Testes de integração

Validar a comunicação entre os componentes, persistência e serviços externos.

### 12.3. Testes de contrato

Validar os formatos e contratos de comunicação, especialmente as representações ActivityStreams e os mecanismos ActivityPub.

### 12.4. Testes de federação

Validar a interoperabilidade com outras implementações do Fediverso.

O objetivo não é apenas confirmar que a aplicação consegue enviar e receber mensagens, mas verificar que outras implementações conseguem interpretar e processar corretamente as atividades.

### 12.5. Testes de segurança

Avaliar autenticação, autorização, exposição de dados, processamento de conteúdo externo e comportamento diante de entradas maliciosas.

## 13. Evolução arquitetural

A arquitetura deverá ser desenvolvida incrementalmente.

A primeira versão deverá priorizar:

1. Modelo de domínio e persistência.
2. Identidade e autenticação.
3. Publicações e interações.
4. Integração básica com ActivityPub.
5. Processamento assíncrono.
6. Segurança e observabilidade.

Os domínios de projetos, grupos, organizações, recursos e atividades deverão ser incorporados conforme a evolução do produto.

A arquitetura não deverá antecipar infraestrutura distribuída ou serviços independentes sem justificativa técnica e operacional.

## 14. Decisões arquiteturais pendentes

As seguintes decisões deverão ser registradas e amadurecidas antes ou durante a implementação:

- Linguagem e framework do backend.
- Tecnologia da interface de usuário.
- Banco de dados principal.
- Estratégia de armazenamento de arquivos.
- Modelo de autenticação e gerenciamento de sessões.
- Estratégia de autorização.
- Tecnologia de filas e processamento assíncrono.
- Estratégia de implantação e distribuição.
- Representação federada de grupos e organizações.
- Política de visibilidade e distribuição de publicações.
- Estratégia de versionamento e compatibilidade de APIs.
- Estratégia de migração e atualização de instâncias.

As decisões deverão ser documentadas preferencialmente por meio de Architecture Decision Records (ADRs).

## 15. Relação com outros documentos

Este documento deve ser interpretado em conjunto com:

- `governance/CHARTER.md` — Carta de Fundação.
- `governance/GOVERNANCE.md` — Modelo de Governança.
- `governance/TRADEMARK_POLICY.md` — Política de Marca e Identidade.
- `governance/CODE_OF_CONDUCT.md` — Código de Conduta.
- `governance/CONTRIBUTING.md` — Guia de Contribuição.
- `docs/VISION.md` — Visão do Produto.
- `docs/ROADMAP.md` — Roadmap do Projeto.

A arquitetura deverá servir à visão do produto e respeitar os princípios institucionais e comunitários do projeto.