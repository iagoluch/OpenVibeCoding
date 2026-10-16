# Contrato visual — réplica do TUI do OpenCode (v2.0.29 / v2.0.31)

> **Status:** FASE 1 (extração/referência) — nenhuma UI implementada.
> **Consumidor:** UI web do OVC servida pelo daemon Go (`go:embed` + stdlib, **zero npm**).
> **Escopo:** aparência. Microcopy permanece em pt-BR (o clone é de *aparência*, não de idioma).
> **Data da extração:** 16/10/2026.

Este documento é a referência visual canônica. Ele **descreve** o alvo; não implementa, não altera produto e não substitui `opencode-ui-inventory.md` (inventário funcional, em inglês) — os dois se complementam: o inventário diz *o que existe*, este contrato diz *como se parece*.

---

## 0. Conceitos que evitam erro de alvo

| Fato | Evidência |
| --- | --- |
| O app em uso é **2.0.29**; o binário npm instalado é **2.0.31** | `~/.local/state/opencode/service.json` → `"version": "2.0.29"`; `opencode.exe` → `v2.0.31` |
| **Tema efetivo = `dracula`, modo `dark`** | `~/.config/opencode/cli.json` → `theme: {mode:"dark", name:"dracula"}`; aplicado em `context/theme.tsx:139` e `:144-149` |
| `~/.config/opencode/tui.json` (`"theme":"dark"`, `plugin: orca-…`) é **legado e inerte** em v2 | v2 lê `cli.json`+`opencode.json`; `tui.json` só alimenta a migração one-time (`packages/cli/src/config/migrate.ts`) |
| `~/.local/state/opencode/kv.json` diz `"theme":"github"` — **não vale** | arquivo de 05/10 (pré-migração); `context/theme.tsx:133-149` sobrescreve `active` com `config.theme.name` |
| O plugin `…/plugins/orca-opencode-status-tui/tui.js` **não existe no disco** e não contribui com nada | `Test-Path` = False; nenhum `Plugin.define` de TUI em `~/.config/opencode` (só o plugin *server* da workforce, em `.bak`) |
| A faixa inferior com `Server / UI / Theme / Tools / Experiments` **é do core**, não de plugin | `app.tsx:1429-1431` → `<DevToolsBar/>`; rótulos em `component/devtools-bar.tsx:245,294,341,357,429`; ligada por `cli.json.debug.devtools: true` |
| A moldura de texto com borda esquerda coral é a **mensagem do usuário** | `routes/session/index.tsx:2244-2249` (`UserMessage`) |
| O chrome da janela (`OC | …` + botões) é do **terminal**, não do TUI | ver §9.2 |

**Diferença .29 → .31 nos componentes observados:** nenhuma que mude aparência. `component/devtools-bar.tsx`, `component/session-tabs*.tsx`, `logo.ts*`, `routes/home.tsx`, `ui/layout.ts` (só +3 linhas) e `component/prompt/index.tsx` (só um toast de erro) são idênticos ou irrelevantes; `feature-plugins/prompt/footer.tsx` muda apenas para *esmaecer* as dicas quando um painel tem foco. Portanto: **o contrato vale para as duas versões**; qualquer citação é de `v2.0.31` (tree igual à observada nos pontos citados).

**Mapa dos estados observados** (prints do usuário, 2 telas: A = conversa ativa, B = tela inicial) — cada elemento visto → seção deste contrato:

| Observado na tela | Onde está | Comentário |
| --- | --- | --- |
| Sidebar esquerda ~255 px com marcador + título `OpenVibeCoding: análise e classificação R0-R3` | §6 | trilha vertical = 42 colunas (`ui/layout.ts`); 42 col × ~6,1 px ≈ 255 px |
| `iago.luchtenberg` na sidebar | §6 | cabeçalho de grupo = `path.basename(directory)` (`util/project.ts:3-7`); confere com `latest/tui/tabs.json` (`cwd: C:\Users\iago.luchtenberg`) |
| `+ New session` na sidebar | §6 | `NEW_SESSION_TAB_TITLE` (`context/session-tabs-model.ts:8`), item `+` no modo compacto |
| `~` | §6 / §8.2 | rótulo de local abreviado (`abbreviateHome`), `component/prompt/index.tsx:1615-1622` |
| Painel central (home vazio / conversa) | §4, §5, §7 | — |
| Faixa inferior de largura total: `Server  UI  Theme  Tools  Experiments` | §9.1 | core + `debug.devtools: on`; ver divergência de cor em §12.1 |
| Wordmark pixelado `open` (lavanda) + `code` (branco) | §5.1 | valida o tema `dracula` |
| `/exit restart to use v2.0.31` | §5.2 | `routes/home.tsx:158-169` |
| Caixa de input com placeholder `ask anything… "Fix a TODO in the codebase"` | §5.3, §8.2 | `component/prompt/index.tsx:1601` |
| `Orchestrator auto · Nemotron 3.5 Super VL Free OpenCode · xhigh` (parte coral) | §8.3 | coral = `categorical[5][200]` = `#ffb86c` (cor do agente) |
| `shift+tab agents  ctrl+p commands` | §8.2 | `feature-plugins/prompt/footer.tsx:99-120` |
| `7 MCP /mcps` (canto inferior esquerdo) | §5.4 | `feature-plugins/home/footer.tsx:38-45` |
| `2.0.29` (canto inferior direito) | §5.4 | `app.version`, só com `width ≥ 64`; confere com `service.json` |
| Texto do assistente com títulos/bullets | §7.2 | markdown renderizado |
| `+ 3 commands, 7 thoughts, 4 tools` | §7.3 | `group-view.tsx:211-222` + `activity-summary.ts:61-95` |
| Bloco com moldura coral + fundo levemente tingido | §7.1 | **é a mensagem do usuário** |
| `↓ 2 subagents · 81.2K (41%) ctrl+p commands` | §8.2 | `prompt/footer.tsx:69-120` + `context/keymap.tsx:463` |
| `OC | OpenVibeCoding: análise e classificaçã…` no topo | §9.2 | chrome do terminal; fora de escopo |

