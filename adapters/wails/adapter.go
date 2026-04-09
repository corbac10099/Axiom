//go:build wails

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
// Build tag : //go:build wails
// Pour les tests sans Wails : utiliser orchestrator.NoopAdapter
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
	ID     string
	Title  string
	Width  int
	Height int
	Visible bool
}

// NewAdapter crée un WailsAdapter.
// ctx doit être le contexte passé par le callback OnStartup de Wails.
func NewAdapter(ctx context.Context, logger *slog.Logger) *Adapter {
	return &Adapter{
		windows: make(map[string]*WindowState),
		ctx:     ctx,
		logger:  logger,
	}
}

// CreateWindow crée une nouvelle fenêtre WebView Wails.
func (a *Adapter) CreateWindow(id, title string, width, height int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.windows[id]; exists {
		return fmt.Errorf("wails: window '%s' already exists", id)
	}

	// Avec Wails v2, on crée une nouvelle fenêtre secondaire.
	// Note: Wails v2 a des limitations pour les fenêtres secondaires.
	// Pour une approche production, utiliser des modales ou des panneaux au sein de la fenêtre principale.
	
	a.windows[id] = &WindowState{
		ID:      id,
		Title:   title,
		Width:   width,
		Height:  height,
		Visible: false,
	}

	a.logger.Info("wails: window created",
		slog.String("id", id),
		slog.String("title", title),
		slog.Int("width", width),
		slog.Int("height", height),
	)
	return nil
}

// ShowWindow rend une fenêtre visible.
func (a *Adapter) ShowWindow(id string) error {
	a.mu.RLock()
	win, exists := a.windows[id]
	a.mu.RUnlock()

	if !exists {
		return fmt.Errorf("wails: window '%s' not found", id)
	}

	a.mu.Lock()
	win.Visible = true
	a.mu.Unlock()

	// Appeler runtime.Show() pour afficher la fenêtre (si applicable à votre fenêtre)
	// Note: Wails v2 gère surtout une fenêtre principale; pour les panneaux secondaires,
	// utiliser des approches de domaine HTML/CSS ou des modales JavaScript.
	
	a.logger.Debug("wails: window shown", slog.String("id", id))
	return nil
}

// HideWindow masque une fenêtre sans la détruire.
func (a *Adapter) HideWindow(id string) error {
	a.mu.RLock()
	win, exists := a.windows[id]
	a.mu.RUnlock()

	if !exists {
		return fmt.Errorf("wails: window '%s' not found", id)
	}

	a.mu.Lock()
	win.Visible = false
	a.mu.Unlock()

	a.logger.Debug("wails: window hidden", slog.String("id", id))
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

	a.logger.Debug("wails: window destroyed", slog.String("id", id))
	return nil
}

// SetWindowContent injecte du HTML dans une WebView Wails.
// Utilise le runtime.Eval() pour modifier le contenu dynamiquement.
func (a *Adapter) SetWindowContent(id, html string) error {
	a.mu.RLock()
	_, exists := a.windows[id]
	a.mu.RUnlock()

	if !exists {
		return fmt.Errorf("wails: window '%s' not found", id)
	}

	// Évaluer du JavaScript pour injecter le contenu HTML.
	// La fonction window.AxiomSetContent() doit être exposée côté frontend.
	jsCode := fmt.Sprintf(`
		if (typeof window.AxiomSetContent === 'function') {
			window.AxiomSetContent('%s', %q);
		} else {
			document.body.innerHTML = %q;
		}
	`, id, html, html)

	_, err := runtime.Eval(a.ctx, jsCode)
	if err != nil {
		a.logger.Warn("wails: SetWindowContent JS eval failed",
			slog.String("id", id),
			slog.String("error", err.Error()),
		)
		return err
	}

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

	// Mettre à jour le titre via runtime.
	// Pour la fenêtre principale, utiliser WindowSetTitle().
	// Pour les panneaux/modales, mettre à jour via JavaScript.
	jsCode := fmt.Sprintf(`
		if (typeof window.AxiomSetTitle === 'function') {
			window.AxiomSetTitle('%s', %q);
		} else {
			document.title = %q;
		}
	`, id, title, title)

	_, _ = runtime.Eval(a.ctx, jsCode)

	a.logger.Debug("wails: window title updated",
		slog.String("id", id),
		slog.String("title", title),
	)
	return nil
}

// EvalJS exécute du JavaScript dans la WebView d'une fenêtre.
func (a *Adapter) EvalJS(id, script string) error {
	a.mu.RLock()
	_, exists := a.windows[id]
	a.mu.RUnlock()

	if !exists {
		return fmt.Errorf("wails: window '%s' not found", id)
	}

	_, err := runtime.Eval(a.ctx, script)
	if err != nil {
		a.logger.Warn("wails: EvalJS failed",
			slog.String("id", id),
			slog.String("error", err.Error()),
		)
		return err
	}

	a.logger.Debug("wails: JavaScript evaluated",
		slog.String("id", id),
		slog.Int("script_len", len(script)),
	)
	return nil
}

// ListWindows retourne toutes les fenêtres actives.
func (a *Adapter) ListWindows() []*WindowState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	result := make([]*WindowState, 0, len(a.windows))
	for _, w := range a.windows {
		result = append(result, w)
	}
	return result
}

// ─────────────────────────────────────────────
// HELPERS POUR LE FRONTEND
// ─────────────────────────────────────────────

// ExposeToFrontend expose des fonctions Axiom au JavaScript du frontend.
// À appeler lors du OnReady de Wails dans main.go.
func (a *Adapter) ExposeToFrontend(app interface{}) {
	// Exemple : exposer des fonctions de dispatch, lecture de fichiers, etc.
	// Voir la section "main.go" pour l'intégration complète.
}