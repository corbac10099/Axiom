//go:build wails

// Package wails — bridge bidirectionnel Go ↔ JS pour Wails v3.
//
// Go → JS : app.EmitEvent("axiom:event", payload)
// JS → Go : app.OnEvent("axiom:input", handler)
//
// Côté frontend (JavaScript) :
//
//	// Écouter les événements Go→JS
//	window.wails.Events.On("axiom:event", (event) => {
//	    console.log(event.topic, event.payload)
//	})
//
//	// Envoyer un événement JS→Go
//	window.wails.Events.Emit("axiom:input", {
//	    window_id: "main", event_type: "keydown", data: { key: "s", ctrl: true }
//	})
package wails

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/axiom-ide/axiom/api"
	"github.com/axiom-ide/axiom/core/bus"
	"github.com/axiom-ide/axiom/pkg/uid"
)

// RegisterBidirectional câble les deux canaux de communication Go ↔ JS.
func (a *Adapter) RegisterBidirectional(eventBus *bus.EventBus) {
	// ── JS → Go ──────────────────────────────────────────────────
	a.app.OnEvent("axiom:input", func(e *application.CustomEvent) {
		raw, err := json.Marshal(e.Data)
		if err != nil {
			a.logger.Warn("wails: marshal JS event failed", slog.String("error", err.Error()))
			return
		}
		var p api.PayloadUIUserInput
		if err := json.Unmarshal(raw, &p); err != nil {
			a.logger.Warn("wails: unmarshal JS event failed", slog.String("error", err.Error()))
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

	// ── Go → JS ──────────────────────────────────────────────────
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
		t := topic
		eventBus.Subscribe(t, func(ev api.Event) {
			a.app.EmitEvent("axiom:event", map[string]any{
				"event_id":  ev.ID,
				"topic":     string(ev.Topic),
				"source":    ev.Source,
				"payload":   ev.Payload,
				"timestamp": ev.Timestamp.UnixMilli(),
			})
			a.logger.Debug("wails: Go→JS event emitted", slog.String("topic", string(ev.Topic)))
		})
	}

	a.logger.Info("wails: bidirectional bridge active",
		slog.Int("subscribed_topics", len(uiTopics)),
	)
}