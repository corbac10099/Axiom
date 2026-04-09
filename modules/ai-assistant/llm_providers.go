package aiassistant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ─────────────────────────────────────────────
// LLM PROVIDER INTERFACE
// ─────────────────────────────────────────────

// LLMProvider est l'interface abstraite pour tous les providers.
type LLMProvider interface {
	Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

// NewLLMProvider crée le provider approprié selon la configuration.
func NewLLMProvider(cfg Config) LLMProvider {
	timeout := time.Duration(cfg.TimeoutSecs) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	baseClient := &http.Client{Timeout: timeout}

	switch strings.ToLower(cfg.Provider) {
	case "ollama":
		return &OllamaProvider{
			baseURL: cfg.BaseURL,
			model:   cfg.ModelID,
			client:  baseClient,
			temp:    cfg.Temperature,
			tokens:  cfg.MaxTokens,
		}
	case "llama_cpp", "llama.cpp":
		return &LlamaCppProvider{
			baseURL: cfg.BaseURL,
			model:   cfg.ModelID,
			client:  baseClient,
			temp:    cfg.Temperature,
			tokens:  cfg.MaxTokens,
		}
	case "mistral":
		return &MistralProvider{
			apiKey:  cfg.APIKey,
			model:   cfg.ModelID,
			client:  baseClient,
			temp:    cfg.Temperature,
			tokens:  cfg.MaxTokens,
		}
	case "openai":
		return &OpenAIProvider{
			apiKey:  cfg.APIKey,
			model:   cfg.ModelID,
			client:  baseClient,
			temp:    cfg.Temperature,
			tokens:  cfg.MaxTokens,
		}
	case "anthropic":
		return &AnthropicProvider{
			apiKey:  cfg.APIKey,
			model:   cfg.ModelID,
			client:  baseClient,
			temp:    cfg.Temperature,
			tokens:  cfg.MaxTokens,
		}
	case "groq":
		return &GroqProvider{
			apiKey:  cfg.APIKey,
			model:   cfg.ModelID,
			client:  baseClient,
			temp:    cfg.Temperature,
			tokens:  cfg.MaxTokens,
		}
	case "none", "":
		return &StubProvider{model: cfg.ModelID}
	default:
		return &StubProvider{model: cfg.ModelID}
	}
}

// ─────────────────────────────────────────────
// OLLAMA PROVIDER
// ─────────────────────────────────────────────

type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
	temp    float64
	tokens  int
}

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

func (p *OllamaProvider) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	req := ollamaRequest{
		Model:  p.model,
		Prompt: userPrompt,
		System: systemPrompt,
		Stream: false,
	}
	req.Options.Temperature = p.temp
	req.Options.NumPredict = p.tokens

	body, _ := json.Marshal(req)
	url := strings.TrimRight(p.baseURL, "/") + "/api/generate"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("ollama HTTP error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(data))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("cannot read ollama response: %w", err)
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(data, &ollamaResp); err != nil {
		return "", fmt.Errorf("cannot parse ollama response: %w", err)
	}
	if ollamaResp.Error != "" {
		return "", fmt.Errorf("ollama error: %s", ollamaResp.Error)
	}
	return ollamaResp.Response, nil
}

// ─────────────────────────────────────────────
// LLAMA.CPP PROVIDER
// ─────────────────────────────────────────────

type LlamaCppProvider struct {
	baseURL string
	model   string
	client  *http.Client
	temp    float64
	tokens  int
}

type llamaCppRequest struct {
	Prompt      string  `json:"prompt"`
	Temperature float64 `json:"temperature"`
	NPredict    int     `json:"n_predict"`
	NCtx        int     `json:"n_ctx"`
}

type llamaCppResponse struct {
	Content string `json:"content"`
	Timings struct {
		PredictingMs float64 `json:"predicting_ms"`
	} `json:"timings"`
}

func (p *LlamaCppProvider) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	fullPrompt := systemPrompt + "\n\nUser: " + userPrompt + "\nAssistant:"

	req := llamaCppRequest{
		Prompt:      fullPrompt,
		Temperature: p.temp,
		NPredict:    p.tokens,
		NCtx:        4096,
	}

	body, _ := json.Marshal(req)
	url := strings.TrimRight(p.baseURL, "/") + "/completion"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("llama.cpp HTTP error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("llama.cpp API error (status %d): %s", resp.StatusCode, string(data))
	}

	data, _ := io.ReadAll(resp.Body)
	var llamaResp llamaCppResponse
	if err := json.Unmarshal(data, &llamaResp); err != nil {
		return "", fmt.Errorf("cannot parse llama.cpp response: %w", err)
	}
	return llamaResp.Content, nil
}

// ─────────────────────────────────────────────
// MISTRAL PROVIDER
// ─────────────────────────────────────────────

type MistralProvider struct {
	apiKey string
	model  string
	client *http.Client
	temp   float64
	tokens int
}

type mistralMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type mistralRequest struct {
	Model       string            `json:"model"`
	Messages    []mistralMessage  `json:"messages"`
	Temperature float64           `json:"temperature"`
	MaxTokens   int               `json:"max_tokens"`
}

type mistralResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *MistralProvider) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if p.apiKey == "" {
		return "", fmt.Errorf("mistral: API key not configured (set AXIOM_AI_KEY)")
	}

	req := mistralRequest{
		Model: p.model,
		Messages: []mistralMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: p.temp,
		MaxTokens:   p.tokens,
	}

	body, _ := json.Marshal(req)
	url := "https://api.mistral.ai/v1/chat/completions"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("mistral HTTP error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("mistral API error (status %d): %s", resp.StatusCode, string(data))
	}

	data, _ := io.ReadAll(resp.Body)
	var mistralResp mistralResponse
	if err := json.Unmarshal(data, &mistralResp); err != nil {
		return "", fmt.Errorf("cannot parse mistral response: %w", err)
	}
	if mistralResp.Error != nil {
		return "", fmt.Errorf("mistral API error: %s", mistralResp.Error.Message)
	}
	if len(mistralResp.Choices) == 0 {
		return "", fmt.Errorf("mistral returned no choices")
	}
	return mistralResp.Choices[0].Message.Content, nil
}

// ─────────────────────────────────────────────
// OPENAI PROVIDER
// ─────────────────────────────────────────────

type OpenAIProvider struct {
	apiKey string
	model  string
	client *http.Client
	temp   float64
	tokens int
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiRequest struct {
	Model       string           `json:"model"`
	Messages    []openaiMessage  `json:"messages"`
	Temperature float64          `json:"temperature"`
	MaxTokens   int              `json:"max_tokens"`
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *OpenAIProvider) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if p.apiKey == "" {
		return "", fmt.Errorf("openai: API key not configured (set AXIOM_AI_KEY)")
	}

	req := openaiRequest{
		Model: p.model,
		Messages: []openaiMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: p.temp,
		MaxTokens:   p.tokens,
	}

	body, _ := json.Marshal(req)
	url := "https://api.openai.com/v1/chat/completions"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("openai HTTP error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai API error (status %d): %s", resp.StatusCode, string(data))
	}

	data, _ := io.ReadAll(resp.Body)
	var openaiResp openaiResponse
	if err := json.Unmarshal(data, &openaiResp); err != nil {
		return "", fmt.Errorf("cannot parse openai response: %w", err)
	}
	if openaiResp.Error != nil {
		return "", fmt.Errorf("openai API error: %s", openaiResp.Error.Message)
	}
	if len(openaiResp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}
	return openaiResp.Choices[0].Message.Content, nil
}

// ─────────────────────────────────────────────
// ANTHROPIC PROVIDER
// ─────────────────────────────────────────────

type AnthropicProvider struct {
	apiKey string
	model  string
	client *http.Client
	temp   float64
	tokens int
}

type anthropicRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	System      string          `json:"system"`
	Messages    []anthropicMsg  `json:"messages"`
	Temperature float64         `json:"temperature"`
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *AnthropicProvider) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if p.apiKey == "" {
		return "", fmt.Errorf("anthropic: API key not configured (set AXIOM_AI_KEY)")
	}

	req := anthropicRequest{
		Model:       p.model,
		MaxTokens:   p.tokens,
		System:      systemPrompt,
		Temperature: p.temp,
		Messages: []anthropicMsg{
			{Role: "user", Content: userPrompt},
		},
	}

	body, _ := json.Marshal(req)
	url := "https://api.anthropic.com/v1/messages"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("anthropic HTTP error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("anthropic API error (status %d): %s", resp.StatusCode, string(data))
	}

	data, _ := io.ReadAll(resp.Body)
	var anthropicResp anthropicResponse
	if err := json.Unmarshal(data, &anthropicResp); err != nil {
		return "", fmt.Errorf("cannot parse anthropic response: %w", err)
	}
	if anthropicResp.Error != nil {
		return "", fmt.Errorf("anthropic API error: %s", anthropicResp.Error.Message)
	}
	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("anthropic returned no content")
	}
	return anthropicResp.Content[0].Text, nil
}

// ─────────────────────────────────────────────
// GROQ PROVIDER
// ─────────────────────────────────────────────

type GroqProvider struct {
	apiKey string
	model  string
	client *http.Client
	temp   float64
	tokens int
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqRequest struct {
	Model       string         `json:"model"`
	Messages    []groqMessage  `json:"messages"`
	Temperature float64        `json:"temperature"`
	MaxTokens   int            `json:"max_tokens"`
}

type groqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *GroqProvider) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if p.apiKey == "" {
		return "", fmt.Errorf("groq: API key not configured (set AXIOM_AI_KEY)")
	}

	req := groqRequest{
		Model: p.model,
		Messages: []groqMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: p.temp,
		MaxTokens:   p.tokens,
	}

	body, _ := json.Marshal(req)
	url := "https://api.groq.com/openai/v1/chat/completions"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("groq HTTP error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("groq API error (status %d): %s", resp.StatusCode, string(data))
	}

	data, _ := io.ReadAll(resp.Body)
	var groqResp groqResponse
	if err := json.Unmarshal(data, &groqResp); err != nil {
		return "", fmt.Errorf("cannot parse groq response: %w", err)
	}
	if groqResp.Error != nil {
		return "", fmt.Errorf("groq API error: %s", groqResp.Error.Message)
	}
	if len(groqResp.Choices) == 0 {
		return "", fmt.Errorf("groq returned no choices")
	}
	return groqResp.Choices[0].Message.Content, nil
}

// ─────────────────────────────────────────────
// STUB PROVIDER (pour tests / mode offline)
// ─────────────────────────────────────────────

type StubProvider struct {
	model string
}

func (p *StubProvider) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return fmt.Sprintf("[AI STUB - provider not configured]\nModel: %s\nUser: %s", p.model, userPrompt), nil
}