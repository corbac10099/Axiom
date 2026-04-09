//go:build wails

// Package wails — extension de l'adaptateur avec notifications bidirectionnelles.
//
// Ce fichier complète adapter.go avec :
//   - Go → JS  : Adapter.EmitToFrontend(windowID, eventType, payload)
//   - JS → Go  : registration d'un handler Wails + publication sur le bus Axiom
//
// Côté JS, le frontend doit appeler :
//
//	window.runtime.EventsOn("axiom:event", handler)
//	window.runtime.EventsEmit("axiom:input", { window_id, event_type, data })
//
// Build tag : //go:build wails
package wails

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/axiom-ide/axiom/api"
	"github.com/axiom-ide/axiom/core/bus"
	"github.com/axiom-ide/axiom/pkg/uid"
)

const (
	// wailsEventToFrontend est le nom d'événement Wails pour Go→JS.
	wailsEventToFrontend = "axiom:event"
	// wailsEventFromFrontend est le nom d'événement Wails pour JS→Go.
	wailsEventFromFrontend = "axiom:input"
)

// frontendEvent est la structure JSON envoyée vers le frontend.
type frontendEvent struct {
	EventID   string      `json:"event_id"`
	Topic     string      `json:"topic"`
	Source    string      `json:"source"`
	Payload   interface{} `json:"payload"`
	Timestamp int64       `json:"timestamp"` // Unix ms
}

// RegisterBidirectional câble les deux canaux de communication.
//
// À appeler dans le OnStartup de Wails, APRÈS NewAdapter() :
//
//	adapter := wails.NewAdapter(ctx, logger)
//	adapter.RegisterBidirectional(eng.Bus())
func (a *Adapter) RegisterBidirectional(eventBus *bus.EventBus) {
	// ── Canal JS → Go ────────────────────────────────────────
	// Wails appelle notre handler quand le JS émet "axiom:input".
	runtime.EventsOn(a.ctx, wailsEventFromFrontend, func(optionalData ...interface{}) {
		if len(optionalData) == 0 {
			return
		}
		raw, err := json.Marshal(optionalData[0])
		if err != nil {
			a.logger.Warn("wails: cannot marshal JS event", slog.String("error", err.Error()))
			return
		}
		var p api.PayloadUIUserInput
		if err := json.Unmarshal(raw, &p); err != nil {
			a.logger.Warn("wails: cannot parse JS event payload", slog.String("error", err.Error()))
			return
		}

		eventBus.Publish(api.Event{
			ID:        uid.New(),
			Topic:     api.TopicUIUserInput,
			Source:    "wails-frontend",
			Payload:   p,
			Timestamp: time.Now().UTC(),
		})

		a.logger.Debug("wails: JS→Go event received",
			slog.String("window", p.WindowID),
			slog.String("type", p.EventType),
		)
	})

	// ── Canal Go → JS ────────────────────────────────────────
	// On s'abonne aux topics qui concernent l'UI et on les relaie au frontend.
	uiTopics := []api.Topic{
		api.TopicUISetTheme,
		api.TopicUIOpenPanel,
		api.TopicUIClosePanel,
		api.TopicEditorTabChanged,
		api.TopicWorkspaceRestored,
		api.TopicFileOpened,
		api.TopicAIResponse,
		api.TopicModuleLoaded,
	}
	for _, topic := range uiTopics {
		t := topic // capture pour la closure
		eventBus.Subscribe(t, func(ev api.Event) {
			a.EmitToFrontend(ev)
		})
	}

	a.logger.Info("wails: bidirectional bridge active",
		slog.Int("subscribed_topics", len(uiTopics)),
	)
}

// EmitToFrontend envoie un événement Axiom vers le WebView JavaScript.
// Thread-safe.
func (a *Adapter) EmitToFrontend(ev api.Event) {
	fe := frontendEvent{
		EventID:   ev.ID,
		Topic:     string(ev.Topic),
		Source:    ev.Source,
		Payload:   ev.Payload,
		Timestamp: ev.Timestamp.UnixMilli(),
	}
	// runtime.EventsEmit est thread-safe dans Wails v2.
	runtime.EventsEmit(a.ctx, wailsEventToFrontend, fe)

	a.logger.Debug("wails: Go→JS event emitted",
		slog.String("topic", string(ev.Topic)),
		slog.String("source", ev.Source),
	)
}

// EmitRaw envoie un événement brut avec un topic et payload arbitraires.
// Utile pour les notifications UI ponctuelles (ex: "file saved", "build done").
func (a *Adapter) EmitRaw(topic string, payload interface{}) {
	fe := frontendEvent{
		EventID:   uid.New(),
		Topic:     topic,
		Source:    "go-backend",
		Payload:   payload,
		Timestamp: time.Now().UnixMilli(),
	}
	runtime.EventsEmit(a.ctx, wailsEventToFrontend, fe)
}

// ─────────────────────────────────────────────
// SNIPPET JS À INTÉGRER DANS LE FRONTEND
// ─────────────────────────────────────────────
//
// Collez ce code dans votre frontend Wails (TypeScript / React) :
//
//  import { EventsOn, EventsEmit } from "@wailsapp/runtime"
//
//  // Écouter les événements Go → JS
//  EventsOn("axiom:event", (event) => {
//    console.log("Axiom event:", event.topic, event.payload)
//    // Dispatcher dans votre store React/Vue
//    axiomStore.dispatch(event)
//  })
//
//  // Envoyer un événement JS → Go
//  function sendToAxiom(windowID: string, eventType: string, data: unknown) {
//    EventsEmit("axiom:input", { window_id: windowID, event_type: eventType, data })
//  }
//
//  // Exemple : l'utilisateur appuie sur Ctrl+S
//  document.addEventListener("keydown", (e) => {
//    if (e.ctrlKey && e.key === "s") {
//      sendToAxiom("main", "keydown", { key: "s", ctrl: true })
//    }
//  })