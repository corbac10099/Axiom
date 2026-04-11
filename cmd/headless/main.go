// Package main — Axiom headless mode (no UI).
// Build: go build -o axiom-headless ./cmd/headless/
// Run:   ./axiom-headless
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
	// ── Config ────────────────────────────────────────────────────
	cfg, warnings := axiomconfig.Load("")
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "[WARN] %s\n", w)
	}

	logger := newLogger(cfg.Core.LogLevel)

	// ── Engine ────────────────────────────────────────────────────
	eng, err := engine.New(engine.Config{
		ModulesDir:    cfg.Core.ModulesDir,
		LogLevel:      cfg.Core.LogLevel,
		BusBufferSize: cfg.Core.BusBufferSize,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: engine init: %v\n", err)
		os.Exit(1)
	}

	// ── Filesystem ────────────────────────────────────────────────
	fsPub := &fsPublisher{eng: eng}
	fsHandler, err := filesystem.NewHandler(filesystem.Config{
		WorkspaceDir:   cfg.Core.WorkspaceDir,
		MaxFileSizeMB:  cfg.FileSystem.MaxFileSizeMB,
		IgnorePatterns: cfg.FileSystem.IgnorePatterns,
		BackupOnWrite:  cfg.FileSystem.BackupOnWrite,
	}, fsPub, logger)
	if err != nil {
		logger.Warn("filesystem handler init failed (non-fatal)", slog.String("error", err.Error()))
	}
	_ = fsHandler

	// ── Orchestrator (noop — headless has no native windows) ──────
	_ = orchestrator.NewOrchestrator(nil, eng.Bus(), logger)

	// ── Tab Manager ───────────────────────────────────────────────
	tabMgr := tabs.NewManager(eng.Bus(), logger)

	// ── Module Runner ─────────────────────────────────────────────
	runner := module.NewRunner(logger)
	runner.Register(aiassistant.New(aiassistant.Config{
		Provider:    cfg.AI.Provider,
		BaseURL:     cfg.AI.BaseURL,
		ModelID:     cfg.AI.ModelID,
		APIKey:      cfg.AI.APIKey,
		MaxTokens:   cfg.AI.MaxTokens,
		Temperature: cfg.AI.Temperature,
		TimeoutSecs: cfg.AI.TimeoutSecs,
	}, logger))

	// ── Workspace Persistence ─────────────────────────────────────
	persist := workspace.NewPersistence(
		eng.Bus(), tabMgr, nil,
		cfg.Core.WorkspaceDir, logger,
	)

	// ── Start ─────────────────────────────────────────────────────
	if err := eng.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: engine start: %v\n", err)
		os.Exit(1)
	}

	tabMgr.Start()
	persist.Start()

	proxy := &engineProxy{eng: eng}
	if errs := runner.InitAll(eng.Context(), proxy, proxy); len(errs) > 0 {
		for _, e := range errs {
			logger.Warn("module init error", slog.String("error", e.Error()))
		}
	}

	logger.Info("axiom headless: ready ✓",
		slog.String("provider", cfg.AI.Provider),
		slog.String("workspace", cfg.Core.WorkspaceDir),
	)

	// ── OS signals ────────────────────────────────────────────────
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-sigCh:
		logger.Info("axiom: signal received", slog.String("signal", sig.String()))
	case <-eng.Context().Done():
	}

	// ── Shutdown ──────────────────────────────────────────────────
	_ = persist.SaveSync()
	_ = runner.StopAll()
	eng.Shutdown()
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}

type fsPublisher struct{ eng *engine.Engine }

func (p *fsPublisher) Subscribe(topic api.Topic, handler func(api.Event)) string {
	return p.eng.Subscribe(topic, handler)
}
func (p *fsPublisher) Publish(event api.Event) {
	_ = p.eng.Dispatch("filesystem", event.Topic, event.Payload)
}

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