# ADR 0013 — Experiência mobile, PWA e redesenho da interface

**Status:** Proposto
**Data:** 2026-11-19

## Contexto

A plataforma entrou em operação com as jornadas de identidade, diário de cultivo,
biblioteca de práticas e federação implementadas. A interface, no entanto, foi
construída incrementalmente: um único `app.css` sem breakpoints explícitos, sem
JavaScript, sem manifest e sem service worker. O resultado funciona em telas
grandes, mas não oferece uma experiência adequada ao uso no campo, quase sempre
feito pelo celular.

O cultivo acontece na varanda, no quintal e na horta. Registrar uma observação
ou consultar uma prática exige poucos toques, boa legibilidade sob luz natural e
tolerância a conexões instáveis. A interface atual também não pode ser instalada
na tela inicial e não oferece nenhuma resposta quando a rede falha.

Restrições consideradas:

- O frontend é renderizado no servidor com templates Go, conforme o ADR 0007.
  Não há SPA, bundler ou etapa de build de frontend.
- JavaScript deve ser mínimo e progressivo: todas as jornadas essenciais
  continuam funcionando sem ele.
- A federação ActivityPub e as regras de visibilidade não podem ser alteradas
  por esta decisão.
- A instância é operada com Nginx e um único binário Go; a solução não pode
  exigir infraestrutura adicional.
- Documentação, comentários e interface seguem em português.

## Decisão

### 1. PWA instalável, com aprimoramento progressivo

A aplicação passa a ser uma **Progressive Web App** instalável, sem deixar de ser
um site renderizado no servidor. A camada PWA é composta por:

| Peça | Papel |
| --- | --- |
| `manifest.webmanifest` | Nome, ícones, cores, `display: standalone`, `start_url` e atalhos. |
| `/sw.js` | Service worker na raiz, com escopo para toda a aplicação. |
| `/offline` | Página de fallback quando uma navegação falha sem rede. |
| `app.js` | Registro do service worker e pequenos aprimoramentos de interface. |

O manifest é servido em `/manifest.webmanifest` e o service worker em `/sw.js`,
fora de `/static`, porque o escopo do service worker é determinado pelo caminho
do script. Os ícones ficam em `/static/icons/`.

O aplicativo usa `display: standalone`, `theme_color` verde e `background_color`
clara, com variação para o modo escuro. A instalação é oferecida pelo navegador;
a interface não cria banners próprios nem bloqueia o uso até a instalação.

### 2. Estratégia de cache e offline

O service worker pré-carrega apenas recursos públicos e estáveis:

- a página `/offline`;
- `app.css`, `app.js` e o manifest;
- os ícones da PWA.

Navegações usam **rede primeiro**. Se a rede falhar, o service worker responde
com `/offline`. Nenhuma página autenticada, resposta de formulário, mídia privada
ou conteúdo federado é armazenado em cache. Isso evita vazar conteúdo entre
sessões e mantém as regras de visibilidade sob controle do servidor.

Recursos estáticos usam **cache primeiro**, com atualização em segundo plano
quando possível. Como os arquivos são versionados por hash, uma nova versão do
binário invalida naturalmente as referências antigas.

O cache do service worker é nomeado com a versão dos assets. Na ativação, caches
de versões anteriores são removidos.

### 3. Cache-busting por conteúdo

A versão exibida em produção é `dev`, porque o binário é compilado sem uma tag
de release. Por isso, o cache-busting não pode depender de `version.Version`.

O servidor calcula, na inicialização, um hash SHA-256 do conteúdo embutido em
`internal/web/static` e o expõe como `AssetVersion`. As referências a `app.css` e
`app.js` incluem `?v=<hash>`, e o service worker recebe o mesmo valor ao ser
servido. Qualquer mudança em um asset muda o hash e força a atualização.

### 4. Redesenho mobile-first com design system próprio

A interface é reescrita em torno de uma fundação compartilhada, sem framework
CSS e sem dependências externas:

- **Tokens** de cor, tipografia, espaçamento, raio, sombra e toque.
- **Tema claro e escuro** via `prefers-color-scheme`, com contraste adequado.
- **Tipografia**: serifada do sistema para títulos, sans-serif do sistema para
  texto corrido. Nenhuma fonte é baixada.
- **Componentes base**: botões, campos, cartões, alertas, etiquetas, linha do
  tempo, tabelas roláveis e estados vazios.
- **Layout fluido** com `min()`, `clamp()` e grid responsivo, sem breakpoints
  rígidos desnecessários.

O CSS mantém as classes existentes nos templates. A reescrita é visual e
estrutural, não uma renomeação em massa, para reduzir o risco de regressão.

### 5. Navegação adaptada ao dispositivo

