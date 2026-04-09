// Package module définit le contrat qu'un plugin Axiom doit implémenter
// pour s'intégrer dans le moteur.
//
// Architecture plugin : un module Axiom peut être :
//   - Un goroutine interne (Go natif, même binaire)    → type InProcessModule
//   - Un process externe communiquant via stdio/IPC    → type ExternalModule (à venir)
//   - Un plugin Go (.so) chargé dynamiquement          → type DylibModule    (à venir)
//
// Tous ces modes implémentent l'interface Module.
// Le Registry instancie le bon type selon le champ EntryPoint du manifest.
package module

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/axiom-ide/axiom/api"
	"github.com/axiom-ide/axiom/core/security"
)

// ─────────────────────────────────────────────
// INTERFACE MODULE — Contrat public
// ─────────────────────────────────────────────

// Module est le contrat qu'implémente tout plugin Axiom.
// Le cœur ne connaît les modules qu'à travers cette interface.
type Module interface {
	// ID retourne l'identifiant unique du module (doit matcher le manifest).
	ID() string

	// Name retourne le nom lisible.
	Name() string

	// Clearance retourne le niveau d'accréditation du module.
	Clearance() security.ClearanceLevel

	// Init est appelé une seule fois par le moteur après enregistrement.
	// Le module reçoit un Dispatcher pour envoyer des commandes et
	// un Subscriber pour écouter les événements qui l'intéressent.
	Init(ctx context.Context, d Dispatcher, s Subscriber) error

	// Stop est appelé lors du shutdown propre du moteur.
	// Le module doit libérer ses ressources et terminer ses goroutines.
	Stop() error
}

// ─────────────────────────────────────────────
// INTERFACES D'INJECTION
// ─────────────────────────────────────────────

// Dispatcher est l'interface qu'un module utilise pour envoyer des actions.
// Elle est la vue "restreinte" de engine.Engine injectée dans les modules.
// Un module ne voit jamais Engine directement — uniquement cette interface.
type Dispatcher interface {
	// Dispatch publie une action sur le bus après vérification des permissions.
	Dispatch(moduleID string, topic api.Topic, payload interface{}) error
}

// Subscriber est l'interface qu'un module utilise pour écouter des événements.
// La lecture ne nécessite pas de clearance — un L0 peut tout écouter.
type Subscriber interface {
	// Subscribe enregistre un handler pour un Topic.
	// Retourne un subID utilisable pour Unsubscribe.
	Subscribe(topic api.Topic, handler func(api.Event)) string

	// Unsubscribe retire un abonnement.
	Unsubscribe(topic api.Topic, subID string)
}

// ─────────────────────────────────────────────
// BASE MODULE — Struct de base réutilisable
// ─────────────────────────────────────────────

// BaseModule est une implémentation partielle de Module.
// Les modules concrets l'embarquent via composition et n'ont qu'à implémenter
// leur logique métier dans Init() et Stop().
//
// Usage :
//
//	type MyModule struct {
//	    module.BaseModule
//	    // champs spécifiques
//	}
type BaseModule struct {
	mu         sync.Mutex
	id         string
	name       string
	clearance  security.ClearanceLevel
	dispatcher Dispatcher
	subscriber Subscriber
	subIDs     map[api.Topic][]string // topic → []subID (pour cleanup dans Stop)
	stopped    bool
	logger     *slog.Logger
}

// NewBase crée un BaseModule initialisé.
func NewBase(id, name string, clearance security.ClearanceLevel, logger *slog.Logger) BaseModule {
	return BaseModule{
		id:        id,
		name:      name,
		clearance: clearance,
		subIDs:    make(map[api.Topic][]string),
		logger:    logger,
	}
}

// ID implémente Module.
func (b *BaseModule) ID() string { return b.id }

// Name implémente Module.
func (b *BaseModule) Name() string { return b.name }

// Clearance implémente Module.
func (b *BaseModule) Clearance() security.ClearanceLevel { return b.clearance }

