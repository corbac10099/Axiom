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

var topicRequirements = map[string]ClearanceLevel{
	"file.read":    L0,
	"system.ready": L0,

	"file.create": L1,
	"file.write":  L1,
	"file.delete": L1,

	"ui.panel.open":  L2,
	"ui.panel.close": L2,
	"ui.theme.set":   L2,
	"ui.window.new":  L2,
	"ai.command":     L2,

	"system.module.loaded": L3,
	"system.shutdown":      L3,

	// BUG FIX: "file.opened" est publié par le filesystem en interne.
	// On lui donne un niveau L3 pour qu'uniquement les composants
	// internes de confiance (enregistrés L3) puissent le publier.
	"file.opened": L3,

	// Les events de sécurité sont publiés par le Security Manager lui-même
	// via publishFn directe — ils ne passent pas par Dispatch().
	// On les déclare quand même pour la complétude de la matrice.
	"security.denied": L3,
	"security.audit":  L3,
}

// RequiredLevelForTopic retourne le niveau minimum requis pour publier sur un Topic.
func RequiredLevelForTopic(topic string) (ClearanceLevel, bool) {
	required, ok := topicRequirements[topic]
	return required, ok
}

// CanPublish vérifie si un module avec le niveau 'actual' peut publier sur un Topic.
func CanPublish(actual, required ClearanceLevel) bool {
	return actual >= required
}