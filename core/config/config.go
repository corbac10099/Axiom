// Package config gère le chargement hiérarchique de la configuration d'Axiom.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Core       CoreConfig       `json:"core"`
	UI         UIConfig         `json:"ui"`
	AI         AIConfig         `json:"ai"`
	FileSystem FileSystemConfig `json:"filesystem"`
	Security   SecurityConfig   `json:"security"`
}

type CoreConfig struct {
	ModulesDir    string `json:"modules_dir"`
	LogLevel      string `json:"log_level"`
	BusBufferSize int    `json:"bus_buffer_size"`
	WorkspaceDir  string `json:"workspace_dir"`
}

type UIConfig struct {
	DefaultTheme string `json:"default_theme"`
	WindowWidth  int    `json:"window_width"`
	WindowHeight int    `json:"window_height"`
	FontSize     int    `json:"font_size"`
	TabSize      int    `json:"tab_size"`
	WordWrap     bool   `json:"word_wrap"`
}

type AIConfig struct {
	Provider    string  `json:"provider"`
	ModelID     string  `json:"model_id"`
	BaseURL     string  `json:"base_url"`
	APIKey      string  `json:"api_key"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	TimeoutSecs int     `json:"timeout_secs"`
}

type FileSystemConfig struct {
	WatchEnabled   bool     `json:"watch_enabled"`
	MaxFileSizeMB  int      `json:"max_file_size_mb"`
	IgnorePatterns []string `json:"ignore_patterns"`
	BackupOnWrite  bool     `json:"backup_on_write"`
}

type SecurityConfig struct {
	RequireApprovalForL2 bool `json:"require_approval_for_l2"`
	AuditLogMaxEntries   int  `json:"audit_log_max_entries"`
	AllowExternalModules bool `json:"allow_external_modules"`
}

func Default() Config {
	return Config{
		Core: CoreConfig{
			ModulesDir:    "./modules",
			LogLevel:      "info",
			BusBufferSize: 128,
			WorkspaceDir:  ".",
		},
		UI: UIConfig{
			DefaultTheme: "dark",
			WindowWidth:  1400,
			WindowHeight: 900,
			FontSize:     14,
			TabSize:      4,
			WordWrap:     false,
		},
		AI: AIConfig{
			Provider:    "ollama",
			ModelID:     "mistral:7b",
			BaseURL:     "http://localhost:11434",
			MaxTokens:   2048,
			Temperature: 0.2,
			TimeoutSecs: 60,
		},
		FileSystem: FileSystemConfig{
			WatchEnabled:   true,
			MaxFileSizeMB:  50,
			IgnorePatterns: []string{".git", "node_modules", ".axiom_cache", "*.exe"},
			BackupOnWrite:  false,
		},
		Security: SecurityConfig{
			RequireApprovalForL2: true,
			AuditLogMaxEntries:   1000,
			AllowExternalModules: false,
		},
	}
}

func Load(configPath string) (Config, []string) {
	cfg := Default()
	var warnings []string

	path := resolveConfigPath(configPath)
	if path != "" {
		if err := loadFromFile(path, &cfg); err != nil {
			warnings = append(warnings, fmt.Sprintf("config: cannot load '%s': %v (using defaults)", path, err))
		}
	} else {
		warnings = append(warnings, "config: no config file found, using defaults")
	}

	applyEnvOverrides(&cfg)

	if errs := validate(&cfg); len(errs) > 0 {
		warnings = append(warnings, errs...)
	}
	return cfg, warnings
}

func Save(cfg Config, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshal failed: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("config: cannot create config dir: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func resolveConfigPath(explicit string) string {
	if explicit != "" {
		if _, err := os.Stat(explicit); err == nil {
			return explicit
		}
		return ""
	}
	candidates := []string{".axiom/config.json", "axiom.config.json"}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".axiom", "config.json"))
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func loadFromFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, cfg)
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("AXIOM_MODULES_DIR"); v != "" {
		cfg.Core.ModulesDir = v
	}
	if v := os.Getenv("AXIOM_LOG_LEVEL"); v != "" {
		cfg.Core.LogLevel = v
	}
	if v := os.Getenv("AXIOM_WORKSPACE"); v != "" {
		cfg.Core.WorkspaceDir = v
	}
	if v := os.Getenv("AXIOM_DEBUG"); v == "1" {
		cfg.Core.LogLevel = "debug"
	}
	if v := os.Getenv("AXIOM_AI_PROVIDER"); v != "" {
		cfg.AI.Provider = v
	}
	if v := os.Getenv("AXIOM_AI_MODEL"); v != "" {
		cfg.AI.ModelID = v
	}
	if v := os.Getenv("AXIOM_AI_BASE_URL"); v != "" {
		cfg.AI.BaseURL = v
	}
	if v := os.Getenv("AXIOM_AI_KEY"); v != "" {
		cfg.AI.APIKey = v
	}
	if v := os.Getenv("AXIOM_THEME"); v != "" {
		cfg.UI.DefaultTheme = v
	}
	if v := os.Getenv("AXIOM_BUS_BUFFER"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Core.BusBufferSize = n
		}
	}
}

func validate(cfg *Config) []string {
	var warns []string
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[strings.ToLower(cfg.Core.LogLevel)] {
		warns = append(warns, fmt.Sprintf("config: invalid log_level '%s', using 'info'", cfg.Core.LogLevel))
		cfg.Core.LogLevel = "info"
	}
	if cfg.Core.BusBufferSize < 16 {
		warns = append(warns, "config: bus_buffer_size too small (<16), using 128")
		cfg.Core.BusBufferSize = 128
	}
	if cfg.UI.FontSize < 8 || cfg.UI.FontSize > 72 {
		warns = append(warns, fmt.Sprintf("config: font_size %d out of range [8-72], using 14", cfg.UI.FontSize))
		cfg.UI.FontSize = 14
	}
	if cfg.AI.Temperature < 0 || cfg.AI.Temperature > 1 {
		warns = append(warns, "config: AI temperature out of range [0-1], using 0.2")
		cfg.AI.Temperature = 0.2
	}
	if cfg.AI.TimeoutSecs < 1 {
		cfg.AI.TimeoutSecs = 60
	}
	return warns
}