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

// ── UI (natifs) ───────────────────────────────────────────────────
const (
	TopicUIOpenPanel  Topic = "ui.panel.open"
	TopicUIClosePanel Topic = "ui.panel.close"
	TopicUISetTheme   Topic = "ui.theme.set"
	TopicUINewWindow  Topic = "ui.window.new"
	TopicUIUserInput  Topic = "ui.user.input"
)

// ── UI Module System ──────────────────────────────────────────────
// Ces topics permettent aux modules d'interagir avec l'interface
// de manière complète : enregistrement de vues, injection de slots,
// branding, badges.
const (
	// TopicUIModuleRegister — enregistre un module complet avec vue HTML/CSS/JS.
	// Payload: PayloadUIModuleRegister
	TopicUIModuleRegister Topic = "ui.module.register"

	// TopicUISlotInject — injecte du HTML dans un slot précis de l'interface.
	// Payload: PayloadUISlotInject
	TopicUISlotInject Topic = "ui.slot.inject"

	// TopicUISlotRemove — retire un élément injecté par son ID.
	// Payload: PayloadUISlotRemove
	TopicUISlotRemove Topic = "ui.slot.remove"

	// TopicUIAppBranding — modifie le logo, le nom et les couleurs de l'app.
	// Payload: PayloadUIAppBranding
	TopicUIAppBranding Topic = "ui.app.branding"

	// TopicUIIconBadge — ajoute/retire un badge numérique sur l'icône d'un module.
	// Payload: PayloadUIIconBadge
	TopicUIIconBadge Topic = "ui.icon.badge"

	// TopicUIViewSwitch — force le switch vers une vue (par module ID ou 'editor').
	// Payload: PayloadUIViewSwitch
	TopicUIViewSwitch Topic = "ui.view.switch"
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
// Payloads UI natifs
// ─────────────────────────────────────────────────────────────────

type PayloadUITheme struct {
	ThemeID string `json:"theme_id"`
}

type PayloadUIPanel struct {
	PanelID  string `json:"panel_id"`
	Position string `json:"position"`
	Title    string `json:"title"`
	// Content peut contenir du HTML brut OU un JSON PayloadUIModuleRegister sérialisé.
	// Si c'est un JSON valide, le bridge l'interprète comme un enregistrement de module complet.
	// Si c'est du HTML brut, il est injecté directement comme contenu de la vue.
	Content  string `json:"content,omitempty"`
}

type PayloadUIUserInput struct {
	WindowID  string      `json:"window_id"`
	EventType string      `json:"event_type"` // "keydown" | "click" | "command"
	Data      interface{} `json:"data"`
}

// ─────────────────────────────────────────────────────────────────
// Payloads UI Module System
// ─────────────────────────────────────────────────────────────────

// ViewType contrôle comment la vue du module s'affiche.
type ViewType string

const (
	// ViewTypeReplace — vue normale dans le view-host, remplace la vue précédente.
	ViewTypeReplace ViewType = "replace"
	// ViewTypeTakeover — position:absolute par-dessus tout le content area.
	ViewTypeTakeover ViewType = "takeover"
	// ViewTypeOverlay — modale semi-transparente par-dessus toute l'app.
	ViewTypeOverlay ViewType = "overlay"
	// ViewTypeNone — pas de vue principale, le module utilise seulement des slots.
	ViewTypeNone ViewType = "none"
)

// SlotName identifie les zones d'injection disponibles dans l'interface.
type SlotName string

const (
	SlotSidebar              SlotName = "sidebar"               // Zone basse de la sidebar
	SlotSidebarHeaderActions SlotName = "sidebar-header-actions" // Boutons dans le header sidebar
	SlotStatusbarLeft        SlotName = "statusbar-left"         // Statusbar gauche (après branch)
	SlotStatusbarRight       SlotName = "statusbar-right"        // Statusbar droite (avant lang)
	SlotTabBar               SlotName = "tab-bar"               // Zone droite de la tab bar
	SlotPanelTab             SlotName = "panel-tab"             // Onglet dans le bottom panel
	SlotPanelContent         SlotName = "panel-content"         // Contenu du bottom panel (remplace)
	SlotActivityBarBottom    SlotName = "activity-bar-bottom"   // Section basse activity bar
)

// PayloadUIModuleRegister enregistre un module UI complet.
// Envoyé via TopicUIModuleRegister (ou sérialisé dans PayloadUIPanel.Content).
type PayloadUIModuleRegister struct {
	// ModuleID doit correspondre à l'ID du module Go (ex: "ai-assistant").
	ModuleID string `json:"moduleId"`

	// Icon : emoji ou texte court (ex: "🤖", "AI", "◈").
	Icon string `json:"icon,omitempty"`

	// IconImageURL : URL d'image pour l'icône (data: URI ou chemin relatif au frontend).
	// Si fourni, remplace Icon.
	IconImageURL string `json:"iconImageUrl,omitempty"`

	// Title : nom affiché dans la sidebar header quand le module est actif.
	Title string `json:"title"`

	// ViewType : comment la vue principale s'affiche.
	ViewType ViewType `json:"viewType"`

	// HTML : contenu HTML de la vue principale.
	// Peut contenir des balises <style> et <script> inline.
	HTML string `json:"html,omitempty"`

	// CSS : feuille de style scopée, injectée dans <head> avec id="style-module-{moduleId}".
	// Rechargeable : un second appel remplace le précédent.
	CSS string `json:"css,omitempty"`

	// JS : code JavaScript exécuté une seule fois dans une IIFE.
	// La variable `moduleId` est disponible à l'intérieur.
	// Utiliser window.AxiomModules.* pour interagir avec le système.
	JS string `json:"js,omitempty"`

	// AutoActivate : si true, switche immédiatement vers cette vue à l'enregistrement.
	AutoActivate bool `json:"autoActivate,omitempty"`

	// ShowInActivityBar : si false, n'ajoute pas d'icône dans l'activity bar.
	ShowInActivityBar *bool `json:"showInActivityBar,omitempty"`

	// Position : 'top' (default) ou 'bottom' dans l'activity bar.
	Position string `json:"position,omitempty"`

	// Closeable : si false, pas de bouton fermer sur la vue.
	Closeable *bool `json:"closeable,omitempty"`
}

// PayloadUISlotInject injecte du HTML dans un slot de l'interface.
type PayloadUISlotInject struct {
	// Slot : nom du slot cible (voir constantes SlotName).
	// Accepte aussi "custom:CSS_SELECTOR" pour cibler n'importe quel élément.
	Slot SlotName `json:"slot"`

	// ModuleID : identifiant du module qui fait l'injection.
	ModuleID string `json:"moduleId"`

	// ID : identifiant unique de l'élément injecté (pour pouvoir le retirer).
	// Si vide, généré automatiquement.
	ID string `json:"id,omitempty"`

	// HTML : contenu à injecter.
	HTML string `json:"html"`

	// CSS : style associé à l'injection.
	CSS string `json:"css,omitempty"`

	// JS : script associé à l'injection.
	JS string `json:"js,omitempty"`

	// Replace : si true, retire l'élément existant avec le même ID avant d'injecter.
	Replace bool `json:"replace,omitempty"`
}

// PayloadUISlotRemove retire un élément injecté.
type PayloadUISlotRemove struct {
	ElementID string `json:"element_id"`
}

// PayloadUIAppBranding modifie le branding global de l'application.
type PayloadUIAppBranding struct {
	// LogoURL : URL du logo (data: URI recommandé pour les ressources embarquées).
	// Ex: "data:image/png;base64,..."
	LogoURL string `json:"logo_url,omitempty"`

	// AppName : texte affiché dans la titlebar à côté du logo.
	AppName string `json:"app_name,omitempty"`

	// TitlebarColor : couleur CSS de fond de la titlebar.
	TitlebarColor string `json:"titlebar_color,omitempty"`

	// StatusbarColor : couleur CSS de fond de la statusbar.
	StatusbarColor string `json:"statusbar_color,omitempty"`
}

// PayloadUIIconBadge ajoute ou retire un badge sur l'icône d'un module.
type PayloadUIIconBadge struct {
	// ModuleID : ID du module dont l'icône doit recevoir un badge.
	ModuleID string `json:"module_id"`

	// Count : valeur du badge. 0 ou "" retire le badge.
	Count interface{} `json:"count"`
}

// PayloadUIViewSwitch force le switch vers une vue.
type PayloadUIViewSwitch struct {
	// ViewID : 'editor' ou l'ID d'un module enregistré.
	ViewID string `json:"view_id"`
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