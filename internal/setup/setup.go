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
	"sync"
	"time"

	"golang.org/x/term"

	"github.com/iagoluch/OpenVibeCoding/internal/config"
	"github.com/iagoluch/OpenVibeCoding/internal/credentials"
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

// Catalog describes one connectable provider as it is presented to beginners.
type Catalog struct {
	ID     string
	Title  string
	Detail string
	KeyURL string
}

// Providers returns the provider catalog in the order beginners should see it:
// the services with a free tier first. The wizard and the browser onboarding
// read the same table, so the two paths cannot drift.
func Providers() []Catalog {
	return []Catalog{
		{ID: "openrouter", Title: "OpenRouter", Detail: "includes free models (:free); key at openrouter.ai/keys", KeyURL: "https://openrouter.ai/keys"},
		{ID: "groq", Title: "Groq", Detail: "free developer tier with rate limits; key at console.groq.com/keys", KeyURL: "https://console.groq.com/keys"},
		{ID: "gemini", Title: "Google Gemini", Detail: "AI Studio free tier; key at aistudio.google.com/apikey", KeyURL: "https://aistudio.google.com/apikey"},
		{ID: "zai", Title: "Z.AI", Detail: "free glm flash models; key at z.ai", KeyURL: "https://z.ai"},
		{ID: "nvidia", Title: "NVIDIA NIM", Detail: "free credits on build.nvidia.com", KeyURL: "https://build.nvidia.com"},
		{ID: "cerebras", Title: "Cerebras", Detail: "free trial tier; key at cloud.cerebras.ai", KeyURL: "https://cloud.cerebras.ai"},
	}
}

// Local lists the models served by the local runners that answer right now.
type Local struct {
	Ollama   []string
	LMStudio []string
}

const (
	ollamaURL   = "http://127.0.0.1:11434/api/tags"
	lmstudioURL = "http://127.0.0.1:1234/v1/models"
)

// Detect probes the local model runners; a missing server is normal and
// reported as no models. The two probes run at once so a silent port costs its
// own timeout instead of adding to the other's.
func Detect() Local {
	var detected Local
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		detected.Ollama = detectModels(ollamaURL, "models", "name")
	}()
	go func() {
		defer wg.Done()
		detected.LMStudio = detectModels(lmstudioURL, "data", "id")
	}()
	wg.Wait()
	return detected
}

// LocalProvider reports whether a provider name is served by a local runner.
func LocalProvider(name string) bool { return name == "local" || name == "lmstudio" }

// wizardOptions builds the interactive menu: running local servers first, then
// the catalog, then the OpenCode import.
func wizardOptions(detected Local) []choice {
	var options []choice
	if len(detected.Ollama) > 0 {
		options = append(options, choice{provider: "local", title: fmt.Sprintf("Ollama (running, %d model(s))", len(detected.Ollama)), detail: "free, runs on this machine"})
	}
	if len(detected.LMStudio) > 0 {
		options = append(options, choice{provider: "lmstudio", title: fmt.Sprintf("LM Studio (running, %d model(s))", len(detected.LMStudio)), detail: "free, runs on this machine"})
	}
	for _, p := range Providers() {
		options = append(options, choice{provider: p.ID, title: p.Title, detail: p.Detail, keyURL: p.KeyURL})
	}
	return append(options, choice{provider: "import", title: "Import credentials from OpenCode", detail: "uses accounts already configured in OpenCode (google, nvidia)"})
}

func Run(ctx context.Context, dir string, c config.Config) error {
	in := bufio.NewReader(os.Stdin)
	fmt.Println("OpenVibeCoding setup")
	fmt.Println("Connect one model provider. Credentials go to your OS keychain; nothing is echoed.")
	fmt.Println()

	detected := Detect()
	options := wizardOptions(detected)

	for i, o := range options {
		fmt.Printf("  %d. %s — %s\n", i+1, o.title, o.detail)
	}
	fmt.Println("  0. Exit without changes")
	fmt.Println()

	pick, err := choose(in, len(options))
	if err != nil {
		return err
	}
	if pick == 0 {
		fmt.Println("No changes made.")
		return nil
	}
	selected := options[pick-1]

	// Import and local paths do not need a pasted key.
	if selected.provider == "import" {
		return importOpenCode(ctx, dir, c, in)
	}
	if selected.provider == "local" || selected.provider == "lmstudio" {
		if activateLocal(ctx, dir, c, in, selected.provider, detected) {
			fmt.Println("\nSetup complete. Next:")
			fmt.Println("  cd <your project>")
			fmt.Println("  ovc")
		}
		return nil
	}

	fmt.Println()
	if selected.keyURL != "" {
		fmt.Printf("Create or copy an API key: %s\n", selected.keyURL)
	}
	key, err := readSecret("Paste the API key (hidden): ")
	if err != nil {
		return err
	}
	if err := Store(dir, &c, selected.provider, key); err != nil {
		return err
	}
	if !activate(ctx, dir, c, in, selected.provider, "") {
		return nil
	}
	fmt.Println("\nSetup complete. Next:")
	fmt.Println("  cd <your project>")
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
		fmt.Fprintf(os.Stderr, "Unknown provider %q.\n", name)
		return false
	}
	if len(p.Models) == 0 {
		fmt.Fprintf(os.Stderr, "Provider %q has no models configured.\n", name)
		return false
	}
	first := model
	if first == "" {
		first = p.Models[0].ID
	}
	active, _ := Probe(ctx, p, first, func(candidate string, err error) {
		if err == nil {
			fmt.Printf("Testing %s/%s with a real request...\n", name, candidate)
			return
		}
		fmt.Fprintf(os.Stderr, "Smoke test failed: %v\n", err)
	})
	if active == "" {
		if !confirm(in, "Activate this provider anyway?") {
			return false
		}
		active = first
	} else {
		fmt.Println("Smoke test passed.")
	}
	if _, err := Select(dir, c, name, active); err != nil {
		fmt.Fprintf(os.Stderr, "Saving configuration: %v\n", err)
		return false
	}
	fmt.Printf("Active provider: %s/%s\n", name, active)
	return true
}