---

## 1. Método de extração (como reconferir)

```powershell
# clone raso na tag, apenas os pacotes que importam
git clone --depth 1 --filter=blob:none --sparse https://github.com/anomalyco/opencode.git opencode-src
cd opencode-src; git sparse-checkout set packages; git fetch --depth=1 origin tag v2.0.29
git diff --name-status v2.0.29 v2.0.31 -- packages/tui/src packages/theme/src
```

A paleta **não** foi copiada à mão: o pipeline de tema (`packages/theme/src/tui/{v1-migrate,resolve,expand,color}.ts` + `packages/tui/src/theme/index.ts`) foi portado para Node e validado contra os asserts de `packages/tui/test/theme/v2/v1-migrate.test.ts` — **36 passed / 0 failed**. Saídas reprodutíveis:

| Artefato | Caminho |
| --- | --- |
| Dump completo do tema resolvido (dracula dark) | `docs/research/assets/dracula-resolved.json` |
| Dump do tema v2 default (`opencode`, fallback) | `…/Temp/opencode/opencode-v2-dump.txt` |
| Port fiel do pipeline | `…/Temp/opencode/theme-resolve.mjs`, `theme-lib.mjs` |
| Cores derivadas (tint/decrease) | `…/Temp/opencode/dracula-extra.txt` |

Toda cor da §2/§3 saiu desse dump; toda medida/glifo da §5–§8 saiu do fonte, com arquivo:linha.

---

## 2. Paleta — dracula / dark (valores finais, prontos para CSS)

Como o tema `dracula` é um asset **v1** (`packages/tui/src/theme/assets/dracula.json`) migrado em runtime para o schema v2, os nomes de papel abaixo são os nomes v2 (`ResolvedTheme`).

### 2.1 Base

| Papel | Hex | Papel | Hex |
| --- | --- | --- | --- |
| `text.base` | `#f8f8f2` | `background.base` | `#282a36` |
| `text.muted` | `#6272a4` | `background.raised.base` | `#21222c` |
| `text.faint` | `#6272a4` | `background.raised.high` | `#44475a` |
| `border.base` | `#44475a` | `background.raised.max` | `#545c7e` |
| `scrollbar.base` | `#bd93f9` | `background.action.primary.base` | `transparent` |
| `background.action.primary.hovered` | `#21222c` | `background.action.primary.focused` | `#bd93f9` |

### 2.2 Feedback / ações

| Papel | Hex |
| --- | --- |
| `text.feedback.error / warning / success / info` | `#ff5555` / `#f1fa8c` / `#50fa7b` / `#ffb86c` |
| `background.feedback.*.base` (todos) | `#282a36` |
| `text.action.primary`: `base`·`hovered`·`pressed` | `#f8f8f2` |
| `text.action.primary`: `disabled` | `#6272a4` |
| `text.action.primary`: `selected` | `#bd93f9` |
| `text.action.primary`: `focused` | `#282a36` |
| `text.action.secondary.*` (`base`/`disabled`/`pressed`/`focused`/`selected`) | `#6272a4`; `hovered` = `#f8f8f2` |
| `text.action.destructive.*` | `#282a36` (com `disabled` = `#6272a4`) |

### 2.3 Diff / sintaxe / markdown

| Papel | Hex |
| --- | --- |
| `diff.added`/`removed` (fundo) | `#1a3a1a` / `#3a1a1a` |
| `syntax.keyword` / `operator` | `#ff79c6` |
| `syntax.function` | `#50fa7b` |
| `syntax.string` | `#f1fa8c` |
| `syntax.number` | `#bd93f9` |
| `syntax.type` | `#8be9fd` |
| `markdown.heading` | `#bd93f9` |
| `markdown.code` | `#50fa7b` |
| `markdown.emphasis` | `#f1fa8c` |
| `markdown.strong` | `#ffb86c` |

> Não existe hue `blue` neste tema. `categorical` = `["red","cyan","green","yellow","purple","orange"]`.

### 2.4 Escalas de hue (100 → 900) — usadas para *derivar* cores

