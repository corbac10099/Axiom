// Package orchestrator implémente le Window Orchestrator d'Axiom.
package orchestrator

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/axiom-ide/axiom/api"
	"github.com/axiom-ide/axiom/core/bus"
)

// WindowID est un identifiant unique de fenêtre.
type WindowID string

// WindowState décrit l'état d'une fenêtre.
type WindowState string

const (
	WindowStateOpen   WindowState = "open"
	WindowStateHidden WindowState = "hidden"
	WindowStateClosed WindowState = "closed"
)

// PanelPosition définit où un panel est ancré.
type PanelPosition string

const (
	PanelBottom PanelPosition = "bottom"
	PanelLeft   PanelPosition = "left"
	PanelRight  PanelPosition = "right"
	PanelCenter PanelPosition = "center"
	PanelFloat  PanelPosition = "float"
)

// Window représente une fenêtre OS gérée par l'Orchestrateur.
type Window struct {
	ID        WindowID      `json:"id"`
	Title     string        `json:"title"`
	PanelID   string        `json:"panel_id"`
	Position  PanelPosition `json:"position"`
	State     WindowState   `json:"state"`
	Width     int           `json:"width"`
	Height    int           `json:"height"`
	X         int           `json:"x"`
	Y         int           `json:"y"`
	commandCh chan WindowCommand
}

// WindowCommand est une commande envoyée à une fenêtre individuelle.
type WindowCommand struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Orchestrator est le gestionnaire de fenêtres d'Axiom.
type Orchestrator struct {
	mu      sync.RWMutex
	windows map[WindowID]*Window
	native  NativeWindowAdapter
	bus     *bus.EventBus
	logger  *slog.Logger
}

// NativeWindowAdapter est l'interface vers le framework UI natif.
type NativeWindowAdapter interface {
	CreateWindow(id string, title string, width, height int) error
	ShowWindow(id string) error
	HideWindow(id string) error
	DestroyWindow(id string) error
	SetWindowContent(id string, html string) error
	SetWindowTitle(id string, title string) error
}

// NewOrchestrator crée un Orchestrateur.
// BUG FIX: vérification nil sur native ET sur bus avant subscribeToEvents.
func NewOrchestrator(native NativeWindowAdapter, eventBus *bus.EventBus, logger *slog.Logger) *Orchestrator {
	if native == nil {
		native = &noopAdapter{logger: logger}
	}
	o := &Orchestrator{
		windows: make(map[WindowID]*Window),
		native:  native,
		bus:     eventBus,
		logger:  logger,
	}
	// BUG FIX: ne pas appeler subscribeToEvents si le bus est nil
	// (cas headless / tests sans bus)
	if eventBus != nil {
		o.subscribeToEvents()
	}
	return o
}

// OpenPanel ouvre (ou crée si inexistant) un panel.
func (o *Orchestrator) OpenPanel(panelID, title, position, content string) error {
	winID := WindowID("panel:" + panelID)

	o.mu.Lock()
	win, exists := o.windows[winID]
	if !exists {
		win = &Window{
			ID:        winID,
			Title:     title,
			PanelID:   panelID,
			Position:  PanelPosition(position),
			State:     WindowStateClosed,
			Width:     400,
			Height:    300,
			commandCh: make(chan WindowCommand, 16),
		}
		o.windows[winID] = win
	}
	o.mu.Unlock()

	if win.State == WindowStateClosed || win.State == WindowStateHidden {
		if err := o.native.CreateWindow(string(winID), title, win.Width, win.Height); err != nil {
			return fmt.Errorf("orchestrator: cannot create window for panel '%s': %w", panelID, err)
		}
		if content != "" {
			_ = o.native.SetWindowContent(string(winID), content)
		}
		_ = o.native.ShowWindow(string(winID))

		o.mu.Lock()
		win.State = WindowStateOpen
		o.mu.Unlock()

		o.logger.Info("orchestrator: panel opened",
			slog.String("panel_id", panelID),
			slog.String("position", position),
		)
	}
	return nil
}

