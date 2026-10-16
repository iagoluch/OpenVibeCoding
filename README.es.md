<p align="center">
  <strong>Un entorno de desarrollo con IA pensado para una experiencia de aplicación de escritorio.</strong><br>
  Revisa cambios, aprueba operaciones y comprueba resultados: tú mantienes el control.
</p>

<p align="center"><strong>Aplicación de escritorio integrada: aún no publicada · CLI/TUI: disponible hoy</strong></p>

<p align="center">
  <a href="https://github.com/iagoluch/OpenVibeCoding/releases/latest">⬇️ Probar la versión actual (CLI/TUI)</a>
  &nbsp;·&nbsp;
  <a href="https://github.com/iagoluch/OpenVibeCoding/stargazers">⭐ Dale una estrella al proyecto</a>
</p>

<p align="center">
  <a href="README.md">🇧🇷 Português (Brasil)</a> ·
  <a href="README.en.md">🇺🇸 English</a> ·
  <a href="README.es.md">🇪🇸 Español</a> ·
  <a href="README.zh-CN.md">🇨🇳 简体中文</a>
</p>

<p align="center">
  <a href="#instalación">Instalar</a> ·
  <a href="docs/es/guia.md">Guía</a> ·
  <a href="docs/es/seguridad.md">Seguridad</a> ·
  <a href="docs/es/arquitectura.md">Arquitectura</a>
</p>

<p align="center"><a href="https://github.com/iagoluch/OpenVibeCoding/stargazers">⭐ ¿Te resulta útil? Dale una estrella al proyecto</a></p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26.2%2B-00ADD8?logo=go&logoColor=white" alt="Go 1.26.2 o posterior">
  <img src="https://img.shields.io/badge/licencia-MIT-brightgreen" alt="Licencia MIT">
  <img src="https://img.shields.io/badge/plataformas-Linux%20%7C%20macOS%20%7C%20Windows-informational" alt="Linux, macOS y Windows">
</p>

## ¿Qué es OpenVibeCoding?

**OpenVibeCoding (OVC)** es un entorno de desarrollo con IA y código abierto que está evolucionando hacia una aplicación de escritorio integrada. El objetivo es ofrecer al desarrollador un único espacio para trabajar con el agente, revisar cambios propuestos y consultar evidencias de ejecución.

**Estado actual, sin ambigüedades:** la aplicación de escritorio integrada todavía no tiene un binario ni un instalador publicado. La versión pública disponible hoy es **CLI/TUI** (línea de comandos con interfaz de terminal). Las instrucciones de descarga e instalación siguientes corresponden a esa versión, no a una aplicación de escritorio ya publicada.

## La experiencia de escritorio que estamos construyendo

La experiencia de escritorio prevista busca reunir en una sola interfaz la selección de proyectos, las conversaciones con el agente, la revisión de diffs, la aprobación por operación, el historial de tareas y la verificación de resultados. Las integraciones de navegador y escritorio dependen de la plataforma y de los permisos concedidos por el usuario.

Esta sección describe la dirección del producto, no funciones de una GUI de escritorio ya publicada.

## Qué ofrece hoy el CLI/TUI

| Comprender | Cambiar con control | Verificar |
| --- | --- | --- |
| Leer archivos, buscar código, cargar instrucciones del proyecto e inspeccionar el estado de Git | Revisar diffs y aprobar o rechazar operaciones individuales sobre archivos | Ejecutar comandos autorizados y consultar su salida real y código de salida |
| Mantener sesiones y contexto persistentes | Restringir las herramientas de archivos al workspace configurado | Utilizar QA de navegador con Playwright e integraciones de escritorio configuradas |

### Elige el modelo y el proveedor

OVC no te obliga a utilizar un único proveedor de IA. Incluye adaptadores nativos para OpenAI, Anthropic/Claude y Google Gemini, además de endpoints compatibles con la API de OpenAI para proveedores y modelos como DeepSeek, Kimi, Qwen, Mistral y servicios locales.

`ovc --setup` muestra opciones de proveedores y presets de nivel gratuito, detecta Ollama y LM Studio y puede importar credenciales ya configuradas en OpenCode. La disponibilidad, las cuotas y las condiciones de cada proveedor pueden cambiar.

### Sesiones persistentes y evidencias

Las conversaciones, los resúmenes de contexto, los registros de auditoría y el uso se almacenan localmente en SQLite. Los resultados deben reflejar lo que realmente ocurrió: un comando rechazado no se considera aprobado y un proceso con código de salida distinto de cero no debe presentarse como exitoso.

### Automatización con aprobación

OVC incluye pruebas de QA de navegador con Playwright e integraciones de escritorio que dependen del sistema operativo. Habilita únicamente las capacidades que necesita tu tarea. Las acciones de automatización están sujetas a permisos explícitos y aprobación según el contrato de cada herramienta.

## Probar la versión pública actual (CLI/TUI)

### Linux y macOS