```
gray/neutral  #ffffff #f8f8f2 #94bdb9 #6272a4 #545c7e #44475a #21222c #282a36 #2f3240
accent/cyan   #f0fcff #8be9fd #71c7d9 #58a6b6 #408695 #286874 #104b55 #012f37 #00151a
interactive   #d1b5fe #bd93f9 #a37cd9 #8966b9 #70519a #583d7d #422961 #2c1745 #18062c
red           #fe8781 #ff5555 #df4445 #c03436 #a12328 #84111a #630b11 #450408 #290203
green         #e0ffe3 #50fa7b #37d663 #19b44c #1c903f #166f2f #0a501f #053211 #021705
yellow        #ffffff #f1fa8c #cdd571 #aab056 #888e3c #676c22 #494c05 #2c2e02 #121300
orange        #fde2c7 #ffb86c #dc9c56 #ba8141 #99662d #794d18 #5a3502 #3a2102 #1c0e01
```

`text.base` = `$hue.neutral.200`, `text.muted` = `$hue.neutral.400`, `background.raised.base` = `$hue.neutral.700`, `background.base` = `$hue.neutral.800` (confirmado por `source(color)`).

### 2.5 Cores **derivadas** (não estão na paleta; calcular, não adivinhar)

Semântica (`packages/theme/src/tui/resolve.ts:172-194`):
`increase/decrease(cor, n)` só funciona se a cor for exatamente um passo de alguma escala — passo `position±n`, **clampado** nos extremos; se a cor não estiver indexada, devolve a própria cor.
`tint(base, overlay, a)` = interpolação linear por canal (`tui/src/theme/color.ts:43-48`).

| Uso | Cálculo | Valor (dracula dark) | Fonte |
| --- | --- | --- | --- |
| Faixa DevTools (fundo) | `decrease(background.base)` | `#21222c` | `devtools-bar.tsx:232` |
| Fundo da mensagem do usuário | `background.raised.base` | `#21222c` | `routes/session/index.tsx:2248` |
| Hover da mensagem do usuário | `decrease(background.raised.base)` | `#44475a` | `:2289` |
| Borda do bloco `execute` expandido | `decrease(text.muted)` | `#94bdb9` | `:3187,3213` |
| Borda esquerda do composer | `tint(border.base, <cor do agente>, α)` | ver §8.2 | `component/prompt/index.tsx:1587` |
| Sombra do logo (esq.) | `tint(background.base, text.muted, .25)` | `#373c52` | `component/logo.tsx` |
| Sombra do logo (dir.) | `tint(background.base, text.base, .25)` | `#5c5e65` | idem |
| Fundo de destaque de tab (hover) | `tint(raised.base, raised.high, raised.high.a)` | ≈ `#44475a` | `session-tabs-rail.tsx:16-18` |

---

## 3. Tipografia e grade

- **A TUI é uma grade de células.** Tudo é múltiplo de 1 célula/1 linha; nada de px fracionário.
- **Fonte:** nenhum override no Windows Terminal do usuário (`…\Packages\Microsoft.WindowsTerminal_8wekyb3d8bbwe\LocalState\settings.json`: `profiles.defaults {}`, `schemes: []`, `themes: []`) → **Cascadia Mono** (padrão do WT) — **[Provável]**.
- **Consequência para o OVC:** usar a stack mono do sistema (`ui-monospace, "Cascadia Mono", "Cascadia Code", Consolas, "DejaVu Sans Mono", monospace`) — **não bundlar fonte**. O visual depende do avanço fixo (1 `ch`) e de os glifos `█ ▀ ▄ ┃ ▌ ● ○ ▪ ◉ ⌕ ↓ · ⊙ ✕ ‹ › ⠋…` existirem na fonte.
- **Sem negrito sintético:** `bold` do TUI = `font-weight: 700` (o logo `right` e o `variant` do metadata usam bold; o resto não).
- Largura de referência observada: sidebar ≈ **255 px** para 42 colunas → ~6,1 px/célula (Cascadia Mono ~11 pt em 100% de zoom). No browser, derive o tamanho da fonte do container, não fixe px.

---

## 4. Geometria / layout do app

Constantes (`ui/layout.ts`): `SESSION_SIDEBAR_WIDTH=42`, `SESSION_TABS_COMPACT_WIDTH=5`, `SESSION_TABS_COMPACT_BREAKPOINT=12`, `SESSION_SIDEBAR_MAX_WIDTH=72`, `SESSION_CONTENT_MIN_WIDTH=44`, `SESSION_CONTENT_PREFERRED_WIDTH=64`.
Largura útil do painel central: `contentWidth = (width - verticalTabsWidth) - 4` (`routes/session/index.tsx:255`).

Esqueleto (de cima para baixo, `app.tsx:1395-1431`):

```
┌───────────────────────────────────────────────────────────────┐
│ sidebar vertical (42 col)   │  painel central (transcript /   │  ← flexDirection: row
│ = trilha de sessões         │  home / plugin)                 │
├───────────────────────────────────────────────────────────────┤
│ <Slot path="app"/>  (conteúdo de plugin)                      │
├───────────────────────────────────────────────────────────────┤
│ DevToolsBar — 1 linha, 100% de largura, fundo decrease(bg)    │  ← só se debug.devtools
└───────────────────────────────────────────────────────────────┘
```

