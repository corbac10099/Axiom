// Package api définit le contrat public d'Axiom :
// tous les types d'événements, commandes et réponses
// qui circulent sur l'Event Bus.
package api

import "time"

// Topic est une chaîne typée pour éviter les fautes de frappe.
type Topic string

// ── Système ───────────────────────────────────────────────────────
const (
	TopicSystemReady    Topic = "system.ready"
	TopicSystemShutdown Topic = "system.shutdown"
	TopicModuleLoaded   Topic = "system.module.loaded"
	TopicModuleError    Topic = "system.module.error"
)

// ── Fichiers ──────────────────────────────────────────────────────
const (
	TopicFileCreate Topic = "file.create"
	TopicFileRead   Topic = "file.read"
	TopicFileWrite  Topic = "file.write"
	TopicFileDelete Topic = "file.delete"
	TopicFileOpened Topic = "file.opened"
)

// ── UI ────────────────────────────────────────────────────────────
const (
	TopicUIOpenPanel  Topic = "ui.panel.open"
	TopicUIClosePanel Topic = "ui.panel.close"
	TopicUISetTheme   Topic = "ui.theme.set"
	TopicUINewWindow  Topic = "ui.window.new"
	TopicUIUserInput  Topic = "ui.user.input"
)

// ── Éditeur ───────────────────────────────────────────────────────
const (
	TopicEditorTabOpen    Topic = "editor.tab.open"
	TopicEditorTabClose   Topic = "editor.tab.close"
	TopicEditorTabFocus   Topic = "editor.tab.focus"
	TopicEditorTabChanged Topic = "editor.tab.changed"
)

// ── Workspace ─────────────────────────────────────────────────────
const (
	TopicWorkspaceSave     Topic = "workspace.save"
	TopicWorkspaceRestore  Topic = "workspace.restore"
	TopicWorkspaceRestored Topic = "workspace.restored"
)

// ── IA ────────────────────────────────────────────────────────────
const (
	TopicAICommand  Topic = "ai.command"
	TopicAIResponse Topic = "ai.response"
)

// ── Sécurité ──────────────────────────────────────────────────────
const (
	TopicSecurityDenied Topic = "security.denied"
	TopicSecurityAudit  Topic = "security.audit"
)

// ─────────────────────────────────────────────────────────────────
// Event — structure atomique transitant sur le bus
// ─────────────────────────────────────────────────────────────────

type Event struct {
	ID            string      `json:"id"`
	Topic         Topic       `json:"topic"`
	Source        string      `json:"source"`
	Payload       interface{} `json:"payload,omitempty"`
	Timestamp     time.Time   `json:"timestamp"`
	ReplyTo       Topic       `json:"reply_to,omitempty"`
	CorrelationID string      `json:"correlation_id,omitempty"`
}

// ─────────────────────────────────────────────────────────────────
// Payloads fichiers
// ─────────────────────────────────────────────────────────────────

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

// ─────────────────────────────────────────────────────────────────
// Payloads UI
// ─────────────────────────────────────────────────────────────────

type PayloadUITheme struct {
	ThemeID string `json:"theme_id"`
}

type PayloadUIPanel struct {
	PanelID  string `json:"panel_id"`
	Position string `json:"position"`
	Title    string `json:"title"`
	Content  string `json:"content,omitempty"`
}

type PayloadUIUserInput struct {
	WindowID  string      `json:"window_id"`
	EventType string      `json:"event_type"` // "keydown" | "click" | "command"
	Data      interface{} `json:"data"`
}

// ─────────────────────────────────────────────────────────────────
// Payloads éditeur
// ─────────────────────────────────────────────────────────────────

type PayloadEditorTab struct {
	TabID    string `json:"tab_id"`
	WindowID string `json:"window_id"`
	FilePath string `json:"file_path"`
	Title    string `json:"title"`
	IsDirty  bool   `json:"is_dirty"`
	Language string `json:"language"`
}

// ─────────────────────────────────────────────────────────────────
// Payloads workspace
// ─────────────────────────────────────────────────────────────────

type PayloadWorkspaceSave struct {
	TargetPath string `json:"target_path"`
}

type PayloadWorkspaceRestored struct {
	TabsRestored   int    `json:"tabs_restored"`
	PanelsRestored int    `json:"panels_restored"`
	SourcePath     string `json:"source_path"`
}

// ─────────────────────────────────────────────────────────────────
// Payloads IA
// ─────────────────────────────────────────────────────────────────

type PayloadAICommand struct {
	RawCommand    string      `json:"raw_command"`
	ParsedTopic   Topic       `json:"parsed_topic"`
	ParsedPayload interface{} `json:"parsed_payload"`
}

// ─────────────────────────────────────────────────────────────────
// Payloads système
// ─────────────────────────────────────────────────────────────────

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