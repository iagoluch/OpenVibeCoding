// Package setup implements the interactive first-run wizard behind ovc --setup.
// It connects one provider end to end: detect local servers, store a key in the
// OS keychain, activate the provider and run a real smoke request.
package setup

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/iagoluch/OpenVibeCoding/internal/config"
	"github.com/iagoluch/OpenVibeCoding/internal/credentials"
	"github.com/iagoluch/OpenVibeCoding/internal/i18n"
	"github.com/iagoluch/OpenVibeCoding/internal/provider"
)

// smokeBudget bounds the whole activation smoke test across model fallbacks;
// smokeTimeout bounds a single model attempt. Variables so tests can shorten
// them.
var (
	smokeBudget  = 60 * time.Second
	smokeTimeout = 30 * time.Second
)

// errNoResponse marks a model that accepted the connection but never answered,
// which is worth retrying with another model of the same provider.
var errNoResponse = errors.New("no response from the provider in time")

type choice struct {
	provider string // key in config.Providers, or "import"/"exit"
	title    string
	detail   string
	keyURL   string
}

func Run(ctx context.Context, dir string, c config.Config) error {
	in := bufio.NewReader(os.Stdin)
	fmt.Println(i18n.T(c.Language, "setup.title"))
	fmt.Println(i18n.T(c.Language, "setup.connect"))
	fmt.Println()

	var options []choice
	ollama := detectModels("http://127.0.0.1:11434/api/tags", "models", "name")
	if len(ollama) > 0 {
		options = append(options, choice{provider: "local", title: fmt.Sprintf(i18n.T(c.Language, "setup.ollamaTitle"), len(ollama)), detail: i18n.T(c.Language, "setup.localFreeDetail")})
	}
	lmstudio := detectModels("http://127.0.0.1:1234/v1/models", "data", "id")
	if len(lmstudio) > 0 {
		options = append(options, choice{provider: "lmstudio", title: fmt.Sprintf(i18n.T(c.Language, "setup.lmstudioTitle"), len(lmstudio)), detail: i18n.T(c.Language, "setup.localFreeDetail")})
	}
	options = append(options,
		choice{provider: "openrouter", title: "OpenRouter", detail: i18n.T(c.Language, "setup.openrouterDetail"), keyURL: "https://openrouter.ai/keys"},
		choice{provider: "groq", title: "Groq", detail: i18n.T(c.Language, "setup.groqDetail"), keyURL: "https://console.groq.com/keys"},
		choice{provider: "gemini", title: "Google Gemini", detail: i18n.T(c.Language, "setup.geminiDetail"), keyURL: "https://aistudio.google.com/apikey"},
		choice{provider: "zai", title: "Z.AI", detail: i18n.T(c.Language, "setup.zaiDetail"), keyURL: "https://z.ai"},
		choice{provider: "nvidia", title: "NVIDIA NIM", detail: i18n.T(c.Language, "setup.nvidiaDetail"), keyURL: "https://build.nvidia.com"},
		choice{provider: "cerebras", title: "Cerebras", detail: i18n.T(c.Language, "setup.cerebrasDetail"), keyURL: "https://cloud.cerebras.ai"},
		choice{provider: "import", title: "Import credentials from OpenCode", detail: i18n.T(c.Language, "setup.importDetail")},
	)

	for i, o := range options {
		fmt.Printf("  %d. %s — %s\n", i+1, o.title, o.detail)
	}
	fmt.Println("  0. " + i18n.T(c.Language, "setup.exit"))
	fmt.Println()

	pick, err := choose(in, len(options), c.Language)
	if err != nil {
		return err
	}
	if pick == 0 {
		fmt.Println(i18n.T(c.Language, "setup.noChanges"))
		return nil
	}
	selected := options[pick-1]

	// Import and local paths do not need a pasted key.
	if selected.provider == "import" {
		return importOpenCode(ctx, dir, c, in)
	}
	if selected.provider == "local" || selected.provider == "lmstudio" {
		if activateLocal(ctx, dir, c, in, selected.provider, ollama, lmstudio) {
			fmt.Println("
" + i18n.T(c.Language, "setup.complete"))
			fmt.Println("  " + i18n.T(c.Language, "setup.projectCommand"))
			fmt.Println("  ovc")
		}
		return nil
	}

	fmt.Println()
	if selected.keyURL != "" {
		fmt.Println(fmt.Sprintf(i18n.T(c.Language, "setup.keyURL"), selected.keyURL))
	}
	key, err := readSecret(i18n.T(c.Language, "setup.keyPrompt"), c.Language)
	if err != nil {
		return err
	}
	if key == "" {
		return fmt.Errorf("%s", i18n.T(c.Language, "setup.emptyCredential"))
	}
	if err := credentials.Set(dir, selected.provider, key, false); err != nil {
		return fmt.Errorf("%s: %w", i18n.T(c.Language, "setup.storingCredential"), err)
	}
	// The configuration was loaded before the credential existed; refresh it so
	// the smoke test authenticates with the key that was just stored.
	stored := c.Providers[selected.provider]
	stored.Secret = key
	c.Providers[selected.provider] = stored
	if !activate(ctx, dir, c, in, selected.provider, "") {
		return nil
	}
	fmt.Println("
" + i18n.T(c.Language, "setup.complete"))
	fmt.Println("  " + i18n.T(c.Language, "setup.projectCommand"))
	fmt.Println("  ovc")
	return nil
}

// activate points the configuration at one provider/model, smoke-tests a real
// request and only keeps the change on success (or explicit confirmation).
// When the preferred model cannot serve requests (retired, rate limited,
// provider down), the provider's other models are tried before giving up.
func activate(ctx context.Context, dir string, c config.Config, in *bufio.Reader, name, model string) bool {
	p, ok := c.Providers[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "%s\n", fmt.Sprintf(i18n.T(c.Language, "setup.unknownProvider"), name))
		return false
	}
	if len(p.Models) == 0 {
		fmt.Fprintf(os.Stderr, "%s\n", fmt.Sprintf(i18n.T(c.Language, "setup.noModels"), name))
		return false
	}
	first := model
	if first == "" {
		first = p.Models[0].ID
	}
	active := ""
	deadline := time.Now().Add(smokeBudget)
	for _, candidate := range modelOrder(p, first) {
		timeout := smokeTimeout
		if remaining := time.Until(deadline); remaining < timeout {
			timeout = remaining
		}
		if timeout <= 0 {
			break
		}
		fmt.Println(fmt.Sprintf(i18n.T(c.Language, "setup.testing"), name, candidate))
		err := smoke(ctx, p, candidate, timeout)
		if err == nil {
			active = candidate
			break
		}
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T(c.Language, "setup.smokeFailed"), err)
		if !provider.ModelUnavailable(err) && !errors.Is(err, errNoResponse) {
			break
		}
	}
	if active == "" {
		if !confirm(in, i18n.T(c.Language, "setup.activateAnyway")) {
			return false
		}
		active = first
	} else {
		fmt.Println(i18n.T(c.Language, "setup.smokePassed"))
	}
	c.Provider, c.Model = name, active
	if err := config.Save(dir, c); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T(c.Language, "setup.saving"), err)
		return false
	}
	fmt.Println(fmt.Sprintf(i18n.T(c.Language, "setup.activeProvider"), name, active))
	return true
}