Breakpoints que mudam a aparência (todos verificados no fonte):

| Condição | Efeito |
| --- | --- |
| `height < 12` | logo do splash não é desenhado |
| `width < 22` | só o fragmento direito do wordmark (bold) |
| `width < 44` | wordmark esquerdo (muted) + metade direita do logo; paddings do home caem de 2 → 1 |
| `width < 44` | metadata do prompt esconde agente e provider; footer do prompt mostra `esc shell` |
| `width ≥ 64` | footer do home mostra `/mcps` e a versão |
| `width ≥ 80` | footer do home mostra `/plugins` |
| `sidebar width ≤ 12` | trilha entra em modo compacto (5 col, só indicadores) |

---

## 5. Splash / tela inicial

Ordem vertical (`routes/home.tsx:83-108`): `flexGrow` → **3 linhas vazias** → logo → **1 linha vazia** → aviso de update → caixa de prompt (`maxWidth: 75`, `paddingTop: 1`) → `flexGrow` → footer do home (1 linha) → DevToolsBar.

### 5.1 Wordmark

Arte literal — **`packages/tui/src/logo.ts`** (objeto com `left`/`right`/`go`), 4 linhas × 21 colunas por metade; render lado a lado com **1 coluna de gap** (regras em **`packages/tui/src/component/logo.tsx`**):

```
esquerda: ["                   ", "█▀▀█ █▀▀█ █▀▀█ █▀▀▄", "█__█ █__█ █^^^ █__█", "▀▀▀▀ █▀▀▀ ▀▀▀▀ ▀~~▀"]
direita:  ["             ▄     ", "█▀▀▀ █▀▀█ █▀▀█ █▀▀█", "█___ █__█ █__█ █^^^", "▀▀▀▀ ▀▀▀▀ ▀▀▀▀ ▀▀▀▀"]
```

Regras de pintura — `packages/tui/src/component/logo.tsx`, `packages/tui/src/util/presentation.ts:13` (mesma tabela aplicada ao logo):

| Caractere | Vira | Cor |
| --- | --- | --- |
| `_` | espaço | **fundo** = `shadow` |
| `^` | `▀` | **frente** = cor da metade |
| `~` | `▀` | **frente** = `shadow` |
| `,` | `▄` | **frente** = `shadow` |
| `█ ▀ ▄` | como está | frente = cor da metade |

`shadow = tint(theme.background.base, <cor da metade>, 0.25)` → **esquerda** `#373c52`, **direita** `#5c5e65`.
Metade esquerda: `text.muted` (`#6272a4`), sem bold. Metade direita: `text.base` (`#f8f8f2`), **bold**.
→ Resultado observado: “open” lavanda-azulado, “code” branco. É exatamente isso que valida o tema `dracula` (§0).

### 5.2 Aviso de versão (abaixo do logo, centralizado na largura do logo)

`/exit` em `text.action.primary.selected` (`#bd93f9`) + `" restart to use v2.0.31"` em `text.muted` (`#6272a4`), com `gap={1}`; hover troca o fundo para `background.action.primary.hovered` (`#21222c`). Texto literal: `` ` restart to use v${version}` `` / `` ` to install v${version}` `` / `remote server update available` (`routes/home.tsx:158-169`).
Quando o aviso está presente, o conjunto logo+aviso é a “largura do logo” usada para centralizar.

### 5.3 Caixa de prompt (mesma do §8.2)

Placeholder exato: `` `Ask anything… "${sugestão}"` `` (`component/prompt/index.tsx:1601`), em `text.muted`; sugestões do home: `"Fix a TODO in the codebase"`, `"What is the tech stack of this project?"`, `"Fix broken tests"`. Cursor de bloco (cursor nativo do terminal).
Abaixo: **metadata row** (§8.3) e, na linha seguinte, **footer** (§5.4).

### 5.4 Footer do home (1 linha, `feature-plugins/home/footer.tsx:83-102`)

`paddingLeft/Right = 2`, `paddingTop/Bottom = 1` (0 se altura < 16), `gap = 2`; só existe se `height ≥ 14 && width ≥ 44`.

```
[⊙ 7 MCP  /mcps]  [⊙ N plugins failed /plugins]   …espaço…              [2.0.31]
 └── "⊙ " verde se conectado; cinza se 0; ⊙ vermelho + "N MCP failed" se falhou
```
- contagem = MCPs `connected` (`:20,38-40`); `/mcps` em `text.muted`, só se `width ≥ 64`;
- versão em `text.muted`, só se `width ≥ 64`;
- o `~` visto à esquerda **não** é do footer do home: é o rótulo de **local**, que vive no prompt (`~` = home abreviado, `component/prompt/index.tsx:1615-1622`).

---

## 6. Trilha de sessões (a “sidebar”, 42 colunas)

`component/session-tabs.tsx` + `session-tabs-rail.tsx` + `context/session-tabs.tsx`. Ligada por `cli.json.tabs = {mode:"on", scope:"global", layout:"vertical", indicators:"status"}`.