// Probe runs a real request against the provider's models, preferred first, and
// returns the first model that answered. budget bounds the whole set and one
// attempt is bounded by timeout. report, when set, is called before each
// attempt with a nil error and again with the failure when an attempt failed.
func Probe(ctx context.Context, p config.Provider, preferred string, report func(model string, err error)) (string, error) {
	if len(p.Models) == 0 {
		return "", errors.New("provider has no models configured")
	}
	if preferred == "" {
		preferred = p.Models[0].ID
	}
	deadline := time.Now().Add(smokeBudget)
	var last error
	for _, candidate := range candidates(p, preferred) {
		timeout := smokeTimeout
		if remaining := time.Until(deadline); remaining < timeout {
			timeout = remaining
		}
		if timeout <= 0 {
			break
		}
		if report != nil {
			report(candidate, nil)
		}
		err := smoke(ctx, p, candidate, timeout)
		if err == nil {
			return candidate, nil
		}
		last = err
		if report != nil {
			report(candidate, err)
		}
		if !provider.ModelUnavailable(err) && !errors.Is(err, errNoResponse) {
			break
		}
	}
	if last == nil {
		last = fmt.Errorf("%w (%s): the provider is slow; retry", errNoResponse, smokeBudget)
	}
	return "", last
}

// Select validates one provider/model pair, persists it as the active choice
// and returns the updated configuration, which the caller adopts.
func Select(dir string, c config.Config, provider, model string) (config.Config, error) {
	p, ok := c.Providers[provider]
	if !ok {
		return c, fmt.Errorf("unknown provider %q", provider)
	}
	for _, m := range p.Models {
		if m.ID == model {
			c.Provider, c.Model = provider, model
			return c, config.Save(dir, c)
		}
	}
	return c, fmt.Errorf("unknown model %q for %s", model, provider)
}

// Store saves one credential for a provider and refreshes the in-memory
// configuration, so a following smoke test authenticates with the key that was
// just stored. The key never leaves the credential store.
func Store(dir string, c *config.Config, provider, key string) error {
	if _, ok := c.Providers[provider]; !ok {
		return fmt.Errorf("unknown provider %q", provider)
	}
	if strings.TrimSpace(key) == "" {
		return errors.New("empty credential; nothing was changed")
	}
	if err := credentials.Set(dir, provider, key, false); err != nil {
		return fmt.Errorf("storing the credential: %w", err)
	}
	p := c.Providers[provider]
	p.Secret = key
	c.Providers[provider] = p
	return nil
}

// candidates lists the provider's models with the preferred one first.
func candidates(p config.Provider, preferred string) []string {
	order := []string{preferred}
	for _, m := range p.Models {
		if m.ID != preferred {
			order = append(order, m.ID)
		}
	}
	return order
}

func activateLocal(ctx context.Context, dir string, c config.Config, in *bufio.Reader, name string, detected Local) bool {
	models := detected.Ollama
	if name == "lmstudio" {
		models = detected.LMStudio
	}
	p := c.Providers[name]
	model := pickLocalModel(models, p.Models[0].ID)
	c.Providers[name] = EnsureModel(c.Providers[name], model, 32768)
	return activate(ctx, dir, c, in, name, model)
}

// EnsureModel adds a locally served model to the provider registry when the
// user's server exposes one that is not in the shipped preset.
func EnsureModel(p config.Provider, model string, context int) config.Provider {
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
			fmt.Printf("Found OpenCode credentials at %s\n", candidate)
			break
		}
	}
	if len(found) == 0 {
		fmt.Println("No importable OpenCode credentials found (expected google and/or nvidia).")
		return nil
	}
	imported := ""
	for _, name := range []string{"gemini", "nvidia"} {
		key, ok := found[name]
		if !ok || key == "" {
			continue
		}
		if !confirm(in, fmt.Sprintf("Import the %s credential into OpenVibeCoding?", name)) {
			continue
		}
		if err := Store(dir, &c, name, key); err != nil {
			fmt.Fprintf(os.Stderr, "Importing %s: %v\n", name, err)
			continue
		}
		fmt.Printf("Imported %s.\n", name)
		if imported == "" {
			imported = name
		}
	}
	if imported == "" {
		fmt.Println("Nothing imported.")
		return nil
	}
	if !activate(ctx, dir, c, in, imported, "") {
		return nil
	}
	fmt.Println("\nSetup complete. Next:")
	fmt.Println("  cd <your project>")
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

func choose(in *bufio.Reader, n int) (int, error) {
	for {
		fmt.Printf("Choose [1-%d] (0 to exit): ", n)
		line, err := in.ReadString('\n')
		if err != nil {
			return 0, err
		}
		v, err := strconv.Atoi(strings.TrimSpace(line))
		if err == nil && v >= 0 && v <= n {
			return v, nil
		}
		fmt.Println("Enter a number between 0 and", n)
	}
}

func confirm(in *bufio.Reader, prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	line, err := in.ReadString('\n')
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}

func readSecret(prompt string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("credential input requires a terminal")
	}
	fmt.Fprint(os.Stderr, prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("reading credential: %w", err)
	}
	return strings.TrimSpace(string(b)), nil
}
