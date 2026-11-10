# ADR 0008 — Autenticação e autorização

**Status:** Aceito  
**Data:** 2026-11-10

## Contexto

O ADR 0003 estabeleceu um modelo de identidade baseado em contas locais, com separação entre conta de acesso, pessoa, perfil público e identidade federada. Também definiu que grupos e organizações terão identidade própria e serão administrados por pessoas autenticadas, sem que isso implique contas de acesso para as entidades.

O ADR 0007 definiu a stack inicial da aplicação. Falta estabelecer os mecanismos concretos de autenticação e autorização, incluindo:

- Como as credenciais serão armazenadas e verificadas.
- Como as sessões serão mantidas e revogadas.
- Como as permissões serão verificadas nos diferentes contextos.
- Como as chaves criptográficas da federação serão protegidas.
- Como administradores de instância e de entidades coletivas serão definidos.

A decisão deve equilibrar segurança, simplicidade de implementação e facilidade de uso, sem introduzir dependência de provedores externos.

## Decisão

### 1. Contas locais

A autenticação inicial será baseada em **e-mail e senha**.

Diretrizes:

- O e-mail será único por conta e deverá ser verificado antes da ativação de funcionalidades federadas.
- Senhas serão armazenadas exclusivamente como hash **Argon2id**, com parâmetros alinhados às recomendações atuais da OWASP.
- O sistema deverá rejeitar senhas fracas, vazadas ou excessivamente comuns, quando a verificação for viável sem comprometer a privacidade.
- Mensagens de erro não deverão revelar se um e-mail está cadastrado.
- A recuperação de senha utilizará token de uso único, com validade curta e armazenamento apenas do hash.
- O cadastro e a recuperação deverão ser protegidos contra automação abusiva por meio de limitação de taxa e, quando necessário, verificação adicional.

### 2. Sessões

A autenticação de pessoas utilizará **sessões no servidor**, com token opaco de alta entropia.

Diretrizes:

- O token de sessão será armazenado no banco apenas em forma de hash.
- O cookie será `Secure`, `HttpOnly` e `SameSite=Lax`.
- A sessão terá prazo de inatividade e prazo absoluto configuráveis.
- O token será rotacionado no login e em mudanças relevantes de privilégio.
- O logout e a alteração de senha revogarão as sessões correspondentes.
- Requisições que alteram estado deverão ser protegidas contra CSRF.
- Sessões poderão ser listadas e revogadas pela própria pessoa em uma etapa posterior.

Não serão utilizados tokens JWT autocontidos como mecanismo principal de sessão na primeira versão.

### 3. Autorização

A autorização será aplicada na camada de aplicação, com verificação explícita por operação.

Modelo inicial:

- **Papéis de instância:** administrador, moderador e membro.
- **Papéis em grupos:** proprietário, administrador, moderador e membro.
- **Papéis em organizações:** administrador, editor e membro.
- **Papéis em projetos:** mantenedor e participante.
- **Regras de relacionamento:** vínculo de participação, autoria, administração e visibilidade do recurso.

Diretrizes:

- Toda operação que altera dados ou age em nome de outra entidade deverá verificar permissão explicitamente.
- O padrão será negar por padrão.
- As regras deverão residir no contexto responsável pelo recurso, por meio de funções de autorização explícitas.
- Papéis sociais, como seguir alguém ou participar de um grupo, não deverão ser confundidos com permissões administrativas.
- Não será introduzido motor externo de políticas, como OPA ou Cedar, na primeira versão.

### 4. Administração da instância

- O primeiro administrador será criado por comando administrativo ou variável de ambiente durante a instalação.
- Não haverá promoção automática a administrador por cadastro público.
- Ações administrativas relevantes deverão ser registradas em log de auditoria.

### 5. Chaves da federação

Cada Actor local possuirá um par de chaves próprio para assinatura de atividades federadas.

Diretrizes:

