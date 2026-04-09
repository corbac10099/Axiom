// Package filesystem implémente le gestionnaire de fichiers d'Axiom.
// Il s'abonne aux Topics FILE_* du bus et effectue les opérations réelles
// sur le système de fichiers de l'OS, en respectant le workspace configuré.
//
// Sécurité additionnelle : path traversal prevention.
// Toute tentative d'accès hors du workspace est bloquée, même si le
// Security Manager a autorisé l'action au niveau clearance.
package filesystem

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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

// FileEntry est la description d'un fichier ou dossier dans le workspace.
type FileEntry struct {
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	IsDir    bool      `json:"is_dir"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"mod_time"`
	MimeType string    `json:"mime_type,omitempty"`
}

// ReadResult est le résultat d'une opération de lecture.
type ReadResult struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Size    int64  `json:"size"`
}

// OperationResult est la réponse générique à une opération fichier.
type OperationResult struct {
	Success bool   `json:"success"`
	Path    string `json:"path"`
	Error   string `json:"error,omitempty"`
}

// ─────────────────────────────────────────────
// HANDLER
// ─────────────────────────────────────────────

// Handler s'abonne aux Topics FILE_* du bus et opère sur le FS réel.
// Il est le seul composant d'Axiom autorisé à toucher le disque directement.
type Handler struct {
	mu           sync.RWMutex
	workspaceDir string  // racine absolue du workspace (anti path-traversal)
	maxFileSizeMB int
	ignorePatterns []string
	backupOnWrite bool

	bus    *bus.EventBus
	logger *slog.Logger
	subIDs []string // pour cleanup
}

// Config regroupe les paramètres du FileSystem Handler.
type Config struct {
	WorkspaceDir   string
	MaxFileSizeMB  int
	IgnorePatterns []string
	BackupOnWrite  bool
}

// NewHandler crée un Handler et l'attache au bus.
func NewHandler(cfg Config, eventBus *bus.EventBus, logger *slog.Logger) (*Handler, error) {
	// Résoudre le workspace en chemin absolu pour l'anti-traversal
	abs, err := filepath.Abs(cfg.WorkspaceDir)
	if err != nil {
		return nil, fmt.Errorf("filesystem: cannot resolve workspace path '%s': %w", cfg.WorkspaceDir, err)
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return nil, fmt.Errorf("filesystem: cannot create workspace '%s': %w", abs, err)
	}

	maxMB := cfg.MaxFileSizeMB
	if maxMB <= 0 {
		maxMB = 50
	}

	h := &Handler{
		workspaceDir:   abs,
		maxFileSizeMB:  maxMB,
		ignorePatterns: cfg.IgnorePatterns,
		backupOnWrite:  cfg.BackupOnWrite,
		bus:            eventBus,
		logger:         logger,
	}

	h.subscribeToEvents()
	logger.Info("filesystem: handler ready", slog.String("workspace", abs))
	return h, nil
}

// WorkspaceDir retourne le chemin absolu du workspace courant.
func (h *Handler) WorkspaceDir() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.workspaceDir
}

// SetWorkspace change le workspace courant. Thread-safe.
func (h *Handler) SetWorkspace(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("filesystem: invalid workspace path: %w", err)
	}
	h.mu.Lock()
	h.workspaceDir = abs
	h.mu.Unlock()
	h.logger.Info("filesystem: workspace changed", slog.String("path", abs))
	return nil
}

// ─────────────────────────────────────────────
// OPÉRATIONS FICHIERS
// ─────────────────────────────────────────────

// ReadFile lit le contenu d'un fichier dans le workspace.
func (h *Handler) ReadFile(relPath string) (ReadResult, error) {
	absPath, err := h.safePath(relPath)
	if err != nil {
		return ReadResult{}, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return ReadResult{}, fmt.Errorf("filesystem: cannot stat '%s': %w", relPath, err)
	}
	if info.IsDir() {
		return ReadResult{}, fmt.Errorf("filesystem: '%s' is a directory", relPath)
	}

	maxBytes := int64(h.maxFileSizeMB) * 1024 * 1024
	if info.Size() > maxBytes {
		return ReadResult{}, fmt.Errorf("filesystem: file '%s' too large (%d MB > %d MB limit)",
			relPath, info.Size()/1024/1024, h.maxFileSizeMB)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return ReadResult{}, fmt.Errorf("filesystem: cannot read '%s': %w", relPath, err)
	}

	return ReadResult{Path: relPath, Content: string(data), Size: info.Size()}, nil
}

