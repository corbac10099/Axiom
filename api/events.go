// Package api définit le contrat public d'Axiom :
// tous les types d'événements, commandes et réponses
// qui circulent sur l'Event Bus.
package api

import (
	"time"
)

// Topic est une chaîne typée pour éviter les fautes de frappe.
type Topic string

const (
	TopicSystemReady    Topic = "system.ready"
	TopicSystemShutdown Topic = "system.shutdown"
	TopicModuleLoaded   Topic = "system.module.loaded"
	TopicModuleError    Topic = "system.module.error"

	TopicFileCreate Topic = "file.create"
	TopicFileRead   Topic = "file.read"
	TopicFileWrite  Topic = "file.write"
	TopicFileDelete Topic = "file.delete"
	TopicFileOpened Topic = "file.opened"

	TopicUIOpenPanel  Topic = "ui.panel.open"
	TopicUIClosePanel Topic = "ui.panel.close"
	TopicUISetTheme   Topic = "ui.theme.set"
	TopicUINewWindow  Topic = "ui.window.new"

	TopicAICommand  Topic = "ai.command"
	TopicAIResponse Topic = "ai.response"

	TopicSecurityDenied Topic = "security.denied"
	TopicSecurityAudit  Topic = "security.audit"
)

// Event est la structure atomique qui transite sur l'Event Bus.
type Event struct {
	ID            string      `json:"id"`
	Topic         Topic       `json:"topic"`
	Source        string      `json:"source"`
	Payload       interface{} `json:"payload,omitempty"`
	Timestamp     time.Time   `json:"timestamp"`
	ReplyTo       Topic       `json:"reply_to,omitempty"`
	CorrelationID string      `json:"correlation_id,omitempty"`
}

// --- Payloads ---

type PayloadFileCreate struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type PayloadFileWrite struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Append  bool   `json:"append"`
}

type PayloadFileRead struct {
	Path string `json:"path"`
}

type PayloadUITheme struct {
	ThemeID string `json:"theme_id"`
}

type PayloadUIPanel struct {
	PanelID  string `json:"panel_id"`
	Position string `json:"position"`
	Title    string `json:"title"`
	Content  string `json:"content,omitempty"`
}

type PayloadAICommand struct {
	RawCommand    string      `json:"raw_command"`
	ParsedTopic   Topic       `json:"parsed_topic"`
	ParsedPayload interface{} `json:"parsed_payload"`
}

type PayloadSecurityDenied struct {
	ModuleID       string `json:"module_id"`
	AttemptedTopic Topic  `json:"attempted_topic"`
	RequiredLevel  int    `json:"required_level"`
	ActualLevel    int    `json:"actual_level"`
	Reason         string `json:"reason"`
}

type PayloadModuleStatus struct {
	ModuleID string `json:"module_id"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Error    string `json:"error,omitempty"`
}