package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vietclaw/internal/config"
	"vietclaw/internal/db"
	"vietclaw/internal/providers"
)

type providerChoice struct {
	id   string
	typ  string
	name string
}

func runSetup() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%sVietClaw setup%s\n", uiBold, uiReset)
	uiHint("Cấu hình lần đầu. Web UI dùng cho chat sau khi chạy daemon.")

	paths, err := config.DefaultPaths()
	if err != nil {
		return err
	}

	var cfg config.Config
	if _, err := os.Stat(paths.ConfigFile); err == nil {
		loaded, loadErr := config.Load(paths.ConfigFile)
		if loadErr == nil {
			cfg = loaded
			uiOk("Đã load config hiện có")
		} else {
			cfg = config.Default(paths)
		}
	} else {
		cfg = config.Default(paths)
	}

	envVars := make(map[string]string)

	// --- Provider ---
	uiTitle("LLM provider")
	choices := []providerChoice{
		{id: "mock", typ: "mock", name: "Mock (demo, không cần API key)"},
		{id: "zen", typ: "opencode-zen", name: "OpenCode Zen"},
		{id: "openai", typ: "openai", name: "OpenAI"},
		{id: "gemini", typ: "gemini", name: "Google Gemini"},
		{id: "anthropic", typ: "anthropic", name: "Anthropic Claude"},
		{id: "ollama", typ: "openai-compatible", name: "Ollama (local)"},
	}
	opts := make([]string, len(choices))
	for i, c := range choices {
		opts[i] = c.name
	}
	idx, err := promptSingleSelectClean("Chọn provider mặc định", opts)
	if err != nil {
		return err
	}
	pick := choices[idx]

	// Disable all optional providers first
	for _, id := range []string{"zen", "openai", "gemini", "anthropic", "ollama"} {
		disableProvider(&cfg, id)
	}
	disableProvider(&cfg, "mock")

	switch pick.id {
	case "mock":
		mock := findOrCreateProvider(&cfg, "mock", "mock")
		mock.Enabled = true
		mock.DefaultModel = "mock-small"
		updateProvider(&cfg, *mock)
		cfg.Router.DefaultProvider = "mock"
		cfg.Router.DefaultModel = "mock-small"
	case "zen":
		uiHint("API key: https://opencode.ai/auth")
		key := promptSecret(reader, "Zen API key (Enter = dùng OPENCODE_ZEN_KEY)")
		if key != "" {
			envVars["OPENCODE_ZEN_KEY"] = key
		}
		p := findOrCreateProvider(&cfg, "zen", "opencode-zen")
		p.Enabled = true
		p.APIKeyEnv = "OPENCODE_ZEN_KEY"
		p.BaseURL = providers.ZenBaseURL
		p.DefaultModel = fetchAndPickModelClean(p.BaseURL, "OPENCODE_ZEN_KEY", key, "deepseek-v4-flash-free")
		updateProvider(&cfg, *p)
		cfg.Router.DefaultProvider = "zen"
		cfg.Router.DefaultModel = p.DefaultModel
	case "openai":
		key := promptSecret(reader, "OpenAI API key (Enter = dùng OPENAI_API_KEY)")
		if key != "" {
			envVars["OPENAI_API_KEY"] = key
		}
		p := findOrCreateProvider(&cfg, "openai", "openai")
		p.Enabled = true
		p.APIKeyEnv = "OPENAI_API_KEY"
		p.BaseURL = "https://api.openai.com/v1"
		p.DefaultModel = fetchAndPickModelClean(p.BaseURL, "OPENAI_API_KEY", key, "gpt-4o-mini")
		updateProvider(&cfg, *p)
		cfg.Router.DefaultProvider = "openai"
		cfg.Router.DefaultModel = p.DefaultModel
	case "gemini":
		key := promptSecret(reader, "Gemini API key (Enter = dùng GEMINI_API_KEY)")
		if key != "" {
			envVars["GEMINI_API_KEY"] = key
		}
		p := findOrCreateProvider(&cfg, "gemini", "gemini")
		p.Enabled = true
		p.APIKeyEnv = "GEMINI_API_KEY"
		geminiModels := []string{
			"gemini-2.5-flash-preview-09-2025",
			"gemini-2.5-pro",
			"gemini-2.0-flash",
		}
		p.DefaultModel = pickModelClean(geminiModels, "gemini-2.5-flash-preview-09-2025", "Model")
		updateProvider(&cfg, *p)
		cfg.Router.DefaultProvider = "gemini"
		cfg.Router.DefaultModel = p.DefaultModel
	case "anthropic":
		key := promptSecret(reader, "Anthropic API key (Enter = dùng ANTHROPIC_API_KEY)")
		if key != "" {
			envVars["ANTHROPIC_API_KEY"] = key
		}
		p := findOrCreateProvider(&cfg, "anthropic", "anthropic")
		p.Enabled = true
		p.APIKeyEnv = "ANTHROPIC_API_KEY"
		claudeModels := []string{
			"claude-sonnet-4-5",
			"claude-3-5-sonnet-20241022",
			"claude-3-5-haiku-20241022",
		}
		p.DefaultModel = pickModelClean(claudeModels, "claude-3-5-sonnet-20241022", "Model")
		updateProvider(&cfg, *p)
		cfg.Router.DefaultProvider = "anthropic"
		cfg.Router.DefaultModel = p.DefaultModel
	case "ollama":
		baseURL := promptString(reader, "Ollama URL", "http://localhost:11434/v1")
		p := findOrCreateProvider(&cfg, "ollama", "openai-compatible")
		p.Enabled = true
		p.BaseURL = baseURL
		p.DefaultModel = fetchAndPickModelClean(baseURL, "", "", "qwen2.5-coder:7b")
		updateProvider(&cfg, *p)
		cfg.Router.DefaultProvider = "ollama"
		cfg.Router.DefaultModel = p.DefaultModel
	}

	uiOk(fmt.Sprintf("Provider: %s · model: %s", cfg.Router.DefaultProvider, cfg.Router.DefaultModel))

	// --- Tools ---
	uiTitle("Tools")
	cfg.Tools.Shell.Enabled = promptBool(reader, "Bật shell_exec?", false)
	if cfg.Tools.Shell.Enabled {
		if promptBool(reader, "Docker sandbox?", true) {
			cfg.Tools.Shell.Sandbox = "docker"
			cfg.Tools.Shell.DockerImage = promptString(reader, "Docker image", "alpine:3.20")
			cfg.Tools.Shell.WorkspaceMode = promptString(reader, "Workspace (ro/rw)", "ro")
		} else {
			cfg.Tools.Shell.Sandbox = "none"
		}
	}

	// --- Channels ---
	uiTitle("Channels (tùy chọn)")
	cfg.Channels.Telegram.Enabled = promptBool(reader, "Telegram bot?", false)
	if cfg.Channels.Telegram.Enabled {
		if t := promptSecret(reader, "Telegram token"); t != "" {
			envVars["VIETCLAW_TELEGRAM_TOKEN"] = t
		}
		cfg.Channels.Telegram.TokenEnv = "VIETCLAW_TELEGRAM_TOKEN"
	}
	cfg.Channels.Discord.Enabled = promptBool(reader, "Discord bot?", false)
	if cfg.Channels.Discord.Enabled {
		if t := promptSecret(reader, "Discord token"); t != "" {
			envVars["VIETCLAW_DISCORD_TOKEN"] = t
		}
		cfg.Channels.Discord.TokenEnv = "VIETCLAW_DISCORD_TOKEN"
	}

	// --- Save ---
	uiTitle("Lưu")
	if err := os.MkdirAll(paths.LogDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.Agent.Workspace, 0o755); err != nil {
		return err
	}

	cfg = config.MergeDefault(cfg, config.Default(paths))
	if err := config.Save(paths.ConfigFile, cfg); err != nil {
		return fmt.Errorf("lưu config: %w", err)
	}
	uiOk("config: " + paths.ConfigFile)

	envPath := filepath.Join(paths.DataDir, ".env")
	if err := writeEnvFile(envPath, envVars); err != nil {
		return fmt.Errorf("lưu .env: %w", err)
	}
	if len(envVars) > 0 {
		uiOk(".env: " + envPath)
	}

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		return err
	}
	defer database.Close()
	if err := db.ApplySchema(database); err != nil {
		return fmt.Errorf("database schema: %w", err)
	}
	uiOk("database: " + cfg.Database.Path)

	fmt.Println()
	fmt.Printf("%sSẵn sàng.%s Chạy: %svietclaw daemon%s → http://127.0.0.1:%d\n", uiBold, uiReset, uiBold, uiReset, cfg.Server.Port)
	return nil
}