- A chave privada será gerada localmente e nunca será exposta em respostas públicas.
- A chave privada será cifrada em repouso com chave mestra fornecida pela configuração da instância.
- A escolha inicial do algoritmo deverá priorizar compatibilidade com as implementações de referência do Fediverso. RSA de 2048 bits atende ao requisito inicial; Ed25519 poderá ser adotado quando houver suporte interoperável consistente.
- Deverá existir procedimento documentado de rotação de chaves.
- A perda da chave mestra deverá ser tratada como incidente operacional, com procedimento de recuperação documentado.

### 6. Tokens de API

Clientes externos e integrações futuras poderão utilizar **tokens pessoais de acesso**, com escopos limitados e armazenamento apenas do hash.

Esses tokens não substituirão as sessões de navegador e não serão implementados na primeira versão, salvo necessidade concreta.

### 7. Proteções adicionais

- Limitação de taxa em login, cadastro, recuperação de senha e endpoints federados.
- Bloqueio temporário após tentativas repetidas de autenticação.
- Cabeçalhos de segurança e política de conteúdo.
- Registro de eventos de autenticação e autorização relevantes.
- Proteção contra fixação de sessão e contra reutilização de tokens de recuperação.
- Verificação de permissão também nos fluxos assíncronos e federados, não apenas nas requisições HTTP.

## Alternativas consideradas

### 1. JWT autocontido como sessão

**Vantagens:**
- Verificação sem consulta ao banco.
- Facilidade de uso em múltiplos serviços.

**Desvantagens:**
- Revogação imediata é difícil.
- Aumenta a complexidade de rotação e proteção de chaves.
- Não traz benefícios relevantes para uma aplicação monolítica inicial.

### 2. Provedor externo de identidade (OAuth2/OIDC)

**Vantagens:**
- Reaproveitamento de autenticação existente.
- Possibilidade de login social.

**Desvantagens:**
- Dependência de terceiros.
- Maior complexidade de configuração para instâncias independentes.
- Menor controle sobre o ciclo de vida das contas.

### 3. Apenas passkeys

**Vantagens:**
- Alta resistência a phishing.
- Boa experiência em dispositivos compatíveis.

**Desvantagens:**
- Cobertura ainda desigual.
- Recuperação de conta mais complexa.
- Pode excluir participantes com dispositivos ou navegadores antigos.

### 4. Motor externo de políticas

**Vantagens:**
- Centralização e testabilidade das regras.
- Padronização de decisões de autorização.

**Desvantagens:**
- Complexidade operacional adicional.
- Excesso de infraestrutura para o estágio inicial.
- As regras de domínio ainda estão em consolidação.

## Consequências

A decisão estabelece:

- Contas locais com e-mail e senha.
- Argon2id como função de hash de senha.
- Sessões no servidor com cookies seguros e revogação.
- Autorização explícita por contexto, com negação por padrão.
- Papéis de instância e de entidades coletivas.
- Chaves federadas por Actor, cifradas em repouso.
- Ausência de dependência de provedor externo de identidade.

A decisão também implica:

- Necessidade de infraestrutura de envio de e-mail para verificação e recuperação.
- Manutenção de tabelas de sessões, tokens e auditoria.
- Cuidado adicional na proteção da chave mestra e no procedimento de rotação.
- Evolução futura para múltiplos fatores e tokens de API.

## Escopo e decisões futuras

Este ADR não determina ainda:

- O provedor ou serviço de envio de e-mail.
- A implementação de autenticação em dois fatores.
- A política de sessões simultâneas e gerenciamento de dispositivos.
- Os escopos e o formato dos tokens de API.
- A federação de autenticação ou login entre instâncias.
- O detalhamento do modelo de permissões por contexto.
- As regras de exclusão, anonimização e transferência de contas.
- A política de retenção de logs de auditoria.
- A recuperação de conta quando o e-mail não estiver mais acessível.

Esses aspectos deverão ser detalhados em decisões posteriores.

## Referências

- [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
- [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- [Argon2 — RFC 9106](https://www.rfc-editor.org/rfc/rfc9106)
- [ActivityPub — W3C Recommendation](https://www.w3.org/TR/activitypub/)
- [HTTP Signatures — RFC 9421](https://www.rfc-editor.org/rfc/rfc9421)
- [OWASP Authorization Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html)
