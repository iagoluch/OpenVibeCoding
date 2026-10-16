# Referências técnicas oficiais

Consultadas em 15/10/2026. As APIs dos módulos instalados também foram verificadas com `go doc` antes de usar as APIs v2 de View, textarea e viewport do Bubble Tea.

- [Notas de lançamento do Go 1.26](https://go.dev/doc/go1.26): toolchain e plataformas suportadas. As APIs `os.Root` dificultam escapes do filesystem, mas não isolam processos.
- [Bubble Tea v2](https://github.com/charmbracelet/bubbletea): arquitetura de eventos/update/view e caminho do módulo v2. [Bubbles](https://pkg.go.dev/charm.land/bubbles/v2) fornece textarea e viewport.
- [OpenAI Chat Completions create](https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create): deltas de streaming, chamadas de ferramentas e eventos opcionais de uso.
- [modernc SQLite](https://pkg.go.dev/modernc.org/sqlite): driver database/sql e caminho de build em Go puro.
- [Introdução ao Playwright](https://playwright.dev/docs/intro): QA de navegador e requisitos por SO.

## Referências complementares de implementação

- [Streaming de mensagens Anthropic](https://platform.claude.com/docs/en/api/messages-streaming): início/deltas de blocos de conteúdo, JSON parcial de ferramentas, uso e conclusão da mensagem.
- [Gemini generate/stream content](https://ai.google.dev/api/generate-content): partes nativas, chamadas/resultados de funções e assinaturas de pensamento.
- [Biblioteca Go keyring](https://github.com/zalando/go-keyring): APIs macOS/Windows/Secret Service, operações Get/Set/Delete e requisitos do SO.
- [Bubblewrap](https://github.com/containers/bubblewrap): namespaces, mounts somente leitura, responsabilidade da política e limitações.
- [API Browser do Playwright](https://playwright.dev/docs/api/class-browser): ciclo de vida de browser/context.
- [xdotool](https://github.com/jordansissel/xdotool): entrada/janelas X11 e limitação em Wayland.
- [Apple UI scripting](https://developer.apple.com/library/archive/documentation/LanguagesUtilities/Conceptual/MacAutomationScriptingGuide/AutomatetheUserInterface.html) e [CoreGraphics CGEvent](https://developer.apple.com/documentation/coregraphics/cgevent): acessibilidade concedida pelo usuário, eventos, ponteiro/rolagem e envio.
- [Entrada de desktop Microsoft](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-sendinput): restrições de entrada e limites de nível de integridade do Windows.
