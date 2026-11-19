# Design system do Cultivar

Este documento descreve a fundação visual e os padrões de interface da
plataforma. Ele acompanha o [ADR 0013](decisions/0013-experiencia-mobile-pwa-e-redesenho.md)
e deve ser consultado antes de criar novas telas ou componentes.

O objetivo é uma interface **mobile-first**, legível sob luz natural, tolerante a
conexões instáveis e coerente com a identidade de cultivo comunitário. A
aplicação continua sendo renderizada no servidor, com HTML semântico e JavaScript
apenas como aprimoramento.

## Princípios

1. **O conteúdo primeiro.** A pessoa deve encontrar o diário, a prática ou o
   perfil sem atravessar camadas de navegação.
2. **Toque antes do ponteiro.** Áreas de toque têm pelo menos 44 px; controles
   importantes ficam ao alcance do polegar.
3. **Contraste e tamanho.** Texto base de 16 px, contraste compatível com WCAG AA
   e nenhuma informação transmitida apenas por cor.
4. **Progressivo por padrão.** Toda jornada essencial funciona sem JavaScript.
5. **Sem dependências externas.** Nenhuma fonte, ícone ou biblioteca é baixada de
   terceiros. Ícones são SVG inline; tipografia usa fontes do sistema.
6. **Natureza sem literalidade.** Verde, terra e papel aparecem em tokens e
   detalhes, não em texturas ou ilustrações pesadas.

## Tokens

Os tokens ficam em `:root` no início de `internal/web/static/app.css`. O tema
escuro sobrescreve apenas as cores.

### Cores

| Token | Claro | Escuro | Uso |
| --- | --- | --- | --- |
| `--bg` | `#f7f5ef` | `#141a14` | Fundo da página |
| `--surface` | `#fffdf8` | `#1b241b` | Cartões, cabeçalho, painéis |
| `--surface-2` | `#efece1` | `#222d22` | Campos, faixas e estados sutis |
| `--fg` | `#1f2a1f` | `#e8ece3` | Texto principal |
| `--muted` | `#5c6b5c` | `#a7b3a3` | Texto secundário |
| `--accent` | `#2f6b3a` | `#8fc99a` | Ações, links e destaque |
| `--accent-strong` | `#24542d` | `#b6e0bd` | Hover e ênfase |
| `--accent-soft` | `rgba(47, 107, 58, 0.10)` | `rgba(143, 201, 154, 0.12)` | Fundos de destaque |
| `--border` | `#d8d4c6` | `#2d3a2d` | Bordas e divisores |
| `--danger` | `#b3261e` | `#f2b8b5` | Erro e ações destrutivas |
| `--warning` | `#8a5a00` | `#f0c36d` | Avisos |
| `--success` | `#24542d` | `#8fc99a` | Confirmação |

Regras:

- Texto sobre `--accent` usa `--bg` no tema claro e `--bg` no escuro, mantendo
  contraste.
- Nunca use cor como único indicador de estado; acompanhe com texto ou ícone.
- Novas cores entram como token, não como valor solto no componente.

### Tipografia

| Token | Valor |
| --- | --- |
| `--font-sans` | `system-ui, -apple-system, "Segoe UI", Roboto, sans-serif` |
| `--font-serif` | `ui-serif, Georgia, "Iowan Old Style", "Palatino Linotype", serif` |

- Títulos (`h1`, `h2`, `h3`, `.brand`) usam `--font-serif`, com peso 600 e
  `line-height` entre 1.15 e 1.3.
- Texto corrido usa `--font-sans`, 1rem, `line-height: 1.65`.
- `h1` usa `clamp(1.85rem, 5vw, 2.75rem)`.
- Textos auxiliares usam `.muted` e no máximo 0.95rem.
- Não use `text-transform: uppercase` em blocos longos.

### Espaçamento e forma

| Token | Valor | Uso |
| --- | --- | --- |
| `--space-1` | `0.25rem` | Ajustes finos |
| `--space-2` | `0.5rem` | Ícone e rótulo |
| `--space-3` | `0.75rem` | Itens de lista |
| `--space-4` | `1rem` | Padding padrão |
| `--space-5` | `1.5rem` | Blocos |
| `--space-6` | `2rem` | Seções |
| `--space-7` | `3rem` | Separação ampla |
| `--radius-sm` | `0.5rem` | Campos e botões pequenos |
| `--radius` | `0.85rem` | Cartões e painéis |
| `--radius-lg` | `1.25rem` | Destaques e mídia |
| `--radius-full` | `999px` | Etiquetas e avatares |
| `--tap` | `2.75rem` | Área mínima de toque |
| `--content` | `72rem` | Largura máxima do conteúdo |
| `--bottom-nav-height` | `4.5rem` | Reserva para a barra inferior |

