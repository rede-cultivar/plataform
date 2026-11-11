# ADR 0009 — Implementação da federação

**Status:** Aceito  
**Data:** 2026-11-10

## Contexto

O ADR 0001 definiu a adoção do ActivityPub como principal protocolo de federação. O ADR 0003 estabeleceu a separação entre identidade interna e identidade federada. O ADR 0004 determinou que a comunicação federada ocorrerá predominantemente por processamento assíncrono, com idempotência, retentativas e observabilidade. O ADR 0007 definiu a stack e a fila inicial baseada em PostgreSQL.

O ActivityPub é uma recomendação do W3C, mas não define todos os detalhes necessários à interoperabilidade prática. Implementações existentes adotam convenções complementares, extensões e comportamentos que precisam ser considerados. Além disso, conceitos específicos do produto, como grupos, organizações, projetos e recursos, não possuem representação padronizada completa no ActivityPub.

É necessário definir:

- Qual implementação será utilizada como referência de compatibilidade.
- Como as entidades do domínio serão representadas como objetos e Actors.
- Como as atividades serão recebidas, validadas, armazenadas e distribuídas.
- Como os riscos de segurança inerentes à federação serão mitigados.

## Decisão

### 1. Referência de compatibilidade

A implementação de referência inicial será o **Mastodon**.

Diretrizes:

- O comportamento do Mastodon será priorizado na implementação de servidor para servidor.
- A compatibilidade será validada também com **GoToSocial**, **Pleroma/Akkoma** e, quando viável, **Misskey/Firefish**.
- A compatibilidade com implementações reais deverá ser testada continuamente, e não apenas por meio de fixtures de protocolo.
- Extensões específicas do produto não poderão impedir a interoperabilidade das funcionalidades básicas.

### 2. Protocolos e padrões

A federação utilizará:

- **ActivityPub** para comunicação servidor para servidor.
- **ActivityStreams 2.0** para representação de objetos e atividades.
- **WebFinger** para descoberta de identidades.
- **HTTP Signatures** para autenticação de mensagens, considerando tanto o padrão atual (RFC 9421) quanto o perfil legado amplamente utilizado pelas implementações existentes.
- **NodeInfo 2.0** para informações da instância.
- **HTTPS** para todas as comunicações.

A aplicação deverá publicar documentos com o tipo de mídia `application/activity+json` e aceitar negociação de conteúdo adequada.

### 3. Representação das entidades

O mapeamento inicial será:

| Entidade do domínio | Representação federada |
| --- | --- |
| Pessoa | `Person` |
| Grupo | `Group` |
| Organização | `Organization` |
| Instância | `Application` |
| Publicação | `Note` |
| Comentário | `Note` |
| Atividade ou evento | Objeto com extensão própria |
| Projeto | Objeto com extensão própria, não Actor na primeira versão |
| Recurso | Objeto com extensão própria |

Diretrizes:

- Pessoas, grupos e organizações poderão possuir Actors próprios.
- Projetos e recursos não serão tratados como Actors na primeira versão, evitando complexidade de participação e moderação que o ActivityPub não padroniza.
- Os identificadores locais de Actor deverão ser estáveis e opacos, no formato `https://{domínio}/actors/{id}`.
- O identificador público poderá ser apresentado como `@nome@domínio`, resolvido por WebFinger.
- Extensões do produto deverão utilizar namespace próprio e permanecer opcionais para consumidores externos.
- A representação externa não substituirá o modelo interno de participação, permissões e administração.

### 4. Objetos e coleções

- O Outbox de cada Actor publicará as atividades produzidas localmente.
- As coleções `followers`, `following` e `liked` serão expostas quando aplicável.
- A paginação utilizará `OrderedCollectionPage`.
- Publicações e demais objetos deverão incluir `attributedTo`, `published`, `to` e `cc` coerentes com a visibilidade.
- A aplicação manterá uma caixa de entrada compartilhada (`sharedInbox`) para reduzir o volume de entregas.
- Objetos recebidos poderão ser armazenados com o payload original e com a representação normalizada utilizada internamente.

### 5. Entrega de atividades

A entrega seguirá o padrão Transactional Outbox definido no ADR 0004.

Diretrizes:

- A atividade será registrada na mesma transação local da operação de domínio.
- Um trabalhador assíncrono realizará a entrega por destinatário ou por caixa de entrada compartilhada.
- As tentativas utilizarão recuo exponencial com variação aleatória e limite máximo configurável.
- Falhas permanentes serão registradas em tabela de falhas para análise e reprocessamento manual.
- A idempotência será garantida pelo identificador da atividade.
- O controle de ordenação será feito por Actor e por destino, quando necessário.
- Requisições deverão incluir `Digest` e assinatura HTTP válida.
- O sistema deverá limitar concorrência e respeitar limites de taxa dos destinos.

### 6. Recebimento de atividades

O recebimento deverá observar as seguintes etapas:

1. Validar a assinatura HTTP e a origem da requisição.
2. Resolver ou atualizar o Actor remoto, com cache e prazo de validade.
3. Verificar duplicidade pelo identificador da atividade.
4. Validar o contexto, os tipos e os campos mínimos.
5. Aplicar as regras de visibilidade e moderação.
6. Persistir o payload e a representação normalizada.
7. Executar o caso de uso correspondente no domínio.

Atividades inicialmente suportadas:

- `Create`
- `Update`
- `Delete`
- `Follow`
- `Accept`
- `Reject`
- `Undo`
- `Like`
- `Announce`

Atividades desconhecidas ou inválidas deverão ser registradas e descartadas de forma segura, sem comprometer o processamento das demais.

### 7. Segurança

A federação deverá tratar explicitamente riscos como:

