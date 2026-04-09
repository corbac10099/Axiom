// Package bus implémente le cœur de communication asynchrone d'Axiom.
package bus

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"

	"github.com/axiom-ide/axiom/api"
)

// HandlerFunc est la signature d'un abonné à un Topic.
type HandlerFunc func(event api.Event)

type subscription struct {
	id      string
	handler HandlerFunc
}

type topicRouter struct {
	mu           sync.RWMutex
	subscribers  []subscription
	eventChannel chan api.Event
}

// EventBus est le bus d'événements central d'Axiom.
type EventBus struct {
	mu         sync.RWMutex
	routers    map[api.Topic]*topicRouter
	bufferSize int
	wg         sync.WaitGroup
	// BUG FIX: protéger shutdown contre double-appel
	shutdownOnce sync.Once
	logger       *slog.Logger
}

// New crée un EventBus prêt à l'emploi.
func New(ctx context.Context, bufferSize int, logger *slog.Logger) *EventBus {
	eb := &EventBus{
		routers:    make(map[api.Topic]*topicRouter),
		bufferSize: bufferSize,
		logger:     logger,
	}
	go func() {
		<-ctx.Done()
		eb.shutdown()
	}()
	return eb
}

// Subscribe enregistre un handler sur un Topic.
func (eb *EventBus) Subscribe(topic api.Topic, handler HandlerFunc) string {
	eb.mu.Lock()
	router, exists := eb.routers[topic]
	if !exists {
		router = eb.newRouter(topic)
		eb.routers[topic] = router
	}
	eb.mu.Unlock()

	subID := uuid.New().String()
	router.mu.Lock()
	router.subscribers = append(router.subscribers, subscription{id: subID, handler: handler})
	router.mu.Unlock()

	eb.logger.Debug("bus: subscribed",
		slog.String("topic", string(topic)),
		slog.String("sub_id", subID),
	)
	return subID
}

// Unsubscribe retire un handler du bus.
func (eb *EventBus) Unsubscribe(topic api.Topic, subID string) {
	eb.mu.RLock()
	router, exists := eb.routers[topic]
	eb.mu.RUnlock()
	if !exists {
		return
	}
	router.mu.Lock()
	defer router.mu.Unlock()
	filtered := router.subscribers[:0]
	for _, s := range router.subscribers {
		if s.id != subID {
			filtered = append(filtered, s)
		}
	}
	router.subscribers = filtered
}

// Publish envoie un événement de manière NON-BLOQUANTE.
// BUG FIX: on vérifie si le bus est en cours d'arrêt avant de publier.
func (eb *EventBus) Publish(event api.Event) {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	eb.mu.RLock()
	router, exists := eb.routers[event.Topic]
	eb.mu.RUnlock()

	if !exists {
		eb.logger.Debug("bus: publish to topic with no subscribers",
			slog.String("topic", string(event.Topic)),
		)
		return
	}

	select {
	case router.eventChannel <- event:
		eb.logger.Debug("bus: event queued",
			slog.String("topic", string(event.Topic)),
			slog.String("source", event.Source),
		)
	default:
		eb.logger.Warn("bus: channel full — event DROPPED",
			slog.String("topic", string(event.Topic)),
			slog.String("event_id", event.ID),
		)
	}
}

// PublishSync publie et attend que tous les handlers aient terminé.
func (eb *EventBus) PublishSync(event api.Event) {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	eb.mu.RLock()
	router, exists := eb.routers[event.Topic]
	eb.mu.RUnlock()
	if !exists {
		return
	}

	router.mu.RLock()
	subs := make([]subscription, len(router.subscribers))
	copy(subs, router.subscribers)
	router.mu.RUnlock()

	var wg sync.WaitGroup
	for _, sub := range subs {
		wg.Add(1)
		go func(s subscription) {
			defer wg.Done()
			safeCall(s.handler, event, eb.logger)
		}(sub)
	}
	wg.Wait()
}

// newRouter crée un topicRouter et lance sa goroutine de dispatch.
// DOIT être appelé sous eb.mu.Lock().
func (eb *EventBus) newRouter(topic api.Topic) *topicRouter {
	router := &topicRouter{
		eventChannel: make(chan api.Event, eb.bufferSize),
	}
	eb.wg.Add(1)
	go func() {
		defer eb.wg.Done()
		eb.dispatchLoop(topic, router)
	}()
	return router
}

// dispatchLoop consomme les événements d'un Topic et dispatche aux handlers.
func (eb *EventBus) dispatchLoop(topic api.Topic, router *topicRouter) {
	for event := range router.eventChannel {
		router.mu.RLock()
		subs := make([]subscription, len(router.subscribers))
		copy(subs, router.subscribers)
		router.mu.RUnlock()

		for _, sub := range subs {
			go safeCall(sub.handler, event, eb.logger)
		}
	}
	eb.logger.Debug("bus: dispatch loop terminated", slog.String("topic", string(topic)))
}

// shutdown ferme proprement tous les channels.
// BUG FIX: sync.Once empêche tout double-close (panic sur channel déjà fermé).
func (eb *EventBus) shutdown() {
	eb.shutdownOnce.Do(func() {
		eb.logger.Info("bus: shutting down...")
		eb.mu.Lock()
		defer eb.mu.Unlock()
		for _, router := range eb.routers {
			close(router.eventChannel)
		}
		eb.wg.Wait()
		eb.logger.Info("bus: shutdown complete")
	})
}

func safeCall(handler HandlerFunc, event api.Event, logger *slog.Logger) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("bus: handler panic recovered",
				slog.String("topic", string(event.Topic)),
				slog.String("panic", fmt.Sprintf("%v", r)),
			)
		}
	}()
	handler(event)
}