package security

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/axiom-ide/axiom/api"
)

// AuditEntry est une entrée dans le journal d'audit de sécurité.
type AuditEntry struct {
	Timestamp  time.Time      `json:"timestamp"`
	ModuleID   string         `json:"module_id"`
	Topic      api.Topic      `json:"topic"`
	Clearance  ClearanceLevel `json:"clearance"`
	Required   ClearanceLevel `json:"required"`
	Authorized bool           `json:"authorized"`
	Reason     string         `json:"reason,omitempty"`
}

type registeredModule struct {
	id        string
	clearance ClearanceLevel
}

// SecurityError est retourné quand une action est refusée.
type SecurityError struct {
	ModuleID      string
	Topic         api.Topic
	ActualLevel   ClearanceLevel
	RequiredLevel ClearanceLevel
}

func (e *SecurityError) Error() string {
	return fmt.Sprintf(
		"SECURITY DENIED: module '%s' (clearance=%s) attempted to publish on '%s' (requires=%s)",
		e.ModuleID, e.ActualLevel, e.Topic, e.RequiredLevel,
	)
}

// Manager est le gardien de la Sovereign API.
type Manager struct {
	mu      sync.RWMutex
	modules map[string]*registeredModule

	auditMu  sync.Mutex
	auditLog []AuditEntry
	maxAudit int

	publishFn func(event api.Event)
	logger    *slog.Logger
}

// NewManager crée un Security Manager.
func NewManager(publishFn func(api.Event), logger *slog.Logger) *Manager {
	return &Manager{
		modules:   make(map[string]*registeredModule),
		auditLog:  make([]AuditEntry, 0, 256),
		maxAudit:  1000,
		publishFn: publishFn,
		logger:    logger,
	}
}

// RegisterModule enregistre un module avec son niveau d'accréditation.
func (m *Manager) RegisterModule(moduleID string, clearance ClearanceLevel) error {
	if moduleID == "" {
		return errors.New("security: moduleID cannot be empty")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.modules[moduleID] = &registeredModule{id: moduleID, clearance: clearance}
	m.logger.Info("security: module registered",
		slog.String("module_id", moduleID),
		slog.String("clearance", clearance.String()),
	)
	return nil
}

// UnregisterModule supprime un module du registre.
func (m *Manager) UnregisterModule(moduleID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.modules, moduleID)
	m.logger.Info("security: module unregistered", slog.String("module_id", moduleID))
}

// Authorize vérifie si un module est autorisé à publier sur un Topic.
func (m *Manager) Authorize(moduleID string, topic api.Topic) error {
	m.mu.RLock()
	mod, exists := m.modules[moduleID]
	m.mu.RUnlock()

	if !exists {
		err := &SecurityError{ModuleID: moduleID, Topic: topic, ActualLevel: -1, RequiredLevel: L0}
		m.recordAudit(moduleID, topic, -1, L0, false, "module not registered")
		m.publishDenied(moduleID, topic, -1, L0, "module not registered")
		return err
	}

	required, ok := RequiredLevelForTopic(string(topic))
	if !ok {
		err := &SecurityError{ModuleID: moduleID, Topic: topic, ActualLevel: mod.clearance, RequiredLevel: L3 + 1}
		m.recordAudit(moduleID, topic, mod.clearance, L3+1, false, "unknown topic")
		m.publishDenied(moduleID, topic, mod.clearance, L3+1, "unknown topic")
		return err
	}

	if !CanPublish(mod.clearance, required) {
		err := &SecurityError{ModuleID: moduleID, Topic: topic, ActualLevel: mod.clearance, RequiredLevel: required}
		m.recordAudit(moduleID, topic, mod.clearance, required, false, "insufficient clearance")
		m.publishDenied(moduleID, topic, mod.clearance, required, "insufficient clearance")
		m.logger.Warn("security: action denied", slog.String("error", err.Error()))
		return err
	}

	m.recordAudit(moduleID, topic, mod.clearance, required, true, "")
	return nil
}

// GetClearance retourne le niveau d'accréditation d'un module.
func (m *Manager) GetClearance(moduleID string) (ClearanceLevel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mod, exists := m.modules[moduleID]
	if !exists {
		return 0, false
	}
	return mod.clearance, true
}

// AuditLog retourne une copie du journal d'audit.
func (m *Manager) AuditLog() []AuditEntry {
	m.auditMu.Lock()
	defer m.auditMu.Unlock()
	result := make([]AuditEntry, len(m.auditLog))
	copy(result, m.auditLog)
	return result
}

func (m *Manager) recordAudit(moduleID string, topic api.Topic,
	actual, required ClearanceLevel, authorized bool, reason string) {

	entry := AuditEntry{
		Timestamp:  time.Now().UTC(),
		ModuleID:   moduleID,
		Topic:      topic,
		Clearance:  actual,
		Required:   required,
		Authorized: authorized,
		Reason:     reason,
	}
	m.auditMu.Lock()
	defer m.auditMu.Unlock()
	if len(m.auditLog) >= m.maxAudit {
		m.auditLog = m.auditLog[1:]
	}
	m.auditLog = append(m.auditLog, entry)
}

func (m *Manager) publishDenied(moduleID string, topic api.Topic,
	actual, required ClearanceLevel, reason string) {
	if m.publishFn == nil {
		return
	}
	m.publishFn(api.Event{
		Topic:  api.TopicSecurityDenied,
		Source: "security.manager",
		Payload: api.PayloadSecurityDenied{
			ModuleID:       moduleID,
			AttemptedTopic: topic,
			RequiredLevel:  int(required),
			ActualLevel:    int(actual),
			Reason:         reason,
		},
	})
}