El instalador descarga el archivo correspondiente al sistema operativo y a la arquitectura del CPU, y verifica su checksum SHA-256. No necesitas Go para ejecutar un binario de una versión publicada.

```sh
git clone https://github.com/iagoluch/OpenVibeCoding.git
cd OpenVibeCoding
bash scripts/install.sh
ovc --setup
ovc --workspace /ruta/a/tu/proyecto
```

Si no se encuentra el comando `ovc`, añade `~/.local/bin` a tu `PATH` y abre una nueva terminal.

### Windows (PowerShell)

Ejecuta desde el directorio del repositorio:

```powershell
./scripts/install.ps1 -AddToPath
```

Cierra y vuelve a abrir PowerShell después de la instalación. Si la política de ejecución bloquea el script:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/install.ps1 -AddToPath
```

También puedes descargar un archivo precompilado para tu sistema y arquitectura desde [Releases](https://github.com/iagoluch/OpenVibeCoding/releases/latest).

### Primera ejecución

Usa `ovc --setup` para configurar proveedores, credenciales y un modelo. Después, indica el directorio del proyecto:

```sh
ovc --workspace /ruta/a/tu/proyecto
```

Ejecuta `ovc --doctor` para revisar la configuración y las capacidades sin mostrar claves.

### Compilar desde el código fuente

Requiere Go **1.26.2 o posterior**:

```sh
go build -o ovc ./cmd/ovc
```

En Linux/macOS, instala el binario generado con `bash scripts/install.sh ./ovc`. En Windows, genera `ovc.exe` y ejecuta `./scripts/install.ps1 -Binary ./ovc.exe -AddToPath`.

## Un ciclo de desarrollo típico

```text
❯ Investiga este parser, propone una corrección acotada y verifícala.

COMPRENDER  Lee la implementación, las instrucciones y las pruebas relevantes.
REVISAR     Examina el diff propuesto antes de cambiar nada.
APROBAR     Acepta o rechaza la operación individual.
VERIFICAR   Ejecuta el comando autorizado y revisa el resultado real.
```

Este ejemplo ilustra el flujo; no es una ejecución real ni garantiza que un cambio vaya a funcionar.

Comandos útiles:

```text
/model <proveedor/modelo>   /harness inspect
/sessions                  /resume <id>
/context                   /cost
/changes                   /permissions
/capabilities              /help
/init · /undo · /doctor
```

Las instrucciones del proyecto de `AGENTS.md` se incluyen en el contexto de trabajo. Ejecuta `/help` para consultar los comandos disponibles en tu versión.

## Seguridad: conoce los límites

- Las herramientas habituales de archivos están restringidas al workspace configurado y aplican exclusiones para rutas sensibles.
- Los cambios de archivos se muestran para revisión y requieren aprobación individual antes de aplicarse.
- En Linux, la ejecución de shell utiliza Bubblewrap por defecto y falla de forma cerrada si el sandbox necesario no está disponible.
- **El sandbox de procesos integrado solo funciona en Linux.** En macOS y Windows, los comandos que requieren confinamiento fallan de forma cerrada. La opción explícita `--allow-unsandboxed-shell` habilita comandos con los privilegios del usuario del sistema anfitrión; esto no equivale a aislamiento.
- Las integraciones de escritorio requieren permisos y capacidades específicas de cada plataforma. La compilación cruzada no certifica el comportamiento nativo.

Lee [Seguridad y límites de ejecución](docs/es/seguridad.md) antes de habilitar shell o automatización. No se garantiza que las operaciones externas, los procesos separados ni las acciones en otras aplicaciones sean reversibles.

## Documentación

| Documento | Contenido |
| --- | --- |
| [Guía de usuario](docs/es/guia.md) | Instalación, configuración, proveedores y comandos |
| [Arquitectura](docs/es/arquitectura.md) | Módulos, ciclo del agente, persistencia y responsabilidades de cada plataforma |
| [Seguridad](docs/es/seguridad.md) | Aprobaciones, límites de ejecución, credenciales y restricciones |
| [Automatización](docs/es/automatizacion.md) | Flujos de navegador, acciones de escritorio y adaptadores |
| [Pruebas y diagnóstico](docs/es/pruebas.md) | Comprobaciones automatizadas, validación del entorno y solución de problemas |
| Referencias técnicas | [Referencias técnicas](docs/es/referencias.md) |

## Contribuir

- **Pruébalo y deja una estrella:** [⭐ OpenVibeCoding en GitHub](https://github.com/iagoluch/OpenVibeCoding/stargazers).
- **Reporta un problema:** [abre una issue](https://github.com/iagoluch/OpenVibeCoding/issues).
- **Explora el código y propone mejoras:** las pull requests son bienvenidas; revisa las pautas del repositorio antes de contribuir.

OpenVibeCoding se distribuye bajo la [licencia MIT](LICENSE).

---

<p align="center"><sub>OpenVibeCoding — de la intención al cambio verificado, con aprobación humana en el proceso.</sub></p>
