// Axiom — Moteur d'ingénierie logicielle modulaire
// ─────────────────────────────────────────────────
// Point d'entrée principal pour le core Axiom.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/axiom-ide/axiom/api"
	axiomconfig "github.com/axiom-ide/axiom/core/config"
	"github.com/axiom-ide/axiom/core/engine"
	"github.com/axiom-ide/axiom/core/filesystem"
	"github.com/axiom-ide/axiom/core/module"
	"github.com/axiom-ide/axiom/core/orchestrator"
	"github.com/axiom-ide/axiom/core/security"
	aiassistant "github.com/axiom-ide/axiom/modules/ai-assistant"
)

const banner = `
  ╔═══════════════════════════════════════════════╗
  ║  █████╗ ██╗  ██╗██╗ ██████╗ ███╗   ███╗      ║
  ║ ██╔══██╗╚██╗██╔╝██║██╔═══██╗████╗ ████║      ║
  ║ ███████║ ╚███╔╝ ██║██║   ██║██╔████╔██║      ║
  ║ ██╔══██║ ██╔██╗ ██║██║   ██║██║╚██╔╝██║      ║
  ║ ██║  ██║██╔╝╚██╗██║╚██████╔╝██║ ╚═╝ ██║      ║
  ║ ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝ ╚═════╝ ╚═╝     ╚═╝      ║
  ║                                               ║
  ║  Modular Software Engineering Platform        ║
  ║  v0.2.0-alpha  ·  Sovereign API Active        ║
  ║  Multi-Provider AI Support (Ollama, OpenAI...) ║
  ╚═══════════════════════════════════════════════╝
`

func main() {
	fmt.Print(banner)

	// ── 1. Configuration hiérarchique ─────────────────────────────
	// Charge depuis : .axiom/config.json → axiom.config.json → env vars
	cfg, warnings := axiomconfig.Load("")
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "[WARN] %s\n", w)
	}

	// ── 2. Moteur central ─────────────────────────────────────────
	engCfg := engine.Config{
		ModulesDir:    cfg.Core.ModulesDir,
		LogLevel:      cfg.Core.LogLevel,
		BusBufferSize: cfg.Core.BusBufferSize,
	}
	eng, err := engine.New(engCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: engine init: %v\n", err)
		os.Exit(1)
	}

	// ── 3. FileSystem Handler ─────────────────────────────────────
	enginePub := &enginePublisherProxy{eng: eng}
	fsHandler, err := filesystem.NewHandler(filesystem.Config{
		WorkspaceDir:   cfg.Core.WorkspaceDir,
		MaxFileSizeMB:  cfg.FileSystem.MaxFileSizeMB,
		IgnorePatterns: cfg.FileSystem.IgnorePatterns,
		BackupOnWrite:  cfg.FileSystem.BackupOnWrite,
	}, enginePub, slog.Default())
	if err != nil {
		slog.Warn("filesystem handler init failed (non-fatal)", slog.String("error", err.Error()))
	}

	// ── 4. Module Runner ──────────────────────────────────────────
	runner := module.NewRunner(slog.Default())

	// Module IA avec support multi-provider
	aiMod := aiassistant.New(aiassistant.Config{
		Provider:    cfg.AI.Provider,
		BaseURL:     cfg.AI.BaseURL,
		ModelID:     cfg.AI.ModelID,
		APIKey:      cfg.AI.APIKey,
		MaxTokens:   cfg.AI.MaxTokens,
		Temperature: cfg.AI.Temperature,
		TimeoutSecs: cfg.AI.TimeoutSecs,
	}, slog.Default())
	runner.Register(aiMod)

	// ── 5. Window Orchestrator ────────────────────────────────────
	// nil → NoopAdapter par défaut (pas de Wails sans build tag)
	orch := orchestrator.NewOrchestrator(nil, eng.Bus(), slog.Default())
	_ = orch

	// ── 6. Démarrage du moteur ────────────────────────────────────
	if err := eng.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: engine start: %v\n", err)
		os.Exit(1)
	}

	// ── 7. Initialisation des modules ─────────────────────────────
	engProxy := &engineDispatcherProxy{eng: eng}
	if errs := runner.InitAll(eng.Context(), engProxy, engProxy); len(errs) > 0 {
		for _, e := range errs {
			slog.Warn("module init error", slog.String("error", e.Error()))
		}
	}

	// ── 8. Démonstration / test ───────────────────────────────────
	go runDemo(eng, fsHandler, aiMod, cfg)

	// ── 9. Attente des signaux OS ─────────────────────────────────
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-sigCh:
		slog.Info("axiom: signal received", slog.String("signal", sig.String()))
	case <-eng.Context().Done():
	}

	// ── 10. Arrêt propre ──────────────────────────────────────────
	slog.Info("axiom: shutting down gracefully...")
	_ = runner.StopAll()
	eng.Shutdown()
	slog.Info("axiom: shutdown complete ✓")
}

// ─────────────────────────────────────────────
// DÉMONSTRATION
// ─────────────────────────────────────────────

