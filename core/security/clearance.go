// Package security implémente la Sovereign API d'Axiom.
package security

// ClearanceLevel représente le niveau d'accréditation d'un module.
type ClearanceLevel int

const (
	L0 ClearanceLevel = iota
	L1
	L2
	L3
)

func (cl ClearanceLevel) String() string {
	switch cl {
	case L0:
		return "L0_Observer"
	case L1:
		return "L1_Editor"
	case L2:
		return "L2_Architect"
	case L3:
		return "L3_Sovereign"
	default:
		return "Unknown"
	}
}

// topicRequirements maps every known topic to its minimum ClearanceLevel.
// ANY topic not listed here is rejected, even for L3.
var topicRequirements = map[string]ClearanceLevel{
	// ── System ──────────────────────────────────────────────────────
	"system.ready":         L0,
	"system.shutdown":      L3,
	"system.module.loaded": L3,
	"system.module.error":  L3,

	// ── Files ────────────────────────────────────────────────────────
	"file.read":   L0,
	"file.create": L1,
	"file.write":  L1,
	"file.delete": L1,
	// file.opened is published by the internal filesystem handler (L3).
	// External modules must not publish on this topic.
	"file.opened": L3,

	// ── UI ────────────────────────────────────────────────────────────
	"ui.panel.open":  L2,
	"ui.panel.close": L2,
	"ui.theme.set":   L2,
	"ui.window.new":  L2,
	// ui.user.input is sent by the frontend bridge (engine, L3).
	// L1 is sufficient for receiving; engine dispatches at L3 which satisfies it.
	"ui.user.input": L1,

	// ── Editor ────────────────────────────────────────────────────────
	"editor.tab.open":    L1,
	"editor.tab.close":   L1,
	"editor.tab.focus":   L1,
	"editor.tab.changed": L1,

	// ── Workspace ────────────────────────────────────────────────────
	"workspace.save":    L1,
	"workspace.restore": L1,
	// workspace.restored is published internally after a successful restore (L3).
	"workspace.restored": L3,

	// ── AI ────────────────────────────────────────────────────────────
	"ai.command": L2,
	// ai.response is published by the engine (L3) after executing an AI command.
	"ai.response": L3,

	// ── Security ─────────────────────────────────────────────────────
	// Published directly by the Security Manager via publishFn — not through Dispatch.
	// Listed for completeness; the manager bypasses the authorization check for these.
	"security.denied": L3,
	"security.audit":  L3,
}

// RequiredLevelForTopic returns the minimum ClearanceLevel needed to publish on topic.
func RequiredLevelForTopic(topic string) (ClearanceLevel, bool) {
	required, ok := topicRequirements[topic]
	return required, ok
}

// CanPublish returns true if a module with clearance actual can publish on a topic
// requiring required.
func CanPublish(actual, required ClearanceLevel) bool {
	return actual >= required
}