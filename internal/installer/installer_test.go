package installer

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"cross-tools-install/internal/config"
	"cross-tools-install/internal/platform"
)

func TestPlanUsesFirstAvailablePackageManager(t *testing.T) {
	manifest := config.Manifest{Tools: []config.Tool{{Name: "Git", Category: "toolchain", Packages: map[string][]config.Package{
		"linux": {{Manager: "apt", Name: "git"}, {Manager: "pip", Name: "git-py"}},
	}}}}
	info := platform.Info{Name: "linux", Managers: []string{"pip"}, Commands: map[string]string{"pip": "pip3"}}
	actions := Plan(manifest, "linux", info)
	if len(actions) != 1 || actions[0].Command != "pip3" || actions[0].Package.Name != "git-py" {
		t.Fatalf("plan inattendu: %#v", actions)
	}
}

func TestPlanHidesUnavailableTools(t *testing.T) {
	manifest := config.Manifest{Tools: []config.Tool{
		{Name: "Git", Category: "toolchain", Packages: map[string][]config.Package{"linux": {{Manager: "apt", Name: "git"}}}},
		{Name: "x64dbg", Category: "Windows", Packages: map[string][]config.Package{"windows": {{Manager: "scoop", Name: "x64dbg"}}}},
	}}
	info := platform.Info{Name: "linux", Managers: []string{"apt"}, Commands: map[string]string{"apt": "apt-get"}}
	actions := Plan(manifest, "linux", info)
	if len(actions) != 1 || actions[0].Tool.Name != "Git" {
		t.Fatalf("les outils indisponibles devraient être masqués: %#v", actions)
	}
}

func TestExpectedCountOnlyIncludesToolsForOS(t *testing.T) {
	manifest := config.Manifest{Tools: []config.Tool{
		{Name: "Git", Category: "common", Packages: map[string][]config.Package{"all": {{Manager: "apt", Name: "git"}}}},
		{Name: "x64dbg", Category: "Windows", Packages: map[string][]config.Package{"windows": {{Manager: "scoop", Name: "x64dbg"}}}},
	}}
	if count := ExpectedCount(manifest, "linux"); count != 1 {
		t.Fatalf("nombre d'outils attendu: 1, reçu: %d", count)
	}
}

func TestWingetUsesNonInteractiveWingetSource(t *testing.T) {
	action := actionFor(
		config.Tool{Name: "Git", Category: "toolchain"},
		config.Package{Manager: "winget", Name: "Git.Git"},
		"windows",
		platform.Info{Commands: map[string]string{"winget": "winget"}},
	)
	command := FormatCommand(action)
	for _, expected := range []string{"--source winget", "--accept-source-agreements", "--disable-interactivity"} {
		if !strings.Contains(command, expected) {
			t.Fatalf("option winget manquante %q dans %s", expected, command)
		}
	}
}

func TestWingetAlreadyInstalledIsSuccessful(t *testing.T) {
	action := Action{Manager: "winget", Tool: config.Tool{Name: "Git"}}
	runner := &fakeRunner{failOn: "winget"}
	runner.error = alreadyInstalledError{}
	results := Install(context.Background(), runner, []Action{action})
	if len(results) != 1 || results[0].Err != nil {
		t.Fatalf("un paquet déjà installé devrait être accepté: %#v", results)
	}
}

func TestInstallContinuesAfterFailure(t *testing.T) {
	actions := []Action{{Tool: config.Tool{Name: "one"}, Command: "one"}, {Tool: config.Tool{Name: "two"}, Command: "two"}}
	runner := &fakeRunner{failOn: "one"}
	results := Install(context.Background(), runner, actions)
	if len(results) != 2 || len(runner.commands) != 2 {
		t.Fatalf("installation devrait continuer: results=%d commands=%d", len(results), len(runner.commands))
	}
	if !reflect.DeepEqual(runner.commands[0], []string{"one"}) {
		t.Fatalf("commande exécutée inattendue: %#v", runner.commands)
	}
}

func TestExecRunnerIncludesCommandOutputOnFailure(t *testing.T) {
	err := (ExecRunner{}).Run(context.Background(), "go", []string{"definitely-not-a-go-command"})
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("la sortie de la commande devrait être conservée: %v", err)
	}
}

type fakeRunner struct {
	commands [][]string
	failOn   string
	error    error
}

func (r *fakeRunner) Run(_ context.Context, command string, args []string) error {
	r.commands = append(r.commands, append([]string{command}, args...))
	if command == r.failOn {
		if r.error != nil {
			return r.error
		}
		return errCommandFailed{}
	}
	return nil
}

type alreadyInstalledError struct{}

func (alreadyInstalledError) Error() string {
	return "exit status 0x8a15002b: Found an existing package already installed. No available upgrade found."
}

type errCommandFailed struct{}

func (errCommandFailed) Error() string { return "command failed" }
