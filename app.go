//go:build wails

package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/axiom-ide/axiom/api"
	axiomconfig "github.com/axiom-ide/axiom/core/config"
	"github.com/axiom-ide/axiom/core/engine"
	"github.com/axiom-ide/axiom/core/filesystem"
	"github.com/axiom-ide/axiom/core/module"
	"github.com/axiom-ide/axiom/core/orchestrator"
	"github.com/axiom-ide/axiom/core/tabs"
	"github.com/axiom-ide/axiom/core/workspace"
	aiassistant "github.com/axiom-ide/axiom/modules/ai-assistant"
	wailsadapter "github.com/axiom-ide/axiom/adapters/wails"
)

// App est le struct central exposé au frontend via Wails.
// Toutes les méthodes publiques deviennent des fonctions appelables depuis le JS.
type App struct {
	ctx     context.Context
	cancel  context.CancelFunc
	eng     *engine.Engine
	tabMgr  *tabs.Manager
	persist *workspace.Persistence
	runner  *module.Runner
	fsHdlr  *filesystem.Handler
	adapter *wailsadapter.Adapter
	cfg     axiomconfig.Config
	logger  *slog.Logger
}

// NewApp crée l'instance App.
func NewApp() *App {
	return &App{logger: slog.Default()}
}

// Startup initialise le moteur Axiom. Appelé depuis main() AVANT app.Run().
func (a *App) Startup(wailsApp *application.App) {
	a.ctx, a.cancel = context.WithCancel(context.Background())

	// ── Config ──────────────────────────────────────────────────
	cfg, warnings := axiomconfig.Load("")
	for _, w := range warnings {
		a.logger.Warn("config", slog.String("msg", w))
	}
	a.cfg = cfg

	// ── Engine ──────────────────────────────────────────────────
	eng, err := engine.New(engine.Config{
		ModulesDir:    cfg.Core.ModulesDir,
		LogLevel:      cfg.Core.LogLevel,
		BusBufferSize: cfg.Core.BusBufferSize,
	})
	if err != nil {
		a.logger.Error("engine init failed", slog.String("error", err.Error()))
		return
	}
	a.eng = eng

	// ── FileSystem ───────────────────────────────────────────────
	fsPub := &appEngineProxy{eng: eng}
	fsHdlr, err := filesystem.NewHandler(filesystem.Config{
		WorkspaceDir:   cfg.Core.WorkspaceDir,
		MaxFileSizeMB:  cfg.FileSystem.MaxFileSizeMB,
		IgnorePatterns: cfg.FileSystem.IgnorePatterns,
		BackupOnWrite:  cfg.FileSystem.BackupOnWrite,
	}, fsPub, a.logger)
	if err != nil {
		a.logger.Warn("filesystem init failed (non-fatal)", slog.String("error", err.Error()))
	}
	a.fsHdlr = fsHdlr

	// ── Wails v3 Adapter ─────────────────────────────────────────
	adapter := wailsadapter.NewAdapter(wailsApp, a.logger)
	adapter.RegisterBidirectional(eng.Bus())
	a.adapter = adapter

	// ── Orchestrator ─────────────────────────────────────────────
	_ = orchestrator.NewOrchestrator(adapter, eng.Bus(), a.logger)

	// ── Tab Manager ───────────────────────────────────────────────
	a.tabMgr = tabs.NewManager(eng.Bus(), a.logger)

	// ── Module Runner ─────────────────────────────────────────────
	a.runner = module.NewRunner(a.logger)
	a.runner.Register(aiassistant.New(aiassistant.Config{
		Provider:    cfg.AI.Provider,
		BaseURL:     cfg.AI.BaseURL,
		ModelID:     cfg.AI.ModelID,
		APIKey:      cfg.AI.APIKey,
		MaxTokens:   cfg.AI.MaxTokens,
		Temperature: cfg.AI.Temperature,
		TimeoutSecs: cfg.AI.TimeoutSecs,
	}, a.logger))

	// ── Workspace Persistence ─────────────────────────────────────
	a.persist = workspace.NewPersistence(
		eng.Bus(), a.tabMgr, nil,
		cfg.Core.WorkspaceDir, a.logger,
	)

	// ── Démarrage ─────────────────────────────────────────────────
	if err := eng.Start(); err != nil {
		a.logger.Error("engine start failed", slog.String("error", err.Error()))
		return
	}
	a.tabMgr.Start()
	a.persist.Start()

	engProxy := &appEngineProxy{eng: eng}
	if errs := a.runner.InitAll(a.ctx, engProxy, engProxy); len(errs) > 0 {
		for _, e := range errs {
			a.logger.Warn("module init error", slog.String("error", e.Error()))
		}
	}

	a.logger.Info("axiom wails v3: ready ✓")
}

// Shutdown est appelé à la fermeture de l'application.
func (a *App) Shutdown() {
	if a.persist != nil {
		_ = a.persist.SaveSync()
	}
	if a.runner != nil {
		_ = a.runner.StopAll()
	}
	if a.eng != nil {
		a.eng.Shutdown()
	}
	if a.cancel != nil {
		a.cancel()
	}
}

// ─────────────────────────────────────────────
// MÉTHODES BINDÉES (appelables depuis le JS)
// ─────────────────────────────────────────────

// ReadFile lit un fichier du workspace.
func (a *App) ReadFile(path string) (string, error) {
	if a.fsHdlr == nil {
		return "", fmt.Errorf("filesystem not initialized")
	}
	result, err := a.fsHdlr.ReadFile(path)
	if err != nil {
		return "", err
	}
	return result.Content, nil
}

// WriteFile écrit du contenu dans un fichier du workspace.
func (a *App) WriteFile(path, content string) error {
	if a.fsHdlr == nil {
		return fmt.Errorf("filesystem not initialized")
	}
	return a.fsHdlr.WriteFile(path, content, false)
}

// ListDir liste le contenu d'un répertoire du workspace.
func (a *App) ListDir(path string) ([]filesystem.FileEntry, error) {
	if a.fsHdlr == nil {
		return nil, fmt.Errorf("filesystem not initialized")
	}
	return a.fsHdlr.ListDir(path)
}

// SetTheme change le thème de l'IDE.
func (a *App) SetTheme(themeID string) error {
	if a.eng == nil {
		return fmt.Errorf("engine not initialized")
	}
	return a.eng.Dispatch("engine", api.TopicUISetTheme, api.PayloadUITheme{ThemeID: themeID})
}

// GetConfig retourne la config courante (sans les clés sensibles).
func (a *App) GetConfig() map[string]any {
	return map[string]any{
		"theme":     a.cfg.UI.DefaultTheme,
		"provider":  a.cfg.AI.Provider,
		"model":     a.cfg.AI.ModelID,
		"workspace": a.cfg.Core.WorkspaceDir,
	}
}

// ─────────────────────────────────────────────
// Proxies internes
// ─────────────────────────────────────────────

type appEngineProxy struct{ eng *engine.Engine }

func (p *appEngineProxy) Subscribe(topic api.Topic, handler func(api.Event)) string {
	return p.eng.Subscribe(topic, handler)
}
func (p *appEngineProxy) Publish(event api.Event) {
	_ = p.eng.Dispatch("filesystem", event.Topic, event.Payload)
}
func (p *appEngineProxy) Dispatch(moduleID string, topic api.Topic, payload any) error {
	return p.eng.Dispatch(moduleID, topic, payload)
}
func (p *appEngineProxy) Unsubscribe(topic api.Topic, subID string) {
	p.eng.Unsubscribe(topic, subID)
}