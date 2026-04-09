// Package tabs implémente le gestionnaire d'onglets de l'éditeur Axiom.
//
// Modèle de données :
//
//	Window  1──* TabGroup  1──* Tab
//
// Un TabGroup correspond à un split-pane (ou à la fenêtre principale si pas de split).
// Chaque Tab pointe vers un fichier du workspace.
//
// Intégration :
//
//	tabMgr := tabs.NewManager(eng.Bus(), logger)
//	tabMgr.Start()   // s'abonne aux topics editor.tab.*
package tabs

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/axiom-ide/axiom/api"
	"github.com/axiom-ide/axiom/core/bus"
	"github.com/axiom-ide/axiom/pkg/uid"
)

// ─────────────────────────────────────────────
// TYPES
// ─────────────────────────────────────────────

// Tab représente un onglet ouvert dans l'éditeur.
type Tab struct {
	ID         string    `json:"id"`
	FilePath   string    `json:"file_path"`
	Title      string    `json:"title"`
	Language   string    `json:"language"`
	IsDirty    bool      `json:"is_dirty"`
	IsActive   bool      `json:"is_active"`
	OpenedAt   time.Time `json:"opened_at"`
	ScrollLine int       `json:"scroll_line"`
	CursorLine int       `json:"cursor_line"`
	CursorCol  int       `json:"cursor_col"`
}

// TabGroup est un groupe d'onglets dans une fenêtre (ou un split).
type TabGroup struct {
	ID       string `json:"id"`
	WindowID string `json:"window_id"`
	Tabs     []*Tab `json:"tabs"`
	ActiveID string `json:"active_id"`
}

// Manager est le gestionnaire central des onglets.
type Manager struct {
	mu     sync.RWMutex
	groups map[string]*TabGroup // key: group_id
	bus    *bus.EventBus
	logger *slog.Logger
}

// ─────────────────────────────────────────────
// CONSTRUCTEUR
// ─────────────────────────────────────────────

// NewManager crée un TabManager et l'attache au bus.
func NewManager(eventBus *bus.EventBus, logger *slog.Logger) *Manager {
	return &Manager{
		groups: make(map[string]*TabGroup),
		bus:    eventBus,
		logger: logger,
	}
}

// Start s'abonne aux topics editor.tab.* du bus.
// À appeler après engine.Start().
func (m *Manager) Start() {
	m.bus.Subscribe(api.TopicEditorTabOpen, func(ev api.Event) {
		p, ok := ev.Payload.(api.PayloadEditorTab)
		if !ok {
			return
		}
		if err := m.OpenTab(p.WindowID, p.FilePath, p.Title, p.Language); err != nil {
			m.logger.Warn("tabs: open failed", slog.String("error", err.Error()))
		}
	})

	m.bus.Subscribe(api.TopicEditorTabClose, func(ev api.Event) {
		p, ok := ev.Payload.(api.PayloadEditorTab)
		if !ok {
			return
		}
		m.CloseTab(p.WindowID, p.TabID)
	})

	m.bus.Subscribe(api.TopicEditorTabFocus, func(ev api.Event) {
		p, ok := ev.Payload.(api.PayloadEditorTab)
		if !ok {
			return
		}
		m.FocusTab(p.WindowID, p.TabID)
	})

	m.logger.Info("tabs: manager started")
}

// ─────────────────────────────────────────────
// API PUBLIQUE
// ─────────────────────────────────────────────

// OpenTab ouvre (ou active si déjà ouvert) un onglet dans la fenêtre donnée.
func (m *Manager) OpenTab(windowID, filePath, title, language string) error {
	if filePath == "" {
		return fmt.Errorf("tabs: filePath cannot be empty")
	}
	groupID := groupKey(windowID)

	m.mu.Lock()
	group, exists := m.groups[groupID]
	if !exists {
		group = &TabGroup{
			ID:       groupID,
			WindowID: windowID,
			Tabs:     make([]*Tab, 0),
		}
		m.groups[groupID] = group
	}

	// Si le fichier est déjà ouvert, juste activer l'onglet.
	for _, t := range group.Tabs {
		if t.FilePath == filePath {
			m.setActive(group, t.ID)
			m.mu.Unlock()
			m.publishChanged(windowID)
			return nil
		}
	}

	// Désactiver tous les onglets existants.
	for _, t := range group.Tabs {
		t.IsActive = false
	}

	if title == "" {
		parts := strings.Split(filePath, "/")
		title = parts[len(parts)-1]
	}
	if language == "" {
		language = detectLanguage(filePath)
	}

	tab := &Tab{
		ID:       uid.New(),
		FilePath: filePath,
		Title:    title,
		Language: language,
		IsActive: true,
		OpenedAt: time.Now().UTC(),
	}
	group.Tabs = append(group.Tabs, tab)
	group.ActiveID = tab.ID
	m.mu.Unlock()

	m.logger.Info("tabs: opened",
		slog.String("window", windowID),
		slog.String("file", filePath),
	)
	m.publishChanged(windowID)
	return nil
}

