package registry

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/axiom-ide/axiom/api"
	"github.com/axiom-ide/axiom/core/security"
)

// ─────────────────────────────────────────────
// MODULE RECORD
// ─────────────────────────────────────────────

// ModuleState représente l'état actuel d'un module dans le Registry.
type ModuleState string

const (
	StateLoading ModuleState = "loading"
	StateActive  ModuleState = "active"
	StateError   ModuleState = "error"
	StateDisabled ModuleState = "disabled"
)

// ModuleRecord est l'entrée complète d'un module dans le Registry.
type ModuleRecord struct {
	Manifest  *Manifest
	State     ModuleState
	LoadedAt  time.Time
	Directory string // chemin absolu du dossier du module
	Error     string // message d'erreur si State == StateError
}

// ─────────────────────────────────────────────
// REGISTRY
// ─────────────────────────────────────────────

// Registry maintient la liste de tous les modules connus d'Axiom.
// Il est le seul composant autorisé à interagir avec le Security Manager
// pour enregistrer de nouveaux modules.
type Registry struct {
	mu      sync.RWMutex
	modules map[string]*ModuleRecord // clé : manifest.ID

	security  *security.Manager
	publishFn func(api.Event)
	logger    *slog.Logger

	// modulesDir est le chemin vers le dossier racine des modules.
	modulesDir string
}

// NewRegistry crée un Registry.
func NewRegistry(
	modulesDir string,
	secMgr *security.Manager,
	publishFn func(api.Event),
	logger *slog.Logger,
) *Registry {
	return &Registry{
		modules:    make(map[string]*ModuleRecord),
		security:   secMgr,
		publishFn:  publishFn,
		logger:     logger,
		modulesDir: modulesDir,
	}
}

// ─────────────────────────────────────────────
// SCAN — Découverte automatique des modules
// ─────────────────────────────────────────────

// ScanAndLoad scanne le dossier modulesDir et charge tous les modules valides.
// Chaque sous-dossier contenant un manifest.json est considéré comme un module.
// Les erreurs de chargement individuelles n'arrêtent PAS le scan global.
func (r *Registry) ScanAndLoad() error {
	r.logger.Info("registry: scanning modules directory",
		slog.String("path", r.modulesDir),
	)

	entries, err := os.ReadDir(r.modulesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			r.logger.Warn("registry: modules directory does not exist, creating it",
				slog.String("path", r.modulesDir),
			)
			return os.MkdirAll(r.modulesDir, 0755)
		}
		return fmt.Errorf("registry: cannot read modules dir: %w", err)
	}

	loaded := 0
	failed := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue // on ignore les fichiers à la racine
		}

		moduleDir := filepath.Join(r.modulesDir, entry.Name())
		if err := r.loadModule(moduleDir); err != nil {
			r.logger.Error("registry: module load failed",
				slog.String("dir", entry.Name()),
				slog.String("error", err.Error()),
			)
			failed++
		} else {
			loaded++
		}
	}

	r.logger.Info("registry: scan complete",
		slog.Int("loaded", loaded),
		slog.Int("failed", failed),
	)
	return nil
}

// loadModule charge un module depuis son répertoire.
// Lit le manifest, vérifie les accréditations, enregistre auprès du Security Manager.
func (r *Registry) loadModule(moduleDir string) error {
	// 1. Lire le manifest
	manifest, err := LoadManifest(moduleDir)
	if err != nil {
		r.publishModuleError(filepath.Base(moduleDir), err.Error())
		return err
	}

	// 2. Vérifier si le module est activé
	if !manifest.IsEnabled() {
		r.logger.Info("registry: module disabled, skipping",
			slog.String("id", manifest.ID),
		)
		r.addRecord(&ModuleRecord{
			Manifest:  manifest,
			State:     StateDisabled,
			Directory: moduleDir,
		})
		return nil
	}

	// 3. Vérification de sécurité : les niveaux L2 et L3 doivent être
	// explicitement autorisés (dans un vrai système, une UI de confirmation
	// serait affichée à l'opérateur humain ici).
	if manifest.ClearanceLevel >= int(security.L2) {
		r.logger.Warn("registry: module requests elevated clearance — verify it is trusted",
			slog.String("id", manifest.ID),
			slog.String("clearance", manifest.ClearanceAsLevel().String()),
		)
	}

	// 4. Enregistrer auprès du Security Manager
	if err := r.security.RegisterModule(manifest.ID, manifest.ClearanceAsLevel()); err != nil {
		r.publishModuleError(manifest.ID, err.Error())
		return fmt.Errorf("registry: security registration failed for '%s': %w", manifest.ID, err)
	}

	// 5. Ajouter au registre local
	record := &ModuleRecord{
		Manifest:  manifest,
		State:     StateActive,
		LoadedAt:  time.Now().UTC(),
		Directory: moduleDir,
	}
	r.addRecord(record)

	// 6. Publier l'événement de chargement réussi sur le bus
	r.publishFn(api.Event{
		Topic:  api.TopicModuleLoaded,
		Source: "registry",
		Payload: api.PayloadModuleStatus{
			ModuleID: manifest.ID,
			Name:     manifest.Name,
			Version:  manifest.Version,
		},
	})

	r.logger.Info("registry: module loaded successfully",
		slog.String("id", manifest.ID),
		slog.String("name", manifest.Name),
		slog.String("version", manifest.Version),
		slog.String("clearance", manifest.ClearanceAsLevel().String()),
	)
	return nil
}

// ─────────────────────────────────────────────
// QUERIES
// ─────────────────────────────────────────────

// Get retourne le ModuleRecord d'un module par son ID.
func (r *Registry) Get(moduleID string) (*ModuleRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.modules[moduleID]
	return rec, ok
}

// ListActive retourne tous les modules actifs.
func (r *Registry) ListActive() []*ModuleRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ModuleRecord, 0, len(r.modules))
	for _, rec := range r.modules {
		if rec.State == StateActive {
			result = append(result, rec)
		}
	}
	return result
}

// Count retourne le nombre total de modules enregistrés.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.modules)
}

// ─────────────────────────────────────────────
// INTERNAL
// ─────────────────────────────────────────────

func (r *Registry) addRecord(rec *ModuleRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modules[rec.Manifest.ID] = rec
}

func (r *Registry) publishModuleError(moduleID, errMsg string) {
	r.publishFn(api.Event{
		Topic:  api.TopicModuleError,
		Source: "registry",
		Payload: api.PayloadModuleStatus{
			ModuleID: moduleID,
			Error:    errMsg,
		},
	})
}