// Package security implémente la Sovereign API d'Axiom :
// le système de permissions à 4 niveaux (L0→L3) qui contrôle
// ce que chaque module est autorisé à faire.
//
// Principe du moindre privilège : un module demande un niveau,
// le Security Manager vérifie, et rejette toute action qui
// dépasse l'accréditation accordée.
package security

// ClearanceLevel représente le niveau d'accréditation d'un module.
// Plus le niveau est élevé, plus les permissions sont étendues.
type ClearanceLevel int

const (
	// L0 — Observer : lecture seule.
	// Usage : analyse de code, linters, statistiques.
	// Accès autorisé : lire des fichiers, écouter des événements.
	L0 ClearanceLevel = iota // valeur = 0

	// L1 — Editor : modification de fichiers texte.
	// Usage : formatters, générateurs de code, auto-complétion.
	// Accès autorisé : L0 + créer/écrire/supprimer des fichiers.
	L1 // valeur = 1

	// L2 — Architect : modification de l'interface.
	// Usage : modules UI, gestionnaires de thèmes, modules IA standards.
	// Accès autorisé : L1 + ouvrir/fermer des panels, changer le thème,
	// créer de nouvelles fenêtres OS.
	L2 // valeur = 2

	// L3 — Sovereign : accès total au système.
	// Usage : module de gestion des plugins, terminal intégré, DevTools.
	// Accès autorisé : L2 + exécution de scripts système,
	// installation/désinstallation de modules.
	// ⚠️  À n'accorder qu'aux modules de confiance absolue.
	L3 // valeur = 3
)

// String retourne le nom lisible du niveau d'accréditation.
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

// topicRequirements mappe chaque Topic à son niveau d'accréditation minimum.
// C'est la "matrice de permissions" centrale d'Axiom.
// Si un Topic n'est pas dans cette map, l'accès est REFUSÉ par défaut.
var topicRequirements = map[string]ClearanceLevel{
	// ── Lecture (L0) ────────────────────────────────────────────
	"file.read":    L0,
	"system.ready": L0,

	// ── Écriture fichiers (L1) ───────────────────────────────────
	"file.create": L1,
	"file.write":  L1,
	"file.delete": L1,

	// ── Interface (L2) ──────────────────────────────────────────
	"ui.panel.open":  L2,
	"ui.panel.close": L2,
	"ui.theme.set":   L2,
	"ui.window.new":  L2,
	"ai.command":     L2,

	// ── Système (L3) ────────────────────────────────────────────
	"system.module.loaded": L3,
	"system.shutdown":      L3,
}

// RequiredLevelForTopic retourne le niveau minimum requis pour publier
// sur un Topic donné.
// Retourne L3+1 (refus total) si le Topic est inconnu.
func RequiredLevelForTopic(topic string) (ClearanceLevel, bool) {
	required, ok := topicRequirements[topic]
	return required, ok
}

// CanPublish vérifie si un module avec le niveau 'actual' est autorisé
// à publier sur un Topic qui requiert le niveau 'required'.
func CanPublish(actual, required ClearanceLevel) bool {
	return actual >= required
}