// modelOrder lists the provider's models with the preferred one first.
func modelOrder(p config.Provider, preferred string) []string {
	order := []string{preferred}
	for _, m := range p.Models {
		if m.ID != preferred {
			order = append(order, m.ID)
		}
	}
	return order
}

func activateLocal(ctx context.Context, dir string, c config.Config, in *bufio.Reader, name string, ollama, lmstudio []string) bool {
	detected := ollama
	if name == "lmstudio" {
		detected = lmstudio
	}
	p := c.Providers[name]
	model := pickLocalModel(detected, p.Models[0].ID)
	c.Providers[name] = ensureModel(c.Providers[name], model, 32768)
	return activate(ctx, dir, c, in, name, model)
}

// ensureModel adds a locally served model to the provider registry when the
// user's server exposes one that is not in the shipped preset.
func ensureModel(p config.Provider, model string, context int) config.Provider {
	for _, m := range p.Models {
		if m.ID == model {
			return p
		}
	}
	zero := 0.0
	p.Models = append(p.Models, config.Model{ID: model, Context: context, Tools: true, InputPrice: &zero, OutputPrice: &zero})
	return p
}

func pickLocalModel(detected []string, fallback string) string {
	for _, m := range detected {
		if m == fallback {
			return fallback
		}
	}
	if len(detected) > 0 {
		return detected[0]
	}
	return fallback
}

func smoke(ctx context.Context, p config.Provider, model string, timeout time.Duration) error {
	p.MaxOutput = 32
	var m config.Model
	for _, candidate := range p.Models {
		if candidate.ID == model {
			m = candidate
			break
		}
	}
	if m.ID == "" {
		m = config.Model{ID: model, Context: 32768, Tools: false}
	}
	sctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	_, err := provider.New(p, m).Stream(sctx, []provider.Message{{Role: "user", Content: "Reply with OK."}}, nil, func(string) {})
	if err != nil && errors.Is(sctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("%w (%s): the service may be busy; retry or choose a different model", errNoResponse, timeout)
	}
	return err
}

