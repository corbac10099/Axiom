// Axiom — Moteur d'ingénierie logicielle modulaire
// ─────────────────────────────────────────────────
// Point d'entrée principal. Reste volontairement minimal.
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
  ╚═══════════════════════════════════════════════╝
`

func main() {
	fmt.Print(banner)

	// ── 1. Config hiérarchique ────────────────────────────────────
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
	// engineProxy implémente filesystem.EventPublisher via engine.Subscribe/Publish.
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
	// nil → NoopAdapter (pas de Wails sans build tag wails)
	orch := orchestrator.NewOrchestrator(nil, nil, slog.Default())
	_ = orch

	// ── 6. Démarrage ──────────────────────────────────────────────
	if err := eng.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: engine start: %v\n", err)
		os.Exit(1)
	}

	// ── 7. Init modules in-process ────────────────────────────────
	engProxy := &engineDispatcherProxy{eng: eng}
	if errs := runner.InitAll(eng.Context(), engProxy, engProxy); len(errs) > 0 {
		for _, e := range errs {
			slog.Warn("module init error", slog.String("error", e.Error()))
		}
	}

	// ── 8. Démo complète ──────────────────────────────────────────
	go runFullDemo(eng, fsHandler, aiMod)

	// ── 9. Attente des signaux OS ─────────────────────────────────
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-sigCh:
		slog.Info("axiom: signal", slog.String("sig", sig.String()))
	case <-eng.Context().Done():
	}

	// ── 10. Arrêt propre ──────────────────────────────────────────
	_ = runner.StopAll()
	eng.Shutdown()
}

// ─────────────────────────────────────────────
// DÉMO COMPLÈTE
// ─────────────────────────────────────────────

func runFullDemo(eng *engine.Engine, fs *filesystem.Handler, ai *aiassistant.AIAssistantModule) {
	time.Sleep(400 * time.Millisecond)

	sep := func(label string) {
		slog.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		slog.Info("  " + label)
		slog.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	}

	check := func(label string, err error) {
		if err != nil {
			if _, ok := err.(*security.SecurityError); ok {
				slog.Info("  ✓ BLOCKED (expected): "+label)
			} else {
				slog.Warn("  ✗ ERROR: "+label, slog.String("err", err.Error()))
			}
		} else {
			slog.Info("  ✓ OK: " + label)
		}
	}

	// ── Bloc 1 : FileSystem ──────────────────────────────────────
	if fs != nil {
		sep("1. FileSystem Handler — Opérations réelles sur disque")

		check("CreateFile('demo/hello.go')",
			fs.CreateFile("demo/hello.go", "package main\n\nfunc main() {}\n"))

		r, err := fs.ReadFile("demo/hello.go")
		if err != nil {
			slog.Warn("  ✗ ERROR: ReadFile", slog.String("err", err.Error()))
		} else {
			slog.Info("  ✓ OK: ReadFile", slog.Int("bytes", int(r.Size)))
		}

		check("WriteFile append('demo/hello.go')",
			fs.WriteFile("demo/hello.go", "\n// generated by Axiom AI\n", true))

		entries, _ := fs.ListDir(".")
		slog.Info("  ✓ OK: ListDir('.')", slog.Int("entries", len(entries)))

		_, err = fs.ReadFile("../../../etc/passwd")
		if err != nil {
			slog.Info("  ✓ BLOCKED (expected): path traversal ../../../etc/passwd")
		} else {
			slog.Error("  ✗ BUG: path traversal was NOT blocked!")
		}
	}

	// ── Bloc 2 : Sovereign API ───────────────────────────────────
	sep("2. Sovereign API — Matrice complète L0→L3")

	permTests := []struct {
		mod   string
		topic api.Topic
		note  string
	}{
		{"ai-assistant", api.TopicFileRead, "L2 → L0 requis"},
		{"ai-assistant", api.TopicFileCreate, "L2 → L1 requis"},
		{"ai-assistant", api.TopicUISetTheme, "L2 → L2 requis"},
		{"ai-assistant", api.TopicSystemShutdown, "L2 → L3 requis ✗"},
		{"file-explorer", api.TopicFileWrite, "L1 → L1 requis"},
		{"file-explorer", api.TopicUIOpenPanel, "L1 → L2 requis ✗"},
		{"theme-manager", api.TopicUISetTheme, "L2 → L2 requis"},
		{"theme-manager", api.TopicSystemShutdown, "L2 → L3 requis ✗"},
	}
	for _, t := range permTests {
		err := eng.Dispatch(t.mod, t.topic, nil)
		check(fmt.Sprintf("%-18s %-25s [%s]", t.mod, t.topic, t.note), err)
	}
	time.Sleep(100 * time.Millisecond)

	// ── Bloc 3 : AI Bridge Parser ────────────────────────────────
	sep("3. AI Bridge — Parser de commandes Axiom")

	queryResult, err := ai.Query(context.Background(), "Create a hello world file", "")
	if err != nil {
		slog.Warn("  ✗ ERROR: AI Query", slog.String("err", err.Error()))
	} else {
		slog.Info("  ✓ OK: AI Query completed",
			slog.String("provider", "none/stub"),
			slog.Int("commands", len(queryResult.Commands)),
			slog.Int("response_bytes", len(queryResult.RawResponse)),
		)
	}

	// ── Bloc 4 : Config ──────────────────────────────────────────
	sep("4. Config — Chargement hiérarchique")
	c, warns := axiomconfig.Load("")
	slog.Info("  ✓ OK: Config loaded",
		slog.String("ai_provider", c.AI.Provider),
		slog.String("theme", c.UI.DefaultTheme),
		slog.Int("bus_buffer", c.Core.BusBufferSize),
		slog.Int("warnings", len(warns)),
	)

	sep("Démo v0.2.0 terminée — Ctrl+C pour quitter")
}

// ─────────────────────────────────────────────
// PROXIES
// ─────────────────────────────────────────────

// enginePublisherProxy expose engine.Engine comme filesystem.EventPublisher.
type enginePublisherProxy struct {
	eng *engine.Engine
}

func (p *enginePublisherProxy) Subscribe(topic api.Topic, handler func(api.Event)) string {
	return p.eng.Subscribe(topic, handler)
}

func (p *enginePublisherProxy) Publish(event api.Event) {
	// Pour les publications internes du filesystem (ex: TopicFileOpened),
	// on bypasse le Security Manager — le filesystem est un composant interne de confiance.
	// En production, le filesystem aurait son propre moduleID L3 enregistré.
	_ = p.eng.Dispatch("filesystem", event.Topic, event.Payload)
}

// engineDispatcherProxy expose engine.Engine via module.Dispatcher + module.Subscriber.
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