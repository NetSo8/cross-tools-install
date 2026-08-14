package main

import "testing"

func TestLoadManifestUsesEmbeddedDefault(t *testing.T) {
	t.Chdir(t.TempDir())

	manifest, err := loadManifest(defaultManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Tools) < 20 {
		t.Fatalf("le manifeste intégré devrait contenir les outils principaux, reçu: %d", len(manifest.Tools))
	}
}