Estado real desta máquina (`.local/state/opencode/latest/tui/tabs.json`):

```json
{"global":{"tabs":[{"sessionID":"ses_ebf88…","title":"OpenVibeCoding: análise e classificação R0-R3"}]},
 "cwd":{"C:\\Users\\iago.luchtenberg":{"tabs":[{"sessionID":"ses_ebf91…","title":"Instalar repositório global opencode"}]}}}
```

- **Cabeçalho de grupo** = `projectName(project, directory)` → nome do projeto **ou** `basename(directory)` (`util/project.ts:3-7`) → é isso que produz o rótulo `iago.luchtenberg`.
- **Item de sessão:** título (fade de 4 células no fim de títulos longos, `FADE_WIDTH=4`), indicador à esquerda com `width=10`, e `✕` no hover dentro de 1 célula da borda direita (`session-tabs.tsx:1109`).
- **Item `+`/“New session”:** `NEW_SESSION_TAB_TITLE = "New session"` (`context/session-tabs-model.ts:8`); na variante compacta o rótulo é apenas `+` (`session-tabs.tsx:1700`), `×` no hover (`:1233`).
- **Controle de busca:** linha de 1 célula com `⌕` centralizado (`session-tabs-rail.tsx:50-52`), fundo `background.raised.base`, hover `background.raised.high`; meia-linha `▄`/`▀` desenha a emenda (`:65,91`).

Marcadores e spinners (constantes exatas):

| | Valores |
| --- | --- |
| Não lidos (`TAB_UNREAD_MARKERS`) | `small-dot: "•"`, `dot: "●"`, `square: "▪"`, `large-square: "■"` |
| Spinners (`TAB_SPINNERS`) | `dots` = `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏` @80ms; `arcs` = `◜◝◞◟` @120ms; `quadrants` = `◴◷◶◵` @120ms; `line` = `|/-\` @120ms |
| Estados | rodando → 1º frame do spinner; permissão → `!`; pergunta → `?` (`:176-178`) |
| Cor de feedback da tab | atenção → `hue.accent[200]` = `#8be9fd`; não lido com erro → `text.feedback.error.base` = `#ff5555` (`:127-129`) |

---

## 7. Transcript

### 7.1 Mensagem do usuário — **é a moldura coral**

`packages/tui/src/routes/session/index.tsx:2242-2344` — função `UserMessage`. A moldura é literalmente este `<box>`, linha por linha:

| Linha | Atributo |
| --- | --- |
| `:2245` | `border={["left"]}` — **só a borda esquerda**, 1 célula |
| `:2246` | `borderColor={delivery() ? theme.border.base : color()}` — `color()` = cor do agente |
| `:2247` | `customBorderChars={SplitBorder.customBorderChars}` → `{...EmptyBorder, vertical: "┃"}` (`ui/border.ts:15-21`) |
| `:2248` | `backgroundColor={theme.background.raised.base}` → `#21222c` |
| `:2286-2288` | `paddingTop/Bottom 1`, `paddingLeft 2` |
| `:2289` | hover → `decrease(background.raised.base)` = `#44475a` |
- Logo: **borda esquerda de 1 célula em `┃`** na **cor do agente ativo**, fundo `#21222c` (levemente mais escuro que a página `#282a36` → “fundo levemente tingido”).
- Cor do agente = `categorical[i][200]` (`context/local.tsx:77-84,128-134`) → no dracula: `red #ff5555`, `cyan #8be9fd`, `green #50fa7b`, `yellow #f1fa8c`, `purple #bd93f9`, **`orange #ffb86c`** — o tom coral das prints vem daqui (agente `Orchestrator`).
- Texto (`paddingTop/Bottom 1`, `paddingLeft 2`) em `text.base`; fundo muda para `decrease(raised.base)` = `#44475a` no hover.
- Chips (skills/arquivos) dentro da mensagem: `" skill "` / `" file "` / `" dir "` com fundo `hue.accent[200]` = `#8be9fd`, texto `background.raised.base`, **bold**; o valor (`nome`) com fundo `decrease(raised.base)` = `#44475a` e texto `text.muted`.
- Antes do texto vem a faixa de imagens, se houver (`SessionImages`, `paddingLeft 2`).

### 7.2 Texto do assistente

Markdown renderizado (config `session.markdown: "rendered"`): títulos em `markdown.heading` (`#bd93f9`) bold, corpo em `text.base`, `code` em `#50fa7b`, ênfase `#f1fa8c`, forte `#ffb86c`; listas com `-`/`*` e indentação de 2 células; largura de quebra = largura útil do painel.

### 7.3 Linha de atividade colapsada (`+ 3 commands, 7 thoughts, 4 tools`)

`routes/session/group-view.tsx:201-231` + `routes/session/activity-summary.ts:61-95`:

- ícone: `+` fechado, `−` aberto (`:212`); cor `text.muted`, hover `text.base`; spinner do grupo quando ativo.
- rótulo: contagens na ordem `command, edit, thought, read, tool, instruction`, pluralizadas, separadas por `", "` — ex.: `3 commands, 1 edit, 2 thoughts, 4 reads`.
- `⌥` Instruction group: ícone fixo `◈` e texto `Instructions: N file[s]` (`:250-258`).