// DetachPanel transforme un panel ancré en fenêtre flottante.
func (o *Orchestrator) DetachPanel(panelID string) error {
	winID := WindowID("panel:" + panelID)
	o.mu.Lock()
	win, exists := o.windows[winID]
	if exists {
		win.Position = PanelFloat
	}
	o.mu.Unlock()
	if !exists {
		return fmt.Errorf("orchestrator: panel '%s' not found", panelID)
	}
	o.logger.Info("orchestrator: panel detached", slog.String("panel_id", panelID))
	return nil
}

// ClosePanel ferme un panel.
func (o *Orchestrator) ClosePanel(panelID string) error {
	winID := WindowID("panel:" + panelID)
	o.mu.Lock()
	win, exists := o.windows[winID]
	o.mu.Unlock()
	if !exists {
		return fmt.Errorf("orchestrator: panel '%s' not found", panelID)
	}
	if err := o.native.DestroyWindow(string(winID)); err != nil {
		return err
	}
	o.mu.Lock()
	win.State = WindowStateClosed
	o.mu.Unlock()
	o.logger.Info("orchestrator: panel closed", slog.String("panel_id", panelID))
	return nil
}

// ListWindows retourne la liste de toutes les fenêtres actives.
func (o *Orchestrator) ListWindows() []*Window {
	o.mu.RLock()
	defer o.mu.RUnlock()
	result := make([]*Window, 0, len(o.windows))
	for _, w := range o.windows {
		result = append(result, w)
	}
	return result
}

// subscribeToEvents écoute les Topics UI sur le bus.
func (o *Orchestrator) subscribeToEvents() {
	o.bus.Subscribe(api.TopicUIOpenPanel, func(ev api.Event) {
		p, ok := ev.Payload.(api.PayloadUIPanel)
		if !ok {
			return
		}
		if err := o.OpenPanel(p.PanelID, p.Title, p.Position, p.Content); err != nil {
			o.logger.Error("orchestrator: OpenPanel failed", slog.String("error", err.Error()))
		}
	})

	o.bus.Subscribe(api.TopicUIClosePanel, func(ev api.Event) {
		p, ok := ev.Payload.(api.PayloadUIPanel)
		if !ok {
			return
		}
		if err := o.ClosePanel(p.PanelID); err != nil {
			o.logger.Error("orchestrator: ClosePanel failed", slog.String("error", err.Error()))
		}
	})

	o.bus.Subscribe(api.TopicUISetTheme, func(ev api.Event) {
		t, ok := ev.Payload.(api.PayloadUITheme)
		if !ok {
			return
		}
		o.logger.Info("orchestrator: theme change requested", slog.String("theme_id", t.ThemeID))
	})
}

// noopAdapter — adapter no-op pour les tests / mode headless.
type noopAdapter struct{ logger *slog.Logger }

func (n *noopAdapter) CreateWindow(id, title string, w, h int) error {
	n.logger.Debug("noop: CreateWindow", slog.String("id", id))
	return nil
}
func (n *noopAdapter) ShowWindow(id string) error {
	n.logger.Debug("noop: ShowWindow", slog.String("id", id))
	return nil
}
func (n *noopAdapter) HideWindow(id string) error {
	n.logger.Debug("noop: HideWindow", slog.String("id", id))
	return nil
}
func (n *noopAdapter) DestroyWindow(id string) error {
	n.logger.Debug("noop: DestroyWindow", slog.String("id", id))
	return nil
}
func (n *noopAdapter) SetWindowContent(id, html string) error {
	n.logger.Debug("noop: SetWindowContent", slog.String("id", id))
	return nil
}
func (n *noopAdapter) SetWindowTitle(id, title string) error {
	n.logger.Debug("noop: SetWindowTitle", slog.String("id", id))
	return nil
}