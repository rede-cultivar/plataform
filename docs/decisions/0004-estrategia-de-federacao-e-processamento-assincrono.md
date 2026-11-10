# ADR 0004 — Estratégia de federação e processamento assíncrono

**Status:** Aceito  
**Data:** 2026-11-05

## Contexto

A plataforma será integrada ao Fediverso por meio do ActivityPub.

Essa integração permitirá que participantes locais interajam com pessoas e comunidades de outras instâncias, recebendo e enviando publicações, comentários, reações e outras atividades compatíveis.

Entretanto, a comunicação entre instâncias ocorre por meio de redes externas, sujeitas a indisponibilidade, latência, falhas de conexão, limitações de tráfego e diferenças de implementação.

A execução dessas operações diretamente nos fluxos principais da aplicação poderia comprometer a disponibilidade e a experiência dos participantes.

É necessário, portanto, estabelecer uma estratégia que desacople as operações locais da distribuição federada.

## Decisão

A comunicação federada será realizada predominantemente por processamento assíncrono.

A aplicação manterá mecanismos próprios de entrada e saída de atividades, utilizando filas de processamento para desacoplar as operações de domínio da comunicação com instâncias externas.

A estratégia será baseada nos seguintes princípios:

### 1. Separação entre domínio e federação

As operações de negócio deverão ser concluídas independentemente da disponibilidade das instâncias externas.

Por exemplo, ao publicar uma experiência de cultivo:

1. A aplicação valida a publicação.
2. A publicação é persistida localmente.
3. Uma atividade federável é registrada para distribuição.
4. Um processo assíncrono identifica os destinatários.
5. A atividade é enviada às instâncias correspondentes.

A publicação não deverá depender de uma resposta imediata dos servidores externos.

### 2. Processamento de saída

As atividades destinadas a outras instâncias serão encaminhadas por uma fila de processamento.

Responsabilidades:

- Identificar os destinatários de cada atividade.
- Resolver os endpoints de entrega.
- Agrupar entregas quando houver suporte a Shared Inbox.
- Assinar requisições conforme os mecanismos de segurança adotados.
- Registrar tentativas e resultados.
- Reprocessar entregas que falharem temporariamente.

A distribuição deverá respeitar as regras de visibilidade e privacidade definidas pela aplicação.

### 3. Processamento de entrada

A aplicação deverá disponibilizar endpoints compatíveis com os mecanismos de recebimento do ActivityPub.

As atividades recebidas deverão passar por validações antes de produzir alterações no domínio.

O processamento deverá considerar:

- Validação de origem e assinatura.
- Identificação do Actor responsável.
- Verificação da integridade da atividade.
- Validação de permissões e contexto.
- Identificação de atividades duplicadas.
- Aplicação das regras de negócio correspondentes.
- Registro do processamento e de eventuais falhas.

Atividades externas não deverão ser consideradas confiáveis apenas por terem sido recebidas de uma instância conhecida.

### 4. Idempotência

A distribuição e o recebimento de atividades deverão considerar a possibilidade de repetição.

Uma mesma atividade poderá ser entregue mais de uma vez devido a retentativas, falhas de comunicação ou comportamento de outras implementações.

A aplicação deverá identificar atividades por seus identificadores e impedir que o reprocessamento provoque efeitos duplicados.

A estratégia deverá ser aplicada tanto à entrada quanto à saída.

### 5. Retentativas e falhas

Falhas temporárias de comunicação deverão gerar novas tentativas de entrega.

A estratégia deverá considerar:

- Retentativas com intervalos progressivos.
- Backoff exponencial e variação aleatória (*jitter*).
- Limites de tentativas configuráveis.
- Registro de erros e histórico de entrega.
- Tratamento separado para falhas permanentes.
- Possibilidade de reprocessamento controlado.

Atividades que não puderem ser entregues após as tentativas previstas deverão ser encaminhadas para uma estrutura de falhas, equivalente a uma *Dead Letter Queue*.