### Sombras

| Token | Uso |
| --- | --- |
| `--shadow-sm` | Cartões em repouso |
| `--shadow` | Painéis elevados, menu **Mais** |

Sombras são discretas e não substituem bordas. No tema escuro, a borda é o
principal separador.

## Estrutura da página

```
<body>
  <a class="skip-link">…</a>
  <header class="site-header">…</header>
  <main id="conteudo">…</main>
  <footer class="site-footer">…</footer>
  <nav class="bottom-nav">…</nav>
</body>
```

- `main` recebe `id="conteudo"` e `tabindex="-1"` para o link de pular navegação.
- Alertas globais (`Error`, `Notice`) aparecem no topo de `main`.
- No celular, `body` reserva `padding-bottom` para a barra inferior e para
  `env(safe-area-inset-bottom)`.
- No desktop, a barra inferior e o menu **Mais** ficam ocultos.

## Navegação

### Celular (até 720 px)

Barra inferior fixa com cinco itens:

| Item | Destino | Ícone |
| --- | --- | --- |
| Início | `/` | casa |
| Diário | `/diario` | broto |
| Práticas | `/praticas` | livro |
| Perfil | `/perfil/{handle}` ou `/entrar` | pessoa |
| Mais | painel `<details>` | três pontos |

O item ativo usa `--accent`, um fundo `--accent-soft` e `aria-current="page"`.
O painel **Mais** contém: minha conta, seguidores, administração (se houver),
links de estado da instância e saída. Ele usa `<details>` e funciona sem
JavaScript; com o script, também fecha ao clicar fora ou pressionar `Esc`.

### Desktop (a partir de 721 px)

A navegação fica no cabeçalho, com os mesmos destinos e um agrupamento de conta.
O cabeçalho é compacto, com borda inferior e fundo `--surface`. Não é fixo.

## Componentes

### Botões

Classes: `.button`, `.button-secondary`, `.button-danger`, `.button-small`,
`.link-button`.

- Altura mínima de `--tap`; padding horizontal de 1rem.
- `.button` é a ação principal da tela; use no máximo uma por bloco.
- `.button-secondary` é contorno; `.button-danger` é destrutiva.
- `.link-button` parece texto, mas mantém área de toque e foco visível.
- Estados: `:hover`, `:active`, `:focus-visible`, `[disabled]`.

### Campos

Classes: `.stack`, `.filters`, `.hint`, `.media-choice`.

- `label` envolve o campo e o texto; não use placeholder como rótulo.
- Campos têm no mínimo 44 px de altura e 16 px de fonte para evitar zoom no iOS.
- `.hint` explica limites e privacidade logo abaixo do campo.
- `.filters` usa grid responsivo e um botão de submissão alinhado ao fim.
- Erros usam `.alert-error` no topo e, quando possível, texto junto ao campo.

### Cartões

Classes: `.cards`, `.card`.

- Grid `repeat(auto-fill, minmax(min(100%, 18rem), 1fr))`.
- Cada cartão tem título, metadados e ações contextuais.
- O cartão inteiro não é clicável; o título é o link principal.
- Não aninhe cartões.

### Linha do tempo do diário

Classes: `.timeline`, `.entry`, `.entry-meta`, `.entry-media`, `.entry-photo`,
`.entry-interactions`, `.replies`, `.reply`.

- Cada registro é um cartão com borda esquerda de destaque.
- Data e hora vêm antes do corpo.
- Fotos usam grid responsivo, com no máximo quatro por registro.
- Interações recebidas ficam separadas por uma borda tracejada.
- Respostas locais usam `.reply-local` e a etiqueta “Sua resposta”.

### Alertas

Classes: `.alert`, `.alert-error`, `.alert-success`, `.alert-info`,
`.alert-warning`.

- Sempre com `role="alert"` para erro e `role="status"` para confirmação.
- Mensagem curta, com ação ou próximo passo quando existir.
- A cor da borda reforça o tipo; o texto continua legível sem ela.

### Etiquetas e avatares

Classes: `.tag`, `.badge`, `.avatar`.

- `.tag` indica visibilidade, estágio ou categoria.
- `.badge` é reservado para o ambiente da instância no cabeçalho.
- `.avatar` tem 48 px, borda e `object-fit: cover`.