func promptString(reader *bufio.Reader, question string, defaultValue string) string {
	fmt.Printf("%s [%s]: ", question, defaultValue)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

func promptSecret(reader *bufio.Reader, question string) string {
	fmt.Printf("%s: ", question)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func promptBool(reader *bufio.Reader, question string, defaultValue bool) bool {
	def := "n"
	if defaultValue {
		def = "y"
	}
	fmt.Printf("%s [y/n, mặc định %s]: ", question, def)
	input, _ := reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return defaultValue
	}
	return input == "y" || input == "yes"
}

func findOrCreateProvider(cfg *config.Config, id, providerType string) *config.ProviderConfig {
	for i, p := range cfg.Providers {
		if p.ID == id {
			return &cfg.Providers[i]
		}
	}
	newProvider := config.ProviderConfig{ID: id, Type: providerType}
	cfg.Providers = append(cfg.Providers, newProvider)
	return &cfg.Providers[len(cfg.Providers)-1]
}

func updateProvider(cfg *config.Config, provider config.ProviderConfig) {
	for i, p := range cfg.Providers {
		if p.ID == provider.ID {
			cfg.Providers[i] = provider
			return
		}
	}
	cfg.Providers = append(cfg.Providers, provider)
}

func disableProvider(cfg *config.Config, id string) {
	for i, p := range cfg.Providers {
		if p.ID == id {
			cfg.Providers[i].Enabled = false
			return
		}
	}
}

func writeEnvFile(path string, vars map[string]string) error {
	existing := make(map[string]string)
	if file, err := os.Open(path); err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				existing[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
		file.Close()
	}
	for k, v := range vars {
		existing[k] = v
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	for k, v := range existing {
		if _, err := writer.WriteString(fmt.Sprintf("%s=%s\n", k, v)); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func readKey() (string, error) {
	var buf [16]byte
	n, err := os.Stdin.Read(buf[:])
	if err != nil {
		return "", err
	}
	if n == 1 {
		switch buf[0] {
		case 13, 10:
			return "enter", nil
		case 32:
			return "space", nil
		case 3:
			return "ctrlc", nil
		case 27:
			return "escape", nil
		}
	}
	if n >= 2 && buf[0] == 224 {
		switch buf[1] {
		case 72:
			return "up", nil
		case 80:
			return "down", nil
		}
	}
	if n >= 3 && buf[0] == 27 && buf[1] == 91 {
		switch buf[2] {
		case 65:
			return "up", nil
		case 66:
			return "down", nil
		}
	}
	return "", nil
}

func fetchAndPickModelClean(baseURL, apiKeyEnv, apiKeyDirect, fallback string) string {
	fmt.Printf("%s  Đang tải model…%s", uiDim, uiReset)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if apiKeyDirect != "" && apiKeyEnv != "" {
		_ = os.Setenv(apiKeyEnv, apiKeyDirect)
	}
	models, err := providers.FetchZenModels(ctx, baseURL, apiKeyEnv)
	fmt.Print("\r\033[K")
	if err != nil || len(models) == 0 {
		uiWarn(fmt.Sprintf("Không lấy được model, dùng mặc định: %s", fallback))
		return fallback
	}
	uiOk(fmt.Sprintf("%d model", len(models)))
	return pickModelClean(models, fallback, "Model")
}
