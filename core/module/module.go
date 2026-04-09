// Package module définit le contrat qu'un plugin Axiom doit implémenter.
package module

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/axiom-ide/axiom/api"
	"github.com/axiom-ide/axiom/core/security"
)

// Module est le contrat qu'implémente tout plugin Axiom.
type Module interface {
	ID() string
	Name() string
	Clearance() security.ClearanceLevel
	Init(ctx context.Context, d Dispatcher, s Subscriber) error
	Stop() error
}

// Dispatcher est l'interface qu'un module utilise pour envoyer des actions.
type Dispatcher interface {
	Dispatch(moduleID string, topic api.Topic, payload interface{}) error
}

// Subscriber est l'interface qu'un module utilise pour écouter des événements.
type Subscriber interface {
	Subscribe(topic api.Topic, handler func(api.Event)) string
	Unsubscribe(topic api.Topic, subID string)
}

// BaseModule est une implémentation partielle de Module.
type BaseModule struct {
	mu         sync.Mutex
	id         string
	name       string
	clearance  security.ClearanceLevel
	dispatcher Dispatcher
	subscriber Subscriber
	subIDs     map[api.Topic][]string
	stopped    bool
	logger     *slog.Logger
}

func NewBase(id, name string, clearance security.ClearanceLevel, logger *slog.Logger) BaseModule {
	return BaseModule{
		id:        id,
		name:      name,
		clearance: clearance,
		subIDs:    make(map[api.Topic][]string),
		logger:    logger,
	}
}

func (b *BaseModule) ID() string                       { return b.id }
func (b *BaseModule) Name() string                     { return b.name }
func (b *BaseModule) Clearance() security.ClearanceLevel { return b.clearance }

func (b *BaseModule) BaseInit(_ context.Context, d Dispatcher, s Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.dispatcher = d
	b.subscriber = s
}

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

func (b *BaseModule) Emit(topic api.Topic, payload interface{}) error {
	b.mu.Lock()
	d := b.dispatcher
	b.mu.Unlock()
	if d == nil {
		return fmt.Errorf("module '%s': not initialized (dispatcher is nil)", b.id)
	}
	return d.Dispatch(b.id, topic, payload)
}

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

func (b *BaseModule) Logger() *slog.Logger { return b.logger }

// Runner gère le cycle de vie d'une collection de modules in-process.
type Runner struct {
	mu      sync.Mutex
	modules map[string]Module
	logger  *slog.Logger
}

func NewRunner(logger *slog.Logger) *Runner {
	return &Runner{modules: make(map[string]Module), logger: logger}
}

func (r *Runner) Register(m Module) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modules[m.ID()] = m
	r.logger.Debug("runner: module registered", slog.String("id", m.ID()))
}

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

func (r *Runner) Get(id string) (Module, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.modules[id]
	return m, ok
}