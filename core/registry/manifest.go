// Package registry gère le cycle de vie des modules d'Axiom.
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/axiom-ide/axiom/core/security"
)

// Manifest est la structure qui décrit un module Axiom.
type Manifest struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Version        string   `json:"version"`
	Description    string   `json:"description"`
	Author         string   `json:"author"`
	ClearanceLevel int      `json:"clearance_level"`
	EntryPoint     string   `json:"entry_point"`
	UISlots        []string `json:"ui_slots"`
	Subscriptions  []string `json:"subscriptions"`
	Capabilities   []string `json:"capabilities"`
	Enabled        *bool    `json:"enabled"`
}

func (m *Manifest) IsEnabled() bool {
	if m.Enabled == nil {
		return true
	}
	return *m.Enabled
}

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

func (m *Manifest) ClearanceAsLevel() security.ClearanceLevel {
	return security.ClearanceLevel(m.ClearanceLevel)
}

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