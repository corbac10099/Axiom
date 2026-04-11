// Code généré automatiquement par tools/sync_modules.go - NE PAS ÉDITER
package modules

import (
	"log/slog"

	"github.com/axiom-ide/axiom/core/module"

	demoui "github.com/axiom-ide/axiom/modules/demo-ui"
)

// AutoRegister charge tous les modules standards trouvés dans le dossier modules/
func AutoRegister(runner *module.Runner, logger *slog.Logger) {
	runner.Register(demoui.New(logger))
}
