//go:build wails

package wails

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Adapter implémente orchestrator.NativeWindowAdapter via Wails v2.
//
// Go → JS : runtime.EventsEmit(ctx, "axiom:event", payload)
// JS → Go : window.runtime.EventsOn("axiom:input", handler)
type Adapter struct {
	mu      sync.RWMutex
	windows map[string]bool
	ctx     context.Context
	logger  *slog.Logger
}

// NewAdapter crée un Adapter Wails v2.
// ctx doit être le contexte fourni par Wails dans OnStartup.
func NewAdapter(ctx context.Context, logger *slog.Logger) *Adapter {
	return &Adapter{
		windows: make(map[string]bool),
		ctx:     ctx,
		logger:  logger,
	}
}

// emit envoie un événement au frontend via le runtime Wails v2.
func (a *Adapter) emit(eventName string, data map[string]any) {
	if a.ctx == nil {
		a.logger.Warn("wails: emit called before context is ready", slog.String("event", eventName))
		return
	}
	runtime.EventsEmit(a.ctx, eventName, data)
}

// ─────────────────────────────────────────────
// NativeWindowAdapter implementation
// ─────────────────────────────────────────────

func (a *Adapter) CreateWindow(id, title string, width, height int) error {
	a.mu.Lock()
	a.windows[id] = true
	a.mu.Unlock()
	a.emit("axiom:window:created", map[string]any{
		"id": id, "title": title, "width": width, "height": height,
	})
	a.logger.Info("wails: window created", slog.String("id", id))
	return nil
}

func (a *Adapter) ShowWindow(id string) error {
	a.emit("axiom:window:show", map[string]any{"id": id})
	return nil
}

func (a *Adapter) HideWindow(id string) error {
	a.emit("axiom:window:hide", map[string]any{"id": id})
	return nil
}

func (a *Adapter) DestroyWindow(id string) error {
	a.mu.Lock()
	delete(a.windows, id)
	a.mu.Unlock()
	a.emit("axiom:window:destroy", map[string]any{"id": id})
	return nil
}

func (a *Adapter) SetWindowContent(id, html string) error {
	a.emit("axiom:window:content", map[string]any{"id": id, "html": html})
	return nil
}

func (a *Adapter) SetWindowTitle(id, title string) error {
	a.emit("axiom:window:title", map[string]any{"id": id, "title": title})
	return nil
}

// marshalPayload sérialise un payload arbitraire en type JSON-safe.
func marshalPayload(payload any) any {
	if payload == nil {
		return nil
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf("%v", payload)
	}
	var m any
	if err := json.Unmarshal(b, &m); err != nil {
		return string(b)
	}
	return m
}