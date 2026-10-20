# 自动化协议

[Português](../pt-BR/automacao.md) · [English](../automation.md) · [Español](../es/automatizacion.md) · [简体中文](automation.md)

浏览器和原生桌面是不同的工具与权限。允许 shell 不会自动授予其中任何一项。即使相关权限已开启，`inspect` harness 仍会拒绝这两类操作。

## 浏览器请求

```json
{
  "origins": ["http://127.0.0.1:3000"],
  "steps": [
    {"action":"open","url":"http://127.0.0.1:3000"},
    {"action":"fill","selector":"#name","text":"合成测试用户"},
    {"action":"click","selector":"button[type=submit]"},
    {"action":"expect_text","selector":"#result","text":"已保存"},
    {"action":"screenshot"}
  ]
}
```

模型会将序列化对象放入浏览器工具的 `workflow` 字符串。用户可以在审批前查看整个工作流。最多运行 30 个步骤，每个定位器的超时为 10 秒，总时限为两分钟。来源必须在 scheme/host/port 上完全匹配；不在允许范围内的路由（包括重定向和子资源）会被拒绝。这是浏览器路由策略，不是操作系统网络沙箱。请使用可信的测试站点；程序不声称隔离浏览器在已路由页面请求之外的所有内部活动。失败会返回明确的失败结果，绝不会伪造通过的断言。

PNG 截图仅保存在临时内存中。智能体会从持久化工具输出中移除图像，将最新图像发送给支持视觉的模型，并在会话中只保留占位标记。恢复会话后需要重新截图。不支持图像的 endpoint/model 可能会拒绝此类请求。

## 原生桌面请求

```json
{"action":"click","x":400,"y":220,"button":1}
```

模型将序列化对象作为桌面工具的 `action` 字符串传入。鼠标、键盘、剪贴板和截图使用平台 API/工具，不通过 shell 工具执行。单个操作限时 15 秒。用户可按 Esc/Ctrl+C 停止并取消当前 worker，但已完成的输入无法撤销。被操作系统阻止或不可用的动作会返回错误，而不会授予额外权限。

Linux 按键名使用 xdotool 语法（`ctrl+l`、`Return`）；Windows 使用 Windows Forms SendKeys 语法；macOS 使用 0–127 的虚拟按键代码。Linux 关闭窗口需要数字窗口 ID；Windows/macOS 目标为当前前台窗口。启动/聚焦要求与配置的允许列表完全匹配。全局指针/键盘动作会影响整个桌面，且逐项审批。

Unix 上取消操作会终止运行中的自动化子进程组；主动分离启动的应用仍属于用户的持久状态。Windows 通过 taskkill 终止进程树。上述机制无法撤销已经完成的动作，也不保证终止故意分离的主机进程。工具不会与操作系统权限提示交互，也不会为自身授予辅助功能权限。

## 外部桌面后端

配置一个可信的本地可执行程序：

```json
{"automation":{"desktop":false,"desktop_backend":["/绝对路径/我的适配器"],"applications":["my-app"]}}
```

启用桌面权限并批准动作后，该程序通过 stdin 接收且只接收一个 JSON 动作对象。它必须通过 stdout 返回观察到的结果，失败时以非零代码退出，遵守终止信号，并实施额外的 OS/应用策略。输出上限为 8 MiB。截图可用 `{"image":"BASE64_PNG","mime":"image/png"}` 返回，并采用同样的临时图像处理。除非经批准的操作明确需要，否则不得输出凭据或私人剪贴板内容。附加参数是用户固定配置，绝不是模型生成的命令文本。

该接口支持特定平台集成（例如 UI Automation、macOS 辅助功能框架或由 compositor 授权的 Wayland 工具），而无需修改智能体核心。可执行文件存在只代表满足前置条件；实际动作仍需验证结果与权限。