// Init stocke le dispatcher et le subscriber — appelé par le moteur.
// Les sous-classes DOIVENT appeler b.BaseInit(ctx, d, s) au début de leur Init().
func (b *BaseModule) BaseInit(ctx context.Context, d Dispatcher, s Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.dispatcher = d
	b.subscriber = s
}

// Stop se désabonne de tous les Topics enregistrés.
// Les sous-classes PEUVENT appeler b.BaseStop() en plus de leur nettoyage propre.
func (b *BaseModule) BaseStop() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.stopped {
		return nil
	}
	b.stopped = true
	for topic, ids := range b.subIDs {
		for _, id := range ids {
			b.subscriber.Unsubscribe(topic, id)
		}
	}
	b.logger.Info("module: stopped", slog.String("id", b.id))
	return nil
}

// Emit envoie une action via le Dispatcher.
// Raccourci type-safe pour éviter la répétition de b.id dans chaque Dispatch().
func (b *BaseModule) Emit(topic api.Topic, payload interface{}) error {
	b.mu.Lock()
	d := b.dispatcher
	b.mu.Unlock()
	if d == nil {
		return fmt.Errorf("module '%s': not initialized (dispatcher is nil)", b.id)
	}
	return d.Dispatch(b.id, topic, payload)
}

// On souscrit à un Topic et mémorise le subID pour le cleanup dans Stop().
func (b *BaseModule) On(topic api.Topic, handler func(api.Event)) {
	b.mu.Lock()
	s := b.subscriber
	b.mu.Unlock()
	if s == nil {
		b.logger.Error("module: cannot subscribe — not initialized", slog.String("id", b.id))
		return
	}
	subID := s.Subscribe(topic, handler)
	b.mu.Lock()
	b.subIDs[topic] = append(b.subIDs[topic], subID)
	b.mu.Unlock()
}

// Logger retourne le logger du module.
func (b *BaseModule) Logger() *slog.Logger { return b.logger }

// ─────────────────────────────────────────────
// RUNNER — Cycle de vie des modules in-process
// ─────────────────────────────────────────────

// Runner gère le cycle de vie d'une collection de modules in-process.
// Il est utilisé par le Registry pour démarrer/arrêter les modules Go natifs.
type Runner struct {
	mu      sync.Mutex
	modules map[string]Module
	logger  *slog.Logger
}

// NewRunner crée un Runner vide.
func NewRunner(logger *slog.Logger) *Runner {
	return &Runner{
		modules: make(map[string]Module),
		logger:  logger,
	}
}

// Register enregistre un module dans le Runner.
func (r *Runner) Register(m Module) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modules[m.ID()] = m
	r.logger.Debug("runner: module registered", slog.String("id", m.ID()))
}

// InitAll initialise tous les modules enregistrés.
// Appelé par le moteur après ScanAndLoad().
func (r *Runner) InitAll(ctx context.Context, d Dispatcher, s Subscriber) []error {
	r.mu.Lock()
	mods := make([]Module, 0, len(r.modules))
	for _, m := range r.modules {
		mods = append(mods, m)
	}
	r.mu.Unlock()

	var errs []error
	for _, m := range mods {
		if err := m.Init(ctx, d, s); err != nil {
			r.logger.Error("runner: module init failed",
				slog.String("id", m.ID()),
				slog.String("error", err.Error()),
			)
			errs = append(errs, fmt.Errorf("module '%s' init: %w", m.ID(), err))
		} else {
			r.logger.Info("runner: module initialized", slog.String("id", m.ID()))
		}
	}
	return errs
}

// StopAll arrête tous les modules proprement.
// Toutes les erreurs sont collectées — un échec n'arrête pas les autres.
func (r *Runner) StopAll() []error {
	r.mu.Lock()
	mods := make([]Module, 0, len(r.modules))
	for _, m := range r.modules {
		mods = append(mods, m)
	}
	r.mu.Unlock()

	var errs []error
	for _, m := range mods {
		if err := m.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("module '%s' stop: %w", m.ID(), err))
		}
	}
	return errs
}

// Get retourne un module par son ID.
func (r *Runner) Get(id string) (Module, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.modules[id]
	return m, ok
}