### 7.4 Linhas de ferramenta (inline)

- Largura do ícone: `INLINE_TOOL_ICON_WIDTH = 2` (`routes/session/message-parts.tsx:19`); padding esquerdo 3 no bloco `execute` (`index.tsx:3196`).
- Glifos observados no fonte: `✓` concluído, `✗` erro, `│` pendente, `›` chamada de código, `+`/`−` grupo, `◈` instruções.
- Nomes canônicos exibidos: `shell → command`, `execute → code`, `task → subagent` (`util/tool-display.ts:3`, `activity-summary.ts:50`).
- Estado ocupado: `Thinking…`, `Preparing <noun>…`, `Running <noun>…` (`activity-summary.ts:47-52`).
- Saída longa é colapsada e expandida por clique; `execute` expandido usa a borda `#94bdb9` (§2.5).

---

## 8. Composer (caixa de entrada) e rodapés

### 8.1 Borda

- Contorno superior: `borderColor = tint(border.base, <cor do agente>, α)` (`prompt/index.tsx:1587`) → com `dracula` e agente coral, um `┃`/borda de tom acobreado suave sobre `#44475a`.
- A borda inferior é fechada por `▀` de meia célula quando há fundo (`:1864-1873`), então a caixa parece “sólida” na base.
- Cursor: cor `text.base`, estilo conforme `config.cursor`; fundo focado `transparent` (`:1812,1830-1831`).

### 8.2 Placeholder e dicas

- `Ask anything… "<sugestão>"` em `text.muted` (`:1594-1606`), truncado ao `maxWidth`.
- Linha final do composer, `justifyContent: space-between` (`:1877-1961`):
  - **esquerda** — rótulo de local: `abbreviateHome(dir, home)` + `:branch` quando houver (`:1615-1622`) → ex. `~`, `~/proj:main`; em `text.muted` (`text.base` no hover).
  - **direita** — `feature-plugins/prompt/footer.tsx`:
    - com trabalho vivo: `↓ 2 subagents · 81.2K (41%) ctrl+p commands`
      - `↓` = display de `session.child.first` (`context/keymap.tsx:463` → `down: "↓"`) em `text.base`; `2 subagents` em `text.muted`; `·` e uso `formatContextUsage(tokens, percent)` em `text.muted`; depois `ctrl+p` (`text.base`) + ` commands` (muted);
      - `shells` aparece como `N shells` (`:29-35`), soma de subagentes + shells com ` · `.
    - sem trabalho vivo: `shift+tab agents  ctrl+p commands` (mesma pintura: atalho em `text.base`, rótulo em `text.muted`).
    - visibilidade: `width ≥ 44` para os atalhos; uso some quando não cabe (`promptFooterLayout`, `:146-156`).

### 8.3 Metadata row (logo abaixo do texto, acima do rodapé)

`component/prompt/metadata.tsx:36-87` + `:138-146`, `gap={1}`:

```
<agent: fg = cor do agente> [auto: text.muted] · [model: text.base] [provider: text.muted] · [variant: #f1fa8c BOLD]
```

- `auto` só aparece com `session.permissions: "autoaccept"`.
- Se não couber, degrada: sem `auto` → provider abreviado → sem provider → trunca o modelo com `…` (`promptMetadataLayout`, `:101-136`).
- Exemplos reais observados: `Orchestrator auto · Nemotron 3.5 Super VL Free OpenCode · xhigh` e `Orchestrator auto · DeepSeek V4.1 Flash Free OpenCode · max`.

---

## 9. Elementos que **não** fazem parte do clone

### 9.1 Faixa DevTools (1 linha, largura total)

**De onde vem (não é plugin):** componente do **core** — `packages/tui/src/component/devtools-bar.tsx` (`export function DevToolsBar`, `:29`), renderizado pelo shell do app em `packages/tui/src/app.tsx:1429-1431`, logo abaixo de todo o resto e acima de toasts/overlays.

**É ligável/desligável:**
- default do binário: `const devtools = createMemo(() => config.data.debug?.devtools ?? app.channel === "local")` — `app.tsx:463`. Ou seja: builds de canal `local` (desenvolvimento) mostram sempre; releases só se a config pedir.
- pela própria UI: diálogo de configuração, item **“Developer tools”**, categoria `Debug`, path `debug.devtools`, valores `off|on` — `packages/tui/src/component/dialog-config.tsx:318-326`.
- estado desta máquina: **`on`** (`~/.config/opencode/cli.json` → `debug.devtools: true`).
- **não** vem de plugin: o `tui.json` legado aponta para `…/plugins/orca-opencode-status-tui/tui.js`, que não existe no disco e não é carregado (§0).

**No clone:** fundo `decrease(background.base)` = **`#21222c`** (`devtools-bar.tsx:232`). Como é opcional e barata de adicionar/remover, a decisão de incluí-la no OVC fica **aberta para depois** (ver §12.7).

