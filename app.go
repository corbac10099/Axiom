package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/axiom-ide/axiom/api"
	axiomconfig "github.com/axiom-ide/axiom/core/config"
	"github.com/axiom-ide/axiom/core/engine"
	"github.com/axiom-ide/axiom/core/filesystem"
	"github.com/axiom-ide/axiom/core/module"
	"github.com/axiom-ide/axiom/core/orchestrator"
	"github.com/axiom-ide/axiom/core/tabs"
	"github.com/axiom-ide/axiom/core/workspace"
	aiassistant "github.com/axiom-ide/axiom/modules/ai-assistant"
	"github.com/axiom-ide/axiom/modules"
)

// App is the main struct bound to the Wails frontend.
// Every public method becomes callable from JS via window.go.main.App.MethodName().
type App struct {
	ctx     context.Context
	eng     *engine.Engine
	tabMgr  *tabs.Manager
	persist *workspace.Persistence
	runner  *module.Runner
	fsHdlr  *filesystem.Handler
	cfg     axiomconfig.Config
	logger  *slog.Logger
}

// NewApp creates the App instance (called before OnStartup).
func NewApp() *App {
	return &App{
		logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
	}
}

// ─── Lifecycle ────────────────────────────────────────────────────

// OnStartup is called by Wails after the window is created.
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx

	if isBindingsGeneration() {
		a.logger.Info("axiom: bindings generation mode, skipping engine init")
		return
	}

	// ── Config ──────────────────────────────────────────────────────
	cfg, warnings := axiomconfig.Load("")
	for _, w := range warnings {
		a.logger.Warn("config warning", slog.String("msg", w))
	}
	a.cfg = cfg
	a.logger = newLogger(cfg.Core.LogLevel)

	// ── Engine ──────────────────────────────────────────────────────
	eng, err := engine.New(engine.Config{
		ModulesDir:    cfg.Core.ModulesDir,
		LogLevel:      cfg.Core.LogLevel,
		BusBufferSize: cfg.Core.BusBufferSize,
	})
	if err != nil {
		a.logger.Error("engine init failed", slog.String("error", err.Error()))
		return
	}
	a.eng = eng

	// ── Filesystem ──────────────────────────────────────────────────
	fsPub := &appFSPublisher{eng: eng}
	fsHdlr, err := filesystem.NewHandler(filesystem.Config{
		WorkspaceDir:   cfg.Core.WorkspaceDir,
		MaxFileSizeMB:  cfg.FileSystem.MaxFileSizeMB,
		IgnorePatterns: cfg.FileSystem.IgnorePatterns,
		BackupOnWrite:  cfg.FileSystem.BackupOnWrite,
	}, fsPub, a.logger)
	if err != nil {
		a.logger.Warn("filesystem init failed (non-fatal)", slog.String("error", err.Error()))
	}
	a.fsHdlr = fsHdlr

	// ── Orchestrator ────────────────────────────────────────────────
	_ = orchestrator.NewOrchestrator(nil, eng.Bus(), a.logger)

	// ── Tab Manager ─────────────────────────────────────────────────
	a.tabMgr = tabs.NewManager(eng.Bus(), a.logger)

	// ── Module Runner ───────────────────────────────────────────────
	a.runner = module.NewRunner(a.logger)
	a.runner.Register(aiassistant.New(aiassistant.Config{
		Provider:    cfg.AI.Provider,
		BaseURL:     cfg.AI.BaseURL,
		ModelID:     cfg.AI.ModelID,
		APIKey:      cfg.AI.APIKey,
		MaxTokens:   cfg.AI.MaxTokens,
		Temperature: cfg.AI.Temperature,
		TimeoutSecs: cfg.AI.TimeoutSecs,
	}, a.logger))
	modules.AutoRegister(a.runner, a.logger)

	// ── Workspace Persistence ───────────────────────────────────────
	a.persist = workspace.NewPersistence(
		eng.Bus(), a.tabMgr, nil,
		cfg.Core.WorkspaceDir, a.logger,
	)

	// ── Start ───────────────────────────────────────────────────────
	if err := eng.Start(); err != nil {
		a.logger.Error("engine start failed", slog.String("error", err.Error()))
		return
	}

	a.tabMgr.Start()
	a.persist.Start()

	proxy := &appEngineProxy{eng: eng}
	if errs := a.runner.InitAll(ctx, proxy, proxy); len(errs) > 0 {
		for _, e := range errs {
			a.logger.Warn("module init error", slog.String("error", e.Error()))
		}
	}

	// ── Go → JS bridge ──────────────────────────────────────────────
	a.setupGoToJSBridge()

	// ── JS → Go bridge ──────────────────────────────────────────────
	runtime.EventsOn(ctx, "axiom:input", func(data ...interface{}) {
		a.handleJSEvent(data...)
	})

	a.logger.Info("axiom: ready ✓",
		slog.String("provider", cfg.AI.Provider),
		slog.String("workspace", cfg.Core.WorkspaceDir),
	)
}

