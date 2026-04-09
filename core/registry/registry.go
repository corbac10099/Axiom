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
}

type Registry struct {
	mu         sync.RWMutex
	modules    map[string]*ModuleRecord
	security   *security.Manager
	publishFn  func(api.Event)
	logger     *slog.Logger
	modulesDir string
}

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
	r.logger.Info("registry: scan complete", slog.Int("loaded", loaded), slog.Int("failed", failed))
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
		slog.String("clearance", manifest.ClearanceAsLevel().String()),
	)
	return nil
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