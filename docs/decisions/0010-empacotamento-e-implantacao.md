# ADR 0010 — Empacotamento, implantação e atualização

**Status:** Aceito  
**Data:** 2026-11-10

## Contexto

O ADR 0006 estabeleceu que a aplicação deverá permitir a implantação de instâncias independentes, com infraestrutura, domínio, banco de dados e administração próprios. O projeto oficial não deverá exercer controle administrativo obrigatório sobre outras instâncias.

O ADR 0007 definiu um binário Go único, PostgreSQL, armazenamento compatível com S3 e processamento assíncrono baseado em outbox. O ADR 0009 definiu a estratégia de federação.

É necessário estabelecer como o software será empacotado, distribuído, instalado, configurado, atualizado e observado. A decisão deve reduzir a barreira de adoção por comunidades com diferentes capacidades técnicas, sem esconder responsabilidades operacionais relevantes.

## Decisão

### 1. Empacotamento

A aplicação será distribuída como:

- **Binário estático** para plataformas suportadas, com ativos embutidos.
- **Imagem OCI** (container), construída em múltiplos estágios, executada com usuário não privilegiado e sistema de arquivos somente leitura quando possível.
- **Código-fonte** sob licença AGPL-3.0, com instruções de build reprodutível.

Diretrizes:

- A versão, o commit e a data de build serão incorporados ao binário.
- As imagens serão publicadas em registro público, com assinatura e SBOM.
- Binários serão publicados com somas de verificação.
- A aplicação não exigirá Node.js, compilador ou runtime adicional em produção.

### 2. Implantação de referência

A forma de implantação de referência será **Docker Compose**, contemplando:

- Aplicação web.
- Trabalhador assíncrono.
- PostgreSQL.
- Armazenamento de objetos compatível com S3 (MinIO no ambiente de referência).
- Proxy reverso com terminação TLS (por exemplo, Caddy ou Traefik).

Diretrizes:

- A configuração de referência será documentada e mantida no repositório.
- Manifestos para Kubernetes poderão ser oferecidos posteriormente, sem se tornarem o caminho principal.
- A instalação deverá funcionar em uma única máquina para comunidades pequenas.
- A aplicação deverá ser compatível com execução em plataformas gerenciadas, sem depender delas.

### 3. Configuração

A configuração seguirá os princípios de Twelve-Factor App.

Diretrizes:

- Variáveis de ambiente serão o mecanismo principal.
- Um arquivo `.env.example` documentará todas as variáveis obrigatórias e opcionais.
- Segredos poderão ser fornecidos por variáveis de ambiente, arquivos ou mecanismos de segredos do ambiente de execução.
- Configurações inválidas deverão impedir a inicialização com mensagens claras.
- A aplicação deverá validar configurações críticas, como domínio local, conexão com banco, armazenamento e chaves.
- O domínio local será a identidade canônica da instância.

### 4. Migrações e evolução do esquema

- As migrações serão versionadas em SQL e embutidas no binário.
- A execução será feita por comando dedicado, como `cultivar migrate up`.
- A aplicação verificará a compatibilidade do esquema na inicialização e recusará operar com versão incompatível.
- As migrações deverão seguir o padrão expandir/contrair para permitir atualizações graduais e reduzir indisponibilidade.
- Alterações destrutivas deverão ser separadas em etapas e documentadas.
- O processo de atualização deverá ser documentado, incluindo backup prévio e procedimento de reversão quando possível.

### 5. Versionamento e compatibilidade

- O software utilizará **Versionamento Semântico**.
- A API pública será versionada explicitamente, inicialmente em `/api/v1`.
- O esquema do banco possuirá versão própria e verificável.
- Cada versão publicará notas de lançamento com mudanças relevantes, requisitos de migração e avisos de segurança.
- O projeto deverá definir e documentar uma janela de suporte para versões anteriores.
- A compatibilidade federada deverá ser considerada parte da compatibilidade do produto, não apenas da API interna.

### 6. Distribuição e atualização

- As versões serão publicadas no GitHub Releases e em registro de containers.
- Não haverá atualização automática obrigatória na primeira versão.
- A atualização será iniciada e controlada pela administração de cada instância.
- O projeto poderá oferecer notificações de novas versões, sem coletar dados desnecessários.
- Instâncias deverão poder permanecer em versões anteriores dentro da janela de suporte documentada.
- A atualização deverá preservar dados, chaves e configurações.

### 7. Backup, recuperação e retenção

