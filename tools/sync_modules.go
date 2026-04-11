package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Ce script scrute le dossier modules/ et génère un autoloader.go
// pour enregistrer automatiquement tous les modules standards détectés.

func main() {
	modulesDir := "modules"
	outPath := filepath.Join(modulesDir, "autoloader.go")

	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		fmt.Printf("❌ Impossible de lire %s: %v\n", modulesDir, err)
		os.Exit(1)
	}

	var imports []string
	var registrations []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		modName := entry.Name()

		// On ignore ai-assistant car il a un constructeur custom (besoin d'une Config)
		if modName == "ai-assistant" {
			continue
		}

		// On vérifie s'il y a des fichiers .go
		goFiles, _ := filepath.Glob(filepath.Join(modulesDir, modName, "*.go"))
		if len(goFiles) == 0 {
			continue // Module sans code Go (ex: UI pure, pas d'enregistrement Go requis)
		}

		// Package name est souvent l'identifiant nettoyer
		pkgName := strings.ReplaceAll(modName, "-", "")

		imports = append(imports, fmt.Sprintf("\t%s \"github.com/axiom-ide/axiom/modules/%s\"", pkgName, modName))
		registrations = append(registrations, fmt.Sprintf("\trunner.Register(%s.New(logger))", pkgName))
	}

	content := `// Code généré automatiquement par tools/sync_modules.go - NE PAS ÉDITER
package modules

import (
	"log/slog"

	"github.com/axiom-ide/axiom/core/module"

` + strings.Join(imports, "\n") + `
)

// AutoRegister charge tous les modules standards trouvés dans le dossier modules/
func AutoRegister(runner *module.Runner, logger *slog.Logger) {
` + strings.Join(registrations, "\n") + `
}
`

	err = os.WriteFile(outPath, []byte(content), 0644)
	if err != nil {
		fmt.Printf("❌ Erreur d'écriture %s: %v\n", outPath, err)
		os.Exit(1)
	}

	fmt.Printf("✅ %d module(s) synchronisé(s) dans %s\n", len(imports), outPath)
}
