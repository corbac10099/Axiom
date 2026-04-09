// Package engine est le cœur d'Axiom : il assemble tous les sous-systèmes
// (bus, security, registry) et expose une interface unifiée aux modules
// via la méthode Dispatch() — le seul point d'entrée sécurisé.
//
// Pattern : Façade + Mediator.
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
	ModulesDir    string // chemin vers le dossier /modules
	BusBufferSize int    // taille du buffer par Topic sur le bus
	LogLevel      string // "debug", "info", "warn", "error"
}

// DefaultConfig retourne une configuration sensée pour le développement.
func DefaultConfig() Config {
	return Config{
		ModulesDir:    "./modules",
		BusBufferSize: 128,
		LogLevel:      "info",
	}
}

// Engine est le moteur central d'Axiom — singleton de l'application.
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
// Séquence : logger → context → bus → security → registry → handlers
func New(cfg Config) (*Engine, error) {
	logger := buildLogger(cfg.LogLevel)
	logger.Info("axiom: initializing engine", slog.String("version", "0.1.0-alpha"))

	ctx, cancel := context.WithCancel(context.Background())

	// Bus d'événements — backbone de toute communication inter-modules
	eventBus := bus.New(ctx, cfg.BusBufferSize, logger)

	// Security Manager — gardien de la Sovereign API
	// publishFn injecté pour éviter import circulaire bus ↔ security
	secMgr := security.NewManager(eventBus.Publish, logger)

	// Registry — découverte et chargement des modules
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

	e.registerInternalHandlers()
	return e, nil
}

// ─────────────────────────────────────────────
// CYCLE DE VIE
// ─────────────────────────────────────────────

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

	// PublishSync : on attend que tous les handlers SystemReady aient fini
	// avant de retourner (garantit que les modules sont initialisés)
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

	// Notifier les modules avant d'arrêter le bus
	e.bus.PublishSync(api.Event{
		ID:        uuid.New().String(),
		Topic:     api.TopicSystemShutdown,
		Source:    "engine",
		Timestamp: time.Now().UTC(),
	})

	e.cancel() // annule ctx → le bus ferme ses goroutines proprement
	e.logger.Info("axiom: shutdown complete")
}

// Wait bloque jusqu'à annulation du contexte (Shutdown ou signal OS).
func (e *Engine) Wait() { <-e.ctx.Done() }

// Context expose le contexte global pour les composants qui en ont besoin.
func (e *Engine) Context() context.Context { return e.ctx }

// ─────────────────────────────────────────────
// DISPATCH — Sovereign API Gateway
// ─────────────────────────────────────────────

// Dispatch est le point d'entrée universel pour toute action d'un module.
//
// Pipeline :
//  1. Authorize(moduleID, topic)  → Security Manager
//  2. Si L_module >= L_topic     → Publish sur le bus
//  3. Si L_module < L_topic      → Retourne *SecurityError + publie TopicSecurityDenied
func (e *Engine) Dispatch(moduleID string, topic api.Topic, payload interface{}) error {
	// ── Contrôle de sécurité — OBLIGATOIRE, non bypassable ──────
	if err := e.security.Authorize(moduleID, topic); err != nil {
		return err // l'audit + SecurityDenied sont déjà publiés par Authorize()
	}

	// ── Publication sur le bus ────────────────────────────────────
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
// La souscription (lecture) ne nécessite PAS de permission.
// Un module L0 (Observer) peut écouter tous les Topics.
func (e *Engine) Subscribe(topic api.Topic, handler bus.HandlerFunc) string {
	return e.bus.Subscribe(topic, handler)
}

// Unsubscribe retire un abonnement par son ID.
func (e *Engine) Unsubscribe(topic api.Topic, subID string) {
	e.bus.Unsubscribe(topic, subID)
}

// ─────────────────────────────────────────────
// HANDLERS INTERNES
// ─────────────────────────────────────────────

// registerInternalHandlers câble les réponses système fondamentales du cœur.
func (e *Engine) registerInternalHandlers() {
	// Refus de sécurité → log structuré
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

	// Modules chargés → log informatif
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

	// Erreurs de modules → log d'erreur
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

	// Commandes IA → interprétation et re-dispatch sécurisé
	// Le module IA publie sur TopicAICommand → ce handler re-dispatche
	// via Dispatch() avec le vrai moduleID source → permissions vérifiées.
	e.bus.Subscribe(api.TopicAICommand, func(ev api.Event) {
		e.handleAICommand(ev)
	})

	e.logger.Debug("axiom: internal handlers registered")
}

// handleAICommand traite les commandes générées par un module IA.
// La commande est re-dispatchée via Dispatch() pour vérification de permissions.
// Un module IA ne peut donc PAS contourner la Sovereign API.
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

	// Re-dispatch avec le moduleID de l'IA comme source
	// → le Security Manager vérifie le clearance du module IA
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

// buildLogger construit un slog.Logger avec le niveau configuré.
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