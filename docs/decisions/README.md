# Architecture Decision Records

Este diretório reúne as decisões arquiteturais do projeto, registradas no formato **ADR** (*Architecture Decision Record*).

Cada decisão documenta o contexto, a decisão adotada, as alternativas consideradas e suas consequências. O objetivo é preservar o histórico das escolhas técnicas e permitir que a comunidade compreenda por que cada caminho foi seguido.

## Índice

| ADR | Título | Status | Data |
| --- | --- | --- | --- |
| [0001](0001-arquitetura-propria-com-activitypub.md) | Aplicação própria integrada ao Fediverso | Aceito | 2026-11-05 |
| [0002](0002-modelo-de-dominio-e-fronteiras-entre-os-contextos.md) | Modelo de domínio e fronteiras entre os contextos | Aceito | 2026-11-05 |
| [0003](0003-identidade-e-autenticacao.md) | Identidade e autenticação | Aceito | 2026-11-05 |
| [0004](0004-estrategia-de-federacao-e-processamento-assincrono.md) | Estratégia de federação e processamento assíncrono | Aceito | 2026-11-05 |
| [0005](0005-persistencia-e-armazenamento.md) | Persistência e armazenamento | Aceito | 2026-11-05 |
| [0006](0006-estrategia-de-implantacao-e-operacao.md) | Estratégia de implantação e operação de instâncias | Aceito | 2026-11-05 |
| [0007](0007-stack-da-aplicacao.md) | Stack da aplicação | Aceito | 2026-11-10 |
| [0008](0008-autenticacao-e-autorizacao.md) | Autenticação e autorização | Aceito | 2026-11-10 |
| [0009](0009-implementacao-da-federacao.md) | Implementação da federação | Aceito | 2026-11-10 |
| [0010](0010-empacotamento-e-implantacao.md) | Empacotamento, implantação e atualização | Aceito | 2026-11-10 |

## Status

- **Proposto:** decisão apresentada para discussão e ainda não consolidada.
- **Aceito:** decisão aprovada e vigente.
- **Substituído:** decisão que foi superada por um ADR posterior.
- **Rejeitado:** proposta descartada.

## Como registrar uma decisão

1. Copie o arquivo [`TEMPLATE.md`](TEMPLATE.md) para um novo arquivo numerado sequencialmente.
2. Preencha as seções com o contexto, a decisão proposta, as alternativas e as consequências.
3. Abra uma discussão ou pull request para revisão da comunidade.
4. Após a consolidação, atualize o status e o índice deste documento.
5. Decisões aceitas não devem ser reescritas: mudanças relevantes devem ser registradas em um novo ADR que substitua o anterior.

## Convenções

- Os arquivos seguem o padrão `NNNN-titulo-em-minusculas.md`.
- A numeração é sequencial e não deve ser reutilizada.
- O texto é redigido em português, mantendo consistência com a documentação do projeto.
- Referências a decisões anteriores devem apontar para o arquivo correspondente.
