# Assets de referência

Arquivos brutos usados pelo contrato `../opencode-tui-visual-contract.md`. **Nada aqui é código de produto.**

| Arquivo | Origem | Nota |
| --- | --- | --- |
| `screenshot.png` | `packages/web/src/assets/lander/screenshot.png` do repositório `https://github.com/anomalyco/opencode`, tag `v2.0.31` (`22728d67461260d56bb9796b9b8f36e1de47202e`) | Screenshot **oficial** do TUI. Capturado na versão **1.1.19** e com o tema default (`opencode`), não com `dracula`. Serve como referência de **layout**; não usar como referência de cor nem de glifos de ferramenta. |
| `screenshot-splash.png` | idem | Tela inicial: wordmark, `Ask anything…`, metadata row, footer do home. Mesmas ressalvas de versão/tema. |
| `dracula-resolved.json` | Dump gerado nesta máquina pelo port fiel do pipeline de tema (`packages/theme/src/tui/*`), a partir de `packages/tui/src/theme/assets/dracula.json` + `~/.config/opencode/cli.json` (`theme: dracula/dark`) | Fonte de verdade da paleta. Validado contra os asserts de `packages/tui/test/theme/v2/v1-migrate.test.ts` (36 passed / 0 failed). |

## Demo animado oficial — existe, mas **não serve** como referência de layout

Verificado em 16/10/2026 pela API de árvore do GitHub na tag exata (`/repos/anomalyco/opencode/git/trees/22728d67461260d56bb9796b9b8f36e1de47202e?recursive=1`): há **3 arquivos de vídeo** no repositório, todos em `packages/console/app/src/asset/lander/` — nenhum deles foi versionado aqui (peso), apenas registrado:

| Arquivo (não incluído) | Tamanho | O que é |
| --- | --- | --- |
| `opencode-min.mp4` | 10.368.838 B | **Demo animado do TUI**: 36,5 s, 1920×1080, 60 fps, h264. Baixado e inspecionado quadro a quadro (ffmpeg). |
| `opencode-comparison-min.mp4` | 16.908.501 B | Comparação do OpenCode com outros agentes; não é captura de TUI. |
| `desktop-tabs-landscape.mp4` | 14.906.820 B | App desktop (abas), não é o TUI. |
| `opencode-poster.png` / `opencode-comparison-poster.png` | 13.705 B cada | Capas dos vídeos; os dois são **byte-a-byte idênticos** (SHA-256 `425EAF71…3CA5B`) e genéricos — sem valor para o clone. |

**Por que o demo não entra como referência:** o próprio vídeo se identifica no canto inferior direito — `▪ OpenCode  1.0.146` (lido em *crop* ampliado do quadro de t=19 s). É o **layout v1**, não o v2.0.29 do usuário:

- sem a trilha esquerda de sessões (§6) e sem a faixa DevTools inferior (§9.1);
- **com** um painel direito de `Context` / `MCP` / `LSP` / `TODO` que não existe mais na v2;
- tema default (não `dracula`), transitórios de ferramenta da v1 (`⊙`, `└`) em vez dos indicadores da v2 (`• ● ▪ ■`).

Ou seja: útil só como amostra histórica de transcript. Quem quiser reconferir: `curl -L https://raw.githubusercontent.com/anomalyco/opencode/22728d67461260d56bb9796b9b8f36e1de47202e/packages/console/app/src/asset/lander/opencode-min.mp4`.

Créditos e licença: o conteúdo visual acima é do projeto OpenCode (repositório citado); o dump de tema é uma medição derivada de artefatos locais. Ambos entram neste repositório apenas como **referência de pesquisa**, com origem declarada.
