// Package bus implémente le cœur de communication asynchrone d'Axiom.
// C'est la "colonne vertébrale" qui relie tous les modules entre eux
// sans qu'ils se connaissent directement — découplage total.
//
// Design : chaque Topic possède un channel Go bufferisé.
// Les handlers sont appelés dans des goroutines dédiées.
// Un EventBus peut être arrêté proprement via un context.
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

// subscription représente un abonné unique à un Topic.
type subscription struct {
	id      string
	handler HandlerFunc
}

// topicRouter gère la liste des abonnés et le channel d'un Topic donné.
type topicRouter struct {
	mu           sync.RWMutex
	subscribers  []subscription
	eventChannel chan api.Event
}

// EventBus est le bus d'événements central d'Axiom.
// Thread-safe, asynchrone, et arrêtable proprement.
type EventBus struct {
	mu         sync.RWMutex
	routers    map[api.Topic]*topicRouter
	bufferSize int
	wg         sync.WaitGroup
	logger     *slog.Logger
}

// New crée un EventBus prêt à l'emploi.
// ctx : contexte global — annuler ce contexte arrête proprement le bus.
// bufferSize : taille du buffer par channel Topic (recommandé : 64-256).
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
// Retourne un subscriptionID utilisable pour se désabonner.
func (eb *EventBus) Subscribe(topic api.Topic, handler HandlerFunc) string {
	eb.mu.Lock()
	router, exists := eb.routers[topic]
	if !exists {
		router = eb.newRouter(topic)
		eb.routers[topic] = router
	}
	eb.mu.Unlock()

	subID := uuid.New().String()
	sub := subscription{id: subID, handler: handler}

	router.mu.Lock()
	router.subscribers = append(router.subscribers, sub)
	router.mu.Unlock()

	eb.logger.Debug("bus: subscribed",
		slog.String("topic", string(topic)),
		slog.String("sub_id", subID),
	)
	return subID
}

// Unsubscribe retire un handler du bus via son subscriptionID.
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

// Publish envoie un événement sur le bus de manière NON-BLOQUANTE.
// Si le channel est plein, l'événement est droppé et une alerte loguée.
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
// Usage : tests ou séquences d'initialisation critiques.
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
func (eb *EventBus) shutdown() {
	eb.logger.Info("bus: shutting down...")
	eb.mu.Lock()
	defer eb.mu.Unlock()
	for _, router := range eb.routers {
		close(router.eventChannel)
	}
	eb.wg.Wait()
	eb.logger.Info("bus: shutdown complete")
}

// safeCall exécute un handler en récupérant les panics potentiels.
// Un handler défaillant ne doit JAMAIS tuer le bus.
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