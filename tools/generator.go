#!/usr/bin/env go run
// ═══════════════════════════════════════════════════════════════
// AXIOM MODULE GENERATOR
// 
// Cet outil génère un nouveau module Axiom en quelques secondes.
// Usage: go run tools/module-generator.go -name my-module -level L1
// ═══════════════════════════════════════════════════════════════

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// ─────────────────────────────────────────────────────────────────
// STRUCTURES
// ─────────────────────────────────────────────────────────────────

type ModuleConfig struct {
	ID           string
	Name         string
	Description  string
	ClearanceLevel int
	Author       string
	Version      string
	ModulePath   string
}

// ─────────────────────────────────────────────────────────────────
// TEMPLATES
// ─────────────────────────────────────────────────────────────────

const manifestTemplate = `{
  "id": "{{.ID}}",
  "name": "{{.Name}}",
  "version": "{{.Version}}",
  "description": "{{.Description}}",
  "author": "{{.Author}}",
  "clearance_level": {{.ClearanceLevel}},
  "entry_point": "plugin.so",
  "ui_slots": ["sidebar"],
  "subscriptions": [
    "system.ready"
  ],
  "capabilities": [
    "read_files"
  ],
  "enabled": true
}
`

const moduleTemplate = `package {{.PackageName}}

import (
	"context"
	"log/slog"

	"github.com/axiom-ide/axiom/api"
	"github.com/axiom-ide/axiom/core/module"
	"github.com/axiom-ide/axiom/core/security"
)

// {{.StructName}} est le module {{.Name}}
type {{.StructName}} struct {
	module.BaseModule
}

// New crée une instance du module
func New(logger *slog.Logger) *{{.StructName}} {
	return &{{.StructName}}{
		BaseModule: module.NewBase(
			"{{.ID}}",
			"{{.Name}}",
			security.ClearanceLevel({{.ClearanceLevel}}),
			logger,
		),
	}
}

// Init initialise le module
func (m *{{.StructName}}) Init(ctx context.Context, d module.Dispatcher, s module.Subscriber) error {
	m.BaseInit(ctx, d, s)

	// S'abonner aux événements système
	m.On(api.TopicSystemReady, func(ev api.Event) {
		m.Logger().Info("{{.ID}}: module initialized")
	})

	return nil
}

// Stop arrête le module
func (m *{{.StructName}}) Stop() error {
	return m.BaseStop()
}
`

const testTemplate = `package {{.PackageName}}_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/axiom-ide/axiom/core/bus"
	"github.com/axiom-ide/axiom/core/module"
	"{{.ImportPath}}"
)

var testLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

func TestModuleInit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eb := bus.New(ctx, 64, testLogger)
	mod := {{.PackageName}}.New(testLogger)

	dispatcher := &testDispatcher{eb: eb}
	subscriber := &testSubscriber{eb: eb}

	if err := mod.Init(ctx, dispatcher, subscriber); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if mod.ID() != "{{.ID}}" {
		t.Errorf("expected ID '{{.ID}}', got '%s'", mod.ID())
	}
}

// Test helpers
type testDispatcher struct{ eb *bus.EventBus }
func (d *testDispatcher) Dispatch(moduleID string, topic interface{ String() string }, payload interface{}) error {
	return nil
}

type testSubscriber struct{ eb *bus.EventBus }
func (s *testSubscriber) Subscribe(topic interface{ String() string }, handler func(interface{ GetTopic() interface{ String() string } })) string {
	return ""
}
func (s *testSubscriber) Unsubscribe(topic interface{ String() string }, subID string) {}
`