O reprocessamento deverá ser possível sem duplicar os efeitos da atividade.

### 6. Consistência entre persistência e distribuição

A aplicação deverá evitar situações em que uma operação local seja persistida, mas sua atividade federada seja perdida antes de entrar na fila.

Deverá ser avaliado o uso do padrão Transactional Outbox.

Nesse modelo, a alteração de domínio e o registro da atividade pendente são persistidos na mesma transação local. Um processo independente posteriormente encaminha a atividade para a fila ou mecanismo de distribuição.

A adoção de um mecanismo específico será definida em decisão posterior.

### 7. Observabilidade

A camada de federação deverá oferecer informações operacionais suficientes para identificar problemas de comunicação.

Deverão ser considerados:

- Quantidade de atividades pendentes.
- Quantidade de entregas concluídas.
- Quantidade de falhas.
- Idade das atividades pendentes.
- Número de retentativas.
- Instâncias com falhas recorrentes.
- Tempo de processamento e entrega.

A observabilidade deverá permitir distinguir problemas locais de indisponibilidade de instâncias externas.

## Alternativas consideradas

### 1. Federação síncrona

A aplicação enviaria atividades diretamente às instâncias externas durante a execução da operação local.

**Vantagens:**
- Implementação inicial aparentemente mais simples.
- Retorno imediato sobre algumas tentativas de comunicação.

**Desvantagens:**
- Dependência da disponibilidade de servidores externos.
- Aumento da latência das operações locais.
- Maior risco de falhas em cascata.
- Dificuldade de controlar retentativas.
- Acoplamento entre domínio e comunicação federada.

### 2. Federação assíncrona com processamento independente

A aplicação registra as atividades localmente e realiza a distribuição por processos independentes.

**Vantagens:**
- Desacoplamento entre operações locais e comunicação externa.
- Maior tolerância a falhas.
- Controle sobre retentativas e processamento.
- Possibilidade de escalar a distribuição independentemente.
- Melhor observabilidade operacional.

**Desvantagens:**
- Maior complexidade arquitetural.
- Necessidade de gerenciar filas e estados de processamento.
- Consistência eventual entre instâncias.
- Necessidade de mecanismos de idempotência e recuperação.

## Consequências

A decisão estabelece:

- A federação como processo assíncrono.
- Separação entre operações locais e distribuição externa.
- Tratamento independente de atividades recebidas e enviadas.
- Idempotência como requisito da integração.
- Retentativas e mecanismos de recuperação.
- Observabilidade específica para a camada federada.
- Consideração do padrão Transactional Outbox para garantir consistência entre persistência e distribuição.

A aplicação deverá aceitar que a comunicação entre instâncias não possui garantia de entrega imediata.

A consistência entre instâncias será eventual, respeitando as limitações e os comportamentos dos sistemas externos.

## Escopo e decisões futuras

Este ADR não determina ainda:

- Qual tecnologia de filas será utilizada.
- Se o processamento ocorrerá no mesmo processo da aplicação ou em workers independentes.
- Qual banco de dados armazenará as atividades pendentes.
- Qual será a política exata de retentativas.
- Como será implementado o Transactional Outbox.
- Como será feito o controle de ordenação de atividades.
- Como serão tratados bloqueios, exclusões e atualizações federadas.
- Como serão distribuídas atividades para grandes quantidades de destinatários.
- Qual será a estratégia de escalabilidade da federação.

Esses aspectos deverão ser detalhados em decisões posteriores.

## Referências

- [ActivityPub — W3C Recommendation](https://www.w3.org/TR/activitypub/)
- [ActivityStreams 2.0 — W3C Recommendation](https://www.w3.org/TR/activitystreams-core/)
- [Transactional Outbox Pattern — Microsoft Architecture Center](https://learn.microsoft.com/en-us/azure/architecture/databases/guide/transactional-out-box-cosmos)