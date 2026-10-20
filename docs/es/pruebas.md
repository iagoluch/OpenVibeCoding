# Validación y solución de problemas

[Português](../pt-BR/testes.md) · [English](../testing.md) · [Español](pruebas.md) · [简体中文](../zh-CN/testing.md)

## Comprobaciones automatizadas

```sh
go test ./...
go test -race ./...
go vet ./...
bash scripts/package.sh
```

La cobertura incluye protocolos de streaming nativos/compatibles, fragmentos de herramientas/uso/errores/redirecciones, validación de endpoints, permisos/eliminación de credenciales fallback, persistencia/recuperación SQLite, resúmenes incrementales sin repetición, aprobación antes de escritura, autenticación del runtime, prevención de instancias duplicadas, exclusión de sesiones, exportación, límites de ignore/rutas/enlaces simbólicos, diffs obsoletos, creación/movimiento/borrado, código de salida/cancelación de comandos, validación de comandos slash y cierre de la interfaz.

Las pruebas de integración del entorno requieren fixtures explícitas:

```sh
# Instala primero los requisitos del navegador:
ovc --setup-browser
OPENVIBECODING_TEST_PLAYWRIGHT=/ruta/a/datos/qa/node_modules/playwright go test ./internal/automation -run TestRealBrowserWorkflow
OPENVIBECODING_TEST_SANDBOX=1 go test ./internal/workspace -run TestSandboxBoundary
# Usa solo un escritorio de pruebas aislado, nunca tu sesión de trabajo normal:
DISPLAY=:93 OPENVIBECODING_TEST_DESKTOP=1 go test ./internal/automation -run TestRealDesktop
```

La prueba real de navegador rellena un formulario sintético, hace clic, verifica el resultado visible, captura una imagen y confirma el bloqueo de un origen no autorizado. La prueba real de escritorio mueve/hace clic/desplaza el puntero, captura la pantalla y escribe en un proceso xterm sintético; verifica la entrada mediante un archivo fixture. Xvfb necesita `-noreset` para que el servidor no reinicie el estado del puntero entre procesos.

La prueba de sandbox verifica el acceso al proyecto seleccionado, deniega un archivo fixture externo, oculta `.env` y observa una escritura en el workspace. La política de namespaces del host puede impedir Bubblewrap; hay que registrarlo como no disponible y no recurrir silenciosamente al host.

## Evidencia local y límites

Durante el desarrollo, todos los paquetes pasaron las pruebas unitarias/de integración y el detector de condiciones de carrera. Pasaron las pruebas reales de navegador Linux, Bubblewrap y X11. Dos clientes se conectaron al mismo daemon; cerrar uno no interrumpió al otro. Las fixtures cubrieron OpenAI, Anthropic y Gemini. La generación de archivos compiló seis destinos con el lanzador `ovc`.

Una primera prueba X11 falló porque Xvfb se reiniciaba cuando los clientes se desconectaban; `-noreset` corrigió la fixture y la prueba real de ratón/teclado pasó. Playwright detecta Amazon Linux como distribución no compatible y descarga una versión fallback de Ubuntu; el flujo de Chromium se ejecutó correctamente en ese entorno.

No se afirma haber ejecutado peticiones pagadas a proveedores externos ni pruebas nativas de GUI/almacén de claves en macOS/Windows. Los adaptadores de proveedores se probaron con respuestas HTTP/SSE sintéticas. Que CI esté configurado para pruebas nativas no significa que una ejecución remota concreta haya pasado; consulta los resultados registrados.

## Aceptación con proveedor real

1. Crea un proyecto desechable con README y datos sintéticos. Guarda credenciales mediante `ovc --credential <provider>`; no las pegues en el proyecto/sesión.
2. Ejecuta `--doctor`, abre con `--workspace` y selecciona `/model provider/model`.
3. Pide inspeccionar README y crear `hello.txt`. Rechaza una propuesta y verifica que el archivo no existe; aprueba otra y revisa el archivo real.
4. Aprueba un comando de prueba inocuo y verifica la salida y el código de salida no cero deliberado. Usa aislamiento si está disponible; el modo host requiere opt-in explícito.
5. Sal, reanuda y verifica la transcripción, `/changes` y `/cost` guardado.
6. Repite con Anthropic/Gemini y un endpoint compatible. Confirma que las respuestas nativas de herramientas continúan coherentemente después de reanudar/cambiar el modelo.
7. Con modelos compatibles con imágenes, habilita navegador/escritorio, aprueba una captura y verifica que el modelo se base en evidencia observada. Al reanudar, confirma que solicita una nueva captura en vez de inventar detalles antiguos.

## Solución de problemas

- **Inicio del runtime:** ejecuta `ovc --runtime serve` en un terminal para ver errores de configuración. `--runtime status` comprueba autenticación. No borres un archivo de bloqueo activo; la propiedad depende de un bloqueo del SO. Después de un fallo, vuelve a conectar normalmente.
- **Credenciales/configuración modificadas:** detén el runtime inactivo y reinícialo. Los cambios mediante comandos slash de modelo/harness/permisos se aplican en el siguiente turno.
- **Almacén de claves no disponible:** desbloquea/configura el servicio de credenciales o elige explícitamente `--credential-file`. Linux requiere una sesión Secret Service. No debilites los permisos de archivos.
- **Errores HTTP/protocolo del proveedor:** verifica cuenta, endpoint, modelo y protocolo; los cuerpos de error se omiten intencionadamente. Desactiva `stream_usage` para proveedores compatibles que no lo admiten. Los modelos solo de texto deben usar `tools:false` y no sirven para flujos de capturas.
- **Falló la preparación de contexto:** los originales siguen guardados. Revisa/exporta la sesión e inicia una tarea más pequeña si el turno excede el presupuesto. Los resúmenes inválidos se rechazan.
- **Navegador no disponible:** ejecuta `--setup-browser`; comprueba Node/npm y bibliotecas de Chromium. Las descargas necesitan red y los orígenes permitidos deben incluir subrecursos.
- **Escritorio no disponible:** comprueba `/capabilities`, DISPLAY/sesión del SO y permisos del usuario. X11 requiere xdotool/ImageMagick/xclip. Windows necesita escritorio interactivo y permite el script PowerShell bajo su política vigente. macOS requiere Accesibilidad/Grabación de pantalla.
- **Cancelación:** Esc/Ctrl+C cancela la acción/worker. Inspecciona el destino; clics, envíos, texto y efectos externos ya completados no se pueden deshacer mediante cancelación.

## Copias de seguridad

Para exportar una sesión inactiva: `ovc --export ID > session.json`. Para un respaldo completo restaurable, detén el runtime y copia todo el directorio de la aplicación, incluidos los archivos auxiliares SQLite si existen. Restaura primero en un `OPENVIBECODING_HOME` separado e inspecciónalo. Nunca copies un archivo SQLite activo por separado.
