// Package engine est le cœur d'Axiom : il assemble tous les sous-systèmes.
package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/axiom-ide/axiom/api"
	"github.com/axiom-ide/axiom/core/bus"
	"github.com/axiom-ide/axiom/core/registry"
	"github.com/axiom-ide/axiom/core/security"
)

// Config regroupe les paramètres de démarrage d'Axiom.
type Config struct {
	ModulesDir    string
	BusBufferSize int
	LogLevel      string
}

// DefaultConfig retourne une configuration sensée pour le développement.
func DefaultConfig() Config {
	return Config{
		ModulesDir:    "./modules",
		BusBufferSize: 128,
		LogLevel:      "info",
	}
}

// Engine est le moteur central d'Axiom.
type Engine struct {
	cfg      Config
	bus      *bus.EventBus
	security *security.Manager
	registry *registry.Registry
	logger   *slog.Logger
	ctx      context.Context
	cancel   context.CancelFunc
}

// New crée et initialise le moteur Axiom.
func New(cfg Config) (*Engine, error) {
	logger := buildLogger(cfg.LogLevel)
	logger.Info("axiom: initializing engine", slog.String("version", "0.2.0-alpha"))

	ctx, cancel := context.WithCancel(context.Background())
	eventBus := bus.New(ctx, cfg.BusBufferSize, logger)
	secMgr := security.NewManager(eventBus.Publish, logger)
	reg := registry.NewRegistry(cfg.ModulesDir, secMgr, eventBus.Publish, logger)

	e := &Engine{
		cfg:      cfg,
		bus:      eventBus,
		security: secMgr,
		registry: reg,
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
	}

	// BUG FIX: enregistrer les composants internes de confiance dans le Security Manager
	// avant tout chargement de module externe.
	// "filesystem" publie sur "file.opened" (L3) et sur "system.*" (via le bus interne).
	// "engine" publie sur "system.*" (L3) et gère le re-dispatch IA.
	// Ces enregistrements sont faits AVANT ScanAndLoad() pour que les publications
	// de ces composants ne soient pas rejetées par le Security Manager.
	if err := secMgr.RegisterModule("filesystem", security.L3); err != nil {
		cancel()
		return nil, fmt.Errorf("engine: cannot register filesystem module: %w", err)
	}
	if err := secMgr.RegisterModule("engine", security.L3); err != nil {
		cancel()
		return nil, fmt.Errorf("engine: cannot register engine module: %w", err)
	}
	if err := secMgr.RegisterModule("registry", security.L3); err != nil {
		cancel()
		return nil, fmt.Errorf("engine: cannot register registry module: %w", err)
	}
	if err := secMgr.RegisterModule("security.manager", security.L3); err != nil {
		cancel()
		return nil, fmt.Errorf("engine: cannot register security.manager: %w", err)
	}

	e.registerInternalHandlers()
	return e, nil
}

// Start lance le moteur : scan des modules + émission SystemReady.
func (e *Engine) Start() error {
	e.logger.Info("axiom: starting module discovery...")
	if err := e.registry.ScanAndLoad(); err != nil {
		return fmt.Errorf("axiom: module scan failed: %w", err)
	}
	e.logger.Info("axiom: all modules processed",
		slog.Int("total", e.registry.Count()),
		slog.Int("active", len(e.registry.ListActive())),
	)
	e.bus.PublishSync(api.Event{
		ID:        uuid.New().String(),
		Topic:     api.TopicSystemReady,
		Source:    "engine",
		Timestamp: time.Now().UTC(),
	})
	e.logger.Info("axiom: engine is ready ✓")
	return nil
}

// Shutdown arrête proprement le moteur.
func (e *Engine) Shutdown() {
	e.logger.Info("axiom: shutdown initiated...")
	e.bus.PublishSync(api.Event{
		ID:        uuid.New().String(),
		Topic:     api.TopicSystemShutdown,
		Source:    "engine",
		Timestamp: time.Now().UTC(),
	})
	e.cancel()
	e.logger.Info("axiom: shutdown complete")
}

