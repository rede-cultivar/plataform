# ADR 0006 — Estratégia de implantação e operação de instâncias

**Status:** Proposto  
**Data:** 2026-11-05

## Contexto

A plataforma será desenvolvida como uma aplicação independente, integrada ao Fediverso por meio do ActivityPub.

Um dos objetivos fundamentais do projeto é permitir que diferentes comunidades utilizem o software para construir e administrar seus próprios espaços de participação.

Essa possibilidade é importante para preservar a descentralização, a autonomia comunitária e a liberdade de organização.

O projeto oficial não deverá depender de uma única infraestrutura centralizada nem exigir que todas as comunidades utilizem uma instância administrada pelos mantenedores originais.

Ao mesmo tempo, a multiplicação de instâncias exige considerar responsabilidades operacionais, segurança, manutenção, interoperabilidade e administração.

É necessário estabelecer uma estratégia que permita diferentes formas de implantação sem comprometer a identidade e a evolução do software.

## Decisão proposta

A aplicação será desenvolvida para permitir a implantação de instâncias independentes.

Cada instância deverá possuir sua própria infraestrutura, configuração, domínio, base de dados e administração.

As instâncias poderão se comunicar por meio do ActivityPub, respeitando os mecanismos de federação e as políticas de cada comunidade.

A arquitetura deverá evitar dependências obrigatórias de uma infraestrutura central mantida pelo projeto oficial.

### 1. Instâncias independentes

Cada instância representará uma implantação autônoma da aplicação.

Responsabilidades de cada instância:

- Administrar suas próprias contas e participantes.
- Manter seus próprios dados e arquivos.
- Definir sua configuração institucional.
- Administrar suas políticas de participação e moderação.
- Gerenciar sua infraestrutura e disponibilidade.
- Controlar seus recursos operacionais.
- Comunicar-se com outras instâncias por meio da federação.

A indisponibilidade de uma instância não deverá impedir o funcionamento das demais.

### 2. Identidade da instância

Cada instância deverá possuir uma identidade própria, associada a um domínio.

Essa identidade permitirá:

- Identificar a origem das atividades federadas.
- Descobrir participantes hospedados na instância.
- Identificar informações institucionais e administrativas.
- Apresentar as políticas e os canais de contato da comunidade.
- Permitir que outras instâncias estabeleçam relações de confiança ou restrição.

A identidade de uma instância deverá ser independente da identidade do projeto oficial.

### 3. Autonomia administrativa

Cada instância será administrada por pessoas responsáveis por sua operação.

Os administradores deverão poder:

- Configurar a instância.
- Gerenciar participantes locais.
- Aplicar políticas de moderação.
- Administrar recursos e serviços.
- Configurar mecanismos de federação.
- Consultar registros operacionais.
- Realizar manutenção e atualização do software.

O projeto oficial poderá oferecer orientações, documentação e mecanismos de apoio, mas não deverá exercer controle administrativo obrigatório sobre instâncias independentes.

### 4. Distribuição do software

O software deverá ser distribuído de maneira que permita instalação e operação por diferentes comunidades.

A documentação deverá contemplar:

- Requisitos de infraestrutura.
- Procedimentos de instalação.
- Configuração inicial.
- Administração da instância.
- Atualização de versões.
- Backup e recuperação.
- Monitoramento e diagnóstico.
- Configuração de domínio e HTTPS.
- Procedimentos de segurança.

A estratégia de distribuição deverá favorecer automação e reprodutibilidade.

O formato específico de empacotamento e implantação será definido posteriormente.

### 5. Atualizações e compatibilidade

O projeto deverá estabelecer uma estratégia de versionamento e distribuição de novas versões.

As atualizações deverão considerar:

- Compatibilidade entre versões.
- Migrações de banco de dados.
- Alterações nos protocolos federados.
- Correções de segurança.
- Notas de versão.
- Procedimentos de atualização e recuperação.

Instâncias independentes poderão adotar novas versões em momentos diferentes.

A aplicação deverá tolerar diferenças razoáveis de versão entre instâncias federadas, dentro dos limites de compatibilidade estabelecidos pelo projeto.

### 6. Moderação e políticas locais

Cada instância terá autonomia para estabelecer suas políticas de participação e moderação.

Essa autonomia não elimina a necessidade de mecanismos de proteção contra abuso e comportamentos incompatíveis com os princípios fundamentais do projeto.

A aplicação deverá permitir:

- Bloqueio de participantes.
- Bloqueio ou restrição de outras instâncias.
- Denúncias e tratamento de conteúdo abusivo.
- Configuração de regras locais de participação.
- Administração de contas e permissões.
- Registro das ações administrativas relevantes.

