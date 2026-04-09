package security_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/axiom-ide/axiom/api"
	"github.com/axiom-ide/axiom/core/security"
)

var testLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

func newManager() *security.Manager {
	return security.NewManager(nil, testLogger)
}

func TestRegisterAndAuthorize(t *testing.T) {
	m := newManager()
	if err := m.RegisterModule("mod-l1", security.L1); err != nil {
		t.Fatalf("RegisterModule failed: %v", err)
	}
	// L1 peut lire (L0 requis)
	if err := m.Authorize("mod-l1", api.TopicFileRead); err != nil {
		t.Errorf("L1 should be allowed to read: %v", err)
	}
	// L1 peut écrire (L1 requis)
	if err := m.Authorize("mod-l1", api.TopicFileCreate); err != nil {
		t.Errorf("L1 should be allowed to create files: %v", err)
	}
}

func TestUnregisteredModuleDenied(t *testing.T) {
	m := newManager()
	err := m.Authorize("ghost-module", api.TopicFileRead)
	if err == nil {
		t.Error("unregistered module should be denied")
	}
	secErr, ok := err.(*security.SecurityError)
	if !ok {
		t.Fatalf("expected *SecurityError, got %T", err)
	}
	if secErr.ModuleID != "ghost-module" {
		t.Errorf("expected module_id 'ghost-module', got '%s'", secErr.ModuleID)
	}
}

func TestInsufficientClearanceDenied(t *testing.T) {
	m := newManager()
	_ = m.RegisterModule("mod-l0", security.L0)

	// L0 ne peut PAS créer de fichiers
	if err := m.Authorize("mod-l0", api.TopicFileCreate); err == nil {
		t.Error("L0 should NOT be allowed to create files")
	}
	// L0 ne peut PAS toucher l'UI
	if err := m.Authorize("mod-l0", api.TopicUISetTheme); err == nil {
		t.Error("L0 should NOT be allowed to set theme")
	}
	// L0 ne peut PAS shutdown
	if err := m.Authorize("mod-l0", api.TopicSystemShutdown); err == nil {
		t.Error("L0 should NOT be allowed to shutdown")
	}
}

func TestL2CanDoUI(t *testing.T) {
	m := newManager()
	_ = m.RegisterModule("mod-l2", security.L2)

	if err := m.Authorize("mod-l2", api.TopicUISetTheme); err != nil {
		t.Errorf("L2 should be allowed to set theme: %v", err)
	}
	if err := m.Authorize("mod-l2", api.TopicUIOpenPanel); err != nil {
		t.Errorf("L2 should be allowed to open panel: %v", err)
	}
	// L2 ne peut PAS shutdown (L3 requis)
	if err := m.Authorize("mod-l2", api.TopicSystemShutdown); err == nil {
		t.Error("L2 should NOT be allowed to shutdown")
	}
}

func TestL3CanDoEverything(t *testing.T) {
	m := newManager()
	_ = m.RegisterModule("mod-l3", security.L3)

	topics := []api.Topic{
		api.TopicFileRead, api.TopicFileCreate, api.TopicFileWrite,
		api.TopicUISetTheme, api.TopicUIOpenPanel,
		api.TopicSystemShutdown, api.TopicModuleLoaded,
	}
	for _, topic := range topics {
		if err := m.Authorize("mod-l3", topic); err != nil {
			t.Errorf("L3 should be allowed on topic '%s': %v", topic, err)
		}
	}
}

func TestUnknownTopicDenied(t *testing.T) {
	m := newManager()
	_ = m.RegisterModule("mod-l3", security.L3)

	// Même L3 ne peut PAS publier sur un topic inconnu de la matrice
	if err := m.Authorize("mod-l3", "random.unknown.topic"); err == nil {
		t.Error("unknown topic should be denied even for L3")
	}
}

func TestAuditLogRecordsEntries(t *testing.T) {
	m := newManager()
	_ = m.RegisterModule("mod-l1", security.L1)

	_ = m.Authorize("mod-l1", api.TopicFileRead)   // autorisé
	_ = m.Authorize("mod-l1", api.TopicUISetTheme) // refusé

	log := m.AuditLog()
	if len(log) < 2 {
		t.Errorf("expected at least 2 audit entries, got %d", len(log))
	}
}

func TestGetClearance(t *testing.T) {
	m := newManager()
	_ = m.RegisterModule("mod-l2", security.L2)

	lvl, ok := m.GetClearance("mod-l2")
	if !ok {
		t.Fatal("module should be found")
	}
	if lvl != security.L2 {
		t.Errorf("expected L2, got %s", lvl)
	}
	_, ok = m.GetClearance("nonexistent")
	if ok {
		t.Error("nonexistent module should not be found")
	}
}

func TestUnregisterModule(t *testing.T) {
	m := newManager()
	_ = m.RegisterModule("mod-tmp", security.L1)
	m.UnregisterModule("mod-tmp")

	if err := m.Authorize("mod-tmp", api.TopicFileRead); err == nil {
		t.Error("unregistered module should be denied after removal")
	}
}

func TestEmptyModuleIDRejected(t *testing.T) {
	m := newManager()
	if err := m.RegisterModule("", security.L1); err == nil {
		t.Error("empty moduleID should be rejected")
	}
}

func TestSecurityErrorType(t *testing.T) {
	m := newManager()
	_ = m.RegisterModule("low", security.L0)
	err := m.Authorize("low", api.TopicSystemShutdown)

	secErr, ok := err.(*security.SecurityError)
	if !ok {
		t.Fatalf("expected *SecurityError")
	}
	if secErr.ActualLevel != security.L0 {
		t.Errorf("expected actual L0, got %s", secErr.ActualLevel)
	}
	if secErr.RequiredLevel != security.L3 {
		t.Errorf("expected required L3, got %s", secErr.RequiredLevel)
	}
}