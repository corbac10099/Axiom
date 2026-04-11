// Code généré automatiquement par tools/sync_modules.go - NE PAS ÉDITER
package modules

import (
	"log/slog"

	"github.com/axiom-ide/axiom/core/module"
)

// AutoRegister charge tous les modules standards trouvés dans le dossier modules/
// Les modules JS (index.js) sont chargés automatiquement par le registry au démarrage.
func AutoRegister(runner *module.Runner, logger *slog.Logger) {
	// Aucun module Go pur à enregistrer ici.
	// demo-ui → modules/demo-ui/index.js (hot-reload JS)
	// ai-assistant → enregistré manuellement dans app.go (Config custom)
	_ = runner
	_ = logger
}go mod tidy