// WriteFile écrit (ou crée) un fichier dans le workspace.
func (h *Handler) WriteFile(relPath, content string, appendMode bool) error {
	absPath, err := h.safePath(relPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("filesystem: cannot create parent dirs for '%s': %w", relPath, err)
	}

	// Backup optionnel avant écrasement
	if h.backupOnWrite {
		if _, statErr := os.Stat(absPath); statErr == nil {
			backupPath := absPath + ".bak"
			if bakData, readErr := os.ReadFile(absPath); readErr == nil {
				_ = os.WriteFile(backupPath, bakData, 0644)
			}
		}
	}

	flag := os.O_WRONLY | os.O_CREATE
	if appendMode {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	f, err := os.OpenFile(absPath, flag, 0644)
	if err != nil {
		return fmt.Errorf("filesystem: cannot open '%s' for writing: %w", relPath, err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("filesystem: write failed for '%s': %w", relPath, err)
	}

	h.logger.Debug("filesystem: file written",
		slog.String("path", relPath),
		slog.Int("bytes", len(content)),
		slog.Bool("append", appendMode),
	)
	return nil
}

// CreateFile crée un nouveau fichier (échoue si déjà existant).
func (h *Handler) CreateFile(relPath, content string) error {
	absPath, err := h.safePath(relPath)
	if err != nil {
		return err
	}

	if _, err := os.Stat(absPath); err == nil {
		return fmt.Errorf("filesystem: file already exists '%s'", relPath)
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("filesystem: cannot create parent dirs: %w", err)
	}

	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("filesystem: cannot create '%s': %w", relPath, err)
	}

	h.logger.Info("filesystem: file created", slog.String("path", relPath))
	return nil
}

// DeleteFile supprime un fichier ou un dossier vide du workspace.
func (h *Handler) DeleteFile(relPath string) error {
	absPath, err := h.safePath(relPath)
	if err != nil {
		return err
	}
	if err := os.Remove(absPath); err != nil {
		return fmt.Errorf("filesystem: cannot delete '%s': %w", relPath, err)
	}
	h.logger.Info("filesystem: file deleted", slog.String("path", relPath))
	return nil
}

// ListDir liste les entrées d'un répertoire du workspace.
func (h *Handler) ListDir(relPath string) ([]FileEntry, error) {
	absPath, err := h.safePath(relPath)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("filesystem: cannot list '%s': %w", relPath, err)
	}

	result := make([]FileEntry, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if h.shouldIgnore(name) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		result = append(result, FileEntry{
			Path:    filepath.Join(relPath, name),
			Name:    name,
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
	return result, nil
}

// ─────────────────────────────────────────────
// BUS SUBSCRIPTIONS
// ─────────────────────────────────────────────

// subscribeToEvents câble le Handler aux Topics FILE_* du bus.
func (h *Handler) subscribeToEvents() {
	// TopicFileRead → lit et publie TopicFileOpened avec le contenu
	id := h.bus.Subscribe(api.TopicFileRead, func(ev api.Event) {
		p, ok := ev.Payload.(api.PayloadFileRead)
		if !ok {
			return
		}
		result, err := h.ReadFile(p.Path)
		if err != nil {
			h.logger.Warn("filesystem: read failed",
				slog.String("path", p.Path),
				slog.String("error", err.Error()),
			)
			h.replyError(ev, p.Path, err)
			return
		}
		// Broadcast du contenu lu — les modules intéressés (éditeur, IA) reçoivent
		h.bus.Publish(api.Event{
			ID:            uid.New(),
			Topic:         api.TopicFileOpened,
			Source:        "filesystem",
			CorrelationID: ev.CorrelationID,
			Payload:       result,
		})
	})
	h.subIDs = append(h.subIDs, id)

	// TopicFileCreate → crée le fichier
	id = h.bus.Subscribe(api.TopicFileCreate, func(ev api.Event) {
		p, ok := ev.Payload.(api.PayloadFileCreate)
		if !ok {
			return
		}
		err := h.CreateFile(p.Path, p.Content)
		h.replyOperation(ev, p.Path, err)
	})
	h.subIDs = append(h.subIDs, id)

	// TopicFileWrite → écrit dans le fichier
	id = h.bus.Subscribe(api.TopicFileWrite, func(ev api.Event) {
		p, ok := ev.Payload.(api.PayloadFileWrite)
		if !ok {
			return
		}
		err := h.WriteFile(p.Path, p.Content, p.Append)
		h.replyOperation(ev, p.Path, err)
	})
	h.subIDs = append(h.subIDs, id)

	// TopicFileDelete → supprime le fichier
	id = h.bus.Subscribe(api.TopicFileDelete, func(ev api.Event) {
		p, ok := ev.Payload.(api.PayloadFileRead) // réutilise PayloadFileRead (juste un Path)
		if !ok {
			return
		}
		err := h.DeleteFile(p.Path)
		h.replyOperation(ev, p.Path, err)
	})
	h.subIDs = append(h.subIDs, id)

	h.logger.Debug("filesystem: subscribed to FILE_* events")
}

// ─────────────────────────────────────────────
// SÉCURITÉ — Anti path traversal
// ─────────────────────────────────────────────

// safePath valide et résout un chemin relatif en chemin absolu sûr.
// Toute tentative de sortir du workspace est bloquée.
func (h *Handler) safePath(relPath string) (string, error) {
	h.mu.RLock()
	workspace := h.workspaceDir
	h.mu.RUnlock()

	// Résoudre le chemin absolu
	var absPath string
	if filepath.IsAbs(relPath) {
		absPath = filepath.Clean(relPath)
	} else {
		absPath = filepath.Clean(filepath.Join(workspace, relPath))
	}

	// Vérification anti-traversal : le chemin résolu doit être
	// DANS le workspace (préfixé par workspace + séparateur)
	if !strings.HasPrefix(absPath, workspace+string(os.PathSeparator)) &&
		absPath != workspace {
		return "", fmt.Errorf("filesystem: path traversal attempt blocked: '%s' is outside workspace '%s'",
			relPath, workspace)
	}

	return absPath, nil
}

// shouldIgnore retourne true si un nom de fichier/dossier doit être ignoré.
func (h *Handler) shouldIgnore(name string) bool {
	for _, pattern := range h.ignorePatterns {
		matched, err := filepath.Match(pattern, name)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// ─────────────────────────────────────────────
// HELPERS
// ─────────────────────────────────────────────

func (h *Handler) replyOperation(ev api.Event, path string, err error) {
	result := OperationResult{Success: err == nil, Path: path}
	if err != nil {
		result.Error = err.Error()
		h.logger.Warn("filesystem: operation failed",
			slog.String("topic", string(ev.Topic)),
			slog.String("path", path),
			slog.String("error", err.Error()),
		)
	}
	if ev.ReplyTo != "" {
		h.bus.Publish(api.Event{
			ID:            uid.New(),
			Topic:         ev.ReplyTo,
			Source:        "filesystem",
			CorrelationID: ev.CorrelationID,
			Payload:       result,
		})
	}
}

func (h *Handler) replyError(ev api.Event, path string, err error) {
	h.replyOperation(ev, path, err)
}

// Stop désabonne le Handler de tous les Topics.
func (h *Handler) Stop() {
	// Le bus est arrêté par le contexte — pas besoin de désabonnement explicite ici
	// (le channel est fermé par bus.shutdown()). Méthode présente pour l'interface.
	h.logger.Debug("filesystem: handler stopped")
}