// Package api définit le contrat public d'Axiom :
// tous les types d'événements, commandes et réponses
// qui circulent sur l'Event Bus. C'est le "langage commun"
// de l'écosystème modulaire.
package api

import (
	"time"
)

// ─────────────────────────────────────────────
// TOPICS — Noms canoniques des canaux du Bus
// ─────────────────────────────────────────────

// Topic est une chaîne typée pour éviter les fautes de frappe
// dans les souscriptions/publications.
type Topic string

const (
	// Système / Cycle de vie
	TopicSystemReady    Topic = "system.ready"
	TopicSystemShutdown Topic = "system.shutdown"
	TopicModuleLoaded   Topic = "system.module.loaded"
	TopicModuleError    Topic = "system.module.error"

	// Fichiers
	TopicFileCreate Topic = "file.create"
	TopicFileRead   Topic = "file.read"
	TopicFileWrite  Topic = "file.write"
	TopicFileDelete Topic = "file.delete"
	TopicFileOpened Topic = "file.opened" // broadcast après ouverture réussie

	// Interface Utilisateur
	TopicUIOpenPanel  Topic = "ui.panel.open"
	TopicUIClosePanel Topic = "ui.panel.close"
	TopicUISetTheme   Topic = "ui.theme.set"
	TopicUINewWindow  Topic = "ui.window.new"

	// IA
	TopicAICommand  Topic = "ai.command"
	TopicAIResponse Topic = "ai.response"

	// Sécurité
	TopicSecurityDenied Topic = "security.denied"
	TopicSecurityAudit  Topic = "security.audit"
)

// ─────────────────────────────────────────────
// EVENT — Enveloppe universelle du Bus
// ─────────────────────────────────────────────

// Event est la structure atomique qui transite sur l'Event Bus.
// Tout module publie et reçoit des Event.
type Event struct {
	// ID unique de cet événement (UUID v4)
	ID string `json:"id"`

	// Topic identifie le "canal" de l'événement (ex: "file.create")
	Topic Topic `json:"topic"`

	// Source est l'identifiant du module émetteur (ex: "ai-assistant")
	Source string `json:"source"`

	// Payload contient les données spécifiques au Topic.
	// On utilise interface{} pour la flexibilité ; chaque handler
	// est responsable du cast vers le type concret attendu.
	Payload interface{} `json:"payload,omitempty"`

	// Timestamp de création de l'événement (UTC)
	Timestamp time.Time `json:"timestamp"`

	// ReplyTo : si défini, la réponse sera publiée sur ce Topic.
	// Permet un pattern request/reply asynchrone.
	ReplyTo Topic `json:"reply_to,omitempty"`

	// CorrelationID permet de lier une réponse à sa requête d'origine.
	CorrelationID string `json:"correlation_id,omitempty"`
}

// ─────────────────────────────────────────────
// PAYLOADS — Structures de données pour chaque Topic
// ─────────────────────────────────────────────

// PayloadFileCreate est le payload attendu pour TopicFileCreate.
type PayloadFileCreate struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// PayloadFileWrite est le payload attendu pour TopicFileWrite.
type PayloadFileWrite struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Append  bool   `json:"append"`
}

// PayloadFileRead est le payload attendu pour TopicFileRead.
type PayloadFileRead struct {
	Path string `json:"path"`
}

// PayloadUITheme est le payload attendu pour TopicUISetTheme.
type PayloadUITheme struct {
	ThemeID string `json:"theme_id"` // ex: "dark", "light", "monokai"
}

// PayloadUIPanel est le payload attendu pour TopicUIOpenPanel / ClosePanel.
type PayloadUIPanel struct {
	PanelID  string `json:"panel_id"`  // ex: "bottom_panel", "sidebar"
	Position string `json:"position"`  // ex: "bottom", "left", "right"
	Title    string `json:"title"`
	Content  string `json:"content,omitempty"` // HTML initial optionnel
}

// PayloadAICommand est le payload pour TopicAICommand.
type PayloadAICommand struct {
	// RawCommand est la commande brute générée par le module IA
	// avant parsing (ex: "UI_SET_THEME_RED")
	RawCommand string `json:"raw_command"`
	// ParsedTopic est le Topic Axiom résolu depuis RawCommand
	ParsedTopic Topic `json:"parsed_topic"`
	// ParsedPayload est le payload résolu
	ParsedPayload interface{} `json:"parsed_payload"`
}

// PayloadSecurityDenied est publié sur TopicSecurityDenied
// quand une action est rejetée par le Security Manager.
type PayloadSecurityDenied struct {
	ModuleID       string `json:"module_id"`
	AttemptedTopic Topic  `json:"attempted_topic"`
	RequiredLevel  int    `json:"required_level"`
	ActualLevel    int    `json:"actual_level"`
	Reason         string `json:"reason"`
}

// PayloadModuleStatus est publié lors du chargement d'un module.
type PayloadModuleStatus struct {
	ModuleID string `json:"module_id"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Error    string `json:"error,omitempty"`
}