# Contrato de automatización

[Português](../pt-BR/automacao.md) · [English](../automation.md) · [Español](automatizacion.md) · [简体中文](../zh-CN/automation.md)

El navegador y el escritorio nativo son herramientas y permisos distintos. Permitir shell no concede ninguno de los dos. El harness `inspect` rechaza ambos incluso si sus permisos están habilitados.

## Solicitud de navegador

```json
{
  "origins": ["http://127.0.0.1:3000"],
  "steps": [
    {"action":"open","url":"http://127.0.0.1:3000"},
    {"action":"fill","selector":"#name","text":"Usuario sintético"},
    {"action":"click","selector":"button[type=submit]"},
    {"action":"expect_text","selector":"#result","text":"Guardado"},
    {"action":"screenshot"}
  ]
}
```

El modelo pasa el objeto serializado en la cadena `workflow` de la herramienta de navegador. La interfaz muestra el flujo completo antes de aprobar. Se admiten como máximo 30 pasos, con timeout de 10 segundos por localizador y un plazo total de dos minutos. Los orígenes deben coincidir exactamente en esquema/host/puerto; se deniegan rutas ajenas a ellos, incluidos redireccionamientos y subrecursos. Es una política de enrutamiento del navegador, no un sandbox de red del SO. Usa sitios de prueba confiables; no se afirma que toda actividad interna del navegador fuera de las peticiones enrutadas esté aislada. Los fallos devuelven un resultado de error explícito, nunca una aprobación inventada.

Una captura PNG solo existe en memoria transitoria. El agente elimina la imagen de la salida persistida, envía las imágenes recientes a un modelo compatible con imágenes y conserva únicamente un marcador en la sesión. Al reanudar hace falta una captura nueva. Un endpoint/modelo sin soporte para imágenes puede rechazar estas peticiones.

## Solicitud nativa

```json
{"action":"click","x":400,"y":220,"button":1}
```

El modelo pasa el objeto serializado como cadena `action` de la herramienta de escritorio. Ratón, teclado, portapapeles y capturas usan APIs/herramientas de la plataforma, no la herramienta shell. Cada acción tiene un plazo de 15 segundos. Esc/Ctrl+C cancela el worker activo, pero no deshace una entrada completada. Las acciones bloqueadas por el SO o no disponibles devuelven errores sin conceder privilegios adicionales.

En Linux los nombres de teclas usan sintaxis xdotool (`ctrl+l`, `Return`); Windows usa sintaxis SendKeys de Windows Forms; macOS utiliza códigos de tecla virtual de 0 a 127. Cerrar una ventana en Linux exige un ID numérico; en Windows/macOS se apunta a la ventana en primer plano. Abrir/enfocar requiere la entrada exacta de la lista permitida configurada. Las acciones globales de puntero/teclado afectan a todo el escritorio y se aprueban individualmente.

En Unix, cancelar termina los grupos de subprocesos de automatización en curso; las aplicaciones separadas intencionadamente siguen siendo estado persistente del usuario. En Windows se usa taskkill sobre el árbol de procesos. Estos mecanismos no deshacen acciones completadas ni garantizan terminar procesos separados deliberadamente. La herramienta no interactúa con diálogos de privilegios ni se concede permisos de Accesibilidad.

## Backend de escritorio externo

Configura un ejecutable local de confianza:

```json
{"automation":{"desktop":false,"desktop_backend":["/ruta/absoluta/a/mi-adaptador"],"applications":["mi-app"]}}
```

Tras habilitar el permiso de escritorio y aprobar la acción, el ejecutable recibe exactamente un objeto JSON por stdin. Debe devolver resultados observados por stdout, salir con código distinto de cero si falla, respetar la terminación y aplicar políticas adicionales del SO/aplicación. La salida está limitada a 8 MiB. Para devolver capturas usa `{"image":"BASE64_PNG","mime":"image/png"}`; se aplica el mismo tratamiento transitorio. No emitas credenciales ni contenido privado del portapapeles salvo que la acción aprobada lo requiera expresamente. Los argumentos adicionales son configuración fija del usuario, nunca texto de comando generado por el modelo.

Esta interfaz admite integraciones específicas de plataforma (por ejemplo UI Automation, frameworks de accesibilidad de macOS o herramientas Wayland autorizadas por el compositor) sin cambiar el núcleo del agente. Que exista el ejecutable solo verifica un requisito previo; cada acción aún comprueba resultados y permisos.
