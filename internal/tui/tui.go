package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/iagoluch/OpenVibeCoding/internal/agent"
	"github.com/iagoluch/OpenVibeCoding/internal/automation"
	"github.com/iagoluch/OpenVibeCoding/internal/provider"
)

var accent = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD")).Bold(true)
var muted = lipgloss.NewStyle().Foreground(lipgloss.Color("#8B92A5"))
var warning = lipgloss.NewStyle().Foreground(lipgloss.Color("#F1C879"))

type done struct{ err error }
type Model struct {
	finished      chan struct{}
	Engine        *agent.Engine
	input         textarea.Model
	output        viewport.Model
	lines         []string
	stream        string
	width, height int
	busy          bool
	status        string
	cancel        context.CancelFunc
	events        chan tea.Msg
	approval      chan bool
	history       []string
	historyIndex  int
	usage         provider.Usage
}

func New(e *agent.Engine) *Model {
	input := textarea.New()
	input.Placeholder = "Describe a task, or /help"
	input.ShowLineNumbers = false
	input.CharLimit = 16000
	input.SetHeight(3)
	input.Focus()
	v := viewport.New()
	v.SoftWrap = true
	m := &Model{Engine: e, input: input, output: v, width: 80, height: 24, status: "idle"}
	m.resize()
	m.restore()
	return m
}
func (m *Model) Init() tea.Cmd { return nil }
func (m *Model) restore() {
	m.lines = nil
	entries, e := m.Engine.Store.History(m.Engine.Session)
	if e != nil {
		m.add("error", e.Error())
		return
	}
	for _, v := range entries {
		m.add(v.Message.Role, v.Message.Content)
	}
	if len(entries) == 0 {
		m.add("welcome", "OpenVibeCoding · review before action\nAsk me to inspect this project. /help lists commands.\nFile tools stay inside the workspace. Commands require approval and use OS isolation when available.")
	}
}
func (m *Model) add(role, text string) {
	m.lines = append(m.lines, accent.Render(strings.ToUpper(role))+"\n"+m.Engine.Config.Redact(text))
	m.refresh(true)
}
func (m *Model) refresh(bottom bool) {
	s := strings.Join(m.lines, "\n\n")
	if m.stream != "" {
		s += "\n\n" + accent.Render("ASSISTANT") + "\n" + m.Engine.Config.Redact(m.stream)
	}
	m.output.SetContent(s)
	if bottom {
		m.output.GotoBottom()
	}
}
func (m *Model) resize() {
	m.input.SetWidth(max(10, m.width-2))
	m.output.SetWidth(max(10, m.width-2))
	m.output.SetHeight(max(1, m.height-9))
	m.refresh(false)
}
func wait(ch <-chan tea.Msg) tea.Cmd { return func() tea.Msg { return <-ch } }
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = v.Width
		m.height = v.Height
		m.resize()
		return m, nil
	case agent.Event:
		switch v.Kind {
		case "delta":
			bottom := m.output.AtBottom()
			m.stream += v.Text
			m.refresh(bottom)
		case "response":
			m.stream = ""
			m.usage.Input += v.Usage.Input
			m.usage.Output += v.Usage.Output
			m.add("assistant", v.Text)
		case "approval":
			m.approval = v.Decision
			m.status = "approval required"
			m.add("review", v.Text+"\n\nCtrl+Y approves this operation · Ctrl+N rejects · PgUp/PgDn inspect diff")
		case "status":
			m.status = v.Text
		default:
			m.add(v.Kind, v.Text)
		}
		return m, wait(m.events)
	case done:
		m.busy = false
		m.approval = nil
		m.status = "idle"
		if m.cancel != nil {
			m.cancel()
			m.cancel = nil
		}
		if v.err != nil {
			m.stream = ""
			m.add("error", v.err.Error())
		}
		return m, nil
	case tea.KeyPressMsg:
		switch v.String() {
		case "ctrl+c":
			if m.busy {
				m.cancel()
				m.status = "cancelling"
				return m, nil
			}
			return m, tea.Quit
		case "esc":
			if m.busy {
				m.cancel()
				m.status = "cancelling"
			}
			return m, nil
		case "ctrl+y", "ctrl+n":
			if m.approval != nil {
				m.approval <- v.String() == "ctrl+y"
				m.approval = nil
				m.status = "running"
			}
			return m, nil
		case "pgup":
			m.output.PageUp()
			return m, nil
		case "pgdown":
			m.output.PageDown()
			return m, nil
		case "alt+up":
			if len(m.history) > 0 {
				m.historyIndex = max(0, m.historyIndex-1)
				m.input.SetValue(m.history[m.historyIndex])
			}
			return m, nil
		case "alt+down":
			if len(m.history) > 0 {
				m.historyIndex = min(len(m.history), m.historyIndex+1)
				if m.historyIndex == len(m.history) {
					m.input.Reset()
				} else {
					m.input.SetValue(m.history[m.historyIndex])
				}
			}
			return m, nil
		case "tab":
			if matches := Complete(m.input.Value()); len(matches) == 1 {
				m.input.SetValue(matches[0] + " ")
			} else if len(matches) > 1 {
				m.add("commands", strings.Join(matches, "  "))
			}
			return m, nil
		case "ctrl+j", "alt+enter":
			m.input.InsertString("\n")
			return m, nil
		case "enter":
			if m.busy {
				return m, nil
			}
			text := strings.TrimSpace(m.input.Value())
			if text == "" {
				return m, nil
			}
			m.input.Reset()
			m.history = append(m.history, text)
			m.historyIndex = len(m.history)
			if strings.HasPrefix(text, "/") {
				return m, m.command(text)
			}
			m.add("user", text)
			m.busy = true
			m.status = "requesting"
			ctx, cancel := context.WithCancel(context.Background())
			m.cancel = cancel
			m.events = make(chan tea.Msg, 64)
			events := m.events
			m.finished = make(chan struct{})
			finished := m.finished
			engine := *m.Engine
			go func() {
				defer close(finished)
				err := engine.Run(ctx, text, func(e agent.Event) {
					select {
					case events <- e:
					case <-ctx.Done():
					}
				})
				events <- done{err}
			}()
			return m, wait(events)
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}
func (m *Model) command(text string) tea.Cmd {
	name, arg, err := Parse(text)
	if err != nil {
		m.add("error", err.Error())
		return nil
	}
	c := m.Engine.Config
	switch name {
	case "/exit":
		return tea.Quit
	case "/help":
		m.add("help", "/model [provider/model] · /provider [name] · /harness [review|inspect]\n/sessions · /resume <id> · /new · /rename <title> · /delete <other-session-id>\n/context · /cost · /permissions · /capabilities · /changes · /config · /status · /clear · /exit\nEnter send · Ctrl+J multiline · Tab complete · Alt+↑/↓ history\nPgUp/PgDn scroll · Esc cancel · Ctrl+C cancel/exit\nApprovals: Ctrl+Y accept one operation · Ctrl+N reject")
	case "/clear":
		m.lines = nil
		m.stream = ""
		m.refresh(true)
	case "/model", "/provider", "/harness":
		if arg == "" {
			var list []string
			for p, v := range c.Providers {
				for _, model := range v.Models {
					list = append(list, p+"/"+model.ID)
				}
			}
			sort.Strings(list)
			m.add("configuration", fmt.Sprintf("Active %s/%s · harness %s\n%s", c.Provider, c.Model, c.Harness, strings.Join(list, "\n")))
			return nil
		}
		if name == "/harness" {
			c.Harness = arg
		} else if name == "/provider" {
			p, ok := c.Providers[arg]
			if !ok {
				m.add("error", "unknown provider")
				return nil
			}
			c.Provider = arg
			c.Model = p.Models[0].ID
		} else {
			parts := strings.SplitN(arg, "/", 2)
			if len(parts) != 2 {
				m.add("error", "usage: /model provider/model")
				return nil
			}
			c.Provider = parts[0]
			c.Model = parts[1]
		}
		if err = c.Validate(); err != nil {
			break
		}
		if err = m.Engine.Store.Audit(m.Engine.Session, fmt.Sprintf("configuration: %s/%s harness=%s", c.Provider, c.Model, c.Harness)); err != nil {
			break
		}
		m.Engine.Config = c
		m.usage = provider.Usage{}
		m.add("configuration", "Active: "+c.Provider+"/"+c.Model+" · "+c.Harness)
	case "/new":
		if arg != "" {
			err = fmt.Errorf("usage: /new")
			break
		}
		var id string
		id, err = m.Engine.Store.New(m.Engine.Workspace.Path)
		if err == nil {
			m.Engine.Session = id
			m.usage = provider.Usage{}
			m.restore()
		}
	case "/resume":
		session, e := m.Engine.Store.Get(arg)
		err = e
		if err != nil {
			break
		}
		if session.Workspace != m.Engine.Workspace.Path {
			err = fmt.Errorf("session belongs to another workspace; launch with --workspace %q", session.Workspace)
			break
		}
		err = m.Engine.Store.Recover(arg)
		if err == nil {
			m.Engine.Session = arg
			m.usage = provider.Usage{}
			m.restore()
		}
	case "/sessions":
		sessions, e := m.Engine.Store.List()
		err = e
		var lines []string
		for _, s := range sessions {
			lines = append(lines, s.ID+"  "+s.Title+"  "+s.Workspace)
		}
		m.add("sessions", strings.Join(lines, "\n"))
	case "/rename":
		if arg == "" {
			err = fmt.Errorf("usage: /rename title")
			break
		}
		err = m.Engine.Store.Rename(m.Engine.Session, c.Redact(arg))
		if err == nil {
			m.add("session", "Renamed")
		}
	case "/delete":
		if arg == "" || arg == m.Engine.Session {
			err = fmt.Errorf("specify another session ID; start /new before deleting this session")
			break
		}
		_, err = m.Engine.Store.Get(arg)
		if err == nil {
			err = m.Engine.Store.Delete(arg)
		}
		if err == nil {
			m.add("session", "Deleted "+arg)
		}
	case "/context":
		entries, e := m.Engine.Store.History(m.Engine.Session)
		err = e
		_, model, _ := c.Active()
		messages, n := agent.Context(entries, model.Context)
		total := 0
		for _, msg := range messages {
			total += agent.Estimate(msg)
		}
		m.add("context", fmt.Sprintf("~%d conservative byte-based tokens / %d configured limit; %d messages indexed as excerpts. Original history remains in SQLite; incremental structured summaries are prepared during requests, and recall retrieves original evidence.", total, model.Context, n))

	case "/changes":
		entries, e := m.Engine.Store.History(m.Engine.Session)
		err = e
		var changes []string
		for _, v := range entries {
			if v.Message.Role == "tool" && (strings.Contains(v.Message.Content, "Applied reviewed") || strings.Contains(v.Message.Content, "Completed")) {
				changes = append(changes, v.Message.Content)
			}
		}
		m.add("changes", strings.Join(changes, "\n\n"))
	case "/capabilities":
		b, _ := json.MarshalIndent(automation.Capabilities(c.Automation), "", "  ")
		m.add("capabilities", string(b)+"\n"+m.Engine.Workspace.SandboxStatus())
	case "/cost":
		usage, e := m.Engine.Store.Usage(m.Engine.Session)
		err = e
		var input, output, unknown int
		var cost float64
		for _, u := range usage {
			input += u.Input
			output += u.Output
			if u.Cost == nil {
				unknown++
			} else {
				cost += *u.Cost
			}
		}
		m.add("usage", fmt.Sprintf("Saved session: %d input / %d output tokens reported. Known estimated cost: $%.6f; %d requests lack configured prices. Requests without usage remain unknown.", input, output, cost, unknown))
	case "/permissions":
		if arg != "" {
			parts := strings.Fields(arg)
			if len(parts) != 2 || (parts[1] != "on" && parts[1] != "off") {
				err = fmt.Errorf("usage: /permissions browser|desktop|clipboard|network|install on|off")
				break
			}
			enabled := parts[1] == "on"
			switch parts[0] {
			case "network":
				c.Permissions.Network = enabled
			case "install":
				c.Permissions.Install = enabled
			case "browser":
				c.Automation.Browser = enabled
			case "desktop":
				c.Automation.Desktop = enabled
			case "clipboard":
				c.Automation.Clipboard = enabled
			default:
				err = fmt.Errorf("unknown permission")
			}
			if err != nil {
				break
			}
			if err = m.Engine.Store.Audit(m.Engine.Session, "user permission change: "+arg); err != nil {
				break
			}
			m.Engine.Config = c
		}
		m.add("permissions", fmt.Sprintf("Harness %s; file changes reviewed individually. Host shell opt-in: %t. Default shell: isolated on supported Linux hosts, otherwise denied. Browser=%t Desktop=%t Clipboard=%t. Automation actions require approval. Network/install grants are visible in /config. Use /permissions browser|desktop|clipboard|network|install on|off.", c.Harness, m.Engine.Workspace.AllowShell, c.Automation.Browser, c.Automation.Desktop, c.Automation.Clipboard))
	case "/config":
		b, _ := json.MarshalIndent(c, "", "  ")
		m.add("configuration", string(b))
	case "/status":
		m.add("status", fmt.Sprintf("Session: %s\nWorkspace: %s\nProvider: %s/%s\nHarness: %s\nRuntime: shared authenticated service\nState: %s", m.Engine.Session, m.Engine.Workspace.Path, c.Provider, c.Model, c.Harness, m.status))
	}
	if err != nil {
		m.add("error", err.Error())
	}
	return nil
}
func (m *Model) View() tea.View {
	title := accent.Render(" OPENVIBECODING ") + muted.Render(" project agent")
	status := fmt.Sprintf(" %s/%s · %s · %s", m.Engine.Config.Provider, m.Engine.Config.Model, m.Engine.Config.Harness, m.status)
	status = lipgloss.NewStyle().Width(max(10, m.width-2)).Render(m.Engine.Config.Redact(status))
	help := "Enter send  Ctrl+J newline  /help  Esc stop"
	if m.approval != nil {
		help = "REVIEW · Ctrl+Y approve · Ctrl+N reject · PgUp/PgDn scroll"
	}
	v := tea.NewView(title + "\n" + muted.Render(m.Engine.Config.Redact(m.Engine.Workspace.Path)) + "\n\n" + m.output.View() + "\n" + warning.Render(status) + "\n" + m.input.View() + "\n" + muted.Render(help))
	v.AltScreen = true
	return v
}

// Shutdown cancels and drains an active turn before storage is closed, including
// program termination caused by terminal loss or an OS signal.
func (m *Model) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
	if m.finished == nil {
		return
	}
	for {
		select {
		case <-m.finished:
			return
		case <-m.events:
		}
	}
}
