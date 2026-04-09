# Axiom — Moteur d'Ingénierie Logicielle Modulaire

Axiom est une plateforme d'automatisation et d'intelligence artificielle pour développeurs. Il combine:

- **Core Go ultra-modulaire** avec Event Bus asynchrone
- **Sovereign API** : contrôle d'accès basé sur les permissions (L0→L3)
- **Support multi-provider IA** : Ollama local, OpenAI, Mistral, Anthropic, Groq, llama_cpp
- **FileSystem Handler** avec protection path-traversal
- **Window Orchestrator** pour UI native (Wails v2)

**Version** : v0.2.0-alpha | **Status** : Production-Ready Core

---

## Table of Contents

1. [Installation & Dépendances](#installation--dépendances)
2. [Configuration](#configuration)
3. [Lancer Axiom](#lancer-axiom)
4. [Architecture](#architecture)
5. [Créer un Module](#créer-un-module)
6. [API & Bus d'Événements](#api--bus-dévénements)
7. [Sovereign API (Système de Permissions)](#sovereign-api-système-de-permissions)
8. [Module IA — Multi-Provider](#module-ia--multi-provider)
9. [Troubleshooting](#troubleshooting)

---

## Installation & Dépendances

### Prérequis

- **Go 1.22+** ([télécharger](https://golang.org/dl/))
- **Git**
- Optionnel : **Wails v2** pour UI native
- Optionnel : **Ollama** pour IA locale

### Clone & Setup

```bash
# Clone le repository
git clone https://github.com/axiom-ide/axiom.git
cd axiom

# Télécharge les dépendances Go (google/uuid uniquement)
go mod download
go mod tidy

# Test compilation
go build -o axiom ./
```

### Installation des Provider IA (optionnel)

#### Ollama (Recommandé pour développement local)

```bash
# macOS / Linux
curl https://ollama.ai/install.sh | sh

# Lancer Ollama
ollama serve

# Dans un autre terminal, télécharger un modèle (par défaut: mistral:7b)
ollama pull mistral:7b
ollama pull llama2:7b
ollama pull neural-chat:7b
```

**URL par défaut** : `http://localhost:11434`

#### llama.cpp Server

```bash
# Cloner llama.cpp
git clone https://github.com/ggerganov/llama.cpp
cd llama.cpp
make

# Lancer le serveur
./server -m /path/to/model.gguf -c 4096 -t 8

# Axiom se connectera à http://localhost:8000
```

#### Mistral Cloud API

```bash
# Créer un compte : https://console.mistral.ai
# Générer une clé API
export AXIOM_AI_KEY="your-mistral-api-key"
```

#### OpenAI API

```bash
# Créer un compte : https://platform.openai.com
# Générer une clé API
export AXIOM_AI_KEY="sk-your-openai-api-key"
```

#### Anthropic (Claude API)

```bash
# Créer un compte : https://console.anthropic.com
# Générer une clé API
export AXIOM_AI_KEY="sk-ant-your-anthropic-api-key"
```

#### Groq API (Ultra-rapide)

```bash
# Créer un compte : https://console.groq.com
# Générer une clé API
export AXIOM_AI_KEY="gsk_your-groq-api-key"
```

---

## Configuration

### Fichier de Configuration

Axiom charge la config dans cet ordre :

1. `.axiom/config.json` (local à votre projet)
2. `axiom.config.json` (root du projet)
3. `~/.axiom/config.json` (home)
4. **Variables d'environnement** (override tout)

### Exemple config.json

```json
{
  "core": {
    "modules_dir": "./modules",
    "log_level": "info",
    "bus_buffer_size": 128,
    "workspace_dir": "."
  },
  "ui": {
    "default_theme": "dark",
    "window_width": 1400,
    "window_height": 900,
    "font_size": 14,
    "tab_size": 4,
    "word_wrap": false
  },
  "ai": {
    "provider": "ollama",
    "model_id": "mistral:7b",
    "base_url": "http://localhost:11434",
    "api_key": "",
    "max_tokens": 2048,
    "temperature": 0.2,
    "timeout_secs": 60
  },
  "filesystem": {
    "watch_enabled": true,
    "max_file_size_mb": 50,
    "ignore_patterns": [".git", "node_modules", ".axiom_cache"],
    "backup_on_write": false
  },
  "security": {
    "require_approval_for_l2": true,
    "audit_log_max_entries": 1000,
    "allow_external_modules": false
  }
}
```

### Variables d'Environnement

```bash
# Core
export AXIOM_MODULES_DIR="./modules"
export AXIOM_LOG_LEVEL="debug"
export AXIOM_WORKSPACE="/path/to/workspace"
export AXIOM_DEBUG="1"              # Force log_level=debug
export AXIOM_BUS_BUFFER="256"

# AI
export AXIOM_AI_PROVIDER="ollama"   # ollama|llama_cpp|mistral|openai|anthropic|groq
export AXIOM_AI_MODEL="mistral:7b"
export AXIOM_AI_BASE_URL="http://localhost:11434"
export AXIOM_AI_KEY="your-api-key"

# UI
export AXIOM_THEME="dark"
```

---

## Lancer Axiom

### Mode Basique (Headless)

```bash
# Compiler et lancer
go run ./main.go

# Ou avec le binaire compilé
go build -o axiom ./
./axiom
```

**Sortie attendue** :

```
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

[INFO] axiom: initializing engine, version=0.2.0-alpha
[INFO] filesystem: handler ready, workspace=/path/to/workspace
[INFO] axiom: starting module discovery...
[INFO] registry: module loaded successfully, id=ai-assistant, clearance=L2_Architect
[INFO] axiom: all modules processed, total=1, active=1
[INFO] axiom: engine is ready ✓
```

### Mode avec Wails (UI Native)

**Prérequis** : Wails v2 installé

```bash
# Installer Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Ajouter le build tag et compiler
wails build -platform windows/amd64
# ou pour macOS
wails build -platform darwin
# ou pour Linux
wails build -platform linux
```

### Tests

```bash
# Lancer tous les tests
go test ./...

# Avec couverture
go test -cover ./...

# Spécifique à un package
go test ./core/bus -v
go test ./core/security -v
go test ./core/filesystem -v
```

---

## Architecture

### Structure du Projet

```
axiom/
├── api/                     # Contrat public (Event Types, Topics)
│   └── events.go           # Topics: file.*, ui.*, ai.*, security.*
│
├── core/
│   ├── bus/                 # Event Bus asynchrone
│   ├── config/              # Config hiérarchique
│   ├── engine/              # Moteur central (orchestration)
│   ├── filesystem/          # FileSystem Handler (sécurisé)
│   ├── module/              # Contrat des modules
│   ├── orchestrator/        # Window Orchestrator (UI)
│   ├── registry/            # Module Registry & Discovery
│   └── security/            # Sovereign API (permissions)
│
├── adapters/
│   └── wails/              # Adapter Wails v2
│
├── modules/
│   ├── ai-assistant/       # Module IA (Multi-provider)
│   ├── file-explorer/      # File Explorer (L1)
│   └── theme-manager/      # Theme Manager (L2)
│
├── pkg/
│   ├── uid/                # UUID v4 Generator
│   └── logger/             # Logging utilities
│
└── main.go                 # Point d'entrée + démo
```

### Flux d'Événement

```
Module A
    ↓
engine.Dispatch(moduleID, topic, payload)
    ↓
security.Authorize(moduleID, topic)  ← Vérification des permissions
    ↓ (OK)
bus.Publish(event)
    ↓
topicRouter.dispatch() → Handler goroutines
    ↓
Module B, C, ... reçoivent l'événement
```

### Clearance Levels (Sovereign API)

| Level | Name | Permissions | Modules |
|-------|------|-------------|---------|
| **L0** | Observer | `file.read`, `system.ready` | Theme Explorer |
| **L1** | Editor | L0 + `file.create`, `file.write`, `file.delete` | File Explorer |
| **L2** | Architect | L1 + `ui.*`, `ai.command` | AI Assistant, Theme Manager |
| **L3** | Sovereign | L2 + `system.shutdown`, `system.module.*` | Engine, Security Manager, Filesystem |

---

## Créer un Module

### 1. Créer la Structure

```bash
mkdir -p modules/my-custom-module
cd modules/my-custom-module

# Créer les fichiers
touch manifest.json my_module.go my_module_test.go
```

### 2. Manifest JSON

**`modules/my-custom-module/manifest.json`** :

```json
{
  "id": "my-custom-module",
  "name": "My Custom Module",
  "version": "1.0.0",
  "description": "Un module exemple avec permutations L1",
  "author": "Axiom Developer",
  "clearance_level": 1,
  "entry_point": "plugin.so",
  "ui_slots": ["sidebar"],
  "subscriptions": [
    "file.created",
    "file.deleted",
    "system.ready"
  ],
  "capabilities": [
    "read_files",
    "create_files",
    "write_files"
  ],
  "enabled": true
}
```

**Règles du manifest** :

- `id` : identifiant unique (kebab-case)
- `clearance_level` : 0 (L0), 1 (L1), 2 (L2), 3 (L3)
- `subscriptions` : Topics que le module écoute
- `capabilities` : Commentaire sur ce que le module peut faire
- `enabled` : `true` pour charger automatiquement

### 3. Implémenter le Module

**`modules/my-custom-module/my_module.go`** :

```go
package mycustommodule

import (
	"context"
	"log/slog"

	"github.com/axiom-ide/axiom/api"
	"github.com/axiom-ide/axiom/core/module"
	"github.com/axiom-ide/axiom/core/security"
)

type MyCustomModule struct {
	module.BaseModule
	config map[string]interface{}
}

// New crée une instance du module
func New(logger *slog.Logger) *MyCustomModule {
	return &MyCustomModule{
		BaseModule: module.NewBase(
			"my-custom-module",
			"My Custom Module",
			security.L1,  // Clearance Level
			logger,
		),
	}
}

// Init est appelé au démarrage du moteur
func (m *MyCustomModule) Init(ctx context.Context, d module.Dispatcher, s module.Subscriber) error {
	m.BaseInit(ctx, d, s)

	// S'abonner aux fichiers créés
	m.On(api.TopicFileOpened, func(ev api.Event) {
		m.Logger().Info("my-module: fichier ouvert",
			slog.String("source", ev.Source),
		)
	})

	// S'abonner au démarrage du système
	m.On(api.TopicSystemReady, func(ev api.Event) {
		m.Logger().Info("my-module: système prêt")

		// Émettre une action (avec vérification des permissions)
		if err := m.Emit(api.TopicFileRead, api.PayloadFileRead{
			Path: "config.json",
		}); err != nil {
			m.Logger().Error("my-module: erreur dispatch", slog.String("error", err.Error()))
		}
	})

	m.Logger().Info("my-module: initialized")
	return nil
}

// Stop arrête proprement le module
func (m *MyCustomModule) Stop() error {
	return m.BaseStop()
}

// Custom methods
func (m *MyCustomModule) DoSomething() {
	m.Logger().Info("my-module: doing something!")
}
```

### 4. Tests

**`modules/my-custom-module/my_module_test.go`** :

```go
package mycustommodule_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/axiom-ide/axiom/core/bus"
	"github.com/axiom-ide/axiom/core/module"
	mycustommodule "github.com/axiom-ide/axiom/modules/my-custom-module"
)

var testLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

func TestModuleInit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Créer un event bus
	eb := bus.New(ctx, 64, testLogger)

	// Créer le module
	mod := mycustommodule.New(testLogger)

	// Initialiser
	dispatcher := &testDispatcher{eb: eb}
	subscriber := &testSubscriber{eb: eb}

	if err := mod.Init(ctx, dispatcher, subscriber); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if mod.ID() != "my-custom-module" {
		t.Errorf("expected ID 'my-custom-module', got '%s'", mod.ID())
	}
}

type testDispatcher struct{ eb *bus.EventBus }
func (d *testDispatcher) Dispatch(moduleID string, topic interface{ String() string }, payload interface{}) error {
	return nil
}

type testSubscriber struct{ eb *bus.EventBus }
func (s *testSubscriber) Subscribe(topic interface{ String() string }, handler func(interface{ GetTopic() interface{ String() string } })) string {
	return ""
}
func (s *testSubscriber) Unsubscribe(topic interface{ String() string }, subID string) {}
```

### 5. Enregistrer le Module

**Dans `main.go`** :

```go
import (
	mycustommod "github.com/axiom-ide/axiom/modules/my-custom-module"
)

// ...

// Ajouter au module runner
mod := mycustommod.New(slog.Default())
runner.Register(mod)
```

Ou automatiquement via le Registry (si `enabled: true` dans manifest.json).

---

## API & Bus d'Événements

### Topics Disponibles

#### File Operations

```go
api.TopicFileCreate   // {"path": "...", "content": "..."}
api.TopicFileRead     // {"path": "..."}
api.TopicFileWrite    // {"path": "...", "content": "...", "append": bool}
api.TopicFileDelete   // {"path": "..."}
api.TopicFileOpened   // Publié par le filesystem après lecture
```

#### UI Operations

```go
api.TopicUIOpenPanel  // {"panel_id": "...", "title": "...", "position": "..."}
api.TopicUIClosePanel // {"panel_id": "..."}
api.TopicUISetTheme   // {"theme_id": "dark|light|monokai"}
api.TopicUINewWindow  // Créer une nouvelle fenêtre
```

#### AI / Automation

```go
api.TopicAICommand    // {"raw_command": "...", "parsed_topic": "...", "parsed_payload": {...}}
api.TopicAIResponse   // Réponse du moteur
```

#### System

```go
api.TopicSystemReady      // Le moteur est prêt
api.TopicSystemShutdown   // Arrêt imminent
api.TopicModuleLoaded     // Un module s'est chargé
api.TopicModuleError      // Erreur lors du chargement
```

#### Security

```go
api.TopicSecurityDenied   // Action refusée (permissions insuffisantes)
api.TopicSecurityAudit    // Événement d'audit
```

### Dispatcher dans un Module

```go
// Envoyer un événement via le module
m.Emit(api.TopicFileCreate, api.PayloadFileCreate{
	Path:    "output.go",
	Content: "package main\n\nfunc main() {}",
})
```

### Subscriber dans un Module

```go
// Écouter un événement
m.On(api.TopicSystemReady, func(ev api.Event) {
	// ev.Topic, ev.Source, ev.Payload, ev.Timestamp
	m.Logger().Info("System ready!", slog.String("source", ev.Source))
})
```

---

## Sovereign API — Système de Permissions

### Vérification des Permissions

Chaque Topic a un niveau requis :

```go
// Dans security/clearance.go
var topicRequirements = map[string]ClearanceLevel{
	"file.read":    L0,
	"file.create": L1,
	"file.write":  L1,
	"ui.panel.open": L2,
	"system.shutdown": L3,
}
```

### Autorisation Automatique

Quand un module appelle `m.Emit()` :

1. Le Security Manager vérifie le clearance du module
2. Compare avec le niveau requis du Topic
3. Si insuffisant → **SecurityError**, l'action est bloquée
4. Sinon → L'événement est publié normalement

### Exemple

```go
// Module avec L1
m.Emit(api.TopicFileCreate, ...)  // ✓ OK (L1 >= L1 requis)
m.Emit(api.TopicUISetTheme, ...)  // ✗ DENIED (L1 < L2 requis)
m.Emit(api.TopicSystemShutdown, ...) // ✗ DENIED (L1 < L3 requis)
```

### Enregistrer un Module dans le Security Manager

```go
secMgr.RegisterModule("my-module", security.L1)
secMgr.RegisterModule("admin-panel", security.L3)
```

---

## Module IA — Multi-Provider

### Providers Supportés

| Provider | Setup | Coût | Vitesse | Qualité |
|----------|-------|------|---------|---------|
| **ollama** | Local + gratuit | 0€ | Rapide (GPU) | Good (Mistral, Llama) |
| **llama_cpp** | Local + gratuit | 0€ | Très rapide | Good |
| **mistral** | API Cloud | ~€0.15/1M tokens | Très rapide | Excellent |
| **openai** | API Cloud | ~€0.03/1K tokens (GPT-4) | Moyen | Meilleur |
| **anthropic** | API Cloud | ~€0.008/1K tokens | Moyen | Excellent (Claude) |
| **groq** | API Cloud (beta) | Gratuit | Ultra-rapide | Bon |

### Configuration par Provider

#### Ollama (Recommandé pour dev)

```bash
# Démarrer Ollama
ollama serve

# Télécharger un modèle
ollama pull mistral:7b

# Config Axiom
export AXIOM_AI_PROVIDER="ollama"
export AXIOM_AI_MODEL="mistral:7b"
export AXIOM_AI_BASE_URL="http://localhost:11434"
```

#### llama.cpp

```bash
# Démarrer le serveur
./server -m model.gguf

# Config Axiom
export AXIOM_AI_PROVIDER="llama_cpp"
export AXIOM_AI_BASE_URL="http://localhost:8000"
```

#### Mistral Cloud

```bash
# Clé API depuis https://console.mistral.ai
export AXIOM_AI_PROVIDER="mistral"
export AXIOM_AI_MODEL="mistral-small"  # ou mistral-medium, mistral-large
export AXIOM_AI_KEY="your-mistral-api-key"
```

#### OpenAI

```bash
# Clé API depuis https://platform.openai.com
export AXIOM_AI_PROVIDER="openai"
export AXIOM_AI_MODEL="gpt-4"  # ou gpt-3.5-turbo
export AXIOM_AI_KEY="sk-..."
```

#### Anthropic

```bash
# Clé API depuis https://console.anthropic.com
export AXIOM_AI_PROVIDER="anthropic"
export AXIOM_AI_MODEL="claude-3-sonnet-20240229"
export AXIOM_AI_KEY="sk-ant-..."
```

#### Groq

```bash
# Clé API depuis https://console.groq.com
export AXIOM_AI_PROVIDER="groq"
export AXIOM_AI_MODEL="mixtral-8x7b-32768"
export AXIOM_AI_KEY="gsk_..."
```

### Utiliser le Module IA

```go
// Dans un module
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := aiMod.Query(ctx, "Create a Go function that sorts arrays", "")
if err != nil {
	m.Logger().Error("AI query failed", slog.String("error", err.Error()))
	return
}

for _, cmd := range result.Commands {
	m.Logger().Info("AI command", slog.String("topic", string(cmd.Topic)))
	// Dispatcher la commande via le moteur
	m.Emit(cmd.Topic, cmd.Payload)
}
```

### Format des Réponses IA

Le module IA extrait les commandes du format :

```
I'll create a sorting function for you.
<axiom:command>FILE_CREATE sort.go package main

func Sort(arr []int) {
    // implementation
}</axiom:command>

Done! The file is ready.
```

Commandes supportées :
- `FILE_CREATE <path> <content>`
- `FILE_WRITE <path> <content>`
- `FILE_READ <path>`
- `UI_SET_THEME <theme_id>`
- `UI_OPEN_PANEL <panel_id> <title>`

---

## Troubleshooting

### "Module not found" lors du chargement

```
❌ ERROR: registry: module load failed, dir=ai-assistant, error=manifest: cannot read...
```

**Solution** :

1. Vérifier que le `manifest.json` existe et est valide
2. Valider le JSON : `jq . modules/ai-assistant/manifest.json`
3. Vérifier l'ID du module dans le manifest

### "Security permission denied"

```
🔒 SECURITY DENIED: module 'my-module' (clearance=L1) attempted to publish on 'ui.panel.open' (requires=L2)
```

**Solution** :

1. Augmenter le `clearance_level` dans le manifest.json
2. Ou réduire les topics que le module utilise

### Ollama ne répond pas

```
❌ ai: LLM query failed: ollama HTTP error: connection refused
```

**Solution** :

```bash
# Vérifier qu'Ollama tourne
ps aux | grep ollama

# Relancer Ollama
ollama serve

# Tester la connexion
curl http://localhost:11434/api/version
```

### Fichiers not created / Permission denied

**Solution** :

1. Vérifier que le répertoire workspace existe
2. Vérifier les permissions sur le répertoire
3. Vérifier que le filepath n'essaye pas de faire une traversée (`../../../etc/passwd` = bloqué)

### Tests échouent

```bash
# Lancer les tests avec output verbose
go test -v ./core/bus

# Ajouter des logs
go test -run TestPublishReceived -v ./core/bus
```

### Compiler pour Production

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o axiom.exe -ldflags="-s -w" ./

# macOS
GOOS=darwin GOARCH=arm64 go build -o axiom-darwin -ldflags="-s -w" ./

# Linux
GOOS=linux GOARCH=amd64 go build -o axiom-linux -ldflags="-s -w" ./
```

---

## Ressources & Liens

- **Documentation Axiom** : [GitHub](https://github.com/axiom-ide/axiom)
- **Event Bus** : `core/bus/eventbus.go`
- **Sovereign API** : `core/security/clearance.go`
- **Module Contrat** : `core/module/module.go`
- **Wails v2** : [wails.io](https://wails.io)
- **Ollama** : [ollama.ai](https://ollama.ai)
- **Mistral API** : [mistral.ai](https://mistral.ai)

---

## Licence

Axiom est sous licence **MIT**. Libre d'usage commerciale et personnelle.

---

**Axiom v0.2.0-alpha** — Modular Software Engineering Platform  
Maintenance: Romain Delassus & Axiom Core Team