A federação não deverá pressupor que todas as instâncias compartilham as mesmas políticas ou decisões de moderação.

### 7. Identidade oficial e instâncias independentes

A instância oficial, quando existir, será uma implantação do software mantida sob a governança do projeto comunitário.

Ela não será considerada uma autoridade administrativa sobre todas as demais instâncias.

O uso do software não implicará autorização automática para utilizar a marca ou a identidade oficial.

Instâncias independentes poderão adotar nomes, marcas e identidades próprias, respeitando a licença do software e a Política de Marca e Identidade.

A utilização da identidade oficial deverá seguir as regras estabelecidas pela comunidade responsável pelo projeto.

### 8. Sustentabilidade operacional

A operação de uma instância envolve custos e responsabilidades que deverão ser considerados por sua comunidade administradora.

Entre eles:

- Infraestrutura e hospedagem.
- Armazenamento e tráfego de dados.
- Backup e recuperação.
- Monitoramento.
- Atualizações e segurança.
- Moderação e suporte aos participantes.

O projeto deverá disponibilizar orientações que ajudem comunidades a avaliar esses custos e responsabilidades.

A existência de uma instância oficial não deverá ser confundida com a obrigação de hospedar gratuitamente todas as comunidades interessadas.

## Alternativas consideradas

### 1. Instância centralizada única

Toda a comunidade utilizaria uma única instância, administrada pelo projeto oficial.

**Vantagens:**
- Operação centralizada.
- Menor duplicação de infraestrutura.
- Facilidade de administração e atualização.
- Experiência inicial mais uniforme.

**Desvantagens:**
- Dependência de uma única infraestrutura.
- Concentração de responsabilidades e poder administrativo.
- Menor autonomia para comunidades.
- Maior impacto em caso de indisponibilidade.
- Contradição com os princípios de descentralização do projeto.

### 2. Instâncias independentes sem federação

Cada comunidade poderia instalar sua própria instância, mas sem comunicação federada.

**Vantagens:**
- Autonomia administrativa.
- Independência operacional.
- Possibilidade de políticas locais diferentes.

**Desvantagens:**
- Fragmentação das comunidades.
- Ausência de interoperabilidade.
- Dificuldade de interação entre participantes de instâncias diferentes.
- Perda de um dos principais benefícios do Fediverso.

### 3. Instâncias independentes e federadas

Cada comunidade poderá operar sua própria instância, mantendo comunicação com outras por meio do ActivityPub.

**Vantagens:**
- Autonomia comunitária.
- Descentralização operacional.
- Interoperabilidade.
- Possibilidade de políticas e identidades próprias.
- Maior resiliência diante de falhas localizadas.
- Possibilidade de crescimento distribuído.

**Desvantagens:**
- Maior complexidade de instalação e manutenção.
- Necessidade de documentação operacional.
- Diferenças de configuração e capacidade entre instâncias.
- Necessidade de gerenciar compatibilidade e segurança na federação.
- Possibilidade de fragmentação e conflitos entre comunidades.

## Consequências

A decisão estabelece:

- O suporte a múltiplas instâncias independentes como requisito arquitetural.
- Autonomia administrativa e operacional para cada instância.
- Identidade própria associada ao domínio de cada implantação.
- Federação como mecanismo de comunicação entre instâncias.
- Necessidade de documentação de instalação e operação.
- Necessidade de versionamento e estratégia de atualização.
- Separação entre a identidade do projeto oficial e a identidade das instâncias independentes.
- Ausência de dependência obrigatória de serviços centrais do projeto oficial.

Essa estratégia amplia a responsabilidade do projeto sobre a qualidade da distribuição, documentação e manutenção do software.

## Escopo e decisões futuras

Este ADR não determina ainda:

- O mecanismo de empacotamento e distribuição.
- A utilização de containers ou orquestradores.
- Os requisitos mínimos de infraestrutura.
- A estratégia de implantação da instância oficial.
- O modelo de configuração inicial.
- O processo de registro ou descoberta de instâncias.
- A política de compatibilidade entre versões.
- A estratégia de atualização automática ou manual.
- Os mecanismos de monitoramento.
- Os procedimentos de recuperação e migração.
- A política de suporte oferecida pelo projeto oficial.

Esses aspectos deverão ser detalhados conforme a evolução do produto.

## Referências

- [ActivityPub — W3C Recommendation](https://www.w3.org/TR/activitypub/)
- [AGPLv3 — GNU Affero General Public License](https://www.gnu.org/licenses/agpl-3.0.html)
- [Mastodon Documentation](https://docs.joinmastodon.org/)
- [The Twelve-Factor App](https://12factor.net/)