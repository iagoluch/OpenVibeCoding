# Guía de usuario de OpenVibeCoding

[Português](../pt-BR/guia.md) · [English](../guide.md) · [Español](guia.md) · [简体中文](../zh-CN/guide.md)

OpenVibeCoding (OVC) es un agente de programación para terminal, escrito en Go con Bubble Tea. Inspecciona proyectos, propone cambios revisables, ejecuta comandos aprobados, conserva el contexto de tareas, prueba sitios web en Chromium y puede realizar acciones aprobadas en el escritorio cuando la plataforma lo permite.

## Instalar e iniciar

Descarga un binario desde [Releases](https://github.com/iagoluch/OpenVibeCoding/releases/latest) o usa el instalador, que descarga el paquete para tu sistema/CPU y verifica su SHA-256.

Linux/macOS:
```sh
git clone https://github.com/iagoluch/OpenVibeCoding.git
cd OpenVibeCoding
bash scripts/install.sh
# Si hace falta, añade ~/.local/bin al PATH:
export PATH="$HOME/.local/bin:$PATH"
ovc --workspace /ruta/del/proyecto
```

Windows (PowerShell): ejecuta `./scripts/install.ps1 -AddToPath` y vuelve a abrir la terminal. Si la política de ejecución bloquea el script, usa `powershell -ExecutionPolicy Bypass -File scripts/install.ps1 -AddToPath`. Para fijar una versión, usa `-Version v0.1.0` en PowerShell o `OPENVIBECODING_VERSION=v0.1.0` en el shell.

Compilar desde el código fuente requiere **Go 1.26.2 o posterior**: `go build -o ovc ./cmd/ovc`, seguido de `bash scripts/install.sh ./ovc` (Windows: `./scripts/install.ps1 -Binary ./ovc.exe -AddToPath`). `bash scripts/package.sh` genera archivos para Linux/macOS/Windows × amd64/arm64 y `SHA256SUMS`; el empaquetado también requiere Python 3. Ejecutar un binario de lanzamiento no requiere Go. Las pruebas de navegador necesitan Node y Chromium.

## Configuración inicial y credenciales

La primera ejecución crea una configuración predeterminada. Ejecuta `ovc --setup` para usar el asistente: detecta servidores Ollama y LM Studio, enumera proveedores con planes gratuitos y enlaces para obtener claves, puede importar credenciales de OpenCode, guarda la clave en el almacén del sistema operativo y realiza una petición real antes de activar la configuración. Si el modelo preferido está retirado, limitado por frecuencia o ocupado, el asistente prueba el siguiente modelo del proveedor.

Para registrar una credencial mediante entrada oculta en la terminal:
```sh
ovc --credential openai
ovc --credential anthropic
ovc --credential gemini
```

El backend predeterminado es el almacén del sistema operativo: Keychain en macOS, Credential Manager en Windows o Secret Service en Linux. Si no está disponible, elige explícitamente `--credential-file`; no existe una alternativa silenciosa en texto plano. Variables como `OPENAI_API_KEY`, `ANTHROPIC_API_KEY` y `GEMINI_API_KEY` sustituyen las credenciales guardadas. No pegues secretos en conversaciones ni en URLs de endpoints.

Usa `ovc --doctor` para comprobar configuración y capacidades sin mostrar claves. Si cambian las credenciales o la configuración mientras el runtime está activo, detén el runtime inactivo con `ovc --runtime stop` y vuelve a conectarte. Los comandos `/model` y `/harness` se aplican inmediatamente.

## Proveedores compatibles

| Adaptador/proveedor | Compatibilidad |
| --- | --- |
| OpenAI | Streaming nativo de Chat Completions, herramientas y uso opcional |
| Anthropic Claude | Streaming nativo de Messages API, herramientas y uso |
| Google Gemini | Streaming nativo, llamadas/resultados de funciones, imágenes y conservación de firmas de pensamiento |
| DeepSeek, Kimi, Qwen, Mistral | Endpoints compatibles con OpenAI |
| Presets gratuitos | OpenRouter (modelos `:free`), Groq, Cerebras, NVIDIA NIM, Z.AI y Google AI Studio; consulta `ovc --setup` |
| Locales/personalizados | Cualquier endpoint compatible; se permite HTTP en loopback y se detectan Ollama/LM Studio |

El catálogo de proveedores define modelos, presupuestos de contexto, compatibilidad con herramientas, límites de salida, temperatura y precios. Verifica la disponibilidad y los endpoints regionales con el proveedor. Las rutas realmente gratuitas se muestran como US$ 0; los servicios de pago muestran los precios publicados por millón de tokens; si no hay precio fiable, se marca como desconocido en vez de inventarlo. Las peticiones tienen timeout, respuestas limitadas, cancelación, reintentos para HTTP 429/5xx y errores redactados. Se rechazan redirecciones para proteger credenciales; HTTP solo se permite en loopback.

Los presets se actualizan con cada lanzamiento. `config.json` conserva únicamente tus cambios; las entradas no editadas reciben automáticamente nuevos identificadores de modelo, contextos y precios.

## Flujo de desarrollo

1. Pide al agente que inspeccione el proyecto y planifique el cambio. Las herramientas respetan nombres protegidos y patrones de `.gitignore`, limitan lecturas/búsquedas y rechazan rutas fuera del workspace.
2. Revisa el diff unificado. **Ctrl+Y** aprueba esa operación; **Ctrl+N** la rechaza. Se pueden reemplazar, crear, mover y eliminar archivos. Los cambios externos invalidan propuestas obsoletas.
3. Aprueba comandos de pruebas/build. En Linux compatible se ejecutan en Bubblewrap, con acceso al workspace, archivos de runtime del sistema de solo lectura y sin red por defecto. Si no se puede aplicar aislamiento, la ejecución falla de forma cerrada.
4. Inspecciona stdout, stderr, código de salida y `/changes`. Puedes retomar la sesión más tarde.

`--allow-unsandboxed-shell` permite explícitamente ejecutar comandos en el host en cualquier plataforma. Cada comando sigue requiriendo aprobación, pero se ejecuta con los privilegios del usuario y puede acceder a archivos/red del host; no es un sandbox del sistema operativo. Consulta las [garantías de seguridad](seguridad.md).

### Controles y comandos

Enter envía; Ctrl+J o Alt+Enter inserta una nueva línea; Alt+Arriba/Abajo recupera entradas; Tab completa comandos slash; PgUp/PgDn desplaza la vista. Esc o Ctrl+C cancela la operación actual. Ctrl+C estando inactivo cierra el cliente. La revisión es por operación/archivo y muestra diffs unificados completos; no hay selección por hunk.

| Comandos | Finalidad |
| --- | --- |
| `/help`, `/status`, `/config` | Ayuda, estado de sesión/runtime y configuración efectiva |
| `/model [provider/model]`, `/provider [name]` | Seleccionar proveedor/modelo registrado |
| `/harness [review\|inspect]` | `review` permite operaciones aprobadas; `inspect` solo permite leer el proyecto |
| `/sessions`, `/resume <id>`, `/new` | Navegar por sesiones persistentes |
| `/rename <title>`, `/delete <other-session-id>` | Administrar sesiones |
| `/init`, `/undo`, `/doctor` | Proponer `AGENTS.md`, deshacer el último cambio aplicado, diagnosticar runtime |
| `/context`, `/cost`, `/changes` | Contexto, uso/coste guardado y cambios de archivos observados |
| `/capabilities` | Comprobar requisitos de automatización y sandbox |
| `/permissions [browser\|desktop\|clipboard\|network\|install on\|off]` | Consultar o modificar permisos de este cliente |
| `/clear`, `/exit` | Limpiar la vista sin borrar datos; salir |

Un `AGENTS.md` en la raíz del workspace se añade al mensaje de sistema de cada turno (máximo 16 KiB). Describe convenciones del proyecto, pero es evidencia, no autorización. `/init` pide al agente inspeccionar el proyecto y proponer el archivo mediante el flujo normal de revisión. Las nuevas sesiones se titulan según la primera solicitud; `/rename` puede cambiar el título.

`/undo` revierte los cambios aplicados uno por uno y sobrevive a reinicios. Se niega a actuar si el archivo cambió después de la operación, si la ruta fue recreada o si hay un turno activo. Las concesiones de permisos no eliminan las aprobaciones por operación. Red e instalación están desactivadas por defecto; la instalación de paquetes requiere un permiso separado. Un shell aprobado es un entorno general de programación, así que los límites efectivos dependen del aislamiento del sistema operativo, no solo de las categorías de herramientas.

## Runtime compartido y sesiones

La primera terminal inicia un runtime local; las siguientes lo reutilizan. El daemon controla SQLite, configuración de proveedores, auditoría y coordinación de tareas. La comunicación utiliza HTTP autenticado en loopback, un token aleatorio en archivo privado y un bloqueo del sistema operativo para impedir daemons duplicados. Los navegadores no pueden invocarlo mediante solicitudes cross-origin.

```sh
ovc --runtime start
ovc --runtime status
ovc --runtime stop
ovc --resume SESSION_ID --workspace /ruta/del/proyecto
ovc --export SESSION_ID > session.json
```

Cada cliente tiene identidad y aprobaciones independientes. Los turnos se serializan por sesión **y workspace**. Cerrar un cliente cancela su solicitud sin interrumpir tareas de otros clientes. El daemon sigue activo hasta detenerlo expresamente; no permite detenerlo mientras haya tareas activas. La terminación del sistema operativo cancela tareas antes de cerrar el almacenamiento.

SQLite almacena mensajes originales, resultados de herramientas, aprobaciones, ciclo de tareas, cambios de configuración, resúmenes y uso/coste de tokens. Las migraciones son transaccionales. Tras una interrupción, el resultado de una herramienta se marca como incierto; las mutaciones no se reproducen automáticamente. Las exportaciones incluyen historial, resumen, uso y auditoría; la sesión debe estar inactiva para exportar. Las capturas de pantalla son transitorias y no se guardan en la base de datos.

La preparación de contexto resume progresivamente intercambios antiguos completos sin borrar los mensajes recientes ni el historial original. Los identificadores de origen permiten recuperar evidencia con `recall`. Las estimaciones de tokens son conservadoras; el uso exacto depende del proveedor. Los resúmenes pueden ser imperfectos: la evidencia original prevalece y las aprobaciones anteriores no autorizan acciones nuevas. Una tarea demasiado grande falla con un error útil en lugar de descartarse silenciosamente.

## Pruebas de navegador

```sh
ovc --setup-browser
# En la TUI:
/permissions browser on
```

La configuración instala Playwright 1.58.2 y Chromium en el directorio QA de la aplicación. Revisa y aprueba un flujo con orígenes permitidos y pasos como `open`, `fill`, `click`, `press`, `select`, `expect_text`, `expect_url` y `screenshot`. Cada flujo usa un contexto nuevo; se bloquean orígenes no autorizados, service workers, WebSockets y descargas. Los resultados incluyen texto DOM observado, título/URL, aserciones, mensajes de consola, errores de página y solicitudes fallidas. Los modelos con soporte de imágenes pueden inspeccionar capturas durante el turno; luego se descartan. Usa datos sintéticos y entornos de prueba dedicados.

## Uso del escritorio

Activa `/permissions desktop on`; el portapapeles necesita `/permissions clipboard on` por separado. Todas las acciones se revisan. Pueden incluir inspección, capturas, movimiento/clic del puntero, desplazamiento, escritura/teclas, abrir/enfocar/cerrar aplicaciones y acceso al portapapeles, según el backend.

| Plataforma | Backend y límites |
| --- | --- |
| Linux X11 | `xdotool`, ImageMagick `import` para capturas y `xclip` para portapapeles; requiere `DISPLAY` activo. Wayland no está soportado por el backend integrado. |
| macOS | AppleScript/System Events, `screencapture` y CoreGraphics mediante JavaScript for Automation. Requiere permisos de Accesibilidad y Grabación de pantalla. |
| Windows | PowerShell, APIs Win32 y Windows Forms para entrada/portapapeles/capturas. Requiere un escritorio interactivo; no controla UAC/escritorios seguros ni evita permisos del SO. |

`automation.applications` es una lista permitida para abrir/enfocar aplicaciones, no aislamiento de procesos para acciones globales del ratón/teclado. Revisa el objetivo visible antes de aprobar una entrada. Los adaptadores externos se configuran en `automation.desktop_backend` como matriz de ejecutable/argumentos. Consulta el [contrato de automatización](automatizacion.md).

## Configuración, mantenimiento y verificación

Consulta [config.example.json](../../config.example.json). Los datos se guardan en `os.UserConfigDir()/openvibecoding` (por ejemplo, `~/.config`, Application Support en macOS o AppData en Windows); `OPENVIBECODING_HOME` sustituye el directorio. `OPENVIBECODING_PROVIDER` y `OPENVIBECODING_MODEL` sustituyen los valores predeterminados. Los secretos de proveedores no se incluyen en JSON ni en las exportaciones.

El directorio contiene `config.json`, archivos SQLite de sesiones, `runtime.json`/`runtime.lock`, `runtime.log` con metadatos, credenciales opcionales y dependencias QA. En Unix se restringen permisos de archivos/directorios; Windows depende de las ACL del perfil. `--debug` registra metadatos de diagnóstico, no cuerpos de peticiones ni secretos. Detén el runtime antes de copiar los datos o actualizar binarios.

Para actualizar, ejecuta el instalador con un binario/release más reciente (`OPENVIBECODING_VERSION=v…` o PowerShell `-Version v…`); se conservan configuración y sesiones. Detén el runtime y usa `scripts/uninstall.sh` o `scripts/uninstall.ps1` para quitar el lanzador. Borra los datos por separado solo si quieres eliminar sesiones y credenciales.

Para validar, consulta [pruebas y diagnóstico](pruebas.md), [arquitectura](arquitectura.md) y [referencias técnicas](referencias.md). Las pruebas locales cubren sandbox, navegador y escritorio en Linux, protocolos de proveedores, concurrencia/autenticación/aprobaciones del runtime, persistencia y recuperación del contexto. La ejecución nativa en macOS/Windows, los almacenes de claves de esos hosts y las peticiones facturables requieren sus entornos/cuentas correspondientes. La compilación cruzada no certifica el comportamiento nativo.
