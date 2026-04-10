//go:build wails

// Package wails — bridge bidirectionnel Go ↔ JS pour Wails v2.
//
// Go → JS : runtime.EventsEmit(ctx, "axiom:event", payload)
//           Côté JS : window.runtime.EventsOn("axiom:event", callback)
//
// JS → Go : window.runtime.EventsEmit("axiom:input", payload)
//           Côté Go : runtime.EventsOn(ctx, "axiom:input", handler)
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

// RegisterBidirectional câble les deux canaux Go ↔ JS.
func (a *Adapter) RegisterBidirectional(eventBus *bus.EventBus) {
	// ── JS → Go ──────────────────────────────────────────────────
	// Le frontend envoie : window.runtime.EventsEmit("axiom:input", {...})
	runtime.EventsOn(a.ctx, "axiom:input", func(optionalData ...interface{}) {
		if len(optionalData) == 0 {
			return
		}
		raw, err := json.Marshal(optionalData[0])
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
	// Les topics UI sont pushés vers le frontend via EventsEmit.
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
			runtime.EventsEmit(a.ctx, "axiom:event", map[string]any{
				"event_id":  ev.ID,
				"topic":     string(ev.Topic),
				"source":    ev.Source,
				"payload":   marshalPayload(ev.Payload),
				"timestamp": ev.Timestamp.UnixMilli(),
			})
			a.logger.Debug("wails: Go→JS event emitted",
				slog.String("topic", string(ev.Topic)),
			)
		})
	}

	a.logger.Info("wails: bidirectional bridge active",
		slog.Int("subscribed_topics", len(uiTopics)),
	)
}