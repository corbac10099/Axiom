//go:build wails

package wails

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Adapter implémente orchestrator.NativeWindowAdapter via Wails v2.
type Adapter struct {
	mu      sync.RWMutex
	windows map[string]*WindowState
	ctx     context.Context
	logger  *slog.Logger
}

// WindowState maintient l'état d'une fenêtre Wails.
type WindowState struct {
	ID      string
	Title   string
	Width   int
	Height  int
	Visible bool
}

// NewAdapter crée un WailsAdapter.
func NewAdapter(ctx context.Context, logger *slog.Logger) *Adapter {
	return &Adapter{
		windows: make(map[string]*WindowState),
		ctx:     ctx,
		logger:  logger,
	}
}

// CreateWindow enregistre une fenêtre logique.
// Dans Wails v2, il n'y a qu'une seule fenêtre native — on gère
// les "fenêtres" comme des panneaux HTML côté frontend.
func (a *Adapter) CreateWindow(id, title string, width, height int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.windows[id]; exists {
		return fmt.Errorf("wails: window '%s' already exists", id)
	}
	a.windows[id] = &WindowState{
		ID:      id,
		Title:   title,
		Width:   width,
		Height:  height,
		Visible: false,
	}
	// Notifier le frontend via EventsEmit
	runtime.EventsEmit(a.ctx, "axiom:window:created", map[string]interface{}{
		"id":     id,
		"title":  title,
		"width":  width,
		"height": height,
	})
	a.logger.Info("wails: window created",
		slog.String("id", id),
		slog.String("title", title),
	)
	return nil
}

// ShowWindow rend une fenêtre visible.
func (a *Adapter) ShowWindow(id string) error {
	a.mu.Lock()
	win, exists := a.windows[id]
	if exists {
		win.Visible = true
	}
	a.mu.Unlock()

	if !exists {
		return fmt.Errorf("wails: window '%s' not found", id)
	}
	runtime.EventsEmit(a.ctx, "axiom:window:show", map[string]interface{}{"id": id})
	a.logger.Debug("wails: window shown", slog.String("id", id))
	return nil
}

// HideWindow masque une fenêtre.
func (a *Adapter) HideWindow(id string) error {
	a.mu.Lock()
	win, exists := a.windows[id]
	if exists {
		win.Visible = false
	}
	a.mu.Unlock()

	if !exists {
		return fmt.Errorf("wails: window '%s' not found", id)
	}
	runtime.EventsEmit(a.ctx, "axiom:window:hide", map[string]interface{}{"id": id})
	a.logger.Debug("wails: window hidden", slog.String("id", id))
	return nil
}

// DestroyWindow supprime une fenêtre logique.
func (a *Adapter) DestroyWindow(id string) error {
	a.mu.Lock()
	_, exists := a.windows[id]
	if exists {
		delete(a.windows, id)
	}
	a.mu.Unlock()

	if !exists {
		return fmt.Errorf("wails: window '%s' not found", id)
	}
	runtime.EventsEmit(a.ctx, "axiom:window:destroy", map[string]interface{}{"id": id})
	a.logger.Debug("wails: window destroyed", slog.String("id", id))
	return nil
}

// SetWindowContent envoie du HTML au frontend via un événement.
// Le frontend doit écouter "axiom:window:content" et injecter le HTML.
func (a *Adapter) SetWindowContent(id, html string) error {
	a.mu.RLock()
	_, exists := a.windows[id]
	a.mu.RUnlock()

	if !exists {
		return fmt.Errorf("wails: window '%s' not found", id)
	}
	// runtime.Eval n'existe plus dans Wails v2 — on passe par les events
	runtime.EventsEmit(a.ctx, "axiom:window:content", map[string]interface{}{
		"id":   id,
		"html": html,
	})
	a.logger.Debug("wails: window content updated",
		slog.String("id", id),
		slog.Int("content_len", len(html)),
	)
	return nil
}

// SetWindowTitle change le titre d'une fenêtre.
func (a *Adapter) SetWindowTitle(id, title string) error {
	a.mu.Lock()
	win, exists := a.windows[id]
	if exists {
		win.Title = title
	}
	a.mu.Unlock()

	if !exists {
		return fmt.Errorf("wails: window '%s' not found", id)
	}
	// Pour la fenêtre principale, on peut utiliser WindowSetTitle
	runtime.WindowSetTitle(a.ctx, title)
	// Pour les panneaux secondaires, on notifie le frontend
	runtime.EventsEmit(a.ctx, "axiom:window:title", map[string]interface{}{
		"id":    id,
		"title": title,
	})
	a.logger.Debug("wails: window title updated",
		slog.String("id", id),
		slog.String("title", title),
	)
	return nil
}

// EvalJS envoie un script JS au frontend via un événement dédié.
// Le frontend doit écouter "axiom:eval" et exécuter le script.
// Note: runtime.Eval n'existe plus dans Wails v2.
func (a *Adapter) EvalJS(id, script string) error {
	a.mu.RLock()
	_, exists := a.windows[id]
	a.mu.RUnlock()

	if !exists {
		return fmt.Errorf("wails: window '%s' not found", id)
	}
	runtime.EventsEmit(a.ctx, "axiom:eval", map[string]interface{}{
		"id":     id,
		"script": script,
	})
	a.logger.Debug("wails: JS eval requested",
		slog.String("id", id),
		slog.Int("script_len", len(script)),
	)
	return nil
}

// ListWindows retourne toutes les fenêtres logiques actives.
func (a *Adapter) ListWindows() []*WindowState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]*WindowState, 0, len(a.windows))
	for _, w := range a.windows {
		result = append(result, w)
	}
	return result
}

// ExposeToFrontend — hook pour exposer des fonctions au JS (optionnel).
func (a *Adapter) ExposeToFrontend(app interface{}) {}