- **SSRF:** bloqueio de endereços privados, locais e de metadados, com validação de DNS e limites de redirecionamento.
- **Abuso de recursos:** limites de tamanho, tempo de resposta, quantidade de objetos e concorrência por origem.
- **Conteúdo malicioso:** validação de HTML, sanitização de campos textuais e tratamento seguro de mídia.
- **Falsificação:** verificação obrigatória de assinaturas e proteção contra reutilização.
- **Spam e abuso:** limites de taxa, listas de bloqueio e mecanismos de moderação.
- **Contextos JSON-LD remotos:** uso de contextos fixos e conhecidos, evitando busca arbitrária de contextos externos.
- **Chaves:** cache de chaves públicas com invalidação controlada e proteção contra respostas excessivamente grandes.

### 8. Visibilidade e distribuição

A visibilidade interna deverá ser mapeada explicitamente para os campos de endereçamento do ActivityPub:

| Visibilidade interna | Endereçamento |
| --- | --- |
| Pública | `to: Public`, `cc: followers` |
| Não listada | `to: followers`, sem `Public` |
| Somente seguidores | `to: followers` |
| Direta | destinatários explícitos |

Diretrizes:

- Conteúdo direto ou restrito não deverá ser distribuído publicamente por engano.
- A aplicação deverá considerar que o controle sobre cópias já entregues a outras instâncias é limitado.
- Alterações de visibilidade deverão gerar `Update` ou `Delete` quando aplicável, sem garantia de remoção em cópias externas.

### 9. Estratégia de implementação

A camada de federação será implementada como um pacote interno próprio, utilizando bibliotecas pequenas e especializadas quando necessário, em vez de um framework completo de ActivityPub.

Diretrizes:

- O modelo de domínio permanecerá independente das estruturas do ActivityPub.
- A tradução entre domínio e ActivityStreams será explícita e testável.
- O comportamento será validado com implementações reais.
- Fixtures de protocolo serão utilizadas para testes de contrato, mas não substituirão testes de interoperabilidade.
- A implementação deverá ser conservadora: enviar apenas o necessário e aceitar variações razoáveis de implementações externas.

## Alternativas consideradas

### 1. Utilizar um framework completo de ActivityPub

**Vantagens:**
- Reaproveitamento de implementação de protocolo.
- Menor esforço inicial em detalhes de serialização.

**Desvantagens:**
- Acoplamento ao modelo do framework.
- Maturidade e manutenção variáveis.
- Dificuldade de acomodar conceitos específicos do produto.
- Menor controle sobre compatibilidade prática.

### 2. Customizar uma implementação existente, como Mastodon

**Vantagens:**
- Compatibilidade federada desde o início.
- Reaproveitamento de moderação e operação.

**Desvantagens:**
- Dependência do modelo de domínio externo.
- Adaptações complexas para projetos, grupos e recursos.
- Menor autonomia sobre a evolução.

### 3. Implementar apenas cliente para servidor

**Vantagens:**
- Menor complexidade inicial.
- Integração com contas existentes.

**Desvantagens:**
- Não atende ao objetivo de identidade federada própria.
- Limita a participação de comunidades que desejam operar sua própria instância.
- Não valida a proposta central do projeto.

### 4. Protocolo próprio sem ActivityPub

**Vantagens:**
- Liberdade total de modelagem.

**Desvantagens:**
- Isolamento do Fediverso.
- Perda de interoperabilidade e de comunidades existentes.
- Contradiz o ADR 0001.

## Consequências

A decisão estabelece:

- Mastodon como referência inicial de compatibilidade.
- Uso de ActivityPub, ActivityStreams, WebFinger, HTTP Signatures e NodeInfo.
- Pessoas, grupos e organizações como Actors; projetos e recursos como objetos.
- Entrega assíncrona com outbox, retentativas e idempotência.
- Validação obrigatória de assinaturas e proteções contra SSRF e abuso.
- Mapeamento explícito de visibilidade para endereçamento ActivityPub.

A decisão também implica:

- Esforço contínuo de compatibilidade com implementações externas.
- Necessidade de testes de interoperabilidade em ambiente real.
- Complexidade adicional no tratamento de grupos e organizações, cuja semântica federada não é padronizada.
- Responsabilidade sobre segurança, moderação e limites de tráfego.
- Evolução cuidadosa das extensões próprias para não quebrar interoperabilidade.

## Escopo e decisões futuras

Este ADR não determina ainda:

- A semântica federada completa de participação em grupos e organizações.
- A estratégia de descoberta e sincronização de membros de grupos remotos.
- O suporte a relays e a federação de grande escala.
- A política de autorização de busca (`authorized fetch`).
- A estratégia de cache e distribuição de mídia.
- A implementação de listas de bloqueio compartilhadas.
- A migração de identidades entre instâncias.
- A rotação e recuperação de chaves federadas.
- A representação federada detalhada de projetos, recursos e atividades.
- A estratégia de busca federada.

Esses aspectos deverão ser detalhados em decisões posteriores.

## Referências

- [ActivityPub — W3C Recommendation](https://www.w3.org/TR/activitypub/)
- [ActivityStreams 2.0 — W3C Recommendation](https://www.w3.org/TR/activitystreams-core/)
- [WebFinger — RFC 7033](https://www.rfc-editor.org/rfc/rfc7033)
- [HTTP Signatures — RFC 9421](https://www.rfc-editor.org/rfc/rfc9421)
- [NodeInfo 2.0](https://nodeinfo.diaspora.software/)
- [Mastodon Documentation](https://docs.joinmastodon.org/)
- [GoToSocial Documentation](https://docs.gotosocial.org/)
- [OWASP Server-Side Request Forgery Prevention](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)
