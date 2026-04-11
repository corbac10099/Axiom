// Package jsruntime fournit un moteur JavaScript embarqué (goja) pour Axiom.
// Chaque module .js tourne dans son propre contexte isolé mais partage le même bus.
package jsruntime

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/dop251/goja"

	"github.com/axiom-ide/axiom/api"
	"github.com/axiom-ide/axiom/core/module"
	"github.com/axiom-ide/axiom/core/security"
)

// JSModule représente un module chargé depuis un fichier .js.
type JSModule struct {
	mu        sync.Mutex
	id        string
	name      string
	clearance security.ClearanceLevel
	source    string // chemin vers index.js
	code      string // contenu du fichier JS

	vm         *goja.Runtime
	dispatcher module.Dispatcher
	subscriber module.Subscriber
	subIDs     map[api.Topic][]string
	logger     *slog.Logger
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewJSModule crée un module JS prêt à être initialisé.
func NewJSModule(id, name string, clearance security.ClearanceLevel, source, code string, logger *slog.Logger) *JSModule {
	ctx, cancel := context.WithCancel(context.Background())
	return &JSModule{
		id:        id,
		name:      name,
		clearance: clearance,
		source:    source,
		code:      code,
		subIDs:    make(map[api.Topic][]string),
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// ID implémente module.Module.
func (m *JSModule) ID() string { return m.id }

// Name implémente module.Module.
func (m *JSModule) Name() string { return m.name }

// Clearance implémente module.Module.
func (m *JSModule) Clearance() security.ClearanceLevel { return m.clearance }

// Init initialise le runtime JS et exécute le code du module.
func (m *JSModule) Init(ctx context.Context, d module.Dispatcher, s module.Subscriber) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.dispatcher = d
	m.subscriber = s

	// Créer un nouveau runtime goja pour ce module
	vm := goja.New()
	m.vm = vm

	// Exposer l'API Axiom dans le contexte JS
	if err := m.exposeAxiomAPI(vm); err != nil {
		return fmt.Errorf("jsruntime[%s]: cannot expose Axiom API: %w", m.id, err)
	}

	// Exécuter le code JS du module
	if _, err := vm.RunString(m.code); err != nil {
		return fmt.Errorf("jsruntime[%s]: script error: %w", m.id, err)
	}

	// Appeler la fonction init() si elle existe
	if initFn, ok := goja.AssertFunction(vm.Get("init")); ok {
		if _, err := initFn(goja.Undefined()); err != nil {
			return fmt.Errorf("jsruntime[%s]: init() failed: %w", m.id, err)
		}
	}

	m.logger.Info("jsruntime: module initialized",
		slog.String("id", m.id),
		slog.String("source", m.source),
	)
	return nil
}

// Reload recharge le module avec un nouveau code JS (hot-reload).
// Les anciens abonnements sont nettoyés avant le rechargement.
func (m *JSModule) Reload(newCode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Appeler stop() JS si elle existe
	if m.vm != nil {
		if stopFn, ok := goja.AssertFunction(m.vm.Get("stop")); ok {
			_, _ = stopFn(goja.Undefined())
		}
	}

	// Désabonner tous les topics
	if m.subscriber != nil {
		for topic, ids := range m.subIDs {
			for _, id := range ids {
				m.subscriber.Unsubscribe(topic, id)
			}
		}
	}
	m.subIDs = make(map[api.Topic][]string)

	// Nouveau runtime propre
	vm := goja.New()
	m.vm = vm
	m.code = newCode

	if err := m.exposeAxiomAPI(vm); err != nil {
		return fmt.Errorf("jsruntime[%s]: reload API error: %w", m.id, err)
	}
	if _, err := vm.RunString(m.code); err != nil {
		return fmt.Errorf("jsruntime[%s]: reload script error: %w", m.id, err)
	}
	if initFn, ok := goja.AssertFunction(vm.Get("init")); ok {
		if _, err := initFn(goja.Undefined()); err != nil {
			return fmt.Errorf("jsruntime[%s]: reload init() failed: %w", m.id, err)
		}
	}

	m.logger.Info("jsruntime: module reloaded (hot-reload)",
		slog.String("id", m.id),
	)
	return nil
}

// Stop arrête proprement le module JS.
func (m *JSModule) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.vm != nil {
		if stopFn, ok := goja.AssertFunction(m.vm.Get("stop")); ok {
			_, _ = stopFn(goja.Undefined())
		}
		m.vm.Interrupt("module stopped")
	}

	if m.subscriber != nil {
		for topic, ids := range m.subIDs {
			for _, id := range ids {
				m.subscriber.Unsubscribe(topic, id)
			}
		}
	}

	m.cancel()
	m.logger.Info("jsruntime: module stopped", slog.String("id", m.id))
	return nil
}

// exposeAxiomAPI injecte l'objet `axiom` global dans le runtime JS du module.
func (m *JSModule) exposeAxiomAPI(vm *goja.Runtime) error {
	axiomObj := vm.NewObject()

	// ── axiom.moduleId ─────────────────────────────────────────────
	if err := axiomObj.Set("moduleId", m.id); err != nil {
		return err
	}

	// ── axiom.emit(topic, payload) ──────────────────────────────────
	// Permet au module JS d'envoyer un événement sur le bus.
	if err := axiomObj.Set("emit", func(topicStr string, payload goja.Value) error {
		if m.dispatcher == nil {
			return fmt.Errorf("dispatcher not ready")
		}
		var p interface{}
		if payload != nil && !goja.IsUndefined(payload) && !goja.IsNull(payload) {
			p = payload.Export()
		}
		return m.dispatcher.Dispatch(m.id, api.Topic(topicStr), p)
	}); err != nil {
		return err
	}

	// ── axiom.on(topic, handler) ────────────────────────────────────
	// Permet au module JS de s'abonner à un topic du bus.
	if err := axiomObj.Set("on", func(topicStr string, handler goja.Callable) {
		if m.subscriber == nil {
			m.logger.Warn("jsruntime: subscriber not ready", slog.String("id", m.id))
			return
		}
		topic := api.Topic(topicStr)
		subID := m.subscriber.Subscribe(topic, func(ev api.Event) {
			// Exécuter le handler JS dans une goroutine sécurisée
			go func() {
				m.mu.Lock()
				defer m.mu.Unlock()
				if m.vm == nil {
					return
				}
				evObj := m.vm.NewObject()
				_ = evObj.Set("id", ev.ID)
				_ = evObj.Set("topic", string(ev.Topic))
				_ = evObj.Set("source", ev.Source)
				_ = evObj.Set("timestamp", ev.Timestamp.UnixMilli())
				if ev.Payload != nil {
					_ = evObj.Set("payload", m.vm.ToValue(ev.Payload))
				}
				if ev.CorrelationID != "" {
					_ = evObj.Set("correlationId", ev.CorrelationID)
				}
				if _, err := handler(goja.Undefined(), evObj); err != nil {
					m.logger.Warn("jsruntime: handler error",
						slog.String("module", m.id),
						slog.String("topic", topicStr),
						slog.String("error", err.Error()),
					)
				}
			}()
		})
		m.subIDs[topic] = append(m.subIDs[topic], subID)
	}); err != nil {
		return err
	}

	// ── axiom.log(level, message, ...args) ──────────────────────────
	logObj := vm.NewObject()
	_ = logObj.Set("info",  func(msg string) { m.logger.Info("[JS:"+m.id+"] "+msg) })
	_ = logObj.Set("debug", func(msg string) { m.logger.Debug("[JS:"+m.id+"] "+msg) })
	_ = logObj.Set("warn",  func(msg string) { m.logger.Warn("[JS:"+m.id+"] "+msg) })
	_ = logObj.Set("error", func(msg string) { m.logger.Error("[JS:"+m.id+"] "+msg) })
	if err := axiomObj.Set("log", logObj); err != nil {
		return err
	}

	// ── axiom.topics ─────────────────────────────────────────────────
	// Expose toutes les constantes de topics pour éviter les fautes de frappe.
	topicsObj := vm.NewObject()
	topicMap := map[string]string{
		"SystemReady":    string(api.TopicSystemReady),
		"SystemShutdown": string(api.TopicSystemShutdown),
		"FileCreate":     string(api.TopicFileCreate),
		"FileRead":       string(api.TopicFileRead),
		"FileWrite":      string(api.TopicFileWrite),
		"FileDelete":     string(api.TopicFileDelete),
		"FileOpened":     string(api.TopicFileOpened),
		"UIOpenPanel":    string(api.TopicUIOpenPanel),
		"UIClosePanel":   string(api.TopicUIClosePanel),
		"UISetTheme":     string(api.TopicUISetTheme),
		"UIModuleReg":    string(api.TopicUIModuleRegister),
		"UISlotInject":   string(api.TopicUISlotInject),
		"UIViewSwitch":   string(api.TopicUIViewSwitch),
		"UIIconBadge":    string(api.TopicUIIconBadge),
		"AICommand":      string(api.TopicAICommand),
		"AIResponse":     string(api.TopicAIResponse),
	}
	for k, v := range topicMap {
		_ = topicsObj.Set(k, v)
	}
	if err := axiomObj.Set("topics", topicsObj); err != nil {
		return err
	}

	// ── axiom.sleep(ms) ──────────────────────────────────────────────
	// Utilitaire pour les délais dans les modules JS.
	if err := axiomObj.Set("sleep", func(ms int64) {
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}); err != nil {
		return err
	}

	// Injecter l'objet global `axiom` dans le VM
	return vm.Set("axiom", axiomObj)
}