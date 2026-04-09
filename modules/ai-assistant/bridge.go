// Package aiassistant implémente le module IA d'Axiom.
//
// Il sert de pont entre le LLM externe (Ollama/Mistral local ou OpenAI)
// et le moteur Axiom via l'Event Bus.
//
// Flux d'une requête IA :
//
//	Utilisateur frappe Ctrl+Space (ou un événement déclencheur)
//	→ Module envoie une requête au LLM via HTTP
//	→ Le LLM retourne du texte structuré (commandes Axiom + réponse)
//	→ Le parser extrait les commandes (ex: FILE_CREATE, UI_SET_THEME)
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

// Config est la configuration du module IA, lue depuis le manifest + config.json.
type Config struct {
	Provider    string  // "ollama" | "openai" | "anthropic"
	BaseURL     string  // ex: "http://localhost:11434"
	ModelID     string  // ex: "mistral:7b"
	APIKey      string  // vide pour Ollama local
	MaxTokens   int
	Temperature float64
	TimeoutSecs int
}

// ─────────────────────────────────────────────
// MODULE IA
// ─────────────────────────────────────────────

// AIAssistantModule est le module IA in-process d'Axiom.
// Il embarque BaseModule pour le boilerplate de cycle de vie.
type AIAssistantModule struct {
	module.BaseModule
	cfg    Config
	client *LLMClient
}

// New crée une instance du module IA.
func New(cfg Config, logger *slog.Logger) *AIAssistantModule {
	return &AIAssistantModule{
		BaseModule: module.NewBase(
			"ai-assistant",
			"AI Assistant",
			security.L2,
			logger,
		),
		cfg:    cfg,
		client: NewLLMClient(cfg),
	}
}

// Init câble le module sur le bus : écoute SystemReady + FileOpened.
func (m *AIAssistantModule) Init(ctx context.Context, d module.Dispatcher, s module.Subscriber) error {
	m.BaseInit(ctx, d, s)

	// Écouter les fichiers ouverts → envoyer au contexte du LLM
	m.On(api.TopicFileOpened, func(ev api.Event) {
		if result, ok := ev.Payload.(interface{ GetPath() string }); ok {
			m.Logger().Debug("ai: file opened context updated", slog.String("path", result.GetPath()))
		}
	})

	// Écouter les réponses du moteur à nos commandes
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
	Raw     string    // texte brut de la commande (ex: "FILE_CREATE path content")
	Topic   api.Topic // Topic Axiom résolu
	Payload interface{} // payload structuré
}

