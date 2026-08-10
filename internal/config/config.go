package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Manifest struct {
	Version int    `json:"version"`
	Tools   []Tool `json:"tools"`
}

type Tool struct {
	Name        string               `json:"name"`
	Category    string               `json:"category"`
	Description string               `json:"description"`
	Packages    map[string][]Package `json:"packages"`
}

type Package struct {
	Manager string   `json:"manager"`
	Name    string   `json:"name"`
	Options []string `json:"options,omitempty"`
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}

func Load(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("lecture du manifeste %q: %w", path, err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("JSON invalide %q: %w", path, err)
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, fmt.Errorf("manifeste invalide: %w", err)
	}
	return manifest, nil
}

func (m Manifest) Validate() error {
	if len(m.Tools) == 0 {
		return fmt.Errorf("aucun outil défini")
	}
	seen := make(map[string]bool, len(m.Tools))
	for _, tool := range m.Tools {
		if tool.Name == "" || tool.Category == "" {
			return fmt.Errorf("nom et catégorie requis pour chaque outil")
		}
		if seen[tool.Name] {
			return fmt.Errorf("outil dupliqué: %s", tool.Name)
		}
		seen[tool.Name] = true
		if len(tool.Packages) == 0 {
			return fmt.Errorf("aucun paquet pour %s", tool.Name)
		}
		for osName, packages := range tool.Packages {
			if osName != "all" && osName != "windows" && osName != "linux" && osName != "darwin" {
				return fmt.Errorf("OS inconnu %q pour %s", osName, tool.Name)
			}
			if len(packages) == 0 {
				return fmt.Errorf("liste de paquets vide pour %s/%s", tool.Name, osName)
			}
			for _, pkg := range packages {
				if pkg.Manager == "" || pkg.Name == "" {
					return fmt.Errorf("gestionnaire et nom requis pour %s", tool.Name)
				}
				if pkg.Manager == "script" && pkg.Command == "" {
					return fmt.Errorf("commande requise pour le script %s", tool.Name)
				}
			}
		}
	}
	return nil
}

func (t Tool) PackagesFor(osName string) []Package {
	packages := append([]Package(nil), t.Packages[osName]...)
	return append(packages, t.Packages["all"]...)
}