// Wait bloque jusqu'à annulation du contexte.
func (e *Engine) Wait() { <-e.ctx.Done() }

// Context expose le contexte global.
func (e *Engine) Context() context.Context { return e.ctx }

// Bus expose le bus pour les composants internes (ex: Orchestrator).
func (e *Engine) Bus() *bus.EventBus { return e.bus }

// Dispatch est le point d'entrée universel pour toute action d'un module.
func (e *Engine) Dispatch(moduleID string, topic api.Topic, payload interface{}) error {
	if err := e.security.Authorize(moduleID, topic); err != nil {
		return err
	}
	e.bus.Publish(api.Event{
		ID:        uuid.New().String(),
		Topic:     topic,
		Source:    moduleID,
		Payload:   payload,
		Timestamp: time.Now().UTC(),
	})
	return nil
}

// Subscribe enregistre un handler sur un Topic.
func (e *Engine) Subscribe(topic api.Topic, handler bus.HandlerFunc) string {
	return e.bus.Subscribe(topic, handler)
}

// Unsubscribe retire un abonnement par son ID.
func (e *Engine) Unsubscribe(topic api.Topic, subID string) {
	e.bus.Unsubscribe(topic, subID)
}

// registerInternalHandlers câble les réponses système fondamentales.
func (e *Engine) registerInternalHandlers() {
	e.bus.Subscribe(api.TopicSecurityDenied, func(ev api.Event) {
		d, ok := ev.Payload.(api.PayloadSecurityDenied)
		if !ok {
			return
		}
		e.logger.Warn("🔒 SECURITY DENIED",
			slog.String("module", d.ModuleID),
			slog.String("topic", string(d.AttemptedTopic)),
			slog.String("reason", d.Reason),
			slog.Int("required", d.RequiredLevel),
			slog.Int("actual", d.ActualLevel),
		)
	})

	e.bus.Subscribe(api.TopicModuleLoaded, func(ev api.Event) {
		s, ok := ev.Payload.(api.PayloadModuleStatus)
		if !ok {
			return
		}
		e.logger.Info("📦 module loaded",
			slog.String("id", s.ModuleID),
			slog.String("version", s.Version),
		)
	})

	e.bus.Subscribe(api.TopicModuleError, func(ev api.Event) {
		s, ok := ev.Payload.(api.PayloadModuleStatus)
		if !ok {
			return
		}
		e.logger.Error("❌ module error",
			slog.String("id", s.ModuleID),
			slog.String("error", s.Error),
		)
	})

	e.bus.Subscribe(api.TopicAICommand, func(ev api.Event) {
		e.handleAICommand(ev)
	})

	e.logger.Debug("axiom: internal handlers registered")
}

// handleAICommand traite les commandes générées par un module IA.
func (e *Engine) handleAICommand(ev api.Event) {
	cmd, ok := ev.Payload.(api.PayloadAICommand)
	if !ok {
		e.logger.Warn("ai: invalid AICommand payload", slog.String("source", ev.Source))
		return
	}
	e.logger.Info("🤖 AI command",
		slog.String("source", ev.Source),
		slog.String("raw", cmd.RawCommand),
		slog.String("→ topic", string(cmd.ParsedTopic)),
	)
	if err := e.Dispatch(ev.Source, cmd.ParsedTopic, cmd.ParsedPayload); err != nil {
		e.bus.Publish(api.Event{
			Topic:         api.TopicAIResponse,
			Source:        "engine",
			CorrelationID: ev.CorrelationID,
			Payload: map[string]interface{}{
				"success": false,
				"error":   err.Error(),
				"command": cmd.RawCommand,
			},
			Timestamp: time.Now().UTC(),
		})
		return
	}
	e.bus.Publish(api.Event{
		Topic:         api.TopicAIResponse,
		Source:        "engine",
		CorrelationID: ev.CorrelationID,
		Payload: map[string]interface{}{
			"success": true,
			"command": cmd.RawCommand,
			"topic":   string(cmd.ParsedTopic),
		},
		Timestamp: time.Now().UTC(),
	})
}

func buildLogger(level string) *slog.Logger {
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