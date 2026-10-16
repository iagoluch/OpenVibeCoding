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
