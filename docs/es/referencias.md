# Referencias oficiales de implementación

Consultadas el 15/10/2026. También se comprobaron con `go doc` las APIs de los módulos instalados antes de utilizar las APIs v2 de View, textarea y viewport de Bubble Tea.

- [Notas de lanzamiento de Go 1.26](https://go.dev/doc/go1.26): toolchain y plataformas compatibles. Las APIs `os.Root` dificultan escapes del sistema de archivos; no aíslan procesos.
- [Bubble Tea v2](https://github.com/charmbracelet/bubbletea): arquitectura de eventos/update/view y ruta del módulo v2. [Bubbles](https://pkg.go.dev/charm.land/bubbles/v2) proporciona textarea y viewport.
- [OpenAI Chat Completions create](https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create): deltas de streaming, llamadas a herramientas y eventos opcionales de uso.
- [modernc SQLite](https://pkg.go.dev/modernc.org/sqlite): driver database/sql y compilación en Go puro.
- [Introducción a Playwright](https://playwright.dev/docs/intro): QA de navegador y requisitos de SO.

## Referencias adicionales de implementación

- [Streaming de mensajes de Anthropic](https://platform.claude.com/docs/en/api/messages-streaming): inicio/deltas de bloques de contenido, JSON parcial de herramientas, uso y finalización de mensajes.
- [Gemini generate/stream content](https://ai.google.dev/api/generate-content): partes nativas, llamadas/resultados de funciones y firmas de pensamiento.
- [Biblioteca Go keyring](https://github.com/zalando/go-keyring): APIs macOS/Windows/Secret Service, Get/Set/Delete y requisitos del SO.
- [Bubblewrap](https://github.com/containers/bubblewrap): namespaces, mounts de solo lectura, responsabilidad de política y limitaciones.
- [API Browser de Playwright](https://playwright.dev/docs/api/class-browser): ciclo de vida del browser/context.
- [xdotool](https://github.com/jordansissel/xdotool): APIs de entrada/ventana X11 y limitación de Wayland.
- [Apple UI scripting](https://developer.apple.com/library/archive/documentation/LanguagesUtilities/Conceptual/MacAutomationScriptingGuide/AutomatetheUserInterface.html) y [CoreGraphics CGEvent](https://developer.apple.com/documentation/coregraphics/cgevent): accesibilidad concedida por el usuario, eventos, campos de puntero/scroll y envío.
- [Entrada de escritorio de Microsoft](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-sendinput): restricciones de entrada y límites de nivel de integridad.