- O projeto deverá documentar backup do banco relacional e do armazenamento de objetos.
- O procedimento de restauração deverá ser documentado e testável.
- A documentação deverá distinguir backup lógico e físico, com recomendações para diferentes escalas.
- Períodos de retenção serão definidos por política e configuráveis pela instância.
- A perda de chaves federadas deverá constar como cenário de recuperação.

### 8. Observabilidade

A aplicação deverá oferecer:

- Logs estruturados em JSON.
- Endpoints de saúde e prontidão.
- Métricas compatíveis com Prometheus.
- Monitoramento de filas, entregas federadas e falhas.
- Identificadores de correlação para requisições e processamento assíncrono.
- Registro de auditoria para ações administrativas e de segurança.

OpenTelemetry e integrações adicionais poderão ser avaliados posteriormente.

### 9. Segurança operacional

- TLS será terminado no proxy reverso, com redirecionamento de HTTP para HTTPS.
- Cabeçalhos de segurança serão aplicados pela aplicação ou pelo proxy.
- Containers executarão com o mínimo de privilégios.
- Segredos não serão incluídos em imagens nem em logs.
- Limites de recursos e de taxa deverão ser configuráveis.
- A documentação deverá orientar sobre exposição segura de banco e armazenamento.
- A instância oficial do projeto não terá papel de autoridade sobre as demais.

## Alternativas consideradas

### 1. Implantação baseada apenas em Kubernetes

**Vantagens:**
- Padronização para ambientes de grande escala.
- Recursos de orquestração, escalonamento e alta disponibilidade.

**Desvantagens:**
- Complexidade excessiva para comunidades pequenas.
- Barreira de adoção elevada.
- Não é necessária para validar o produto.

### 2. Pacotes nativos (deb, rpm, Homebrew)

**Vantagens:**
- Integração com gerenciadores de pacotes do sistema.
- Atualizações familiares para administradores.

**Desvantagens:**
- Manutenção de múltiplos formatos.
- Diferenças entre distribuições.
- Dependência de repositórios e assinatura de pacotes.

### 3. Aplicação serverless

**Vantagens:**
- Escalonamento gerenciado.
- Menor administração de servidores.

**Desvantagens:**
- Dependência de provedores específicos.
- Dificuldade com processamento assíncrono e conexões persistentes.
- Conflito com o princípio de autonomia e portabilidade.

### 4. Atualização automática

**Vantagens:**
- Correções de segurança mais rápidas.
- Menor esforço administrativo.

**Desvantagens:**
- Risco de indisponibilidade e regressões sem intervenção.
- Menor controle pelas comunidades.
- Necessidade de infraestrutura de distribuição confiável.

### 5. Banco embutido no mesmo container

**Vantagens:**
- Instalação simplificada.

**Desvantagens:**
- Dificulta backup, escalonamento e operação independente.
- Mistura ciclos de vida de componentes.
- Contraria a separação definida no ADR 0005.

## Consequências

A decisão estabelece:

- Distribuição em binário e imagem OCI.
- Docker Compose como implantação de referência.
- Configuração por variáveis de ambiente.
- Migrações versionadas e explícitas.
- Versionamento semântico e API versionada.
- Atualização controlada pela administração da instância.
- Procedimentos documentados de backup e recuperação.
- Observabilidade mínima baseada em logs, métricas e endpoints de saúde.

A decisão também implica:

- Responsabilidade contínua sobre documentação de instalação e atualização.
- Necessidade de manter imagens e binários para múltiplas arquiteturas.
- Custo de suporte a diferentes formas de implantação.
- Disciplina nas migrações para permitir atualizações seguras.
- Compromisso com a portabilidade entre ambientes.

## Escopo e decisões futuras

Este ADR não determina ainda:

- A lista definitiva de arquiteturas e plataformas suportadas.
- A estratégia de alta disponibilidade e replicação.
- A topologia de escalonamento horizontal da aplicação e dos trabalhadores.
- O uso de CDN para mídia.
- A política de suporte oficial a versões anteriores.
- O processo de registro ou descoberta de instâncias.
- A oferta de serviços gerenciados pelo projeto oficial.
- A estratégia de monitoramento centralizado.
- Os procedimentos detalhados de resposta a incidentes.
- A política de compatibilidade entre versões do esquema e da federação.

Esses aspectos deverão ser detalhados conforme a evolução do produto e as necessidades reais das comunidades.

## Referências

- [The Twelve-Factor App](https://12factor.net/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Open Container Initiative](https://opencontainers.org/)
- [Semantic Versioning](https://semver.org/)
- [PostgreSQL Backup and Restore](https://www.postgresql.org/docs/current/backup.html)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [AGPLv3 — GNU Affero General Public License](https://www.gnu.org/licenses/agpl-3.0.html)