Ordem e conteúdo (`devtools-bar.tsx:245-441`): `[ícone] Server` · `[ícone] UI` · `Theme` · `Tools` · `Experiments` · (espaço flexível, timeline de “Time to first draw”).
- `Server`: ícone de conexão (`✓` conectado em `text.feedback.success.base` = `#50fa7b`), rótulo `Server` em `text.muted`.
- `UI`: ícone de runtime (`○` normal) + `UI`, ambos `text.muted` (`text.feedback.error.base` se `runtime === "high"`).
- `Theme`, `Tools`, `Experiments`: `text.muted`; item ativo recebe `text.action.primary.focused`.
- Item ativo/hover: fundo `background.action.primary.focused` (`#bd93f9`) / `hovered` (`#21222c`), `paddingLeft/Right 1` (`:445-473`).
- Painéis abrem **acima** da faixa: `position absolute; bottom 1; left -1; width 42; padding 1/2; backgroundColor raised.base` (`:476-491`).
- ⚠️ **Divergência registrada:** nos prints do usuário a faixa aparece **laranja**; pelo fonte o fundo é `#21222c` (§2.5). Nenhum token usado nessa linha é laranja. Hipóteses: (a) leitura de cor do print (a única coisa coral na região é a metadata row imediatamente acima, §8.3); (b) o print é de outro tema/estado. **Pendente de conferência com o arquivo de imagem** — ver §12.

### 9.2 Chrome do terminal (fora de escopo)

**Fonte da conclusão:** `%LOCALAPPDATA%\Packages\Microsoft.WindowsTerminal_8wekyb3d8bbwe\LocalState\settings.json` desta máquina —

| Chave | Valor | Consequência |
| --- | --- | --- |
| `profiles.defaults` | `{}` | nenhuma fonte/tamanho/opacidade customizados |
| `schemes` | `[]` | nenhum esquema de cores próprio |
| `themes` | `[]` | nenhum tema de janela (cor de aba/título) |
| `profiles.list[].font` | ausente em todos os perfis | fonte padrão do Windows Terminal |
| `defaultProfile` | `{61c54bbd-…}` = “Windows PowerShell” | perfil padrão, sem customização |

→ O topo é **chrome do Windows Terminal** (barra de título + botões de janela), inclusive o texto `OC | …`: o *string* é gerado pelo TUI (`packages/tui/src/app.tsx:621`: `` renderer.setTerminalTitle(`OC | ${title.length > 40 ? title.slice(0,37) + "…" : title}`) ``, corte em 37 células + `…`), mas quem o desenha é a janela. **O OVC não deve desenhar título nem botões de janela.**

---

## 10. Mapeamento para a UI web (Go + `go:embed`, sem npm)

1. **Tokens → variáveis CSS** em um único `tokens.css` gerado a partir de `docs/research/assets/dracula-resolved.json` (§2). Camada 1: papéis semânticos; nunca usar hex solto no componente.
2. **Grade**: um “cell” = `1ch` × `1em` com `line-height` inteiro. Todos os paddings/gaps em `ch`. Sidebar `42ch`, painel central `calc(100% - 42ch - 4ch)`.
3. **Glifos**: usar os literais exatos deste contrato (não substituir por ícones SVG) — `█ ▀ ▄ ┃ ▌ ● ○ ▪ ◉ ⌕ ↓ · ⊙ ✕ › + − ◈ ⠋…`. São a assinatura visual.
4. **Sem animação obrigatória**: spinner a 80 ms é a única animação estrutural; `prefers-reduced-motion` → frame fixo (`⠋`), igual ao fallback do TUI (`[⋯]` quando `animations: false`).
5. **Derivadas**: implementar `tint()`/`increase()`/`decrease()` como funções puras (ou pré-computar as 7 da §2.5) — **nunca** “chutar” um hex próximo.
6. **Microcopy em pt-BR** (§11) mantendo as métricas de largura/truncamento (ex.: título de sessão cortado em 40 com `…`).
7. **Estados a portar**: home vazia, conversa em andamento, mensagem do usuário (normal/hover), grupo de atividade fechado/aberto, composer focado/desabilitado, rodapé com e sem trabalho vivo.

---

## 11. Strings exatas (referência) e proposta pt-BR

| Original (en-US) | Local | Proposta pt-BR |
| --- | --- | --- |
| `Ask anything… "Fix a TODO in the codebase"` | `component/prompt/index.tsx:1601` | `Pergunte qualquer coisa… "Corrija um TODO no código"` |
| `shift+tab agents` / `ctrl+p commands` | `prompt/footer.tsx:105-118` | `shift+tab agentes` / `ctrl+p comandos` |
| `↓ 2 subagents · 81.2K (41%)` | `prompt/footer.tsx:82-95` | `↓ 2 subagentes · 81,2K (41%)` |
| `/exit restart to use v2.0.31` | `routes/home.tsx:158-169` | `/exit reinicie para usar v2.0.31` |
| `+ 3 commands, 7 thoughts, 4 tools` | `group-view.tsx:211-222` | `+ 3 comandos, 7 pensamentos, 4 ferramentas` |
| `⊙ 7 MCP /mcps` | `home/footer.tsx:38-45` | `⊙ 7 MCP /mcps` (sigla não traduz) |
| `+ New session` (item de adicionar da trilha; `+` no modo compacto) | `session-tabs-model.ts:8`, `session-tabs.tsx:1700` | `+ Nova sessão` |
| `Server · UI · Theme · Tools · Experiments` | `devtools-bar.tsx` | se portado: `Servidor · UI · Tema · Ferramentas · Experimentos` |
| `Thinking…` / `Running command…` | `activity-summary.ts:47-52` | `Pensando…` / `Executando comando…` |