const readmeTemplate = `# {{.Name}}

Bienvenue dans le module **{{.Name}}**!

## 📋 Informations

- **ID:** {{.ID}}
- **Version:** {{.Version}}
- **Clearance Level:** L{{.ClearanceLevel}}
- **Description:** {{.Description}}

## 🚀 Installation

Ce module est automatiquement chargé lors du démarrage d'Axiom si `enabled: true` dans le manifest.

## 📖 Usage

### Dans le Code

\`\`\`go
import "github.com/axiom-ide/axiom/modules/{{.ID}}"

mod := {{.PackageName}}.New(logger)
mod.Init(ctx, dispatcher, subscriber)
\`\`\`

### Événements Disponibles

Le module écoute:
- \`system.ready\` — Au démarrage du moteur
- Ajoute tes propres abonnements ici!

### Émettre des Actions

\`\`\`go
// Émettre une action via le module
m.Emit(api.TopicFileCreate, api.PayloadFileCreate{
    Path: "output.txt",
    Content: "Hello World",
})
\`\`\`

## 🧪 Tests

\`\`\`bash
go test ./...
go test -v ./modules/{{.ID}}
\`\`\`

## 🔐 Permissions (Clearance Level)

**L{{.ClearanceLevel}}** permet:
- Accès à tous les topics jusqu'à ce niveau
- Voir la liste complète: \`core/security/clearance.go\`

## 📝 Modification

Pour modifier ce module:

1. Édite \`{{.ModuleName}}.go\`
2. Ajoute tes handlers d'événements dans \`Init()\`
3. Lance: \`go run ../../main.go\`
4. Les changements se rechargent automatiquement

## 🤝 Contribution

Si tu améliores ce module, n'oublie pas:
- Ajouter des tests
- Documenter les nouveaux événements
- Mettre à jour la version dans manifest.json

---

**Created with Axiom Module Generator** ✨
`

// ─────────────────────────────────────────────────────────────────
// FUNCTIONS
// ─────────────────────────────────────────────────────────────────

func main() {
	// Parse arguments
	name := flag.String("name", "", "Module name (kebab-case, ex: my-module)")
	description := flag.String("desc", "A custom Axiom module", "Module description")
	level := flag.String("level", "L1", "Clearance level (L0, L1, L2, L3)")
	author := flag.String("author", "Axiom Developer", "Module author")
	outputDir := flag.String("out", "./modules", "Output directory")

	flag.Parse()

	// Validation
	if *name == "" {
		fmt.Println("❌ Error: -name is required")
		fmt.Println("Usage: go run tools/module-generator.go -name my-module -level L1")
		os.Exit(1)
	}

	if !isValidModuleName(*name) {
		fmt.Println("❌ Error: Invalid module name. Use kebab-case (ex: my-module)")
		os.Exit(1)
	}

	clearanceLevel := parseClearanceLevel(*level)
	if clearanceLevel == -1 {
		fmt.Println("❌ Error: Invalid clearance level. Use L0, L1, L2, or L3")
		os.Exit(1)
	}

	// Create config
	config := ModuleConfig{
		ID:             *name,
		Name:           humanizeName(*name),
		Description:    *description,
		ClearanceLevel: clearanceLevel,
		Author:         *author,
		Version:        "1.0.0",
		ModulePath:     filepath.Join(*outputDir, *name),
	}

	// Generate module
	if err := generateModule(config); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}

	// Success!
	printSuccess(config)
}

