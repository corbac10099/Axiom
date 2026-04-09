// Package aiassistant implémente le module IA d'Axiom.
//
// Il sert de pont entre le LLM externe (local ou cloud)
// et le moteur Axiom via l'Event Bus.
//
// Providers supportés:
// - ollama         : Ollama local (http://localhost:11434)
// - llama_cpp      : llama.cpp server (http://localhost:8000)
// - mistral        : Mistral API cloud (api.mistral.ai)
// - openai         : OpenAI API (api.openai.com)
// - anthropic      : Claude API (api.anthropic.com)
// - groq           : Groq API (api.groq.com)
// - local-cortex   : LocalAI / Cortex avec auto-download
//
// Flux d'une requête IA :
//
//	Utilisateur frappe Ctrl+Space
//	→ Module envoie requête au LLM (via HTTP/streaming)
//	→ LLM retourne texte structuré (commandes Axiom + réponse)
//	→ Parser extrait les commandes (FILE_CREATE, UI_SET_THEME, etc.)
//	→ Chaque commande est dispatchée via engine.Dispatch()
//	→ Résultat renvoyé au LLM comme contexte (loop d'agentivité)
package aiassistant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/axiom-ide/axiom/api"
	"github.com/axiom-ide/axiom/core/module"
	"github.com/axiom-ide/axiom/core/security"
	"github.com/axiom-ide/axiom/pkg/uid"
)

// ─────────────────────────────────────────────
// CONFIG DU MODULE
// ─────────────────────────────────────────────

// Config est la configuration du module IA.
type Config struct {
	// Provider : "ollama" | "llama_cpp" | "mistral" | "openai" | "anthropic" | "groq" | "none"
	Provider    string
	BaseURL     string  // ex: "http://localhost:11434" pour Ollama
	ModelID     string  // ex: "mistral:7b" pour Ollama, "gpt-4" pour OpenAI
	APIKey      string  // clé API (Mistral, OpenAI, Anthropic, Groq)
	MaxTokens   int
	Temperature float64
	TimeoutSecs int
}

// ─────────────────────────────────────────────
// MODULE IA
// ─────────────────────────────────────────────

// AIAssistantModule est le module IA in-process d'Axiom.
type AIAssistantModule struct {
	module.BaseModule
	cfg      Config
	provider LLMProvider
}

// New crée une instance du module IA.
func New(cfg Config, logger *slog.Logger) *AIAssistantModule {
	provider := NewLLMProvider(cfg)
	return &AIAssistantModule{
		BaseModule: module.NewBase(
			"ai-assistant",
			"AI Assistant",
			security.L2,
			logger,
		),
		cfg:      cfg,
		provider: provider,
	}
}

// Init câble le module sur le bus.
func (m *AIAssistantModule) Init(ctx context.Context, d module.Dispatcher, s module.Subscriber) error {
	m.BaseInit(ctx, d, s)

	m.On(api.TopicFileOpened, func(ev api.Event) {
		if result, ok := ev.Payload.(interface{ GetPath() string }); ok {
			m.Logger().Debug("ai: file opened context updated", slog.String("path", result.GetPath()))
		}
	})

	m.On(api.TopicAIResponse, func(ev api.Event) {
		if ev.Source == "engine" {
			m.Logger().Debug("ai: command result received", slog.String("correlation", ev.CorrelationID))
		}
	})

	m.Logger().Info("ai-assistant: module initialized",
		slog.String("provider", m.cfg.Provider),
		slog.String("model", m.cfg.ModelID),
	)
	return nil
}

// Stop arrête proprement le module.
func (m *AIAssistantModule) Stop() error {
	return m.BaseStop()
}

// ─────────────────────────────────────────────
// QUERY — Interface publique du module
// ─────────────────────────────────────────────

// QueryResult est le résultat d'une requête au LLM.
type QueryResult struct {
	RawResponse  string         // texte brut retourné par le LLM
	Commands     []ParsedCommand // commandes Axiom extraites
	ThinkingText string         // réponse textuelle (hors commandes)
}

// ParsedCommand est une commande Axiom extraite de la réponse du LLM.
type ParsedCommand struct {
	Raw     string        // texte brut de la commande
	Topic   api.Topic     // Topic Axiom résolu
	Payload interface{}   // payload structuré
}

// Query envoie un prompt au LLM, parse la réponse, et dispatche les commandes.
func (m *AIAssistantModule) Query(ctx context.Context, userPrompt string, codeContext string) (QueryResult, error) {
	systemPrompt := buildSystemPrompt()
	fullPrompt := buildUserPrompt(userPrompt, codeContext)

	m.Logger().Info("ai: sending query",
		slog.String("model", m.cfg.ModelID),
		slog.String("provider", m.cfg.Provider),
		slog.Int("prompt_len", len(fullPrompt)),
	)

	rawResponse, err := m.provider.Complete(ctx, systemPrompt, fullPrompt)
	if err != nil {
		m.Logger().Error("ai: LLM query failed", slog.String("error", err.Error()))
		return QueryResult{}, fmt.Errorf("ai: LLM query failed: %w", err)
	}

	m.Logger().Debug("ai: response received", slog.Int("response_len", len(rawResponse)))

	result := parseResponse(rawResponse)

	for _, cmd := range result.Commands {
		corrID := uid.New()
		dispatchErr := m.Emit(api.TopicAICommand, api.PayloadAICommand{
			RawCommand:    cmd.Raw,
			ParsedTopic:   cmd.Topic,
			ParsedPayload: cmd.Payload,
		})
		if dispatchErr != nil {
			m.Logger().Warn("ai: command dispatch failed",
				slog.String("command", cmd.Raw),
				slog.String("error", dispatchErr.Error()),
				slog.String("correlation_id", corrID),
			)
		}
	}

	return result, nil
}

