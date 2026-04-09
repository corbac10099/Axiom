package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/axiom-ide/axiom/core/config"
)

func TestDefaultValues(t *testing.T) {
	cfg := config.Default()
	if cfg.Core.ModulesDir != "./modules" {
		t.Errorf("expected default modules_dir './modules', got '%s'", cfg.Core.ModulesDir)
	}
	if cfg.Core.BusBufferSize != 128 {
		t.Errorf("expected buffer size 128, got %d", cfg.Core.BusBufferSize)
	}
	if cfg.AI.Provider != "ollama" {
		t.Errorf("expected default AI provider 'ollama', got '%s'", cfg.AI.Provider)
	}
	if cfg.UI.DefaultTheme != "dark" {
		t.Errorf("expected default theme 'dark', got '%s'", cfg.UI.DefaultTheme)
	}
}

func TestLoadFromFile(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.json")
	data := `{
		"core": {"log_level": "debug", "bus_buffer_size": 256},
		"ui":   {"default_theme": "monokai", "font_size": 16}
	}`
	if err := os.WriteFile(cfgPath, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, warns := config.Load(cfgPath)
	for _, w := range warns {
		t.Logf("warning: %s", w)
	}
	if cfg.Core.LogLevel != "debug" {
		t.Errorf("expected log_level 'debug', got '%s'", cfg.Core.LogLevel)
	}
	if cfg.Core.BusBufferSize != 256 {
		t.Errorf("expected bus_buffer_size 256, got %d", cfg.Core.BusBufferSize)
	}
	if cfg.UI.DefaultTheme != "monokai" {
		t.Errorf("expected theme 'monokai', got '%s'", cfg.UI.DefaultTheme)
	}
	if cfg.UI.FontSize != 16 {
		t.Errorf("expected font_size 16, got %d", cfg.UI.FontSize)
	}
	if cfg.AI.Provider != "ollama" {
		t.Errorf("expected default AI provider 'ollama', got '%s'", cfg.AI.Provider)
	}
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv("AXIOM_LOG_LEVEL", "warn")
	t.Setenv("AXIOM_AI_PROVIDER", "openai")
	t.Setenv("AXIOM_THEME", "light")
	t.Setenv("AXIOM_BUS_BUFFER", "512")
	cfg, _ := config.Load("")
	if cfg.Core.LogLevel != "warn" {
		t.Errorf("expected log_level 'warn' from env, got '%s'", cfg.Core.LogLevel)
	}
	if cfg.AI.Provider != "openai" {
		t.Errorf("expected AI provider 'openai' from env, got '%s'", cfg.AI.Provider)
	}
	if cfg.UI.DefaultTheme != "light" {
		t.Errorf("expected theme 'light' from env, got '%s'", cfg.UI.DefaultTheme)
	}
	if cfg.Core.BusBufferSize != 512 {
		t.Errorf("expected bus_buffer_size 512 from env, got %d", cfg.Core.BusBufferSize)
	}
}

func TestValidationFixes(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "bad.json")
	bad := `{"core": {"log_level": "INVALID", "bus_buffer_size": 2}, "ui": {"font_size": 200}, "ai": {"temperature": 5.0}}`
	if err := os.WriteFile(cfgPath, []byte(bad), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, warns := config.Load(cfgPath)
	if len(warns) == 0 {
		t.Error("expected validation warnings for bad config")
	}
	if cfg.Core.LogLevel != "info" {
		t.Errorf("invalid log_level should fall back to 'info', got '%s'", cfg.Core.LogLevel)
	}
	if cfg.Core.BusBufferSize != 128 {
		t.Errorf("too-small bus_buffer_size should fall back to 128, got %d", cfg.Core.BusBufferSize)
	}
	if cfg.UI.FontSize != 14 {
		t.Errorf("out-of-range font_size should fall back to 14, got %d", cfg.UI.FontSize)
	}
	if cfg.AI.Temperature != 0.2 {
		t.Errorf("out-of-range temperature should fall back to 0.2, got %f", cfg.AI.Temperature)
	}
}

func TestSaveAndReload(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "saved.json")
	original := config.Default()
	original.Core.LogLevel = "warn"
	original.UI.DefaultTheme = "solarized"
	if err := config.Save(original, path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	data, _ := os.ReadFile(path)
	var roundtrip map[string]interface{}
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatalf("saved JSON is invalid: %v", err)
	}
	reloaded, _ := config.Load(path)
	if reloaded.Core.LogLevel != "warn" {
		t.Errorf("reloaded log_level mismatch: got '%s'", reloaded.Core.LogLevel)
	}
	if reloaded.UI.DefaultTheme != "solarized" {
		t.Errorf("reloaded theme mismatch: got '%s'", reloaded.UI.DefaultTheme)
	}
}