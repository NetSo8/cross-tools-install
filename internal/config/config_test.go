package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndPackagesFor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tools.json")
	content := `{"version":1,"tools":[{"name":"Git","category":"toolchain","packages":{"linux":[{"manager":"apt","name":"git"}],"all":[{"manager":"pip","name":"example"}]}}]}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	packages := manifest.Tools[0].PackagesFor("linux")
	if len(packages) != 2 || packages[0].Name != "git" || packages[1].Name != "example" {
		t.Fatalf("packages inattendus: %#v", packages)
	}
}

func TestLoadBytes(t *testing.T) {
	content := []byte(`{"version":1,"tools":[{"name":"Git","category":"toolchain","packages":{"linux":[{"manager":"apt","name":"git"}]}}]}`)
	manifest, err := LoadBytes("embedded", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Tools) != 1 || manifest.Tools[0].Name != "Git" {
		t.Fatalf("manifeste inattendu: %#v", manifest)
	}
}

func TestValidateRejectsDuplicateTools(t *testing.T) {
	manifest := Manifest{Tools: []Tool{{Name: "Git", Category: "x", Packages: map[string][]Package{"all": {{Manager: "apt", Name: "git"}}}}, {Name: "Git", Category: "x", Packages: map[string][]Package{"all": {{Manager: "apt", Name: "git"}}}}}}
	if err := manifest.Validate(); err == nil {
		t.Fatal("un manifeste dupliqué devrait être refusé")
	}
}

func TestRepositoryManifestIsValid(t *testing.T) {
	manifest, err := Load(filepath.Join("..", "..", "tools.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Tools) < 20 {
		t.Fatalf("le manifeste devrait contenir les outils principaux, reçu: %d", len(manifest.Tools))
	}
}

func TestSmokeManifestIsValid(t *testing.T) {
	if _, err := Load(filepath.Join("..", "..", "testdata", "smoke-tools.json")); err != nil {
		t.Fatal(err)
	}
}
