# Seguridad y garantías de ejecución

[Português](../pt-BR/seguranca.md) · [English](../security.md) · [Español](seguridad.md) · [简体中文](../zh-CN/security.md)

## Herramientas de archivos del workspace

`os.Root` dificulta escapar de la raíz del workspace. Se rechazan rutas absolutas, componentes `..` y componentes de enlaces simbólicos. Las ubicaciones conocidas de secretos (`.env*`, `.ssh`, `.aws`, nombres de credenciales y extensiones de claves privadas), directorios generados y coincidencias de `.gitignore` quedan excluidos de las herramientas ordinarias de lectura/búsqueda/listado. Las lecturas aceptan archivos de texto UTF-8 regulares de hasta 128 KiB. Es una política de exclusión/ignore, no un detector universal de secretos. El manejo de exclusiones anidadas es conservador: una excepción dentro del árbol no reabre una ruta excluida por un ancestro.

Una propuesta no escribe datos. Cada sustitución/eliminación muestra un diff unificado; cada movimiento indica origen/destino y muestra el contenido afectado. La aprobación se aplica a una sola operación. El digest del archivo original se verifica de nuevo antes de modificarlo. Las sustituciones usan archivos temporales exclusivos, fsync y rename; los movimientos crean el destino de forma exclusiva antes de eliminar el origen. Si no se puede eliminar el origen, se informa del resultado parcial. Se pueden crear directorios padre dentro de la raíz; un movimiento nunca sobrescribe un destino no relacionado.

Esta frontera protege contra rutas proporcionadas por el modelo, no contra procesos locales hostiles concurrentes, hard links o sistemas de archivos montados y controlados por el mismo usuario del SO. Verificar el digest no equivale a un compare-and-swap del sistema de archivos completo; evita ediciones externas simultáneas durante la aplicación. La exclusión por workspace del runtime evita que dos turnos del agente modifiquen el mismo workspace a la vez.

## Aislamiento de procesos en Linux

La ejecución predeterminada de shell usa Bubblewrap y falla de forma cerrada si no está disponible o la política del host lo deniega. El sandbox expone `/usr` y los directorios de runtime del sistema necesarios en solo lectura, un `/tmp` efímero, dispositivos limitados y el proyecto seleccionado en `/workspace`. El home del host y `/proc` no están disponibles. Las entradas secretas protegidas dentro del proyecto se enmascaran; se mantienen dependencias para permitir builds. Se solicitan espacios de nombres separados, una sesión nueva y terminación cuando termina el proceso padre. No se afirma que mitigue vulnerabilidades del kernel ni canales laterales.

La red está aislada por defecto. Un permiso explícito de red comparte la red del host y monta archivos necesarios de resolución/certificados; la revisión indica que esa ejecución tiene red. Un permiso de instalación habilita la herramienta de instalación de paquetes. El shell general puede realizar operaciones equivalentes dentro de los límites permitidos de filesystem/red; las categorías de herramientas por sí solas no aplican una política semántica a cualquier programa.

No hay un sandbox de procesos integrado en macOS o Windows. Las solicitudes que requieren confinamiento fallan de forma cerrada en esas plataformas. `--allow-unsandboxed-shell` habilita deliberadamente la ejecución como usuario del host en cualquier plataforma, con aprobación por comando y una advertencia explícita. Puede acceder a rutas del host, red y aplicaciones instaladas. No lo habilites si necesitas una barrera fuerte del SO. No se elevan privilegios automáticamente.

Los procesos hijos reciben una pequeña lista permitida de variables de entorno en lugar de las credenciales del proveedor. Aun así, el modo host puede leer credenciales en archivos locales. Los comandos tienen un límite de 60 segundos y salida stdout/stderr acotada. En Unix, cancelar el shell del host termina su grupo de procesos; los workers de automatización también usan grupos. Windows usa taskkill para el árbol del proceso. Los procesos separados intencionalmente y las acciones ya completadas pueden no ser reversibles o terminables. Son mecanismos de cancelación, no aislamiento del SO en macOS/Windows.

