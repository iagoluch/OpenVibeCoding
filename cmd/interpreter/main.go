package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/iagoluch/OpenVibeCoding/internal/credentials"
	daemon "github.com/iagoluch/OpenVibeCoding/internal/runtime"
	"golang.org/x/term"
	"os"
	"os/signal"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/iagoluch/OpenVibeCoding/internal/agent"
	"github.com/iagoluch/OpenVibeCoding/internal/automation"
	"github.com/iagoluch/OpenVibeCoding/internal/config"

	"github.com/iagoluch/OpenVibeCoding/internal/tui"
	"github.com/iagoluch/OpenVibeCoding/internal/workspace"
)

var version = "0.1.0-dev"

func run() error {
	project := flag.String("workspace", ".", "project root")
	shell := flag.Bool("allow-unsandboxed-shell", false, "enable per-command approval for shell with host access (no OS isolation)")
	ver := flag.Bool("version", false, "show version")
	doctor := flag.Bool("doctor", false, "show configuration and capability diagnostics")
	resume := flag.String("resume", "", "resume session ID in selected workspace")
	runtimeOp := flag.String("runtime", "", "start, status, stop, or serve")
	credential := flag.String("credential", "", "provider whose credential to store (hidden terminal input)")
	credentialFile := flag.Bool("credential-file", false, "explicitly use restricted local file instead of OS keychain")
	export := flag.String("export", "", "export session JSON to stdout")
	setupBrowser := flag.Bool("setup-browser", false, "install pinned Playwright and Chromium into application data directory")
	debug := flag.Bool("debug", false, "enable runtime diagnostic metadata logging")
	flag.Parse()
	if *debug {
		os.Setenv("INTERPRETER_DEBUG", "1")
	}
	if *ver {
		fmt.Println("Open Interpreter " + version)
		return nil
	}
	dir, e := config.Dir()
	if e != nil {
		return e
	}
	c, e := config.Load(dir)
	if e != nil {
		return fmt.Errorf("configuration: %w", e)
	}
	if *setupBrowser {
		return automation.SetupBrowser(context.Background(), dir)
	}
	if *credential != "" {
		if _, ok := c.Providers[*credential]; !ok {
			return fmt.Errorf("unknown provider")
		}
		fmt.Fprint(os.Stderr, "API key (hidden): ")
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return fmt.Errorf("credential input requires a terminal")
		}
		if len(strings.TrimSpace(string(b))) == 0 {
			return fmt.Errorf("empty credential")
		}
		return credentials.Set(dir, *credential, string(b), *credentialFile)
	}
	if *runtimeOp == "serve" {
		ctx, cancel := signal.NotifyContext(context.Background(), terminationSignals()...)
		defer cancel()
		return daemon.Serve(ctx, dir, c)
	}
	if *runtimeOp != "" {
		var client *daemon.Client
		if *runtimeOp == "start" {
			client, e = daemon.Ensure(context.Background(), dir)
		} else {
			client, e = daemon.Connect(dir)
		}
		if e != nil {
			return e
		}
		if *runtimeOp != "start" && *runtimeOp != "status" && *runtimeOp != "stop" {
			return fmt.Errorf("unknown runtime command")
		}
		status, err := client.Status(*runtimeOp == "stop")
		if err == nil {
			fmt.Print(status)
		}
		return err
	}
	w, e := workspace.Open(*project, *shell)
	if e != nil {
		return e
	}
	defer w.Close()
	if *doctor {
		p, m, _ := c.Active()
		fmt.Printf("Open Interpreter %s\nData: %s\nWorkspace: %s\nProvider: %s/%s\nTool calls configured: %t\nCredentials available: %t\nShell host access: %t\nOS sandbox: %s\nShared runtime: authenticated loopback service\n", version, dir, w.Path, c.Provider, m.ID, m.Tools, p.KeyEnv == "" || p.Secret != "", *shell, w.SandboxStatus())
		b, _ := json.MarshalIndent(automation.Capabilities(c.Automation), "", "  ")
		fmt.Println(string(b))
		return nil
	}
	client, e := daemon.Ensure(context.Background(), dir)
	if e != nil {
		return e
	}
	db := client
	if *export != "" {
		b, err := db.Export(*export)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}
	id := *resume
	if id == "" {
		id, e = db.New(w.Path)
	} else {
		s, err := db.Get(id)
		e = err
		if e == nil && s.Workspace != w.Path {
			return fmt.Errorf("session workspace differs; select %s", s.Workspace)
		}
		if e == nil {
			e = db.Recover(id)
		}
	}
	if e != nil {
		return e
	}
	engine := &agent.Engine{Config: c, Store: db, Workspace: w, Session: id, Remote: client.Run}
	model := tui.New(engine)
	_, e = tea.NewProgram(model).Run()
	model.Shutdown()
	return e
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, "interpreter:", e)
		os.Exit(1)
	}
}