// CloseTab ferme un onglet. Active le suivant ou le précédent automatiquement.
func (m *Manager) CloseTab(windowID, tabID string) {
	groupID := groupKey(windowID)
	m.mu.Lock()
	group, exists := m.groups[groupID]
	if !exists {
		m.mu.Unlock()
		return
	}

	idx := -1
	for i, t := range group.Tabs {
		if t.ID == tabID {
			idx = i
			break
		}
	}
	if idx == -1 {
		m.mu.Unlock()
		return
	}

	wasActive := group.Tabs[idx].IsActive
	group.Tabs = append(group.Tabs[:idx], group.Tabs[idx+1:]...)

	// Réactiver un onglet adjacent si celui qu'on ferme était actif.
	if wasActive && len(group.Tabs) > 0 {
		newIdx := idx
		if newIdx >= len(group.Tabs) {
			newIdx = len(group.Tabs) - 1
		}
		group.Tabs[newIdx].IsActive = true
		group.ActiveID = group.Tabs[newIdx].ID
	} else if len(group.Tabs) == 0 {
		group.ActiveID = ""
	}
	m.mu.Unlock()

	m.logger.Info("tabs: closed", slog.String("tab_id", tabID))
	m.publishChanged(windowID)
}

// FocusTab rend un onglet actif.
func (m *Manager) FocusTab(windowID, tabID string) {
	groupID := groupKey(windowID)
	m.mu.Lock()
	group, exists := m.groups[groupID]
	if !exists {
		m.mu.Unlock()
		return
	}
	m.setActive(group, tabID)
	m.mu.Unlock()
	m.publishChanged(windowID)
}

// MarkDirty marque un onglet comme modifié (bullet • dans le titre).
func (m *Manager) MarkDirty(windowID, tabID string, dirty bool) {
	groupID := groupKey(windowID)
	m.mu.Lock()
	group, exists := m.groups[groupID]
	if !exists {
		m.mu.Unlock()
		return
	}
	for _, t := range group.Tabs {
		if t.ID == tabID {
			t.IsDirty = dirty
			break
		}
	}
	m.mu.Unlock()
	m.publishChanged(windowID)
}

// UpdateCursor met à jour la position curseur d'un onglet (pour persistence).
func (m *Manager) UpdateCursor(windowID, tabID string, line, col, scrollLine int) {
	groupID := groupKey(windowID)
	m.mu.Lock()
	defer m.mu.Unlock()
	group, exists := m.groups[groupID]
	if !exists {
		return
	}
	for _, t := range group.Tabs {
		if t.ID == tabID {
			t.CursorLine = line
			t.CursorCol = col
			t.ScrollLine = scrollLine
			break
		}
	}
}

// ListTabs retourne les onglets d'une fenêtre (copie thread-safe).
func (m *Manager) ListTabs(windowID string) []*Tab {
	groupID := groupKey(windowID)
	m.mu.RLock()
	defer m.mu.RUnlock()
	group, exists := m.groups[groupID]
	if !exists {
		return nil
	}
	result := make([]*Tab, len(group.Tabs))
	copy(result, group.Tabs)
	return result
}

// ActiveTab retourne l'onglet actif d'une fenêtre.
func (m *Manager) ActiveTab(windowID string) (*Tab, bool) {
	groupID := groupKey(windowID)
	m.mu.RLock()
	defer m.mu.RUnlock()
	group, exists := m.groups[groupID]
	if !exists || group.ActiveID == "" {
		return nil, false
	}
	for _, t := range group.Tabs {
		if t.ID == group.ActiveID {
			cp := *t
			return &cp, true
		}
	}
	return nil, false
}

// Snapshot retourne l'état complet pour la persistence.
func (m *Manager) Snapshot() map[string]*TabGroup {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]*TabGroup, len(m.groups))
	for k, g := range m.groups {
		tabs := make([]*Tab, len(g.Tabs))
		copy(tabs, g.Tabs)
		result[k] = &TabGroup{
			ID:       g.ID,
			WindowID: g.WindowID,
			Tabs:     tabs,
			ActiveID: g.ActiveID,
		}
	}
	return result
}

// Restore charge un snapshot (appelé par WorkspacePersistence.Restore).
func (m *Manager) Restore(snapshot map[string]*TabGroup) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.groups = make(map[string]*TabGroup, len(snapshot))
	for k, g := range snapshot {
		m.groups[k] = g
	}
	m.logger.Info("tabs: state restored", slog.Int("groups", len(m.groups)))
}

// ─────────────────────────────────────────────
// HELPERS INTERNES
// ─────────────────────────────────────────────

func (m *Manager) setActive(group *TabGroup, tabID string) {
	for _, t := range group.Tabs {
		t.IsActive = t.ID == tabID
	}
	group.ActiveID = tabID
}

func (m *Manager) publishChanged(windowID string) {
	if m.bus == nil {
		return
	}
	m.bus.Publish(api.Event{
		ID:        uid.New(),
		Topic:     api.TopicEditorTabChanged,
		Source:    "tab-manager",
		Timestamp: time.Now().UTC(),
		Payload:   map[string]string{"window_id": windowID},
	})
}

func groupKey(windowID string) string {
	if windowID == "" {
		return "main"
	}
	return windowID
}

// detectLanguage devine le langage depuis l'extension du fichier.
func detectLanguage(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		switch strings.ToLower(path[i:]) {
		case ".go":
			return "go"
		case ".ts", ".tsx":
			return "typescript"
		case ".js", ".jsx":
			return "javascript"
		case ".json":
			return "json"
		case ".md":
			return "markdown"
		case ".yaml", ".yml":
			return "yaml"
		case ".html":
			return "html"
		case ".css":
			return "css"
		case ".py":
			return "python"
		case ".rs":
			return "rust"
		}
	}
	return "plaintext"
}