func runDemo(eng *engine.Engine, fs *filesystem.Handler, ai *aiassistant.AIAssistantModule, cfg axiomconfig.Config) {
	time.Sleep(400 * time.Millisecond)

	sep := func(label string) {
		slog.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		slog.Info("  " + label)
		slog.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	}

	check := func(label string, err error) {
		if err != nil {
			if _, ok := err.(*security.SecurityError); ok {
				slog.Info("  ✓ BLOCKED (expected): " + label)
			} else {
				slog.Warn("  ✗ ERROR: "+label, slog.String("error", err.Error()))
			}
		} else {
			slog.Info("  ✓ OK: " + label)
		}
	}

	// ── Bloc 1 : FileSystem ───────────────────────────────────
	if fs != nil {
		sep("1. FileSystem Handler — Opérations disque")

		check("CreateFile('demo/hello.go')",
			fs.CreateFile("demo/hello.go", "package main\n\nfunc main() {}\n"))

		r, err := fs.ReadFile("demo/hello.go")
		if err != nil {
			slog.Warn("  ✗ ERROR: ReadFile", slog.String("error", err.Error()))
		} else {
			slog.Info("  ✓ OK: ReadFile", slog.Int("bytes", int(r.Size)))
		}

		check("WriteFile append",
			fs.WriteFile("demo/hello.go", "\n// Generated by Axiom\n", true))

		entries, _ := fs.ListDir(".")
		slog.Info("  ✓ OK: ListDir('.')", slog.Int("entries", len(entries)))

		_, err = fs.ReadFile("../../../etc/passwd")
		if err != nil {
			slog.Info("  ✓ BLOCKED (expected): path traversal detected")
		} else {
			slog.Error("  ✗ BUG: path traversal not blocked!")
		}
	}

	// ── Bloc 2 : Sovereign API ────────────────────────────────
	sep("2. Sovereign API — Matrice de permissions L0→L3")

	permTests := []struct {
		mod   string
		topic api.Topic
		note  string
	}{
		{"ai-assistant", api.TopicFileRead, "L2 → L0 requis ✓"},
		{"ai-assistant", api.TopicFileCreate, "L2 → L1 requis ✓"},
		{"ai-assistant", api.TopicUISetTheme, "L2 → L2 requis ✓"},
		{"ai-assistant", api.TopicSystemShutdown, "L2 → L3 requis ✗"},
		{"file-explorer", api.TopicFileWrite, "L1 → L1 requis ✓"},
		{"file-explorer", api.TopicUIOpenPanel, "L1 → L2 requis ✗"},
		{"theme-manager", api.TopicUISetTheme, "L2 → L2 requis ✓"},
		{"theme-manager", api.TopicSystemShutdown, "L2 → L3 requis ✗"},
	}
	for _, t := range permTests {
		err := eng.Dispatch(t.mod, t.topic, nil)
		check(fmt.Sprintf("%-18s %-25s [%s]", t.mod, t.topic, t.note), err)
	}
	time.Sleep(100 * time.Millisecond)

	// ── Bloc 3 : AI Module ────────────────────────────────────
	sep("3. AI Module — Multi-Provider Support")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	slog.Info("  AI Configuration:",
		slog.String("provider", cfg.AI.Provider),
		slog.String("model", cfg.AI.ModelID),
		slog.String("base_url", cfg.AI.BaseURL),
	)

	queryResult, err := ai.Query(ctx, "Create a hello world file", "")
	if err != nil {
		slog.Warn("  ✗ ERROR: AI Query failed", slog.String("error", err.Error()))
	} else {
		slog.Info("  ✓ OK: AI Query completed",
			slog.Int("commands", len(queryResult.Commands)),
			slog.Int("response_bytes", len(queryResult.RawResponse)),
		)
		if len(queryResult.Commands) > 0 {
			for i, cmd := range queryResult.Commands {
				slog.Info(fmt.Sprintf("    Command %d:", i+1),
					slog.String("topic", string(cmd.Topic)),
					slog.String("raw", cmd.Raw),
				)
			}
		}
	}

	// ── Bloc 4 : Configuration ────────────────────────────────
	sep("4. Configuration — Chargement hiérarchique")
	slog.Info("  ✓ OK: Config loaded",
		slog.String("ai_provider", cfg.AI.Provider),
		slog.String("theme", cfg.UI.DefaultTheme),
		slog.Int("bus_buffer", cfg.Core.BusBufferSize),
		slog.Int("window_width", cfg.UI.WindowWidth),
		slog.Int("window_height", cfg.UI.WindowHeight),
	)

	sep("Démonstration v0.2.0 terminée")
	slog.Info("  Appuyez sur Ctrl+C pour quitter")
}

// ─────────────────────────────────────────────
// ADAPTERS / PROXIES
// ─────────────────────────────────────────────

// enginePublisherProxy expose Engine comme filesystem.EventPublisher.
type enginePublisherProxy struct {
	eng *engine.Engine
}

func (p *enginePublisherProxy) Subscribe(topic api.Topic, handler func(api.Event)) string {
	return p.eng.Subscribe(topic, handler)
}

func (p *enginePublisherProxy) Publish(event api.Event) {
	// Les publications internes du filesystem contournent le Security Manager.
	// Le filesystem est un composant interne de confiance (L3).
	_ = p.eng.Dispatch("filesystem", event.Topic, event.Payload)
}

// engineDispatcherProxy expose Engine via module.Dispatcher + module.Subscriber.
type engineDispatcherProxy struct {
	eng *engine.Engine
}

func (p *engineDispatcherProxy) Dispatch(moduleID string, topic api.Topic, payload interface{}) error {
	return p.eng.Dispatch(moduleID, topic, payload)
}

func (p *engineDispatcherProxy) Subscribe(topic api.Topic, handler func(api.Event)) string {
	return p.eng.Subscribe(topic, handler)
}

func (p *engineDispatcherProxy) Unsubscribe(topic api.Topic, subID string) {
	p.eng.Unsubscribe(topic, subID)
}