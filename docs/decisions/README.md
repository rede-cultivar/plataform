# Architecture Decision Records

Este diretório reúne as decisões arquiteturais do projeto, registradas no formato **ADR** (*Architecture Decision Record*).

Cada decisão documenta o contexto, a decisão adotada, as alternativas consideradas e suas consequências. O objetivo é preservar o histórico das escolhas técnicas e permitir que a comunidade compreenda por que cada caminho foi seguido.

## Índice

| ADR | Título | Status | Data |
| --- | --- | --- | --- |
| [0001](0001-arquitetura-propria-com-activitypub.md) | Aplicação própria integrada ao Fediverso | Proposto | 2026-11-05 |
| [0002](0002-modelo-de-dominio-e-fronteiras-entre-os-contextos.md) | Modelo de domínio e fronteiras entre os contextos | Proposto | 2026-11-05 |
| [0003](0003-identidade-e-autenticacao.md) | Identidade e autenticação | Proposto | 2026-11-05 |
| [0004](0004-estrategia-de-federacao-e-processamento-assincrono.md) | Estratégia de federação e processamento assíncrono | Proposto | 2026-11-05 |
| [0005](0005-persistencia-e-armazenamento.md) | Persistência e armazenamento | Proposto | 2026-11-05 |
| [0006](0006-estrategia-de-implantacao-e-operacao.md) | Estratégia de implantação e operação de instâncias | Proposto | 2026-11-05 |
| [0007](0007-stack-da-aplicacao.md) | Stack da aplicação | Proposto | 2026-11-10 |
| [0008](0008-autenticacao-e-autorizacao.md) | Autenticação e autorização | Proposto | 2026-11-10 |
| [0009](0009-implementacao-da-federacao.md) | Implementação da federação | Proposto | 2026-11-10 |
| [0010](0010-empacotamento-e-implantacao.md) | Empacotamento, implantação e atualização | Proposto | 2026-11-10 |
| [0011](0011-diario-de-cultivo-e-biblioteca-de-praticas.md) | Diário de cultivo e biblioteca de práticas | Proposto | 2026-11-14 |
| [0012](0012-modos-de-cadastro-e-administracao-da-instancia.md) | Modos de cadastro e administração da instância | Proposto | 2026-11-16 |
| [0013](0013-experiencia-mobile-pwa-e-redesenho.md) | Experiência mobile, PWA e redesenho da interface | Proposto | 2026-11-19 |

Todas as decisões permanecem **abertas para revisão**. O projeto ainda não
realizou a rodada de discussão com a comunidade e nenhum ADR deve ser
considerado consolidado. As decisões já refletidas no código descrevem o estado
atual da implementação, mas podem ser revistas ou substituídas.

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
