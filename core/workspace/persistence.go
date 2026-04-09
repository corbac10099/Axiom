// Package workspace implémente la persistence de l'état UI d'Axiom.
//
// Ce qui est sauvegardé :
//   - Onglets ouverts (path, curseur, scroll) par fenêtre
//   - Panels ouverts (id, position, titre)
//   - Thème courant
//   - Répertoire de travail courant
//
// Format : JSON dans .axiom/workspace_state.json
// (même convention que config.json)
//
// Intégration dans main.go :
//
//	persistence := workspace.NewPersistence(
//	    eng.Bus(), tabMgr, orch, cfg.Core.WorkspaceDir, logger,
//	)
//	persistence.Start()
//	defer persistence.SaveSync()  // sauvegarde avant exit
package workspace

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/axiom-ide/axiom/api"
	"github.com/axiom-ide/axiom/core/bus"
	"github.com/axiom-ide/axiom/core/tabs"
	"github.com/axiom-ide/axiom/pkg/uid"
)

const defaultStateFile = ".axiom/workspace_state.json"

// ─────────────────────────────────────────────
// STATE SCHEMA
// ─────────────────────────────────────────────

// PanelState décrit un panel persisté.
type PanelState struct {
	PanelID  string `json:"panel_id"`
	Title    string `json:"title"`
	Position string `json:"position"`
	IsOpen   bool   `json:"is_open"`
}

// WorkspaceState est le document JSON complet sauvegardé sur disque.
type WorkspaceState struct {
	Version      string                        `json:"version"`
	SavedAt      time.Time                     `json:"saved_at"`
	WorkspaceDir string                        `json:"workspace_dir"`
	Theme        string                        `json:"theme"`
	TabGroups    map[string]*tabs.TabGroup     `json:"tab_groups"`
	Panels       []PanelState                  `json:"panels"`
}

// ─────────────────────────────────────────────
// INTERFACES MINIMALISTES (évite import circulaire)
// ─────────────────────────────────────────────

// TabSnapshotter est implémenté par tabs.Manager.
type TabSnapshotter interface {
	Snapshot() map[string]*tabs.TabGroup
	Restore(map[string]*tabs.TabGroup)
}

// PanelLister est implémenté par orchestrator.Orchestrator
// (il expose ListWindows — on extrait les panels actifs).
type PanelLister interface {
	ListWindows() []PanelInfo
}

// PanelInfo est un résumé minimal d'une fenêtre/panel exposé par l'orchestrateur.
type PanelInfo struct {
	PanelID  string
	Title    string
	Position string
	IsOpen   bool
}

// ─────────────────────────────────────────────
// PERSISTENCE MANAGER
// ─────────────────────────────────────────────

// Persistence orchestre la sauvegarde et la restauration de l'état.
type Persistence struct {
	mu           sync.Mutex
	bus          *bus.EventBus
	tabs         TabSnapshotter
	panels       PanelLister
	workspaceDir string
	statePath    string
	currentTheme string
	logger       *slog.Logger
}

// NewPersistence crée un gestionnaire de persistence.
// panels peut être nil (panels non sauvegardés).
func NewPersistence(
	eventBus *bus.EventBus,
	tabMgr TabSnapshotter,
	panelLister PanelLister, // peut être nil
	workspaceDir string,
	logger *slog.Logger,
) *Persistence {
	statePath := filepath.Join(workspaceDir, defaultStateFile)
	return &Persistence{
		bus:          eventBus,
		tabs:         tabMgr,
		panels:       panelLister,
		workspaceDir: workspaceDir,
		statePath:    statePath,
		currentTheme: "dark",
		logger:       logger,
	}
}