// Query envoie un prompt au LLM, parse la réponse, et dispatche les commandes.
// C'est le point d'entrée principal du module IA.
func (m *AIAssistantModule) Query(ctx context.Context, userPrompt string, codeContext string) (QueryResult, error) {
	// Construire le prompt système avec les instructions de formatage des commandes
	systemPrompt := buildSystemPrompt()
	fullPrompt := buildUserPrompt(userPrompt, codeContext)

	m.Logger().Info("ai: sending query",
		slog.String("model", m.cfg.ModelID),
		slog.Int("prompt_len", len(fullPrompt)),
	)

	// Appel au LLM
	rawResponse, err := m.client.Complete(ctx, systemPrompt, fullPrompt)
	if err != nil {
		return QueryResult{}, fmt.Errorf("ai: LLM query failed: %w", err)
	}

	m.Logger().Debug("ai: response received", slog.Int("response_len", len(rawResponse)))

	// Parser la réponse pour extraire les commandes Axiom
	result := parseResponse(rawResponse)

	// Dispatcher chaque commande via le bus (permissions vérifiées)
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
// Format attendu dans la réponse du LLM :
//
//	<axiom:command>FILE_CREATE path/to/file.go package main\n\nfunc main(){}</axiom:command>
//	<axiom:command>UI_SET_THEME monokai</axiom:command>
//
// Le texte hors balises est la réponse textuelle à afficher à l'utilisateur.
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
		// Texte avant la commande → partie thinking
		if start > 0 {
			thinkingParts = append(thinkingParts, remaining[:start])
		}
		remaining = remaining[start+len(openTag):]

		end := strings.Index(remaining, closeTag)
		if end == -1 {
			break // balise non fermée — on ignore le reste
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
//
// Format des commandes reconnues :
//
//	FILE_CREATE <path> <content>
//	FILE_WRITE  <path> <content>
//	FILE_READ   <path>
//	UI_SET_THEME <theme_id>
//	UI_OPEN_PANEL <panel_id> [title]
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
// LLM CLIENT
// ─────────────────────────────────────────────

// LLMClient est le client HTTP générique vers un LLM.
// Il supporte Ollama (local) et OpenAI-compatible APIs.
type LLMClient struct {
	cfg        Config
	httpClient *http.Client
}

// NewLLMClient crée un client HTTP configuré.
func NewLLMClient(cfg Config) *LLMClient {
	timeout := time.Duration(cfg.TimeoutSecs) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	return &LLMClient{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// Complete envoie un prompt au LLM et retourne la réponse textuelle complète.
// Supporte : Ollama (provider="ollama") et OpenAI-compatible (provider="openai").
func (c *LLMClient) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	switch c.cfg.Provider {
	case "ollama":
		return c.completeOllama(ctx, systemPrompt, userPrompt)
	case "openai", "anthropic", "groq":
		return c.completeOpenAI(ctx, systemPrompt, userPrompt)
	case "none", "":
		// Mode "none" — retourne un stub pour les tests
		return fmt.Sprintf("[AI STUB] Provider is 'none'. Prompt was: %s", userPrompt), nil
	default:
		return "", fmt.Errorf("ai: unknown provider '%s'", c.cfg.Provider)
	}
}

// ── Ollama API (http://localhost:11434/api/generate) ─────────────────────────

type ollamaRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	System  string `json:"system,omitempty"`
	Stream  bool   `json:"stream"`
	Options struct {
		Temperature float64 `json:"temperature"`
		NumPredict  int     `json:"num_predict"`
	} `json:"options"`
}

type ollamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
	Error    string `json:"error,omitempty"`
}

func (c *LLMClient) completeOllama(ctx context.Context, system, user string) (string, error) {
	req := ollamaRequest{
		Model:  c.cfg.ModelID,
		Prompt: user,
		System: system,
		Stream: false,
	}
	req.Options.Temperature = c.cfg.Temperature
	req.Options.NumPredict = c.cfg.MaxTokens

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/api/generate"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("ai: ollama HTTP error: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ai: cannot read ollama response: %w", err)
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(data, &ollamaResp); err != nil {
		return "", fmt.Errorf("ai: cannot parse ollama response: %w", err)
	}
	if ollamaResp.Error != "" {
		return "", fmt.Errorf("ai: ollama error: %s", ollamaResp.Error)
	}
	return ollamaResp.Response, nil
}

// ── OpenAI-compatible API (/v1/chat/completions) ────────────────────────────

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *LLMClient) completeOpenAI(ctx context.Context, system, user string) (string, error) {
	req := openAIRequest{
		Model: c.cfg.ModelID,
		Messages: []openAIMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		MaxTokens:   c.cfg.MaxTokens,
		Temperature: c.cfg.Temperature,
	}

	body, _ := json.Marshal(req)
	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/v1/chat/completions"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("ai: openai HTTP error: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var openAIResp openAIResponse
	if err := json.Unmarshal(data, &openAIResp); err != nil {
		return "", fmt.Errorf("ai: cannot parse openai response: %w", err)
	}
	if openAIResp.Error != nil {
		return "", fmt.Errorf("ai: openai API error: %s", openAIResp.Error.Message)
	}
	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("ai: openai returned no choices")
	}
	return openAIResp.Choices[0].Message.Content, nil
}

// ─────────────────────────────────────────────
// PROMPTS
// ─────────────────────────────────────────────

// buildSystemPrompt construit le prompt système qui instruit le LLM
// sur le format des commandes Axiom.
func buildSystemPrompt() string {
	return `You are Axiom AI, an intelligent coding assistant integrated into the Axiom IDE.

You can interact with the IDE by emitting commands inside <axiom:command> tags.
Available commands (emit them exactly as shown):

FILE_CREATE <path> <content>     — Create a new file
FILE_WRITE <path> <content>      — Overwrite an existing file  
FILE_READ <path>                 — Read a file (result shown to you next turn)
UI_SET_THEME <theme_id>          — Change editor theme (dark/light/monokai/solarized)
UI_OPEN_PANEL <panel_id> <title> — Open a UI panel

Rules:
- Emit commands only when the user explicitly asks for an action.
- Think step by step before emitting commands.
- Explain what you are doing in plain text OUTSIDE the command tags.
- Never emit commands that could harm the user's system.
- Commands are security-checked; actions outside your clearance will be rejected.

Example response:
I'll create the main.go file for you.
<axiom:command>FILE_CREATE main.go package main\n\nimport "fmt"\n\nfunc main() {\n\tfmt.Println("Hello, Axiom!")\n}</axiom:command>
The file has been created with a basic Hello World program.`
}

// buildUserPrompt construit le prompt utilisateur avec contexte de code.
func buildUserPrompt(userMessage, codeContext string) string {
	if codeContext == "" {
		return userMessage
	}
	return fmt.Sprintf("Code context:\n```\n%s\n```\n\nUser request: %s", codeContext, userMessage)
}