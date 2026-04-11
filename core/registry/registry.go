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
	"github.com/axiom-ide/axiom/core/jsruntime"
	"github.com/axiom-ide/axiom/core/module"
	"github.com/axiom-ide/axiom/core/security"
)

type ModuleState string

const (
	StateLoading  ModuleState = "loading"
	StateActive   ModuleState = "active"
	StateError    ModuleState = "error"
	StateDisabled ModuleState = "disabled"
)

type ModuleRecord struct {
	Manifest  *Manifest
	State     ModuleState
	LoadedAt  time.Time
	Directory string
	Error     string
	IsJS      bool // true si module JavaScript
}

// ModuleRunner est l'interface du Runner Go existant + ajout du rechargement JS.
type ModuleRunner interface {
	Register(m module.Module)
}

type Registry struct {
	mu         sync.RWMutex
	modules    map[string]*ModuleRecord
	jsModules  map[string]*jsruntime.JSModule // modules JS actifs
	security   *security.Manager
	publishFn  func(api.Event)
	logger     *slog.Logger
	modulesDir string
	watcher    *jsruntime.Watcher
	dispatcher module.Dispatcher
	subscriber module.Subscriber
}

func NewRegistry(
	modulesDir string,
	secMgr *security.Manager,
	publishFn func(api.Event),
	logger *slog.Logger,
) *Registry {
	watcher, err := jsruntime.NewWatcher(logger)
	if err != nil {
		logger.Warn("registry: cannot create watcher (hot-reload disabled)", slog.String("error", err.Error()))
	}
	return &Registry{
		modules:    make(map[string]*ModuleRecord),
		jsModules:  make(map[string]*jsruntime.JSModule),
		security:   secMgr,
		publishFn:  publishFn,
		logger:     logger,
		modulesDir: modulesDir,
		watcher:    watcher,
	}
}

// SetDispatcherSubscriber injecte le dispatcher/subscriber pour les modules JS.
// Doit être appelé avant ScanAndLoad().
func (r *Registry) SetDispatcherSubscriber(d module.Dispatcher, s module.Subscriber) {
	r.dispatcher = d
	r.subscriber = s
}

func (r *Registry) ScanAndLoad() error {
	r.logger.Info("registry: scanning modules directory", slog.String("path", r.modulesDir))

	entries, err := os.ReadDir(r.modulesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			r.logger.Warn("registry: modules directory does not exist, creating it",
				slog.String("path", r.modulesDir))
			return os.MkdirAll(r.modulesDir, 0755)
		}
		return fmt.Errorf("registry: cannot read modules dir: %w", err)
	}

	// Démarrer le watcher de hot-reload
	if r.watcher != nil {
		r.watcher.Start()
	}

	loaded, failed := 0, 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
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

func (r *Registry) loadModule(moduleDir string) error {
	manifest, err := LoadManifest(moduleDir)
	if err != nil {
		r.publishModuleError(filepath.Base(moduleDir), err.Error())
		return err
	}
	if !manifest.IsEnabled() {
		r.logger.Info("registry: module disabled, skipping", slog.String("id", manifest.ID))
		r.addRecord(&ModuleRecord{Manifest: manifest, State: StateDisabled, Directory: moduleDir})
		return nil
	}

	// ── Détecter si c'est un module JS ─────────────────────────────
	jsPath := filepath.Join(moduleDir, "index.js")
	if _, err := os.Stat(jsPath); err == nil {
		return r.loadJSModule(manifest, moduleDir, jsPath)
	}

	// ── Sinon : module Go classique (manifest only, enregistrement sécurité) ──
	if manifest.ClearanceLevel >= int(security.L2) {
		r.logger.Warn("registry: module requests elevated clearance — verify it is trusted",
			slog.String("id", manifest.ID),
			slog.String("clearance", manifest.ClearanceAsLevel().String()),
		)
	}
	if err := r.security.RegisterModule(manifest.ID, manifest.ClearanceAsLevel()); err != nil {
		r.publishModuleError(manifest.ID, err.Error())
		return fmt.Errorf("registry: security registration failed for '%s': %w", manifest.ID, err)
	}
	record := &ModuleRecord{
		Manifest:  manifest,
		State:     StateActive,
		LoadedAt:  time.Now().UTC(),
		Directory: moduleDir,
	}
	r.addRecord(record)
	r.publishModuleLoaded(manifest)
	return nil
}

