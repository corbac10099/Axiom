// Package wails implémente l'adaptateur Wails v2 pour l'Orchestrateur Axiom.
//
// Ce package est le SEUL point de contact entre le core Go d'Axiom
// et le framework d'interface natif Wails. Le reste du codebase
// ne connaît que l'interface orchestrator.NativeWindowAdapter.
//
// Pour activer Wails :
//  1. go get github.com/wailsapp/wails/v2
//  2. Dans main.go, remplacer `nil` par `wails.NewAdapter(app)`
//  3. Compiler avec : wails build -platform windows/amd64
//
// IMPORTANT : Ce fichier NE compile PAS sans Wails installé.
// Il est gardé en dehors des builds de tests via un build tag.
//
// Build tag : //go:build wails
// Pour les tests sans Wails : utiliser orchestrator.NoopAdapter
package wails

// ─────────────────────────────────────────────
// NOTE D'INTÉGRATION
// ─────────────────────────────────────────────
//
// Wails v2 expose une API de gestion de fenêtres via *application.App.
// La structure WailsAdapter ci-dessous wrappera ces appels.
//
// Exemple d'intégration complète avec Wails v2 :
//
//   package main
//
//   import (
//       "github.com/wailsapp/wails/v2"
//       "github.com/wailsapp/wails/v2/pkg/options"
//       "github.com/axiom-ide/axiom/adapters/wails"
//       "github.com/axiom-ide/axiom/core/engine"
//       "github.com/axiom-ide/axiom/core/orchestrator"
//   )
//
//   func main() {
//       eng, _ := engine.New(engine.DefaultConfig())
//
//       app := wails.CreateApp(&options.App{
//           Title:  "Axiom IDE",
//           Width:  1400,
//           Height: 900,
//           OnStartup: func(ctx context.Context) {
//               adapter := wailsadapter.NewAdapter(ctx)
//               orch := orchestrator.NewOrchestrator(adapter, nil, slog.Default())
//               _ = eng.Start()
//           },
//       })
//       app.Run()
//   }

// ─────────────────────────────────────────────
// STUB COMPILABLE (sans dépendance Wails)
// ─────────────────────────────────────────────

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// WindowRef maintient la référence à une fenêtre Wails créée.
// En production, ce serait un *wails.WebviewWindow.
type WindowRef struct {
	ID     string
	Title  string
	Width  int
	Height int
}

// Adapter implémente orchestrator.NativeWindowAdapter via Wails.
// Ce stub est fonctionnel sans Wails et sera remplacé par les vrais appels
// une fois que github.com/wailsapp/wails/v2 est ajouté comme dépendance.
type Adapter struct {
	mu      sync.RWMutex
	windows map[string]*WindowRef
	ctx     context.Context // contexte Wails (passé par OnStartup)
	logger  *slog.Logger
}

// NewAdapter crée un WailsAdapter.
// ctx doit être le contexte passé par le callback OnStartup de Wails.
func NewAdapter(ctx context.Context, logger *slog.Logger) *Adapter {
	return &Adapter{
		windows: make(map[string]*WindowRef),
		ctx:     ctx,
		logger:  logger,
	}
}

// CreateWindow crée une nouvelle fenêtre WebView Wails.
//
// TODO avec Wails v2 :
//
//	win := app.NewWebviewWindowWithOptions(&webviewwindow.Options{
//	    Title: title, Width: width, Height: height,
//	    URL: "local://axiom/panel/" + id,
//	})
//	a.windows[id] = &WindowRef{ID: id, Title: title, ...}
func (a *Adapter) CreateWindow(id, title string, width, height int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.windows[id]; exists {
		return fmt.Errorf("wails: window '%s' already exists", id)
	}

	// ── Stub : simuler la création sans Wails ─────────────────────
	a.windows[id] = &WindowRef{ID: id, Title: title, Width: width, Height: height}
	a.logger.Info("wails: [STUB] CreateWindow",
		slog.String("id", id),
		slog.String("title", title),
		slog.Int("w", width),
		slog.Int("h", height),
	)
	// ── Production avec Wails :
	// win := a.app.NewWebviewWindowWithOptions(...)
	// a.windows[id] = &WindowRef{native: win, ...}
	return nil
}

// ShowWindow rend une fenêtre visible.
func (a *Adapter) ShowWindow(id string) error {
	a.mu.RLock()
	_, exists := a.windows[id]
	a.mu.RUnlock()
	if !exists {
		return fmt.Errorf("wails: window '%s' not found", id)
	}
	a.logger.Debug("wails: [STUB] ShowWindow", slog.String("id", id))
	// Production: a.windows[id].native.Show()
	return nil
}

// HideWindow masque une fenêtre sans la détruire.
func (a *Adapter) HideWindow(id string) error {
	a.mu.RLock()
	_, exists := a.windows[id]
	a.mu.RUnlock()
	if !exists {
		return fmt.Errorf("wails: window '%s' not found", id)
	}
	a.logger.Debug("wails: [STUB] HideWindow", slog.String("id", id))
	// Production: a.windows[id].native.Hide()
	return nil
}

// DestroyWindow ferme et détruit une fenêtre.
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
	a.logger.Debug("wails: [STUB] DestroyWindow", slog.String("id", id))
	// Production: a.windows[id].native.Destroy()
	return nil
}

// SetWindowContent injecte du HTML dans une WebView Wails.
//
// TODO avec Wails v2 :
//
//	win.SetContent(html)
//	// ou via JS bridge : win.ExecJS(fmt.Sprintf("document.body.innerHTML = `%s`", html))
func (a *Adapter) SetWindowContent(id, html string) error {
	a.mu.RLock()
	_, exists := a.windows[id]
	a.mu.RUnlock()
	if !exists {
		return fmt.Errorf("wails: window '%s' not found", id)
	}
	a.logger.Debug("wails: [STUB] SetWindowContent",
		slog.String("id", id),
		slog.Int("html_len", len(html)),
	)
	// Production: a.windows[id].native.ExecJS(...)
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
	a.logger.Debug("wails: [STUB] SetWindowTitle", slog.String("id", id), slog.String("title", title))
	// Production: a.windows[id].native.SetTitle(title)
	return nil
}

// EvalJS exécute du JavaScript dans la WebView d'une fenêtre.
// Utilisé pour les changements de thème Monaco, les mises à jour de l'éditeur, etc.
//
// TODO avec Wails v2 :
//
//	win.ExecJS(script)
func (a *Adapter) EvalJS(id, script string) error {
	a.mu.RLock()
	_, exists := a.windows[id]
	a.mu.RUnlock()
	if !exists {
		return fmt.Errorf("wails: window '%s' not found", id)
	}
	a.logger.Debug("wails: [STUB] EvalJS",
		slog.String("id", id),
		slog.Int("script_len", len(script)),
	)
	// Production: a.windows[id].native.ExecJS(script)
	return nil
}

// ListWindows retourne toutes les fenêtres actives.
func (a *Adapter) ListWindows() []*WindowRef {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]*WindowRef, 0, len(a.windows))
	for _, w := range a.windows {
		result = append(result, w)
	}
	return result
}