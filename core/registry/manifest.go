// Package registry gère le cycle de vie des modules d'Axiom :
// scan du dossier /modules, lecture des manifestes JSON,
// enregistrement auprès du Security Manager.
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/axiom-ide/axiom/core/security"
)

// ─────────────────────────────────────────────
// MANIFEST — Contrat JSON d'un module
// ─────────────────────────────────────────────

// Manifest est la structure qui décrit un module Axiom.
// Elle est lue depuis le fichier manifest.json à la racine du module.
//
// Exemple de manifest.json :
//
//	{
//	  "id": "ai-assistant",
//	  "name": "AI Assistant (Mistral)",
//	  "version": "1.0.0",
//	  "description": "Module IA local basé sur Mistral 7B",
//	  "author": "Axiom Team",
//	  "clearance_level": 2,
//	  "entry_point": "plugin.so",
//	  "ui_slots": ["bottom_panel", "sidebar"],
//	  "subscriptions": ["file.opened", "ai.response"],
//	  "capabilities": ["read_files", "set_theme", "open_panel"]
//	}
type Manifest struct {
	// ID unique du module (snake_case, ex: "ai-assistant").
	// Utilisé comme clé dans le Registry et le Security Manager.
	ID string `json:"id"`

	// Name est le nom lisible affiché dans l'interface.
	Name string `json:"name"`

	// Version sémantique du module (ex: "1.2.3").
	Version string `json:"version"`

	// Description courte du rôle du module.
	Description string `json:"description"`

	// Author de ce module.
	Author string `json:"author"`

	// ClearanceLevel est le niveau d'accréditation demandé par le module (0-3).
	// ⚠️  L'opérateur humain doit valider tout niveau ≥ L2 manuellement.
	ClearanceLevel int `json:"clearance_level"`

	// EntryPoint est le chemin relatif vers le binaire/plugin à charger.
	// Pour Go plugins : "plugin.so". Pour HTTP : "http://localhost:PORT".
	EntryPoint string `json:"entry_point"`

	// UISlots liste les zones de l'interface que ce module souhaite occuper.
	// Valeurs possibles : "bottom_panel", "sidebar", "toolbar", "status_bar".
	UISlots []string `json:"ui_slots"`

	// Subscriptions liste les Topics sur lesquels ce module veut s'abonner.
	Subscriptions []string `json:"subscriptions"`

	// Capabilities liste les actions que ce module souhaite effectuer
	// (pour affichage/audit, la vérification réelle est faite par le Security Manager).
	Capabilities []string `json:"capabilities"`

	// Enabled permet de désactiver un module sans le supprimer.
	Enabled *bool `json:"enabled"` // pointeur pour distinguer false de non-défini
}

// IsEnabled retourne true si le module est actif (true par défaut si non défini).
func (m *Manifest) IsEnabled() bool {
	if m.Enabled == nil {
		return true
	}
	return *m.Enabled
}

// Validate vérifie la cohérence d'un manifeste.
func (m *Manifest) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("manifest: 'id' is required")
	}
	if m.Name == "" {
		return fmt.Errorf("manifest[%s]: 'name' is required", m.ID)
	}
	if m.Version == "" {
		return fmt.Errorf("manifest[%s]: 'version' is required", m.ID)
	}
	if m.ClearanceLevel < int(security.L0) || m.ClearanceLevel > int(security.L3) {
		return fmt.Errorf("manifest[%s]: 'clearance_level' must be 0-3, got %d",
			m.ID, m.ClearanceLevel)
	}
	return nil
}

// ClearanceAsLevel convertit l'int du JSON en type ClearanceLevel typé.
func (m *Manifest) ClearanceAsLevel() security.ClearanceLevel {
	return security.ClearanceLevel(m.ClearanceLevel)
}

// ─────────────────────────────────────────────
// LOADER — Lecture du manifest.json
// ─────────────────────────────────────────────

// LoadManifest lit et parse le manifest.json situé dans moduleDir.
func LoadManifest(moduleDir string) (*Manifest, error) {
	manifestPath := filepath.Join(moduleDir, "manifest.json")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("manifest: cannot read '%s': %w", manifestPath, err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("manifest: cannot parse '%s': %w", manifestPath, err)
	}

	if err := manifest.Validate(); err != nil {
		return nil, err
	}

	return &manifest, nil
}