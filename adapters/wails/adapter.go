//go:build wails

package wails

import (
	"log/slog"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Adapter implémente orchestrator.NativeWindowAdapter via Wails v3.
// En v3, les "fenêtres logiques" sont des panneaux HTML gérés côté frontend.
// Le Go notifie via des événements ; le JS fait le rendu.
type Adapter struct {
	mu      sync.RWMutex
	windows map[string]bool
	app     *application.App
	logger  *slog.Logger
}

// NewAdapter crée un Adapter Wails v3.
func NewAdapter(app *application.App, logger *slog.Logger) *Adapter {
	return &Adapter{
		windows: make(map[string]bool),
		app:     app,
		logger:  logger,
	}
}

// CreateWindow enregistre une fenêtre logique et notifie le frontend.
func (a *Adapter) CreateWindow(id, title string, width, height int) error {
	a.mu.Lock()
	a.windows[id] = true
	a.mu.Unlock()
	a.app.EmitEvent("axiom:window:created", map[string]any{
		"id": id, "title": title, "width": width, "height": height,
	})
	a.logger.Info("wails: window created", slog.String("id", id))
	return nil
}

// ShowWindow rend une fenêtre visible.
func (a *Adapter) ShowWindow(id string) error {
	a.app.EmitEvent("axiom:window:show", map[string]any{"id": id})
	return nil
}

// HideWindow masque une fenêtre.
func (a *Adapter) HideWindow(id string) error {
	a.app.EmitEvent("axiom:window:hide", map[string]any{"id": id})
	return nil
}

// DestroyWindow supprime une fenêtre logique.
func (a *Adapter) DestroyWindow(id string) error {
	a.mu.Lock()
	delete(a.windows, id)
	a.mu.Unlock()
	a.app.EmitEvent("axiom:window:destroy", map[string]any{"id": id})
	return nil
}

// SetWindowContent envoie du HTML au frontend.
func (a *Adapter) SetWindowContent(id, html string) error {
	a.app.EmitEvent("axiom:window:content", map[string]any{"id": id, "html": html})
	return nil
}

// SetWindowTitle change le titre d'une fenêtre.
func (a *Adapter) SetWindowTitle(id, title string) error {
	a.app.EmitEvent("axiom:window:title", map[string]any{"id": id, "title": title})
	return nil
}