// Start s'abonne aux topics workspace.save / workspace.restore.
// Écoute aussi ui.theme.set pour tracker le thème courant.
func (p *Persistence) Start() {
	p.bus.Subscribe(api.TopicWorkspaceSave, func(ev api.Event) {
		payload, _ := ev.Payload.(api.PayloadWorkspaceSave)
		target := payload.TargetPath
		if target == "" {
			target = p.statePath
		}
		if err := p.SaveTo(target); err != nil {
			p.logger.Error("workspace: save failed", slog.String("error", err.Error()))
		}
	})

	p.bus.Subscribe(api.TopicWorkspaceRestore, func(ev api.Event) {
		payload, _ := ev.Payload.(api.PayloadWorkspaceSave)
		source := payload.TargetPath
		if source == "" {
			source = p.statePath
		}
		n, err := p.RestoreFrom(source)
		if err != nil {
			p.logger.Warn("workspace: restore failed", slog.String("error", err.Error()))
			return
		}
		p.bus.Publish(api.Event{
			ID:        uid.New(),
			Topic:     api.TopicWorkspaceRestored,
			Source:    "workspace-persistence",
			Timestamp: time.Now().UTC(),
			Payload: api.PayloadWorkspaceRestored{
				TabsRestored:   n,
				SourcePath:     source,
			},
		})
	})

	// Suivre le thème courant pour l'inclure dans la sauvegarde.
	p.bus.Subscribe(api.TopicUISetTheme, func(ev api.Event) {
		if t, ok := ev.Payload.(api.PayloadUITheme); ok {
			p.mu.Lock()
			p.currentTheme = t.ThemeID
			p.mu.Unlock()
		}
	})

	p.logger.Info("workspace: persistence manager started",
		slog.String("state_path", p.statePath))
}

// SaveSync sauvegarde synchrone vers le chemin par défaut.
// À appeler dans le defer de main() avant shutdown.
func (p *Persistence) SaveSync() error {
	return p.SaveTo(p.statePath)
}

// SaveTo sauvegarde l'état complet vers un chemin explicite.
func (p *Persistence) SaveTo(path string) error {
	p.mu.Lock()
	theme := p.currentTheme
	p.mu.Unlock()

	snapshot := p.tabs.Snapshot()

	var panelStates []PanelState
	if p.panels != nil {
		for _, w := range p.panels.ListWindows() {
			panelStates = append(panelStates, PanelState{
				PanelID:  w.PanelID,
				Title:    w.Title,
				Position: w.Position,
				IsOpen:   w.IsOpen,
			})
		}
	}

	state := WorkspaceState{
		Version:      "1",
		SavedAt:      time.Now().UTC(),
		WorkspaceDir: p.workspaceDir,
		Theme:        theme,
		TabGroups:    snapshot,
		Panels:       panelStates,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("workspace: marshal failed: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("workspace: cannot create state dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("workspace: write failed: %w", err)
	}

	p.logger.Info("workspace: state saved",
		slog.String("path", path),
		slog.Int("tab_groups", len(snapshot)),
		slog.Int("panels", len(panelStates)),
	)
	return nil
}

// RestoreFrom charge l'état depuis un fichier JSON.
// Retourne le nombre d'onglets restaurés.
func (p *Persistence) RestoreFrom(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("workspace: cannot read state file '%s': %w", path, err)
	}

	var state WorkspaceState
	if err := json.Unmarshal(data, &state); err != nil {
		return 0, fmt.Errorf("workspace: cannot parse state file: %w", err)
	}

	// Restaurer les onglets.
	if state.TabGroups != nil {
		p.tabs.Restore(state.TabGroups)
	}

	// Restaurer le thème.
	if state.Theme != "" {
		p.mu.Lock()
		p.currentTheme = state.Theme
		p.mu.Unlock()
		p.bus.Publish(api.Event{
			ID:        uid.New(),
			Topic:     api.TopicUISetTheme,
			Source:    "workspace-persistence",
			Timestamp: time.Now().UTC(),
			Payload:   api.PayloadUITheme{ThemeID: state.Theme},
		})
	}

	// Compter les onglets restaurés.
	totalTabs := 0
	for _, g := range state.TabGroups {
		totalTabs += len(g.Tabs)
	}

	p.logger.Info("workspace: state restored",
		slog.String("path", path),
		slog.Int("tabs", totalTabs),
		slog.String("theme", state.Theme),
		slog.Time("saved_at", state.SavedAt),
	)
	return totalTabs, nil
}

// StatePath retourne le chemin du fichier d'état par défaut.
func (p *Persistence) StatePath() string { return p.statePath }