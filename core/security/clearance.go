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
	// file.opened est publié par le filesystem handler interne (L3).
	"file.opened": L3,

	// ── UI natifs ─────────────────────────────────────────────────────
	"ui.panel.open":  L2,
	"ui.panel.close": L2,
	"ui.theme.set":   L2,
	"ui.window.new":  L2,
	// ui.user.input est envoyé par le bridge frontend (engine, L3).
	"ui.user.input": L1,

	// ── UI Module System ──────────────────────────────────────────────
	// Enregistrement d'une vue module complète — L2 (Architect).
	"ui.module.register": L2,
	// Injection dans un slot — L2.
	"ui.slot.inject": L2,
	// Retrait d'un élément injecté — L2.
	"ui.slot.remove": L2,
	// Modification du branding app — L3 (Sovereign uniquement).
	"ui.app.branding": L3,
	// Badge sur icône — L2.
	"ui.icon.badge": L2,
	// Switch de vue — L2.
	"ui.view.switch": L2,

	// ── Editor ────────────────────────────────────────────────────────
	"editor.tab.open":    L1,
	"editor.tab.close":   L1,
	"editor.tab.focus":   L1,
	"editor.tab.changed": L1,

	// ── Workspace ────────────────────────────────────────────────────
	"workspace.save":    L1,
	"workspace.restore": L1,
	// workspace.restored est publié en interne après restore (L3).
	"workspace.restored": L3,

	// ── AI ────────────────────────────────────────────────────────────
	"ai.command": L2,
	// ai.response est publié par l'engine (L3) après une commande AI.
	"ai.response": L3,

	// ── Security ─────────────────────────────────────────────────────
	// Publiés directement par le Security Manager via publishFn (bypass Dispatch).
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