---

## 12. Pendências e divergências registradas

1. **Cor da faixa DevTools** — fonte diz `#21222c`; os prints do usuário descrevem “fundo laranja”. Precisa do arquivo de imagem para fechar (§9.1). *Não inventar valor.*
2. **Fonte tipográfica** — Cascadia Mono é **[Provável]** (padrão do WT sem override). Confirmar visualmente comparando o desenho de `█`/`▀` e a largura da sidebar (42 col ≈ 255 px).
3. **Screenshots oficiais** (`docs/research/assets/screenshot.png`, `screenshot-splash.png`) são de **1.1.19** e do tema `opencode` (não dracula). Servem para *layout* (§5–§8), **não** para cor nem para glifos de ferramenta (lá aparecem `*` e `→`, hoje substituídos por `+` e `›`).
4. **Cabeçalho do transcript** (título da sessão + custo/contexto alinhados à direita): existe na screenshot 1.1.19 (`# <título> … 39,413 20% ($0.29)`), mas **não foi verificado** no fonte v2 nesta passagem — não portar sem conferir.
5. **`.29` vs `.31`** — divergência visual nula nos componentes observados (§0). Se o OVC quiser o alvo literal do print, fixar `v2.0.29` (`042d083…`); caso contrário `v2.0.31` (`22728d6…`) é equivalente e é o binário instalado.
6. **QA/release segurados** — QA em background e release v0.3.0 permanecem suspensos até aprovação visual; nada fora de `docs/research/**` foi tocado nesta fase.
7. **Faixa DevTools no clone** — decisão **em aberto**: ela aparece no estado atual do usuário (`debug.devtools: on`) e é barata de incluir/remover (§9.1). Se o OVC não tiver um rodapé de diagnóstico, basta não portar §9.1.
8. **Frames de demonstração** — reconferido **com rede** (API de árvore do GitHub na tag `v2.0.31`): existem 3 vídeos oficiais, todos em `packages/console/app/src/asset/lander/` — `opencode-min.mp4` (10,4 MB), `opencode-comparison-min.mp4` (16,9 MB) e `desktop-tabs-landscape.mp4` (14,9 MB). O único do TUI (`opencode-min.mp4`, 36,5 s / 1920×1080 / 60 fps) foi baixado e inspecionado quadro a quadro e **se auto-identifica como `OpenCode 1.0.146`** → é o **layout v1** (painel direito de Context/MCP/LSP/TODO, sem trilha de sessões, sem faixa DevTools) e **não deve** ser usado como referência do v2. Nenhum foi versionado (peso); registro completo em `assets/README.md`. Para um registro em movimento da v2, o caminho é gravar o TUI local (a versão do usuário já está instalada).
9. **Captura do próprio usuário** — os 2 prints reais (tela inicial + conversa) não existem como arquivo nesta máquina/sessão. Se forem fornecidos, entram em `docs/research/assets/` com o mesmo README de origem e servem para fechar §12.1.

---

## 13. Checklist de fidelidade (usar como critério de aceite)

- [ ] Wordmark: `open` = `#6272a4`, `code` = `#f8f8f2` bold, sombras `#373c52`/`#5c5e65`, gap de 1 célula.
- [ ] Splash: 3 linhas acima / 1 abaixo, prompt `maxWidth 75`, corte do logo por altura 12 e larguras 22/44.
- [ ] Aviso de versão com `/exit` em `#bd93f9` e o resto em `#6272a4`.
- [ ] Home footer: `⊙ N MCP` + `/mcps` e versão à direita, `padding 2/1`, `gap 2`.
- [ ] Composer: borda esquerda tingida pela cor do agente, base fechada por `▀`, placeholder `Ask anything… "…"` em `#6272a4`.
- [ ] Metadata row: agente (cor do agente) + `auto` (`#6272a4`) + `·` + modelo (`#f8f8f2`) + provider (`#6272a4`) + `·` + variant (`#f1fa8c` **bold**).
- [ ] Rodapé do composer: local à esquerda, `↓ N subagents · <tokens> (<%>) ctrl+p commands` à direita.
- [ ] Mensagem do usuário: moldura `┃` de 1 célula na cor do agente + fundo `#21222c` + hover `#44475a`.
- [ ] Linha de atividade `+ 3 commands, 7 thoughts, 4 tools` em `#6272a4`, vira `−` ao abrir.
- [ ] Sidebar 42 col com cabeçalho de grupo, item `+ New session`, `✕`/`×` no hover e controle `⌕`.
- [ ] Nenhum hexadecimal fora da §2/§3, nenhuma derivada calculada “a olho”.
