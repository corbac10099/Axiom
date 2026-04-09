package aiassistant_test

import (
	"testing"

	aiassistant "github.com/axiom-ide/axiom/modules/ai-assistant"
	"github.com/axiom-ide/axiom/api"
)

// Tests du parser de commandes — pas de dépendance réseau.

func TestParseFileCreate(t *testing.T) {
	raw := `I'll create the file for you.
<axiom:command>FILE_CREATE src/main.go package main

func main() {}</axiom:command>
Done!`

	result := aiassistant.ExportParseResponse(raw)

	if len(result.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(result.Commands))
	}
	cmd := result.Commands[0]
	if cmd.Topic != api.TopicFileCreate {
		t.Errorf("expected TopicFileCreate, got %s", cmd.Topic)
	}
	p, ok := cmd.Payload.(api.PayloadFileCreate)
	if !ok {
		t.Fatalf("expected PayloadFileCreate, got %T", cmd.Payload)
	}
	if p.Path != "src/main.go" {
		t.Errorf("expected path 'src/main.go', got '%s'", p.Path)
	}
	if result.ThinkingText == "" {
		t.Error("expected non-empty thinking text")
	}
}

func TestParseUISetTheme(t *testing.T) {
	raw := `<axiom:command>UI_SET_THEME monokai</axiom:command>`
	result := aiassistant.ExportParseResponse(raw)

	if len(result.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(result.Commands))
	}
	p, ok := result.Commands[0].Payload.(api.PayloadUITheme)
	if !ok {
		t.Fatal("expected PayloadUITheme")
	}
	if p.ThemeID != "monokai" {
		t.Errorf("expected theme 'monokai', got '%s'", p.ThemeID)
	}
}

func TestParseMultipleCommands(t *testing.T) {
	raw := `Let me create two files.
<axiom:command>FILE_CREATE a.go package a</axiom:command>
<axiom:command>FILE_CREATE b.go package b</axiom:command>
Both files created.`

	result := aiassistant.ExportParseResponse(raw)
	if len(result.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(result.Commands))
	}
}

func TestParseNoCommands(t *testing.T) {
	raw := "Here's some information about Go interfaces. No actions needed."
	result := aiassistant.ExportParseResponse(raw)
	if len(result.Commands) != 0 {
		t.Errorf("expected 0 commands, got %d", len(result.Commands))
	}
	if result.ThinkingText == "" {
		t.Error("expected non-empty thinking text")
	}
}

func TestParseUnknownCommand(t *testing.T) {
	raw := `<axiom:command>UNKNOWN_VERB some args</axiom:command>`
	result := aiassistant.ExportParseResponse(raw)
	// Les commandes inconnues doivent être ignorées silencieusement
	if len(result.Commands) != 0 {
		t.Errorf("expected 0 commands for unknown verb, got %d", len(result.Commands))
	}
}

func TestParseUnclosedTag(t *testing.T) {
	raw := `<axiom:command>FILE_READ foo.go` // balise non fermée
	result := aiassistant.ExportParseResponse(raw)
	// Doit être robuste — pas de panic
	if len(result.Commands) != 0 {
		t.Errorf("expected 0 commands for unclosed tag, got %d", len(result.Commands))
	}
}

func TestParseUIOpenPanel(t *testing.T) {
	raw := `<axiom:command>UI_OPEN_PANEL ai-chat AI Chat</axiom:command>`
	result := aiassistant.ExportParseResponse(raw)
	if len(result.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(result.Commands))
	}
	p, ok := result.Commands[0].Payload.(api.PayloadUIPanel)
	if !ok {
		t.Fatal("expected PayloadUIPanel")
	}
	if p.PanelID != "ai-chat" {
		t.Errorf("expected panel_id 'ai-chat', got '%s'", p.PanelID)
	}
	if p.Title != "AI Chat" {
		t.Errorf("expected title 'AI Chat', got '%s'", p.Title)
	}
}