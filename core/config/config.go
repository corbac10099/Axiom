// Package config gère le chargement hiérarchique de la configuration d'Axiom.
//
// Priorité (du plus faible au plus fort) :
//   1. Valeurs par défaut codées en dur
//   2. Fichier config.json (workspace courant ou répertoire home)
//   3. Variables d'environnement AXIOM_*
//   4. Flags CLI (parsing externe, transmis via Apply)
//
// La config est immuable après Load() — les modules lisent via GetConfig().
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ─────────────────────────────────────────────
// STRUCTURES
// ─────────────────────────────────────────────

// Config est la configuration complète d'Axiom.
// Chaque section est un sous-struct pour un découplage propre.
type Config struct {
	Core      CoreConfig      `json:"core"`
	UI        UIConfig        `json:"ui"`
	AI        AIConfig        `json:"ai"`
	FileSystem FileSystemConfig `json:"filesystem"`
	Security  SecurityConfig  `json:"security"`
}

// CoreConfig regroupe les paramètres du moteur principal.
type CoreConfig struct {
	ModulesDir    string `json:"modules_dir"`    // chemin vers /modules
	LogLevel      string `json:"log_level"`      // "debug"|"info"|"warn"|"error"
	BusBufferSize int    `json:"bus_buffer_size"` // buffer channel par Topic
	WorkspaceDir  string `json:"workspace_dir"`  // répertoire de travail ouvert
}

// UIConfig regroupe les paramètres d'interface.
type UIConfig struct {
	DefaultTheme  string `json:"default_theme"`   // "dark"|"light"|"monokai"
	WindowWidth   int    `json:"window_width"`    // largeur fenêtre principale (px)
	WindowHeight  int    `json:"window_height"`   // hauteur fenêtre principale (px)
	FontSize      int    `json:"font_size"`       // taille de police Monaco (pt)
	TabSize       int    `json:"tab_size"`        // taille indentation
	WordWrap      bool   `json:"word_wrap"`       // retour à la ligne automatique
}

// AIConfig regroupe les paramètres du bridge IA.
type AIConfig struct {
	Provider    string `json:"provider"`     // "ollama"|"openai"|"anthropic"|"none"
	ModelID     string `json:"model_id"`     // ex: "mistral:7b", "gpt-4o"
	BaseURL     string `json:"base_url"`     // ex: "http://localhost:11434"
	APIKey      string `json:"api_key"`      // laisser vide pour Ollama local
	MaxTokens   int    `json:"max_tokens"`   // limite de tokens par requête
	Temperature float64 `json:"temperature"` // 0.0–1.0
	TimeoutSecs int    `json:"timeout_secs"` // timeout HTTP en secondes
}

// FileSystemConfig regroupe les paramètres du gestionnaire de fichiers.
type FileSystemConfig struct {
	WatchEnabled  bool     `json:"watch_enabled"`   // activer le watcher de fichiers
	MaxFileSizeMB int      `json:"max_file_size_mb"` // limite lecture (MB)
	IgnorePatterns []string `json:"ignore_patterns"` // ex: [".git", "node_modules"]
	BackupOnWrite bool     `json:"backup_on_write"`  // créer .bak avant écrasement
}

// SecurityConfig regroupe les paramètres de sécurité.
type SecurityConfig struct {
	// RequireApprovalForL2 : si true, un dialog de confirmation s'affiche
	// pour tout module demandant L2 ou plus.
	RequireApprovalForL2 bool `json:"require_approval_for_l2"`
	// AuditLogMaxEntries : nombre max d'entrées en mémoire dans l'audit log.
	AuditLogMaxEntries int  `json:"audit_log_max_entries"`
	// AllowExternalModules : si false, seuls les modules dans /modules sont chargés.
	AllowExternalModules bool `json:"allow_external_modules"`
}

// ─────────────────────────────────────────────
// DEFAULTS
// ─────────────────────────────────────────────

// Default retourne la configuration par défaut d'Axiom.
// Toutes les valeurs sont sensées pour un environnement de développement.
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

// ─────────────────────────────────────────────
// LOADER
// ─────────────────────────────────────────────

// Load charge la configuration depuis les sources disponibles dans l'ordre de priorité.
// Ne retourne jamais d'erreur fatale — en cas de problème, les défauts sont utilisés.
func Load(configPath string) (Config, []string) {
	cfg := Default()
	var warnings []string

	// ── Étape 1 : Chercher et lire le fichier de config ───────────
	path := resolveConfigPath(configPath)
	if path != "" {
		if err := loadFromFile(path, &cfg); err != nil {
			warnings = append(warnings, fmt.Sprintf("config: cannot load '%s': %v (using defaults)", path, err))
		}
	} else {
		warnings = append(warnings, "config: no config file found, using defaults")
	}

	// ── Étape 2 : Surcharges via variables d'environnement ────────
	applyEnvOverrides(&cfg)

	// ── Étape 3 : Validation ─────────────────────────────────────
	if errs := validate(&cfg); len(errs) > 0 {
		warnings = append(warnings, errs...)
	}

	return cfg, warnings
}

// Save sérialise la configuration courante dans un fichier JSON.
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

// ─────────────────────────────────────────────
// INTERNAL
// ─────────────────────────────────────────────

// resolveConfigPath trouve le chemin du fichier de configuration.
// Cherche dans : configPath explicite → ./.axiom/config.json → ~/.axiom/config.json
func resolveConfigPath(explicit string) string {
	if explicit != "" {
		if _, err := os.Stat(explicit); err == nil {
			return explicit
		}
		return ""
	}
	candidates := []string{
		".axiom/config.json",
		"axiom.config.json",
	}
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

// loadFromFile lit et parse un fichier JSON dans cfg (merge — les zéro-valeurs ne sont pas écrasées).
func loadFromFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	// Unmarshal dans une map intermédiaire pour un merge propre
	return json.Unmarshal(data, cfg)
}

// applyEnvOverrides lit les variables d'environnement AXIOM_* et surcharge cfg.
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

// validate vérifie la cohérence de la configuration et retourne des avertissements.
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