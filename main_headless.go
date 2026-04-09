// Axiom — Moteur d'ingénierie logicielle modulaire
// Point d'entrée principal.
//go:build !wails
package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/axiom-ide/axiom/api"
	axiomconfig "github.com/axiom-ide/axiom/core/config"
	"github.com/axiom-ide/axiom/core/engine"
	"github.com/axiom-ide/axiom/core/filesystem"
	"github.com/axiom-ide/axiom/core/module"
	"github.com/axiom-ide/axiom/core/orchestrator"
	"github.com/axiom-ide/axiom/core/tabs"
	"github.com/axiom-ide/axiom/core/workspace"
	aiassistant "github.com/axiom-ide/axiom/modules/ai-assistant"
)

func main() {
	// ── 1. Configuration ──────────────────────────────────────────
	cfg, warnings := axiomconfig.Load("")
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "[WARN] %s\n", w)
	}

	// ── 2. Moteur central ─────────────────────────────────────────
	eng, err := engine.New(engine.Config{
		ModulesDir:    cfg.Core.ModulesDir,
		LogLevel:      cfg.Core.LogLevel,
		BusBufferSize: cfg.Core.BusBufferSize,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: engine init: %v\n", err)
		os.Exit(1)
	}

	// ── 3. FileSystem Handler ─────────────────────────────────────
	fsPub := &enginePublisherProxy{eng: eng}
	fsHandler, err := filesystem.NewHandler(filesystem.Config{
		WorkspaceDir:   cfg.Core.WorkspaceDir,
		MaxFileSizeMB:  cfg.FileSystem.MaxFileSizeMB,
		IgnorePatterns: cfg.FileSystem.IgnorePatterns,
		BackupOnWrite:  cfg.FileSystem.BackupOnWrite,
	}, fsPub, slog.Default())
	if err != nil {
		slog.Warn("filesystem handler init failed (non-fatal)", slog.String("error", err.Error()))
	}
	_ = fsHandler

	// ── 4. Tab Manager ────────────────────────────────────────────
	tabMgr := tabs.NewManager(eng.Bus(), slog.Default())

	// ── 5. Module Runner ──────────────────────────────────────────
	runner := module.NewRunner(slog.Default())
	runner.Register(aiassistant.New(aiassistant.Config{
		Provider:    cfg.AI.Provider,
		BaseURL:     cfg.AI.BaseURL,
		ModelID:     cfg.AI.ModelID,
		APIKey:      cfg.AI.APIKey,
		MaxTokens:   cfg.AI.MaxTokens,
		Temperature: cfg.AI.Temperature,
		TimeoutSecs: cfg.AI.TimeoutSecs,
	}, slog.Default()))

	// ── 6. Window Orchestrator ────────────────────────────────────
	// nil → NoopAdapter par défaut (sans build tag wails)
	orch := orchestrator.NewOrchestrator(nil, eng.Bus(), slog.Default())
	_ = orch

	// ── 7. Workspace Persistence ──────────────────────────────────
	persist := workspace.NewPersistence(
		eng.Bus(),
		tabMgr,
		nil,
		cfg.Core.WorkspaceDir,
		slog.Default(),
	)

	// ── 8. Démarrage ──────────────────────────────────────────────
	if err := eng.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: engine start: %v\n", err)
		os.Exit(1)
	}

	tabMgr.Start()
	persist.Start()

	engProxy := &engineProxy{eng: eng}
	if errs := runner.InitAll(eng.Context(), engProxy, engProxy); len(errs) > 0 {
		for _, e := range errs {
			slog.Warn("module init error", slog.String("error", e.Error()))
		}
	}

	slog.Info("axiom: ready ✓",
		slog.String("provider", cfg.AI.Provider),
		slog.String("workspace", cfg.Core.WorkspaceDir),
	)

	// ── 9. Signaux OS ─────────────────────────────────────────────
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-sigCh:
		slog.Info("axiom: signal received", slog.String("signal", sig.String()))
	case <-eng.Context().Done():
	}

	// ── 10. Arrêt propre ──────────────────────────────────────────
	_ = persist.SaveSync()
	_ = runner.StopAll()
	eng.Shutdown()
}

// ─────────────────────────────────────────────
// Proxies internes
// ─────────────────────────────────────────────

// enginePublisherProxy expose Engine comme filesystem.EventPublisher.
// Le filesystem est un composant interne L3 — il bypasse le security check.
type enginePublisherProxy struct{ eng *engine.Engine }

func (p *enginePublisherProxy) Subscribe(topic api.Topic, handler func(api.Event)) string {
	return p.eng.Subscribe(topic, handler)
}

func (p *enginePublisherProxy) Publish(event api.Event) {
	_ = p.eng.Dispatch("filesystem", event.Topic, event.Payload)
}

// engineProxy expose Engine via module.Dispatcher + module.Subscriber.
type engineProxy struct{ eng *engine.Engine }

func (p *engineProxy) Dispatch(moduleID string, topic api.Topic, payload interface{}) error {
	return p.eng.Dispatch(moduleID, topic, payload)
}

func (p *engineProxy) Subscribe(topic api.Topic, handler func(api.Event)) string {
	return p.eng.Subscribe(topic, handler)
}

func (p *engineProxy) Unsubscribe(topic api.Topic, subID string) {
	p.eng.Unsubscribe(topic, subID)
}