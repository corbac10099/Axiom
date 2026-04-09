package security

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/axiom-ide/axiom/api"
)

// ─────────────────────────────────────────────
// TYPES
// ─────────────────────────────────────────────

// AuditEntry est une entrée dans le journal d'audit de sécurité.
// Chaque action (autorisée ou rejetée) est enregistrée.
type AuditEntry struct {
	Timestamp  time.Time        `json:"timestamp"`
	ModuleID   string           `json:"module_id"`
	Topic      api.Topic        `json:"topic"`
	Clearance  ClearanceLevel   `json:"clearance"`
	Required   ClearanceLevel   `json:"required"`
	Authorized bool             `json:"authorized"`
	Reason     string           `json:"reason,omitempty"`
}

// registeredModule est la représentation interne d'un module enregistré
// avec son niveau d'accréditation validé.
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

// ─────────────────────────────────────────────
// SECURITY MANAGER
// ─────────────────────────────────────────────

// Manager est le gardien de la Sovereign API.
// Toute action d'un module passe par lui avant d'atteindre le bus.
type Manager struct {
	mu      sync.RWMutex
	modules map[string]*registeredModule // clé : moduleID

	// auditLog est le journal d'audit en mémoire (FIFO, taille limitée).
	auditMu  sync.Mutex
	auditLog []AuditEntry
	maxAudit int // nombre max d'entrées en mémoire

	// bus est utilisé pour publier les événements de sécurité
	// (SecurityDenied, Audit) sans créer de dépendance cyclique.
	publishFn func(event api.Event)

	logger *slog.Logger
}

// NewManager crée un Security Manager.
// publishFn : fonction de publication sur le bus (injection de dépendance
// pour éviter un import circulaire entre security et bus).
func NewManager(publishFn func(api.Event), logger *slog.Logger) *Manager {
	return &Manager{
		modules:   make(map[string]*registeredModule),
		auditLog:  make([]AuditEntry, 0, 256),
		maxAudit:  1000,
		publishFn: publishFn,
		logger:    logger,
	}
}

// ─────────────────────────────────────────────
// REGISTER
// ─────────────────────────────────────────────

// RegisterModule enregistre un module avec son niveau d'accréditation.
// Si le module est déjà enregistré, son niveau est mis à jour.
// Cette opération est réservée au Registry (appelée au chargement d'un module).
func (m *Manager) RegisterModule(moduleID string, clearance ClearanceLevel) error {
	if moduleID == "" {
		return errors.New("security: moduleID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.modules[moduleID] = &registeredModule{
		id:        moduleID,
		clearance: clearance,
	}

	m.logger.Info("security: module registered",
		slog.String("module_id", moduleID),
		slog.String("clearance", clearance.String()),
	)
	return nil
}

// UnregisterModule supprime un module du registre de sécurité.
func (m *Manager) UnregisterModule(moduleID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.modules, moduleID)
	m.logger.Info("security: module unregistered", slog.String("module_id", moduleID))
}

// ─────────────────────────────────────────────
// AUTHORIZE
// ─────────────────────────────────────────────

// Authorize vérifie si un module est autorisé à publier sur un Topic.
// C'est le point de contrôle central de la Sovereign API.
//
// Retourne nil si autorisé, *SecurityError si refusé.
// Publie automatiquement un événement TopicSecurityDenied en cas de refus.
func (m *Manager) Authorize(moduleID string, topic api.Topic) error {
	m.mu.RLock()
	mod, exists := m.modules[moduleID]
	m.mu.RUnlock()

	// Module inconnu = refus immédiat
	if !exists {
		err := &SecurityError{
			ModuleID:      moduleID,
			Topic:         topic,
			ActualLevel:   -1,
			RequiredLevel: L0,
		}
		m.recordAudit(moduleID, topic, -1, L0, false, "module not registered")
		m.publishDenied(moduleID, topic, -1, L0, "module not registered")
		return err
	}

	// Récupère le niveau requis pour ce Topic
	required, ok := RequiredLevelForTopic(string(topic))
	if !ok {
		// Topic inconnu de la matrice = refus par sécurité par défaut
		err := &SecurityError{
			ModuleID:      moduleID,
			Topic:         topic,
			ActualLevel:   mod.clearance,
			RequiredLevel: L3 + 1,
		}
		m.recordAudit(moduleID, topic, mod.clearance, L3+1, false, "unknown topic")
		m.publishDenied(moduleID, topic, mod.clearance, L3+1, "unknown topic")
		return err
	}

	// Vérification du niveau
	if !CanPublish(mod.clearance, required) {
		err := &SecurityError{
			ModuleID:      moduleID,
			Topic:         topic,
			ActualLevel:   mod.clearance,
			RequiredLevel: required,
		}
		m.recordAudit(moduleID, topic, mod.clearance, required, false, "insufficient clearance")
		m.publishDenied(moduleID, topic, mod.clearance, required, "insufficient clearance")
		m.logger.Warn("security: action denied", slog.String("error", err.Error()))
		return err
	}

	// Autorisé — on logue quand même pour l'audit trail
	m.recordAudit(moduleID, topic, mod.clearance, required, true, "")
	return nil
}

// GetClearance retourne le niveau d'accréditation d'un module enregistré.
func (m *Manager) GetClearance(moduleID string) (ClearanceLevel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mod, exists := m.modules[moduleID]
	if !exists {
		return 0, false
	}
	return mod.clearance, true
}

// AuditLog retourne une copie du journal d'audit récent.
func (m *Manager) AuditLog() []AuditEntry {
	m.auditMu.Lock()
	defer m.auditMu.Unlock()
	copy := make([]AuditEntry, len(m.auditLog))
	copy = append(copy[:0], m.auditLog...)
	return copy
}

// ─────────────────────────────────────────────
// INTERNAL
// ─────────────────────────────────────────────

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
		// Rotation FIFO : on supprime le plus ancien
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