// loadJSModule charge et démarre un module JavaScript.
func (r *Registry) loadJSModule(manifest *Manifest, moduleDir, jsPath string) error {
	code, err := os.ReadFile(jsPath)
	if err != nil {
		return fmt.Errorf("registry: cannot read JS module '%s': %w", jsPath, err)
	}

	// Enregistrer dans le Security Manager
	if err := r.security.RegisterModule(manifest.ID, manifest.ClearanceAsLevel()); err != nil {
		return fmt.Errorf("registry: security registration failed for JS module '%s': %w", manifest.ID, err)
	}

	// Créer le module JS
	jsMod := jsruntime.NewJSModule(
		manifest.ID,
		manifest.Name,
		manifest.ClearanceAsLevel(),
		jsPath,
		string(code),
		r.logger,
	)

	// Initialiser avec le dispatcher/subscriber si disponibles
	if r.dispatcher != nil && r.subscriber != nil {
		if err := jsMod.Init(nil, r.dispatcher, r.subscriber); err != nil {
			r.publishModuleError(manifest.ID, err.Error())
			return fmt.Errorf("registry: JS module init failed '%s': %w", manifest.ID, err)
		}
	}

	// Enregistrer le watcher pour le hot-reload
	if r.watcher != nil {
		modID := manifest.ID // capture pour la closure
		mod := jsMod
		_ = r.watcher.Watch(jsPath, func(newCode string) {
			r.logger.Info("registry: hot-reloading JS module", slog.String("id", modID))
			if err := mod.Reload(newCode); err != nil {
				r.logger.Error("registry: hot-reload failed",
					slog.String("id", modID),
					slog.String("error", err.Error()),
				)
				return
			}
			r.publishFn(api.Event{
				Topic:  api.TopicModuleLoaded,
				Source: "registry",
				Payload: api.PayloadModuleStatus{
					ModuleID: modID,
					Name:     manifest.Name,
					Version:  manifest.Version + " (reloaded)",
				},
			})
		})
	}

	r.mu.Lock()
	r.jsModules[manifest.ID] = jsMod
	r.mu.Unlock()

	record := &ModuleRecord{
		Manifest:  manifest,
		State:     StateActive,
		LoadedAt:  time.Now().UTC(),
		Directory: moduleDir,
		IsJS:      true,
	}
	r.addRecord(record)
	r.publishModuleLoaded(manifest)

	r.logger.Info("registry: JS module loaded",
		slog.String("id", manifest.ID),
		slog.String("clearance", manifest.ClearanceAsLevel().String()),
	)
	return nil
}

// StopAll arrête tous les modules JS.
func (r *Registry) StopAll() {
	r.mu.RLock()
	mods := make([]*jsruntime.JSModule, 0, len(r.jsModules))
	for _, m := range r.jsModules {
		mods = append(mods, m)
	}
	r.mu.RUnlock()
	for _, m := range mods {
		_ = m.Stop()
	}
	if r.watcher != nil {
		_ = r.watcher.Stop()
	}
}

func (r *Registry) Get(moduleID string) (*ModuleRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.modules[moduleID]
	return rec, ok
}

func (r *Registry) ListActive() []*ModuleRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ModuleRecord, 0)
	for _, rec := range r.modules {
		if rec.State == StateActive {
			result = append(result, rec)
		}
	}
	return result
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.modules)
}

func (r *Registry) addRecord(rec *ModuleRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modules[rec.Manifest.ID] = rec
}

func (r *Registry) publishModuleLoaded(manifest *Manifest) {
	r.publishFn(api.Event{
		Topic:  api.TopicModuleLoaded,
		Source: "registry",
		Payload: api.PayloadModuleStatus{
			ModuleID: manifest.ID,
			Name:     manifest.Name,
			Version:  manifest.Version,
		},
	})
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