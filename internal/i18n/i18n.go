// Package i18n provides the user-interface language catalog for OpenVibeCoding.
// Language tags identify UI translations; command names, flags, provider IDs and
// model IDs remain stable across locales.
package i18n

import (
	"os"
	"strings"
)

const Auto = "auto"

var supported = []string{"auto", "en", "pt-BR", "es", "zh-CN"}

var catalogs = map[string]map[string]string{
	"en": {
		"app.title": "OPENVIBECODING",
		"app.subtitle": " project agent",
		"input.placeholder": "Describe a task, or /help",
		"welcome.body": "OpenVibeCoding · review before action\nAsk me to inspect this project. /help lists commands.\nFile tools stay inside the workspace. Commands require approval and use OS isolation when available.",
		"approval.hint": "Ctrl+Y approves this operation · Ctrl+N rejects · PgUp/PgDn inspect diff",
		"view.help": "Enter send  Ctrl+J newline  /help  Esc stop",
		"view.review": "REVIEW · Ctrl+Y approve · Ctrl+N reject · PgUp/PgDn scroll",
		"help.body": "/model [provider/model] · /provider [name] · /harness [review|inspect]\n/sessions · /resume <id> · /new · /rename <title> · /delete <other-session-id>\n/init · /undo · /doctor\n/context · /cost · /permissions · /capabilities · /changes · /config · /status · /language · /clear · /exit\nEnter send · Ctrl+J multiline · Tab complete · Alt+↑/↓ history\nPgUp/PgDn scroll · Esc cancel · Ctrl+C cancel/exit\nApprovals: Ctrl+Y accept one operation · Ctrl+N reject",
		"language.usage": "Usage: /language [auto|en|pt-BR|es|zh-CN]",
		"language.options": "Available languages: auto (system), en (English), pt-BR (Português do Brasil), es (Español), zh-CN (简体中文)",
		"language.current": "Interface language: %s (resolved: %s)",
		"language.auto": "Automatic (system locale)",
		"language.changed": "Interface language saved: %s",
		"language.invalid": "Unsupported language %q. Choose auto, en, pt-BR, es, or zh-CN.",
		"command.unknown": "unknown command %q; use /help",
		"command.usage": "usage: %s (no arguments)",
		"model.usage": "usage: /model provider/model",
		"provider.unknown": "unknown provider",
		"undo.runtime": "undo needs the shared runtime change log",
		"session.workspace": "session belongs to another workspace; launch with --workspace %q",
		"session.deleteOther": "specify another session ID; start /new before deleting this session",
		"permissions.usage": "usage: /permissions browser|desktop|clipboard|network|install on|off",
		"permissions.unknown": "unknown permission",
		"error.configSave": "saving configuration: %v",
		"config.active": "Active %s/%s · harness %s\\n%s",
		"config.activeLabel": "Active: %s/%s · %s",
		"usage.new": "usage: /new",
		"usage.rename": "usage: /rename title",
		"session.renamed": "Renamed",
		"session.deleted": "Deleted %s",
		"context.summary": "~%d conservative byte-based tokens / %d configured limit; %d messages indexed as excerpts. Original history remains in SQLite; incremental structured summaries are prepared during requests, and recall retrieves original evidence.",
		"cost.summary": "Saved session: %d input / %d output tokens reported. Known estimated cost: $%.6f; %d requests lack configured prices. Requests without usage remain unknown.",
		"permissions.summary": "Harness %s; file changes reviewed individually. Host shell opt-in: %t. Default shell: isolated on supported Linux hosts, otherwise denied (opt-in: restart with --allow-unsandboxed-shell). Browser=%t Desktop=%t Clipboard=%t. Automation actions require approval. Network/install grants are visible in /config. Use /permissions browser|desktop|clipboard|network|install on|off.",
		"doctor.summary": "OpenVibeCoding %s\\nWorkspace: %s\\nProvider: %s/%s · tools %t\\nHarness: %s\\nCredential available: %t\\nHost shell opt-in: %t\\nOS sandbox: %s\\nNetwork grant: %t · install grant: %t\\nRuntime: shared authenticated service",
		"status.summary": "Session: %s\\nWorkspace: %s\\nProvider: %s/%s\\nHarness: %s\\nRuntime: shared authenticated service\\nState: %s",
		"setup.title": "OpenVibeCoding setup",
		"setup.connect": "Connect one model provider. Credentials go to your OS keychain; nothing is echoed.",
		"setup.exit": "Exit without changes",
		"setup.noChanges": "No changes made.",
		"setup.complete": "Setup complete. Next:",
		"setup.projectCommand": "cd <your project>",
		"setup.keyURL": "Create or copy an API key: %s",
		"setup.keyPrompt": "Paste the API key (hidden): ",
		"setup.emptyCredential": "empty credential; nothing was changed",
		"setup.storingCredential": "storing the credential",
		"setup.unknownProvider": "Unknown provider %q.",
		"setup.noModels": "Provider %q has no models configured.",
		"setup.testing": "Testing %s/%s with a real request...",
		"setup.smokeFailed": "Smoke test failed",
		"setup.activateAnyway": "Activate this provider anyway?",
		"setup.smokePassed": "Smoke test passed.",
		"setup.saving": "Saving configuration",
		"setup.activeProvider": "Active provider: %s/%s",
		"setup.opencodeFound": "Found OpenCode credentials at %s",
		"setup.noOpenCodeCredentials": "No importable OpenCode credentials found (expected google and/or nvidia).",
		"setup.importPrompt": "Import the %s credential into OpenVibeCoding?",
		"setup.importing": "Importing %s",
		"setup.imported": "Imported %s.",
		"setup.nothingImported": "Nothing imported.",
		"setup.choose": "Choose [1-%d] (0 to exit): ",
		"setup.invalidChoice": "Enter a number between 0 and %d",
		"setup.credentialTerminal": "credential input requires a terminal",
		"setup.readingCredential": "reading credential",
		"setup.ollamaTitle": "Ollama (running, %d model(s))",
		"setup.lmstudioTitle": "LM Studio (running, %d model(s))",
		"setup.localFreeDetail": "free, runs on this machine",
		"setup.openrouterDetail": "includes free models (:free); key at openrouter.ai/keys",
		"setup.groqDetail": "free developer tier with rate limits; key at console.groq.com/keys",
		"setup.geminiDetail": "AI Studio free tier; key at aistudio.google.com/apikey",
		"setup.zaiDetail": "free GLM Flash models; key at z.ai",
		"setup.nvidiaDetail": "free credits on build.nvidia.com",
		"setup.cerebrasDetail": "free trial tier; key at cloud.cerebras.ai",
		"setup.importDetail": "uses accounts already configured in OpenCode (google, nvidia)",
		"setup.ollamaTitleFallback": "Ollama",
		"setup.lmstudioTitleFallback": "LM Studio",
	},
	"pt-BR": {
		"app.title": "OPENVIBECODING",
		"app.subtitle": " agente de projetos",
		"input.placeholder": "Descreva uma tarefa ou use /help",
		"welcome.body": "OpenVibeCoding · revise antes de executar\nPeça para eu inspecionar este projeto. /help lista os comandos.\nAs ferramentas de arquivos ficam dentro do workspace. Comandos exigem aprovação e usam isolamento do sistema quando disponível.",
		"approval.hint": "Ctrl+Y aprova esta operação · Ctrl+N rejeita · PgUp/PgDn inspecionam o diff",
		"view.help": "Enter enviar  Ctrl+J nova linha  /help  Esc parar",
		"view.review": "REVISÃO · Ctrl+Y aprovar · Ctrl+N rejeitar · PgUp/PgDn rolar",
		"help.body": "/model [provedor/modelo] · /provider [nome] · /harness [review|inspect]\n/sessions · /resume <id> · /new · /rename <título> · /delete <id-de-outra-sessão>\n/init · /undo · /doctor\n/context · /cost · /permissions · /capabilities · /changes · /config · /status · /language · /clear · /exit\nEnter envia · Ctrl+J nova linha · Tab completa · Alt+↑/↓ histórico\nPgUp/PgDn rolam · Esc cancela · Ctrl+C cancela/sai\nAprovações: Ctrl+Y aceita uma operação · Ctrl+N rejeita",
		"language.usage": "Uso: /language [auto|en|pt-BR|es|zh-CN]",
		"language.options": "Idiomas: auto (sistema), en (inglês), pt-BR (português brasileiro), es (espanhol), zh-CN (chinês simplificado)",
		"language.current": "Idioma da interface: %s (resolvido: %s)",
		"language.auto": "Automático (idioma do sistema)",
		"language.changed": "Idioma da interface salvo: %s",
		"language.invalid": "Idioma não suportado %q. Escolha auto, en, pt-BR, es ou zh-CN.",
		"command.unknown": "comando desconhecido %q; use /help",
		"command.usage": "uso: %s (sem argumentos)",
		"model.usage": "uso: /model provedor/modelo",
		"provider.unknown": "provedor desconhecido",
		"undo.runtime": "desfazer exige o histórico de alterações do runtime compartilhado",
		"session.workspace": "a sessão pertence a outro workspace; inicie com --workspace %q",
		"session.deleteOther": "informe o ID de outra sessão; inicie /new antes de excluir esta sessão",
		"permissions.usage": "uso: /permissions browser|desktop|clipboard|network|install on|off",
		"permissions.unknown": "permissão desconhecida",
		"error.configSave": "salvando configuração: %v",
		"config.active": "Ativo %s/%s · modo %s\\n%s",
		"config.activeLabel": "Ativo: %s/%s · %s",
		"usage.new": "uso: /new",
		"usage.rename": "uso: /rename título",
		"session.renamed": "Renomeada",
		"session.deleted": "Excluída %s",
		"context.summary": "~%d tokens estimados por bytes / limite configurado de %d; %d mensagens indexadas como trechos. O histórico original permanece no SQLite; resumos estruturados incrementais são preparados durante as solicitações, e /recall recupera evidências originais.",
		"cost.summary": "Sessão salva: %d tokens de entrada / %d de saída informados. Custo estimado conhecido: $%.6f; %d solicitações não têm preços configurados. Solicitações sem dados de uso permanecem desconhecidas.",
		"permissions.summary": "Modo %s; alterações em arquivos são revisadas individualmente. Acesso shell ao host habilitado: %t. Shell padrão: isolado em Linux compatível; caso contrário, bloqueado (para habilitar, reinicie com --allow-unsandboxed-shell). Browser=%t Desktop=%t Clipboard=%t. Ações de automação exigem aprovação. Permissões de rede/instalação aparecem em /config. Use /permissions browser|desktop|clipboard|network|install on|off.",
		"doctor.summary": "OpenVibeCoding %s\\nWorkspace: %s\\nProvedor: %s/%s · ferramentas %t\\nModo: %s\\nCredencial disponível: %t\\nShell no host habilitado: %t\\nSandbox do sistema: %s\\nRede autorizada: %t · instalação autorizada: %t\\nRuntime: serviço compartilhado autenticado",
		"status.summary": "Sessão: %s\\nWorkspace: %s\\nProvedor: %s/%s\\nModo: %s\\nRuntime: serviço compartilhado autenticado\\nEstado: %s",
		"setup.title": "Configuração do OpenVibeCoding",
		"setup.connect": "Conecte um provedor de modelo. As credenciais ficam no chaveiro do sistema operacional; nada é exibido ao digitar.",
		"setup.exit": "Sair sem alterações",
		"setup.noChanges": "Nenhuma alteração realizada.",
		"setup.complete": "Configuração concluída. Próximos passos:",
		"setup.projectCommand": "cd <seu projeto>",
		"setup.keyURL": "Crie ou copie uma chave de API: %s",
		"setup.keyPrompt": "Cole a chave de API (entrada oculta): ",
		"setup.emptyCredential": "credencial vazia; nada foi alterado",
		"setup.storingCredential": "armazenando a credencial",
		"setup.unknownProvider": "Provedor desconhecido %q.",
		"setup.noModels": "O provedor %q não possui modelos configurados.",
		"setup.testing": "Testando %s/%s com uma solicitação real...",
		"setup.smokeFailed": "O teste de conexão falhou",
		"setup.activateAnyway": "Ativar este provedor mesmo assim?",
		"setup.smokePassed": "Teste de conexão aprovado.",
		"setup.saving": "Salvando configuração",
		"setup.activeProvider": "Provedor ativo: %s/%s",
		"setup.opencodeFound": "Credenciais do OpenCode encontradas em %s",
		"setup.noOpenCodeCredentials": "Nenhuma credencial importável do OpenCode encontrada (esperado: google e/ou nvidia).",
		"setup.importPrompt": "Importar a credencial %s para o OpenVibeCoding?",
		"setup.importing": "Importando %s",
		"setup.imported": "%s importado.",
		"setup.nothingImported": "Nada foi importado.",
		"setup.choose": "Escolha [1-%d] (0 para sair): ",
		"setup.invalidChoice": "Digite um número entre 0 e %d",
		"setup.credentialTerminal": "a entrada da credencial exige um terminal",
		"setup.readingCredential": "lendo credencial",
		"setup.ollamaTitle": "Ollama (em execução, %d modelo(s))",
		"setup.lmstudioTitle": "LM Studio (em execução, %d modelo(s))",
		"setup.localFreeDetail": "gratuito, executa nesta máquina",
		"setup.openrouterDetail": "inclui modelos gratuitos (:free); chave em openrouter.ai/keys",
		"setup.groqDetail": "plano gratuito para desenvolvedores com limites; chave em console.groq.com/keys",
		"setup.geminiDetail": "nível gratuito do AI Studio; chave em aistudio.google.com/apikey",
		"setup.zaiDetail": "modelos GLM Flash gratuitos; chave em z.ai",
		"setup.nvidiaDetail": "créditos gratuitos em build.nvidia.com",
		"setup.cerebrasDetail": "plano de teste gratuito; chave em cloud.cerebras.ai",
		"setup.importDetail": "usa contas já configuradas no OpenCode (google, nvidia)",
		"setup.ollamaTitleFallback": "Ollama",
		"setup.lmstudioTitleFallback": "LM Studio",
	},
	"es": {
		"app.title": "OPENVIBECODING",
		"app.subtitle": " agente de proyectos",
		"input.placeholder": "Describe una tarea o usa /help",
		"welcome.body": "OpenVibeCoding · revisa antes de actuar\nPídeme que inspeccione este proyecto. /help muestra los comandos.\nLas herramientas de archivos permanecen dentro del workspace. Los comandos requieren aprobación y usan aislamiento del sistema cuando está disponible.",
		"approval.hint": "Ctrl+Y aprueba esta operación · Ctrl+N la rechaza · PgUp/PgDn revisa el diff",
		"view.help": "Enter enviar  Ctrl+J nueva línea  /help  Esc detener",
		"view.review": "REVISIÓN · Ctrl+Y aprobar · Ctrl+N rechazar · PgUp/PgDn desplazarse",
		"help.body": "/model [proveedor/modelo] · /provider [nombre] · /harness [review|inspect]\n/sessions · /resume <id> · /new · /rename <título> · /delete <id-de-otra-sesión>\n/init · /undo · /doctor\n/context · /cost · /permissions · /capabilities · /changes · /config · /status · /language · /clear · /exit\nEnter envía · Ctrl+J nueva línea · Tab completa · Alt+↑/↓ historial\nPgUp/PgDn desplaza · Esc cancela · Ctrl+C cancela/sale\nAprobaciones: Ctrl+Y acepta una operación · Ctrl+N rechaza",
		"language.usage": "Uso: /language [auto|en|pt-BR|es|zh-CN]",
		"language.options": "Idiomas: auto (sistema), en (inglés), pt-BR (portugués brasileño), es (español), zh-CN (chino simplificado)",
		"language.current": "Idioma de la interfaz: %s (resuelto: %s)",
		"language.auto": "Automático (idioma del sistema)",
		"language.changed": "Idioma de la interfaz guardado: %s",
		"language.invalid": "Idioma no compatible %q. Elige auto, en, pt-BR, es o zh-CN.",
		"command.unknown": "comando desconocido %q; usa /help",
		"command.usage": "uso: %s (sin argumentos)",
		"model.usage": "uso: /model proveedor/modelo",
		"provider.unknown": "proveedor desconocido",
		"undo.runtime": "deshacer requiere el registro de cambios del runtime compartido",
		"session.workspace": "la sesión pertenece a otro workspace; inicia con --workspace %q",
		"session.deleteOther": "indica otro ID de sesión; inicia /new antes de eliminar esta sesión",
		"permissions.usage": "uso: /permissions browser|desktop|clipboard|network|install on|off",
		"permissions.unknown": "permiso desconocido",
		"error.configSave": "guardando la configuración: %v",
		"config.active": "Activo %s/%s · modo %s\\n%s",
		"config.activeLabel": "Activo: %s/%s · %s",
		"usage.new": "uso: /new",
		"usage.rename": "uso: /rename título",
		"session.renamed": "Renombrada",
		"session.deleted": "Eliminada %s",
		"context.summary": "~%d tokens estimados por bytes / límite configurado de %d; %d mensajes indexados como extractos. El historial original permanece en SQLite; los resúmenes estructurados incrementales se preparan durante las solicitudes y /recall recupera la evidencia original.",
		"cost.summary": "Sesión guardada: %d tokens de entrada / %d de salida informados. Coste estimado conocido: $%.6f; %d solicitudes no tienen precios configurados. Las solicitudes sin datos de uso siguen siendo desconocidas.",
		"permissions.summary": "Modo %s; los cambios en archivos se revisan individualmente. Shell con acceso al host habilitado: %t. Shell predeterminado: aislado en Linux compatible; de lo contrario, bloqueado (para habilitarlo, reinicia con --allow-unsandboxed-shell). Browser=%t Desktop=%t Clipboard=%t. Las acciones de automatización requieren aprobación. Los permisos de red/instalación aparecen en /config. Usa /permissions browser|desktop|clipboard|network|install on|off.",
		"doctor.summary": "OpenVibeCoding %s\\nWorkspace: %s\\nProveedor: %s/%s · herramientas %t\\nModo: %s\\nCredencial disponible: %t\\nShell en el host habilitado: %t\\nSandbox del sistema: %s\\nRed autorizada: %t · instalación autorizada: %t\\nRuntime: servicio compartido autenticado",
		"status.summary": "Sesión: %s\\nWorkspace: %s\\nProveedor: %s/%s\\nModo: %s\\nRuntime: servicio compartido autenticado\\nEstado: %s",
		"setup.title": "Configuración de OpenVibeCoding",
		"setup.connect": "Conecta un proveedor de modelos. Las credenciales se guardan en el llavero del sistema operativo; no se muestra lo que escribes.",
		"setup.exit": "Salir sin cambios",
		"setup.noChanges": "No se realizaron cambios.",
		"setup.complete": "Configuración completada. Siguientes pasos:",
		"setup.projectCommand": "cd <tu proyecto>",
		"setup.keyURL": "Crea o copia una clave de API: %s",
		"setup.keyPrompt": "Pega la clave de API (entrada oculta): ",
		"setup.emptyCredential": "credencial vacía; no se cambió nada",
		"setup.storingCredential": "guardando la credencial",
		"setup.unknownProvider": "Proveedor desconocido %q.",
		"setup.noModels": "El proveedor %q no tiene modelos configurados.",
		"setup.testing": "Probando %s/%s con una solicitud real...",
		"setup.smokeFailed": "Falló la prueba de conexión",
		"setup.activateAnyway": "¿Activar este proveedor de todos modos?",
		"setup.smokePassed": "Prueba de conexión superada.",
		"setup.saving": "Guardando la configuración",
		"setup.activeProvider": "Proveedor activo: %s/%s",
		"setup.opencodeFound": "Se encontraron credenciales de OpenCode en %s",
		"setup.noOpenCodeCredentials": "No se encontraron credenciales importables de OpenCode (se esperaba google y/o nvidia).",
		"setup.importPrompt": "¿Importar la credencial %s a OpenVibeCoding?",
		"setup.importing": "Importando %s",
		"setup.imported": "Se importó %s.",
		"setup.nothingImported": "No se importó nada.",
		"setup.choose": "Elige [1-%d] (0 para salir): ",
		"setup.invalidChoice": "Introduce un número entre 0 y %d",
		"setup.credentialTerminal": "introducir credenciales requiere una terminal",
		"setup.readingCredential": "leyendo la credencial",
		"setup.ollamaTitle": "Ollama (en ejecución, %d modelo(s))",
		"setup.lmstudioTitle": "LM Studio (en ejecución, %d modelo(s))",
		"setup.localFreeDetail": "gratis, se ejecuta en este equipo",
		"setup.openrouterDetail": "incluye modelos gratuitos (:free); clave en openrouter.ai/keys",
		"setup.groqDetail": "plan gratuito para desarrolladores con límites; clave en console.groq.com/keys",
		"setup.geminiDetail": "nivel gratuito de AI Studio; clave en aistudio.google.com/apikey",
		"setup.zaiDetail": "modelos GLM Flash gratuitos; clave en z.ai",
		"setup.nvidiaDetail": "créditos gratuitos en build.nvidia.com",
		"setup.cerebrasDetail": "nivel de prueba gratuito; clave en cloud.cerebras.ai",
		"setup.importDetail": "usa cuentas ya configuradas en OpenCode (google, nvidia)",
		"setup.ollamaTitleFallback": "Ollama",
		"setup.lmstudioTitleFallback": "LM Studio",
	},
	"zh-CN": {
		"app.title": "OPENVIBECODING",
		"app.subtitle": " 项目编程代理",
		"input.placeholder": "描述任务，或输入 /help",
		"welcome.body": "OpenVibeCoding · 操作前先审查\n请让我检查此项目。输入 /help 查看命令。\n文件工具仅在工作区内操作。命令需要批准，并在可用时使用操作系统隔离。",
		"approval.hint": "Ctrl+Y 批准此操作 · Ctrl+N 拒绝 · PgUp/PgDn 查看 diff",
		"view.help": "Enter 发送  Ctrl+J 换行  /help  Esc 停止",
		"view.review": "审查 · Ctrl+Y 批准 · Ctrl+N 拒绝 · PgUp/PgDn 滚动",
		"help.body": "/model [provider/model] · /provider [name] · /harness [review|inspect]\n/sessions · /resume <id> · /new · /rename <title> · /delete <other-session-id>\n/init · /undo · /doctor\n/context · /cost · /permissions · /capabilities · /changes · /config · /status · /language · /clear · /exit\nEnter 发送 · Ctrl+J 换行 · Tab 补全 · Alt+↑/↓ 历史记录\nPgUp/PgDn 滚动 · Esc 取消 · Ctrl+C 取消/退出\n审批：Ctrl+Y 批准单项操作 · Ctrl+N 拒绝",
		"language.usage": "用法：/language [auto|en|pt-BR|es|zh-CN]",
		"language.options": "可用语言：auto（系统）、en（英语）、pt-BR（巴西葡萄牙语）、es（西班牙语）、zh-CN（简体中文）",
		"language.current": "界面语言：%s（当前解析为：%s）",
		"language.auto": "自动（系统语言）",
		"language.changed": "已保存界面语言：%s",
		"language.invalid": "不支持的语言 %q。请选择 auto、en、pt-BR、es 或 zh-CN。",
		"command.unknown": "未知命令 %q；请使用 /help",
		"command.usage": "用法：%s（不接受参数）",
		"model.usage": "用法：/model provider/model",
		"provider.unknown": "未知服务提供商",
		"undo.runtime": "撤销操作需要共享运行时的变更记录",
		"session.workspace": "此会话属于其他工作区；请使用 --workspace %q 启动",
		"session.deleteOther": "请指定其他会话 ID；删除此会话前请先运行 /new",
		"permissions.usage": "用法：/permissions browser|desktop|clipboard|network|install on|off",
		"permissions.unknown": "未知权限",
		"error.configSave": "保存配置时出错：%v",
		"config.active": "当前模型 %s/%s · 模式 %s\\n%s",
		"config.activeLabel": "当前：%s/%s · %s",
		"usage.new": "用法：/new",
		"usage.rename": "用法：/rename 标题",
		"session.renamed": "已重命名",
		"session.deleted": "已删除 %s",
		"context.summary": "约 %d 个按字节估算的 token / 配置上限 %d；%d 条消息以摘录形式索引。原始历史保存在 SQLite 中；请求期间会生成增量结构化摘要，/recall 可检索原始证据。",
		"cost.summary": "已保存会话：报告了 %d 个输入 token / %d 个输出 token。已知估算费用：$%.6f；%d 个请求没有配置价格。未提供用量的请求仍标记为未知。",
		"permissions.summary": "模式 %s；文件修改逐项审查。允许主机 Shell：%t。默认 Shell：支持的 Linux 上使用隔离；其他情况拒绝执行（如需启用，请使用 --allow-unsandboxed-shell 重启）。Browser=%t Desktop=%t Clipboard=%t。自动化操作需要批准。网络/安装授权显示在 /config 中。使用 /permissions browser|desktop|clipboard|network|install on|off。",
		"doctor.summary": "OpenVibeCoding %s\\n工作区：%s\\n服务提供商：%s/%s · 工具 %t\\n模式：%s\\n凭据可用：%t\\n允许主机 Shell：%t\\n系统沙箱：%s\\n网络授权：%t · 安装授权：%t\\n运行时：共享认证服务",
		"status.summary": "会话：%s\\n工作区：%s\\n服务提供商：%s/%s\\n模式：%s\\n运行时：共享认证服务\\n状态：%s",
		"setup.title": "OpenVibeCoding 配置向导",
		"setup.connect": "连接一个模型服务提供商。凭据将保存到操作系统密钥库；输入内容不会显示。",
		"setup.exit": "退出且不做更改",
		"setup.noChanges": "未进行任何更改。",
		"setup.complete": "配置完成。接下来：",
		"setup.projectCommand": "cd <你的项目目录>",
		"setup.keyURL": "请创建或复制 API 密钥：%s",
		"setup.keyPrompt": "粘贴 API 密钥（输入隐藏）：",
		"setup.emptyCredential": "凭据为空；未进行任何更改",
		"setup.storingCredential": "正在保存凭据",
		"setup.unknownProvider": "未知服务提供商 %q。",
		"setup.noModels": "服务提供商 %q 未配置模型。",
		"setup.testing": "正在发送真实请求测试 %s/%s...",
		"setup.smokeFailed": "连接测试失败",
		"setup.activateAnyway": "仍要启用此服务提供商吗？",
		"setup.smokePassed": "连接测试通过。",
		"setup.saving": "正在保存配置",
		"setup.activeProvider": "当前服务提供商：%s/%s",
		"setup.opencodeFound": "在 %s 找到了 OpenCode 凭据",
		"setup.noOpenCodeCredentials": "未找到可导入的 OpenCode 凭据（预期包含 google 和/或 nvidia）。",
		"setup.importPrompt": "要将 %s 凭据导入 OpenVibeCoding 吗？",
		"setup.importing": "正在导入 %s",
		"setup.imported": "已导入 %s。",
		"setup.nothingImported": "未导入任何内容。",
		"setup.choose": "请选择 [1-%d]（输入 0 退出）：",
		"setup.invalidChoice": "请输入 0 到 %d 之间的数字",
		"setup.credentialTerminal": "输入凭据需要交互式终端",
		"setup.readingCredential": "读取凭据时出错",
		"setup.ollamaTitle": "Ollama（运行中，%d 个模型）",
		"setup.lmstudioTitle": "LM Studio（运行中，%d 个模型）",
		"setup.localFreeDetail": "免费，在本机运行",
		"setup.openrouterDetail": "包含免费模型（:free）；密钥页面：openrouter.ai/keys",
		"setup.groqDetail": "开发者免费层（有速率限制）；密钥页面：console.groq.com/keys",
		"setup.geminiDetail": "AI Studio 免费层；密钥页面：aistudio.google.com/apikey",
		"setup.zaiDetail": "免费 GLM Flash 模型；密钥页面：z.ai",
		"setup.nvidiaDetail": "build.nvidia.com 提供免费额度",
		"setup.cerebrasDetail": "免费试用层；密钥页面：cloud.cerebras.ai",
		"setup.importDetail": "使用已在 OpenCode 中配置的账户（google、nvidia）",
		"setup.ollamaTitleFallback": "Ollama",
		"setup.lmstudioTitleFallback": "LM Studio",
	},
}