### Dados e tabelas

Classes: `.details`, `.measurements`, `.table-scroll`.

- `.details` usa duas colunas no desktop e uma no celular.
- Medidas usam `.measurements`, com nome e valor.
- Tabelas ficam dentro de `.table-scroll` e rolam horizontalmente.

### Estados vazios

Use um parágrafo `.muted` com uma ação quando fizer sentido:

```html
<p class="muted">Você ainda não registrou nenhum ciclo de cultivo.</p>
<p class="actions"><a class="button" href="/diario/novo">Criar primeiro ciclo</a></p>
```

Não use ilustrações ou mensagens genéricas.

### Aviso de rede

O elemento `.offline-banner` fica oculto por padrão e é exibido pelo `app.js`
quando `navigator.onLine` é falso. Ele não bloqueia a página e desaparece ao
voltar a conexão.

## Padrões por fluxo

### Início

- `h1` com o nome da instância e uma frase curta sobre o propósito.
- Ações principais: entrar/criar conta ou conta/perfil.
- Links de estado (`/healthz`, `/readyz`) ficam no rodapé, não no herói.

### Diário

- Cabeçalho com título, descrição e ações “Registrar cultivo” e “Meu diário”.
- Filtros em `.filters`, com botão “Filtrar”.
- Ciclos em `.cards`; cada cartão mostra espécie, ambiente, estágio, autoria e
  número de registros.
- O detalhe do ciclo usa `.page-head`, painéis de contexto e `.timeline`.
- O formulário de registro mantém fotos, medidas e visibilidade no mesmo fluxo.

### Práticas

- Biblioteca em `.cards`, com resumo, categoria, dificuldade e autoria.
- O detalhe usa `.practice-body`, `.lead`, `.steps` e avisos em `.alert-warning`.
- Rascunhos e arquivadas exibem alerta informativo e ações de publicação.
- O formulário separa materiais, etapas, segurança, adaptações e licença.

### Perfil e conta

- Perfil público começa com identificador, nome e biografia.
- Estatísticas de seguidores são links para as listas.
- Ações de seguir/deixar de seguir ficam junto ao cabeçalho do perfil.
- A conta reúne dados cadastrais e atalhos para perfil e seguidores.

### Administração

- Configuração de cadastros em formulário `.stack`.
- Contas pendentes em `.table-scroll`, com ações explícitas.
- Em telas estreitas, a tabela rola em vez de comprimir colunas.

### Offline

- Título “Sem conexão”.
- Explicação curta de que a página não pôde ser carregada.
- Botão “Tentar novamente” apontando para `/` ou para a página anterior.
- Nenhum dado privado ou específico de sessão.

## Acessibilidade

- Todo controle tem rótulo visível ou `aria-label`.
- Foco visível com contorno de 2 px e deslocamento de 2 px.
- Link “Pular para o conteúdo” é o primeiro elemento focável.
- A ordem de tabulação segue a ordem visual.
- Ícones decorativos usam `aria-hidden="true"`; ícones com significado têm texto.
- Contraste mínimo de 4.5:1 para texto e 3:1 para componentes.
- Respeite `prefers-reduced-motion`; animações são curtas e opcionais.

## PWA

- Manifest em `/manifest.webmanifest`; service worker em `/sw.js`.
- Ícones: 192×192, 512×512, 512×512 maskable e `apple-touch-icon` 180×180.
- `start_url` e `scope` são `/`.
- O service worker pré-cacheia apenas `/offline`, CSS, JS, manifest e ícones.
- Navegação é rede-primeiro, com fallback para `/offline`.
- HTML autenticado nunca entra no cache.
- A versão dos assets é um hash do conteúdo; `?v=<hash>` aparece no CSS e no JS.

## Checklist para novas telas

- [ ] A tela funciona sem JavaScript.
- [ ] O título principal é um `h1` e a hierarquia usa `h2`/`h3` em ordem.
- [ ] Todos os campos têm rótulo e texto de ajuda quando necessário.
- [ ] As ações principais têm área de toque de pelo menos 44 px.
- [ ] O layout foi conferido em 360 px, 768 px e 1280 px.
- [ ] O tema claro e o escuro foram conferidos.
- [ ] Estados vazio, erro, sucesso e carregamento estão previstos.
- [ ] Nenhuma informação depende apenas de cor.
- [ ] Nenhum recurso externo foi adicionado.
- [ ] Classes existentes foram preservadas ou o CSS foi atualizado junto.
