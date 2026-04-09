# Axiom IDE — Architecture

## Structure

```
axiom/
├── main.go                     ← Point d'entrée, assemble tout
├── go.mod
│
├── api/
│   └── events.go               ← Tous les types/topics (source unique de vérité)
│
├── core/
│   ├── bus/eventbus.go         ← Event Bus async (pub/sub, goroutines)
│   ├── config/config.go        ← Config hiérarchique (.axiom/ → env vars)
│   ├── engine/engine.go        ← Cœur : assemble bus + security + registry
│   ├── filesystem/handler.go   ← Ops fichiers sécurisées (path traversal protégé)
│   ├── module/module.go        ← Interface Module + BaseModule + Runner
│   ├── orchestrator/           ← Window manager (Wails adapter)
│   ├── registry/               ← Scan + chargement modules depuis manifests
│   ├── security/               ← Sovereign API (clearance L0→L3)
│   ├── tabs/tab_manager.go     ← État des onglets éditeur
│   └── workspace/persistence.go← Save/restore état UI (JSON)
│
├── modules/
│   ├── ai-assistant/           ← Multi-provider LLM (Ollama, OpenAI, Claude…)
│   ├── file-explorer/          ← Manifest uniquement (L1)
│   └── theme-manager/          ← Manifest uniquement (L2)
│
├── adapters/
│   └── wails/                  ← Bridge Go↔WebView (build tag: wails)
│
├── frontend/
│   ├── index.html              ← UI principale (VSCode-like shell)
│   ├── axiom-ui-components.css ← Design tokens + composants CSS
│   └── axiom-ui-components.js  ← API JS (axiomPanel, axiomAlert…)
│
├── pkg/uid/                    ← UUID v4
└── tools/generator.go          ← CLI : génère un module en 30 sec
```

## Flux d'un événement

```
Module.Emit(topic, payload)
  → engine.Dispatch(moduleID, topic, payload)
  → security.Authorize(moduleID, topic)       ← vérifie clearance
  → bus.Publish(event)
  → topicRouter → handler goroutines
  → Module(s) abonnés reçoivent l'event
```

## Clearance levels (Sovereign API)

| Level | Nom        | Topics autorisés |
|-------|------------|------------------|
| L0    | Observer   | `file.read`, `system.ready` |
| L1    | Editor     | L0 + `file.create/write/delete` |
| L2    | Architect  | L1 + `ui.*`, `ai.command` |
| L3    | Sovereign  | L2 + `system.shutdown`, `system.module.*` |

Les composants internes (`engine`, `filesystem`, `registry`) sont enregistrés L3.

## Configuration

Ordre de priorité (le dernier l'emporte) :

1. Valeurs par défaut (`core/config/config.go`)
2. `.axiom/config.json`
3. `axiom.config.json`
4. Variables d'environnement (`AXIOM_AI_PROVIDER`, `AXIOM_AI_KEY`…)

## Ajouter un module

```bash
go run tools/generator.go -name mon-module -level L1
# Génère : modules/mon-module/{manifest.json, monmodule.go, _test.go, README.md}
```

Puis enregistrer dans `main.go` :
```go
runner.Register(monmodule.New(slog.Default()))
```

## Build Wails (UI native)

```bash
wails dev                           # Dev avec hot-reload
wails build -platform windows/amd64 # Build final
```

Sans le build tag `wails`, l'orchestrateur utilise le `NoopAdapter` — le core
fonctionne en mode headless.

## Ce qui reste à faire côté frontend

Le shell VSCode-like est en place (`frontend/index.html`). Pour un éditeur complet :

- [ ] Intégrer **Monaco Editor** (`npm i monaco-editor` ou CDN)
- [ ] Brancher l'arbre de fichiers sur l'API Go via `window.runtime.EventsEmit`
- [ ] Implémenter le terminal via `xterm.js`
- [ ] LSP client (déjà prévu dans Nexus — réutilisable)
- [ ] Syntax highlighting Tree-sitter si pas Monaco

Le core Go n'a **pas besoin d'être touché** pour ces ajouts.