// OnShutdown is called by Wails before the window closes.
func (a *App) OnShutdown(_ context.Context) {
	if a.persist != nil {
		_ = a.persist.SaveSync()
	}
	if a.runner != nil {
		_ = a.runner.StopAll()
	}
	if a.eng != nil {
		a.eng.Shutdown()
	}
}

// ─── Go → JS bridge ──────────────────────────────────────────────

// setupGoToJSBridge subscribes to internal topics and pushes them to the frontend.
func (a *App) setupGoToJSBridge() {
	// Topics natifs existants
	nativeTopics := []api.Topic{
		api.TopicUISetTheme,
		api.TopicUIOpenPanel,
		api.TopicUIClosePanel,
		api.TopicEditorTabChanged,
		api.TopicWorkspaceRestored,
		api.TopicFileOpened,
		api.TopicAIResponse,
		api.TopicModuleLoaded,
		api.TopicSecurityDenied,
	}

	// Nouveaux topics Module System
	moduleUITopics := []api.Topic{
		api.TopicUIModuleRegister,
		api.TopicUISlotInject,
		api.TopicUISlotRemove,
		api.TopicUIAppBranding,
		api.TopicUIIconBadge,
		api.TopicUIViewSwitch,
	}

	allTopics := append(nativeTopics, moduleUITopics...)

	for _, topic := range allTopics {
		t := topic
		a.eng.Subscribe(t, func(ev api.Event) {
			if a.ctx == nil {
				return
			}
			payloadJSON, _ := json.Marshal(ev.Payload)
			var payloadMap interface{}
			_ = json.Unmarshal(payloadJSON, &payloadMap)

			runtime.EventsEmit(a.ctx, "axiom:event", map[string]interface{}{
				"event_id":  ev.ID,
				"topic":     string(ev.Topic),
				"source":    ev.Source,
				"payload":   payloadMap,
				"timestamp": ev.Timestamp.UnixMilli(),
			})
		})
	}

	a.logger.Info("axiom: Go→JS bridge active",
		slog.Int("topics", len(allTopics)),
	)
}

// handleJSEvent processes events emitted by the JS frontend via axiom:input.
func (a *App) handleJSEvent(data ...interface{}) {
	if len(data) == 0 || a.eng == nil {
		return
	}
	raw, err := json.Marshal(data[0])
	if err != nil {
		a.logger.Warn("axiom: cannot marshal JS event", slog.String("error", err.Error()))
		return
	}
	var p api.PayloadUIUserInput
	if err := json.Unmarshal(raw, &p); err != nil {
		a.logger.Warn("axiom: cannot unmarshal JS event", slog.String("error", err.Error()))
		return
	}
	_ = a.eng.Dispatch("engine", api.TopicUIUserInput, p)
}

// ─── Bound Methods — callable from JS ────────────────────────────

// ReadFile reads a file from the workspace and returns its content.
func (a *App) ReadFile(path string) (string, error) {
	if a.fsHdlr == nil {
		return "", fmt.Errorf("filesystem not ready")
	}
	result, err := a.fsHdlr.ReadFile(path)
	if err != nil {
		return "", err
	}
	return result.Content, nil
}

// WriteFile writes (or overwrites) a file in the workspace.
func (a *App) WriteFile(path, content string) error {
	if a.fsHdlr == nil {
		return fmt.Errorf("filesystem not ready")
	}
	return a.fsHdlr.WriteFile(path, content, false)
}

// CreateFile creates a new file (fails if it already exists).
func (a *App) CreateFile(path, content string) error {
	if a.fsHdlr == nil {
		return fmt.Errorf("filesystem not ready")
	}
	return a.fsHdlr.CreateFile(path, content)
}

// DeleteFile deletes a file from the workspace.
func (a *App) DeleteFile(path string) error {
	if a.fsHdlr == nil {
		return fmt.Errorf("filesystem not ready")
	}
	return a.fsHdlr.DeleteFile(path)
}

// ListDir lists the contents of a workspace directory.
func (a *App) ListDir(path string) ([]filesystem.FileEntry, error) {
	if a.fsHdlr == nil {
		return nil, fmt.Errorf("filesystem not ready")
	}
	return a.fsHdlr.ListDir(path)
}

// SetTheme dispatches a theme change through the engine.
func (a *App) SetTheme(themeID string) error {
	if a.eng == nil {
		return fmt.Errorf("engine not ready")
	}
	return a.eng.Dispatch("engine", api.TopicUISetTheme, api.PayloadUITheme{ThemeID: themeID})
}

// OpenTab opens a file as an editor tab.
func (a *App) OpenTab(windowID, filePath, title, language string) error {
	if a.tabMgr == nil {
		return fmt.Errorf("tab manager not ready")
	}
	return a.tabMgr.OpenTab(windowID, filePath, title, language)
}