No celular, a navegação principal fica em uma **barra inferior fixa** com cinco
destinos: **Início**, **Diário**, **Práticas**, **Perfil** e **Mais**. A barra
respeita `env(safe-area-inset-bottom)` e usa áreas de toque de pelo menos 44 px.

O item **Mais** abre um painel com conta, seguidores, administração (quando
aplicável), saída e links de estado da instância. O painel usa o elemento
`<details>`, portanto funciona sem JavaScript; o script apenas melhora o
comportamento de fechar ao clicar fora ou pressionar `Esc`.

No desktop, a navegação continua no cabeçalho. A barra inferior e o painel
**Mais** ficam ocultos a partir de 721 px.

### 6. Página offline e estados de rede

`/offline` é uma página renderizada pelo servidor, sem dados privados, com uma
mensagem curta e um botão para tentar novamente. Ela é pré-cacheada e exibida
somente quando uma navegação falha.

O `app.js` também alterna um aviso discreto de “sem conexão” quando o navegador
fica offline, sem impedir o uso do conteúdo já carregado.

### 7. Câmera e fotos

O envio de fotos continua usando `<input type="file" accept="image/jpeg,image/png,image/gif" multiple>`.
Em navegadores móveis, esse campo já oferece a câmera como origem. A aplicação
não força `capture`, porque isso impediria escolher imagens da galeria e enviar
mais de uma foto por registro.

O redesenho deixa a ação de adicionar fotos visível e explica os limites antes do
envio. A validação, a remoção de metadados e o limite de quatro imagens continuam
no servidor.

### 8. Política de segurança de conteúdo

A CSP passa a declarar explicitamente:

```
default-src 'self';
img-src 'self' data: https:;
style-src 'self';
script-src 'self';
worker-src 'self';
manifest-src 'self';
base-uri 'self';
form-action 'self'
```

`worker-src` é necessário para registrar `/sw.js` e `manifest-src` para carregar
o manifest. Não há scripts ou estilos inline; o `app.js` é um arquivo externo.

## Alternativas consideradas

### 1. Aplicativo nativo

**Vantagens:**
- Acesso a câmera, notificações e armazenamento com mais controle.
- Presença em lojas de aplicativos.

**Desvantagens:**
- Duplica a interface e a manutenção em pelo menos duas plataformas.
- Exige processo de publicação, assinatura e atualização independente.
- O ganho atual não justifica o custo; o uso principal é registrar texto e fotos.

A opção fica reservada para quando houver necessidade comprovada de notificações
push e uso offline intenso.

### 2. SPA com framework JavaScript

**Vantagens:**
- Transições ricas e estado de cliente mais elaborado.
- Ecossistema amplo de componentes.

**Desvantagens:**
- Contraria o ADR 0007 e adiciona uma etapa de build.
- Aumenta o tempo até o primeiro conteúdo em conexões ruins.
- Duplica validações e regras de visibilidade no cliente.

### 3. Manter a interface atual e adicionar apenas um manifest

**Vantagens:**
- Menor esforço inicial.

**Desvantagens:**
- Não resolve a navegação ruim no celular, a legibilidade e os alvos de toque.
- A instalação levaria a uma experiência que não parece um aplicativo.
- O débito visual continuaria crescendo com as próximas jornadas.

## Consequências

- A aplicação passa a ser instalável e a oferecer uma resposta digna sem rede.
- A experiência no celular ganha navegação fixa, tipografia maior e alvos de
  toque adequados.
- O CSS e o shell global são reescritos, mas as classes e os templates existentes
  são preservados para limitar regressões.
- O servidor passa a expor `/manifest.webmanifest`, `/sw.js` e `/offline`, além
  de calcular a versão dos assets na inicialização.
- A CSP fica mais restritiva e explícita; qualquer script futuro precisa ser
  externo e servido pela própria origem.
- O service worker não cacheia HTML autenticado. Conteúdo já carregado continua
  visível enquanto a aba estiver aberta, mas recarregar sem rede mostra
  `/offline`.
- Novos assets estáticos entram automaticamente no hash de versão.
- A documentação de desenvolvimento precisa listar as novas rotas e a operação
  do cache.

## Escopo e decisões futuras

Esta decisão **não** determina:

- notificações push;
- sincronização em segundo plano de registros criados offline;
- upload offline de fotos;
- aplicativo nativo;
- internacionalização da interface;
- migração para Tailwind ou outro framework CSS.

Esses temas devem ser tratados em decisões próprias, com medição de uso real.

## Referências

- [ADR 0007 — Stack da aplicação](0007-stack-da-aplicacao.md)
- [ADR 0011 — Diário de cultivo e biblioteca de práticas](0011-diario-de-cultivo-e-biblioteca-de-praticas.md)
- [MDN — Progressive web apps](https://developer.mozilla.org/pt-BR/docs/Web/Progressive_web_apps)
- [web.dev — Service worker caching](https://web.dev/articles/service-worker-caching-and-http-caching)
