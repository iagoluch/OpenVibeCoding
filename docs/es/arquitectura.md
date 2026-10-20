# Arquitectura

[Português](../pt-BR/arquitetura.md) · [English](../architecture.md) · [Español](arquitectura.md) · [简体中文](../zh-CN/architecture.md)

## Módulos

| Módulo | Responsabilidad |
| --- | --- |
| `cmd/ovc` | CLI, configuración inicial, entrada oculta de credenciales, ciclo del runtime, diagnóstico/exportación/configuración del navegador |
| `internal/config` | Catálogo de modelos/proveedores, selección de protocolo, parámetros, permisos y configuración segura de credenciales |
| `internal/credentials` | Almacén de claves del SO y fallback explícito mediante archivo restringido |
| `internal/provider` | Adaptadores de streaming OpenAI-compatible, Anthropic y Gemini nativos; herramientas, imágenes, uso y reintentos/errores |
| `internal/store` | Migraciones SQLite, sesiones, intercambios originales, auditoría, resúmenes, uso y exportaciones |
| `internal/runtime` | IPC local autenticado, bloqueo de propiedad del SO, reutilización del daemon, coordinación por sesión/workspace y flujo de aprobaciones |
| `internal/agent` | Bucle limitado de modelo/herramientas, instrucciones de planificación, política del harness, aprobaciones, contexto incremental y recuperación de evidencia |
| `internal/workspace` | Herramientas de archivos confinadas, reglas ignore, búsqueda, inspección Git, escrituras/movimientos/borrados revisables, shell y sandbox Linux |
| `internal/automation` | Worker Playwright integrado, backends nativos, capturas, comprobaciones de capacidades y contrato para adaptadores externos |
| `internal/tui` | Transcripción de streaming Bubble Tea, entrada/historial, selección de modelo/harness, permisos, revisión y navegación de sesiones |
| `scripts` | Build, archivos/checksums, descarga/instalación, actualización y desinstalación |

## Ciclo de un turno

Un cliente se conecta al runtime existente o inicia uno. Selecciona una sesión persistida y un proveedor/modelo/harness. El runtime valida la selección y coordina el workspace/sesión durante el turno. Los mensajes originales se guardan antes de las peticiones; la preparación de contexto resume gradualmente intercambios antiguos sin borrarlos. Las llamadas de herramientas del proveedor se construyen y validan. Las herramientas de lectura actúan dentro de sus límites; las operaciones que requieren aprobación se envían al cliente propietario y quedan pausadas hasta recibir respuesta. La auditoría se registra antes de la ejecución. Los resultados observados y el uso se guardan. La cancelación se propaga a proveedores, procesos y workers.

La TUI sigue gestionando redimensionamiento, desplazamiento, cancelación y revisión mientras se ejecutan peticiones. Los flujos de herramientas se limitan a dieciséis peticiones de modelo por turno. Los pasos de navegador/escritorio son operaciones reales de worker/API; los entornos no compatibles devuelven errores. Las capturas se eliminan de la salida persistente y se adjuntan a una petición del modelo únicamente en memoria transitoria.

## Persistencia y recuperación

SQLite es la fuente autoritativa. El esquema 1 introdujo sesiones/mensajes/auditoría; el esquema 2 añade resúmenes y uso. Las migraciones son transaccionales. WAL y una única conexión a la base de datos serializan escrituras dentro del servicio; el bloqueo del daemon evita runtimes competidores. Las exportaciones requieren una sesión inactiva e incluyen historial, auditoría, resumen y evidencia de coste. Un intercambio de herramienta interrumpido recibe un resultado explícitamente incierto durante la recuperación; las acciones nunca se reproducen a ciegas.

## Límites de plataforma y verificación

El sandbox de shell integrado es Bubblewrap en Linux. En otras plataformas, el confinamiento requerido no está disponible y falla de forma cerrada; el modo host es una opción explícita. La automatización de navegador usa Node/Playwright en hosts compatibles. El control nativo usa herramientas X11 en Linux, AppleScript/CoreGraphics en macOS y PowerShell/Win32 en Windows, además de adaptadores JSON externos. Las APIs de escritorio nativo y el almacén de claves deben validarse en el host real con permisos concedidos por el usuario.

El repositorio incluye pruebas unitarias, de integración y seguridad, además de CI para ejecución nativa por SO. Las pruebas locales de Linux verifican operaciones reales de navegador, sandbox y X11. Las compilaciones cruzadas no demuestran comportamiento nativo en macOS/Windows. La aceptación con proveedores reales requiere credenciales configuradas. Consulta [pruebas y diagnóstico](pruebas.md) y [referencias técnicas](referencias.md).