// SaveWorkspace saves the current UI state to disk.
func (a *App) SaveWorkspace() error {
	if a.persist == nil {
		return fmt.Errorf("persistence not ready")
	}
	return a.persist.SaveSync()
}

// GetConfig returns the current (non-sensitive) configuration.
func (a *App) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"theme":     a.cfg.UI.DefaultTheme,
		"provider":  a.cfg.AI.Provider,
		"model":     a.cfg.AI.ModelID,
		"workspace": a.cfg.Core.WorkspaceDir,
	}
}

// EmitEvent lets the frontend dispatch a typed event through the engine.
func (a *App) EmitEvent(topicStr, payloadJSON string) error {
	if a.eng == nil {
		return fmt.Errorf("engine not ready")
	}
	var payload interface{}
	if payloadJSON != "" {
		if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
			return fmt.Errorf("invalid payload JSON: %w", err)
		}
	}
	return a.eng.Dispatch("engine", api.Topic(topicStr), payload)
}

// RegisterModuleUI enregistre une vue module depuis Go en une seule méthode.
// C'est le raccourci direct sans passer par le bus.
// moduleJSON doit être un PayloadUIModuleRegister sérialisé.
func (a *App) RegisterModuleUI(moduleJSON string) error {
	if a.ctx == nil {
		return fmt.Errorf("context not ready")
	}
	var payload api.PayloadUIModuleRegister
	if err := json.Unmarshal([]byte(moduleJSON), &payload); err != nil {
		return fmt.Errorf("invalid module JSON: %w", err)
	}

	payloadMap := make(map[string]interface{})
	data, _ := json.Marshal(payload)
	_ = json.Unmarshal(data, &payloadMap)

	runtime.EventsEmit(a.ctx, "axiom:event", map[string]interface{}{
		"topic":   "ui.module.register",
		"source":  "app",
		"payload": payloadMap,
	})
	return nil
}

// InjectSlot injecte du HTML dans un slot de l'interface.
func (a *App) InjectSlot(slotName, moduleID, elementID, html, css, js string, replace bool) error {
	if a.ctx == nil {
		return fmt.Errorf("context not ready")
	}
	runtime.EventsEmit(a.ctx, "axiom:event", map[string]interface{}{
		"topic":  "ui.slot.inject",
		"source": "app",
		"payload": map[string]interface{}{
			"slot":     slotName,
			"moduleId": moduleID,
			"id":       elementID,
			"html":     html,
			"css":      css,
			"js":       js,
			"replace":  replace,
		},
	})
	return nil
}

// SetAppBranding modifie le logo et les couleurs de l'application.
func (a *App) SetAppBranding(logoURL, appName, titlebarColor, statusbarColor string) error {
	if a.eng == nil {
		return fmt.Errorf("engine not ready")
	}
	return a.eng.Dispatch("engine", api.TopicUIAppBranding, api.PayloadUIAppBranding{
		LogoURL:        logoURL,
		AppName:        appName,
		TitlebarColor:  titlebarColor,
		StatusbarColor: statusbarColor,
	})
}

// SetIconBadge met à jour le badge d'une icône de module.
func (a *App) SetIconBadge(moduleID string, count int) error {
	if a.eng == nil {
		return fmt.Errorf("engine not ready")
	}
	return a.eng.Dispatch("engine", api.TopicUIIconBadge, api.PayloadUIIconBadge{
		ModuleID: moduleID,
		Count:    count,
	})
}

// SwitchView force le switch vers une vue (moduleID ou "editor").
func (a *App) SwitchView(viewID string) error {
	if a.ctx == nil {
		return fmt.Errorf("context not ready")
	}
	runtime.EventsEmit(a.ctx, "axiom:event", map[string]interface{}{
		"topic":  "ui.view.switch",
		"source": "app",
		"payload": map[string]interface{}{
			"view_id": viewID,
		},
	})
	return nil
}

// ─── Internal helpers ─────────────────────────────────────────────

func isBindingsGeneration() bool {
	for _, arg := range os.Args {
		if strings.Contains(arg, "wailsbindings") || strings.Contains(arg, "bindings") {
			return true
		}
	}
	return false
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
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

type appFSPublisher struct{ eng *engine.Engine }

func (p *appFSPublisher) Subscribe(topic api.Topic, handler func(api.Event)) string {
	return p.eng.Subscribe(topic, handler)
}
func (p *appFSPublisher) Publish(event api.Event) {
	_ = p.eng.Dispatch("filesystem", event.Topic, event.Payload)
}

type appEngineProxy struct{ eng *engine.Engine }

func (p *appEngineProxy) Dispatch(moduleID string, topic api.Topic, payload interface{}) error {
	return p.eng.Dispatch(moduleID, topic, payload)
}
func (p *appEngineProxy) Subscribe(topic api.Topic, handler func(api.Event)) string {
	return p.eng.Subscribe(topic, handler)
}
func (p *appEngineProxy) Unsubscribe(topic api.Topic, subID string) {
	p.eng.Unsubscribe(topic, subID)
}