// ─────────────────────────────────────────────
// COMMAND PARSER
// ─────────────────────────────────────────────

// parseResponse extrait les commandes Axiom d'une réponse LLM.
//
// Format attendu :
//	<axiom:command>FILE_CREATE path/to/file.go package main\n\nfunc main(){}</axiom:command>
func parseResponse(raw string) QueryResult {
	result := QueryResult{RawResponse: raw}

	const openTag = "<axiom:command>"
	const closeTag = "</axiom:command>"

	remaining := raw
	var thinkingParts []string

	for {
		start := strings.Index(remaining, openTag)
		if start == -1 {
			thinkingParts = append(thinkingParts, remaining)
			break
		}
		if start > 0 {
			thinkingParts = append(thinkingParts, remaining[:start])
		}
		remaining = remaining[start+len(openTag):]

		end := strings.Index(remaining, closeTag)
		if end == -1 {
			break
		}

		rawCmd := strings.TrimSpace(remaining[:end])
		remaining = remaining[end+len(closeTag):]

		if cmd, ok := parseCommand(rawCmd); ok {
			result.Commands = append(result.Commands, cmd)
		}
	}

	result.ThinkingText = strings.TrimSpace(strings.Join(thinkingParts, ""))
	return result
}

// parseCommand convertit une commande brute en ParsedCommand structuré.
func parseCommand(raw string) (ParsedCommand, bool) {
	parts := strings.SplitN(raw, " ", 3)
	if len(parts) == 0 {
		return ParsedCommand{}, false
	}

	verb := strings.ToUpper(strings.TrimSpace(parts[0]))

	switch verb {
	case "FILE_CREATE":
		if len(parts) < 3 {
			return ParsedCommand{}, false
		}
		return ParsedCommand{
			Raw:   raw,
			Topic: api.TopicFileCreate,
			Payload: api.PayloadFileCreate{
				Path:    strings.TrimSpace(parts[1]),
				Content: parts[2],
			},
		}, true

	case "FILE_WRITE":
		if len(parts) < 3 {
			return ParsedCommand{}, false
		}
		return ParsedCommand{
			Raw:   raw,
			Topic: api.TopicFileWrite,
			Payload: api.PayloadFileWrite{
				Path:    strings.TrimSpace(parts[1]),
				Content: parts[2],
				Append:  false,
			},
		}, true

	case "FILE_READ":
		if len(parts) < 2 {
			return ParsedCommand{}, false
		}
		return ParsedCommand{
			Raw:   raw,
			Topic: api.TopicFileRead,
			Payload: api.PayloadFileRead{
				Path: strings.TrimSpace(parts[1]),
			},
		}, true

	case "UI_SET_THEME":
		if len(parts) < 2 {
			return ParsedCommand{}, false
		}
		return ParsedCommand{
			Raw:   raw,
			Topic: api.TopicUISetTheme,
			Payload: api.PayloadUITheme{
				ThemeID: strings.TrimSpace(parts[1]),
			},
		}, true

	case "UI_OPEN_PANEL":
		if len(parts) < 2 {
			return ParsedCommand{}, false
		}
		title := "Panel"
		if len(parts) >= 3 {
			title = parts[2]
		}
		return ParsedCommand{
			Raw:   raw,
			Topic: api.TopicUIOpenPanel,
			Payload: api.PayloadUIPanel{
				PanelID:  strings.TrimSpace(parts[1]),
				Title:    title,
				Position: "bottom",
			},
		}, true
	}

	return ParsedCommand{}, false
}

// ─────────────────────────────────────────────
// PROMPTS
// ─────────────────────────────────────────────

func buildSystemPrompt() string {
	return `You are Axiom AI, an intelligent coding assistant integrated into the Axiom IDE.

You can interact with the IDE by emitting commands inside <axiom:command> tags.

Available commands:
FILE_CREATE <path> <content>     — Create a new file
FILE_WRITE <path> <content>      — Overwrite a file
FILE_READ <path>                 — Read a file
UI_SET_THEME <theme_id>          — Change theme (dark/light/monokai/solarized)
UI_OPEN_PANEL <panel_id> <title> — Open a UI panel

Rules:
- Emit commands only when explicitly asked
- Think step by step before emitting commands
- Explain actions in plain text OUTSIDE command tags
- Never emit commands that could harm the system
- All commands are security-checked; actions outside clearance are rejected

Example:
I'll create the main.go file for you.
<axiom:command>FILE_CREATE main.go package main\n\nimport "fmt"\n\nfunc main() {\n\tfmt.Println("Hello, Axiom!")\n}</axiom:command>
The file has been created.`
}

func buildUserPrompt(userMessage, codeContext string) string {
	if codeContext == "" {
		return userMessage
	}
	return fmt.Sprintf("Code context:\n```\n%s\n```\n\nUser request: %s", codeContext, userMessage)
}