func generateModule(cfg ModuleConfig) error {
	// Create directory
	if err := os.MkdirAll(cfg.ModulePath, 0755); err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}

	// Get package name (kebab-case to camelCase)
	packageName := toCamelCase(cfg.ID)
	structName := toTitleCase(packageName)
	importPath := fmt.Sprintf("github.com/axiom-ide/axiom/modules/%s", cfg.ID)

	// Create manifest.json
	manifestPath := filepath.Join(cfg.ModulePath, "manifest.json")
	if err := writeFile(manifestPath, manifestTemplate, cfg); err != nil {
		return fmt.Errorf("cannot write manifest: %w", err)
	}

	// Create module.go
	modPath := filepath.Join(cfg.ModulePath, fmt.Sprintf("%s.go", packageName))
	data := map[string]interface{}{
		"ID":             cfg.ID,
		"Name":           cfg.Name,
		"PackageName":    packageName,
		"StructName":     structName,
		"ClearanceLevel": cfg.ClearanceLevel,
	}
	if err := writeFileData(modPath, moduleTemplate, data); err != nil {
		return fmt.Errorf("cannot write module: %w", err)
	}

	// Create module_test.go
	testPath := filepath.Join(cfg.ModulePath, fmt.Sprintf("%s_test.go", packageName))
	testData := map[string]interface{}{
		"ID":         cfg.ID,
		"PackageName": packageName,
		"ImportPath": importPath,
	}
	if err := writeFileData(testPath, testTemplate, testData); err != nil {
		return fmt.Errorf("cannot write tests: %w", err)
	}

	// Create README.md
	readmePath := filepath.Join(cfg.ModulePath, "README.md")
	readmeData := map[string]interface{}{
		"ID":         cfg.ID,
		"Name":       cfg.Name,
		"Version":    "1.0.0",
		"ClearanceLevel": cfg.ClearanceLevel,
		"Description": cfg.Description,
		"PackageName": packageName,
		"ModuleName": fmt.Sprintf("%s.go", packageName),
	}
	if err := writeFileData(readmePath, readmeTemplate, readmeData); err != nil {
		return fmt.Errorf("cannot write README: %w", err)
	}

	return nil
}

func writeFile(path, content string, data interface{}) error {
	tmpl, err := template.New("file").Parse(content)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

func writeFileData(path, content string, data map[string]interface{}) error {
	tmpl, err := template.New("file").Parse(content)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

// ─────────────────────────────────────────────────────────────────
// HELPERS
// ─────────────────────────────────────────────────────────────────

func isValidModuleName(name string) bool {
	// Kebab-case validation
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return !strings.HasPrefix(name, "-") && !strings.HasSuffix(name, "-")
}

func parseClearanceLevel(level string) int {
	switch level {
	case "L0":
		return 0
	case "L1":
		return 1
	case "L2":
		return 2
	case "L3":
		return 3
	default:
		return -1
	}
}

func humanizeName(kebab string) string {
	parts := strings.Split(kebab, "-")
	for i, p := range parts {
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

func toCamelCase(kebab string) string {
	parts := strings.Split(kebab, "-")
	if len(parts) == 1 {
		return parts[0]
	}
	result := parts[0]
	for _, p := range parts[1:] {
		result += strings.ToUpper(p[:1]) + p[1:]
	}
	return result
}

func toTitleCase(camel string) string {
	return strings.ToUpper(camel[:1]) + camel[1:]
}

func printSuccess(cfg ModuleConfig) {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════╗")
	fmt.Println("║  ✨ Module Created Successfully!          ║")
	fmt.Println("╚═══════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("📁 Location: %s\n", cfg.ModulePath)
	fmt.Printf("📝 Module:   %s\n", cfg.ID)
	fmt.Printf("📊 Level:    L%d\n", cfg.ClearanceLevel)
	fmt.Println()
	fmt.Println("📂 Files created:")
	fmt.Printf("  - manifest.json\n")
	fmt.Printf("  - %s.go\n", toCamelCase(cfg.ID))
	fmt.Printf("  - %s_test.go\n", toCamelCase(cfg.ID))
	fmt.Printf("  - README.md\n")
	fmt.Println()
	fmt.Println("🚀 Next steps:")
	fmt.Println("  1. Edit the module files in " + cfg.ModulePath)
	fmt.Println("  2. Add event handlers in Init()")
	fmt.Println("  3. Run: go run ./main.go")
	fmt.Println()
	fmt.Println("📚 Documentation:")
	fmt.Printf("  - See %s/README.md\n", cfg.ModulePath)
	fmt.Println("  - See AXIOM_DETAILED_ANALYSIS.md")
	fmt.Println()
}