func importOpenCode(ctx context.Context, dir string, c config.Config, in *bufio.Reader) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	var found map[string]string
	for _, candidate := range []string{
		filepath.Join(home, ".local", "share", "opencode", "auth.json"),
		filepath.Join(home, ".config", "opencode", "auth.json"),
	} {
		b, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		if found = parseOpenCodeAuth(b); len(found) > 0 {
			fmt.Println(fmt.Sprintf(i18n.T(c.Language, "setup.opencodeFound"), candidate))
			break
		}
	}
	if len(found) == 0 {
		fmt.Println(i18n.T(c.Language, "setup.noOpenCodeCredentials"))
		return nil
	}
	imported := ""
	for _, name := range []string{"gemini", "nvidia"} {
		key, ok := found[name]
		if !ok || key == "" {
			continue
		}
		if !confirm(in, fmt.Sprintf(i18n.T(c.Language, "setup.importPrompt"), name)) {
			continue
		}
		if err := credentials.Set(dir, name, key, false); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", fmt.Sprintf(i18n.T(c.Language, "setup.importing"), name), err)
			continue
		}
		// Keep the in-memory configuration consistent with the stored credential
		// so the smoke test can authenticate immediately.
		stored := c.Providers[name]
		stored.Secret = key
		c.Providers[name] = stored
		fmt.Println(fmt.Sprintf(i18n.T(c.Language, "setup.imported"), name))
		if imported == "" {
			imported = name
		}
	}
	if imported == "" {
		fmt.Println(i18n.T(c.Language, "setup.nothingImported"))
		return nil
	}
	if !activate(ctx, dir, c, in, imported, "") {
		return nil
	}
	fmt.Println("
" + i18n.T(c.Language, "setup.complete"))
	fmt.Println("  " + i18n.T(c.Language, "setup.projectCommand"))
	fmt.Println("  ovc")
	return nil
}

// parseOpenCodeAuth maps OpenCode provider entries to OpenVibeCoding provider
// names. Unknown entries are ignored; secret values never leave this function.
func parseOpenCodeAuth(data []byte) map[string]string {
	var raw map[string]struct {
		Key string `json:"key"`
	}
	if json.Unmarshal(data, &raw) != nil {
		return nil
	}
	mapped := map[string]string{}
	if v, ok := raw["google"]; ok && strings.TrimSpace(v.Key) != "" {
		mapped["gemini"] = strings.TrimSpace(v.Key)
	}
	if v, ok := raw["nvidia"]; ok && strings.TrimSpace(v.Key) != "" {
		mapped["nvidia"] = strings.TrimSpace(v.Key)
	}
	return mapped
}

// detectModels probes a local server; absence is normal and returns nil.
func detectModels(url, listKey, nameKey string) []string {
	client := &http.Client{Timeout: 900 * time.Millisecond}
	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	body := struct {
		Models []map[string]any `json:"models"`
		Data   []map[string]any `json:"data"`
	}{}
	if json.NewDecoder(resp.Body).Decode(&body) != nil {
		return nil
	}
	items := body.Models
	if listKey == "data" {
		items = body.Data
	}
	var names []string
	for _, item := range items {
		if name, ok := item[nameKey].(string); ok && name != "" {
			names = append(names, name)
		}
	}
	return names
}

func choose(in *bufio.Reader, n int, language string) (int, error) {
	for {
		fmt.Print(fmt.Sprintf(i18n.T(language, "setup.choose"), n))
		line, err := in.ReadString('\n')
		if err != nil {
			return 0, err
		}
		v, err := strconv.Atoi(strings.TrimSpace(line))
		if err == nil && v >= 0 && v <= n {
			return v, nil
		}
		fmt.Println(fmt.Sprintf(i18n.T(language, "setup.invalidChoice"), n))
	}
}

func confirm(in *bufio.Reader, prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	line, err := in.ReadString('\n')
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes" || answer == "s" || answer == "sim" || answer == "si" || answer == "sí" || answer == "oui" || answer == "ja" || answer == "是" || answer == "是的"
}

func readSecret(prompt, language string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("%s", i18n.T(language, "setup.credentialTerminal"))
	}
	fmt.Fprint(os.Stderr, prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T(language, "setup.readingCredential"), err)
	}
	return strings.TrimSpace(string(b)), nil
}