## Runtime

Un token aleatorio de 256 bits autentica el IPC de loopback. El archivo es privado, nunca se muestra y los clientes rechazan direcciones/redirecciones fuera de loopback. Las solicitudes están limitadas, se validan métodos/rutas y se rechazan peticiones de navegador cross-origin. Los bloqueos del SO evitan daemons duplicados y se liberan tras un fallo. Los metadatos solo se sustituyen después de adquirir el bloqueo. Las incompatibilidades del protocolo fallan en vez de reutilizar estados incompatibles.

Los identificadores de aprobación son aleatorios, de un solo uso y vinculados a la identidad del cliente que originó la solicitud. Solo el runtime puede modificar el almacenamiento; se rechazan escrituras directas del cliente de mensajes/resúmenes/uso. Las sesiones activas no se pueden borrar, recuperar, renombrar ni exportar. Los turnos se serializan por sesión y workspace. Desconectar un cliente cancela su solicitud, pero los demás continúan. La detención del runtime se rechaza si hay tareas activas; la terminación del SO las cancela antes del cierre del almacenamiento.

El usuario local autenticado puede configurar permisos y endpoints. La autenticación no vuelve confiable un comando generado por IA: toda mutación, comando shell y acción de automatización requiere aprobación, y el modo `inspect` los rechaza.

## Secretos, auditoría y contexto

Las credenciales provienen de variables de entorno, del almacén del SO o de un fallback explícitamente elegido en archivo. Se rechazan archivos fallback Unix con permisos para grupo/otros. Los archivos nuevos usan 0600 y los directorios 0700; Windows depende de ACL del perfil. No se incluye ningún secreto en la configuración JSON. Los valores de credenciales conocidos y controles de terminal se redactan antes de mostrarse/guardarse. No es posible detectar con fiabilidad secretos arbitrarios de aplicaciones: no pidas a las herramientas que los impriman. Las sesiones no se cifran en reposo.

SQLite registra propuestas/aprobaciones antes de ejecutar. Un fallo de persistencia detiene la operación. Se auditan los inicios/finales de tareas y cambios de configuración. Un fallo entre la mutación y el resultado duradero produce un mensaje de recuperación con resultado desconocido; nunca reproduce automáticamente la mutación. Los logs del runtime contienen metadatos operativos, no cuerpos de solicitudes ni datos de capturas.

El historial original es la fuente autoritativa. Los resúmenes estructurados conservan IDs de origen y no son permisos. `recall` recupera originales en fragmentos limitados. Las capturas son transitorias: solo se guarda un marcador y la imagen se envía al modelo configurado durante la tarea actual. El usuario debe autorizar la captura, elegir un modelo compatible con imágenes y tener en cuenta la información visible.

## Navegador y escritorio nativo

Navegador y escritorio son herramientas separadas y requieren permisos/aprobaciones independientes. El portapapeles necesita un permiso adicional. El worker Playwright usa un perfil nuevo y una política de enrutamiento por origen exacto; bloquea descargas, service workers y WebSockets, limita pasos/tiempo/salida e informa de fallos observados. El enrutamiento no es aislamiento de red a nivel del SO.

La entrada nativa actúa en el escritorio visible del usuario. Abrir/enfocar aplicaciones usa una lista permitida; los gestos globales de ratón/teclado no quedan confinados a un proceso. El usuario revisa el destino/acción antes de aprobar. Se respetan los permisos de Accesibilidad/Grabación de pantalla, escritorios bloqueados/UAC y restricciones del compositor. Wayland no está soportado por el backend X11 integrado de Linux. Los adaptadores externos son configuración confiable del usuario y deben imponer su propia política del SO. Consulta el [contrato de automatización](automatizacion.md).
