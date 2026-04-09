// Package api — topics supplémentaires pour les fonctionnalités manquantes.
// Ce fichier complète events.go sans le modifier.
package api

const (
	// Bidirectionnel WebView → Go
	// Émis par le JS frontend quand l'utilisateur tape / clique dans l'éditeur.
	TopicUIUserInput Topic = "ui.user.input"

	// Onglets éditeur
	TopicEditorTabOpen    Topic = "editor.tab.open"
	TopicEditorTabClose   Topic = "editor.tab.close"
	TopicEditorTabFocus   Topic = "editor.tab.focus"
	TopicEditorTabChanged Topic = "editor.tab.changed" // état global mis à jour

	// Persistence du workspace
	TopicWorkspaceSave    Topic = "workspace.save"
	TopicWorkspaceRestore Topic = "workspace.restore"
	TopicWorkspaceRestored Topic = "workspace.restored" // ack après restauration
)

// --- Payloads nouveaux ---

// PayloadUIUserInput est émis par le frontend JS via runtime.EventsEmit.
type PayloadUIUserInput struct {
	WindowID  string      `json:"window_id"`
	EventType string      `json:"event_type"` // "keydown" | "click" | "command"
	Data      interface{} `json:"data"`
}

// PayloadEditorTab décrit un onglet éditeur.
type PayloadEditorTab struct {
	TabID    string `json:"tab_id"`
	WindowID string `json:"window_id"`
	FilePath string `json:"file_path"`
	Title    string `json:"title"`
	IsDirty  bool   `json:"is_dirty"`
	Language string `json:"language"` // "go" | "json" | "md" | ...
}

// PayloadWorkspaceSave déclenche une sauvegarde de l'état UI complet.
type PayloadWorkspaceSave struct {
	TargetPath string `json:"target_path"` // vide = chemin par défaut
}

// PayloadWorkspaceRestored est publié après une restauration réussie.
type PayloadWorkspaceRestored struct {
	TabsRestored  int    `json:"tabs_restored"`
	PanelsRestored int   `json:"panels_restored"`
	SourcePath    string `json:"source_path"`
}