// Normalize validates and canonicalizes a user-selected language tag.
func Normalize(tag string) (string, bool) {
	tag = strings.TrimSpace(tag)
	// Locale variables commonly include an encoding or modifier, e.g. pt_BR.UTF-8.
	if cut := strings.IndexAny(tag, ".@"); cut >= 0 {
		tag = tag[:cut]
	}
	switch strings.ToLower(strings.ReplaceAll(tag, "_", "-")) {
	case "", "auto", "system":
		return Auto, true
	case "en", "en-us", "en-gb":
		return "en", true
	case "pt", "pt-br", "pt-pt":
		return "pt-BR", true
	case "es", "es-es", "es-mx", "es-ar":
		return "es", true
	case "zh", "zh-cn", "zh-hans", "zh-hans-cn":
		return "zh-CN", true
	default:
		return "", false
	}
}

// Resolve selects a concrete catalog for a saved preference. Auto mode follows
// common system locale environment variables, falling back to English.
func Resolve(preference string) string {
	normalized, ok := Normalize(preference)
	if !ok {
		normalized = Auto
	}
	if normalized != Auto {
		return normalized
	}
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if value := os.Getenv(key); value != "" {
			if language, ok := Normalize(value); ok && language != Auto {
				return language
			}
		}
	}
	return "en"
}

// T returns a translated message. Unknown keys fall back to English, then to
// the key itself so missing catalog entries remain visible during development.
func T(preference, key string) string {
	lang := Resolve(preference)
	if value, ok := catalogs[lang][key]; ok {
		return value
	}
	if value, ok := catalogs["en"][key]; ok {
		return value
	}
	return key
}

// Supported returns a defensive copy of the selectable language tags.
func Supported() []string {
	return append([]string(nil), supported...)
}

// Name returns a localized display label for a language tag.
func Name(preference string) string {
	normalized, ok := Normalize(preference)
	if !ok {
		return T("en", "language.current")
	}
	switch normalized {
	case Auto:
		return T(preference, "language.auto")
	case "en":
		return "English"
	case "pt-BR":
		return "Português do Brasil"
	case "es":
		return "Español"
	case "zh-CN":
		return "简体中文"
	default:
		return normalized
	}
}
