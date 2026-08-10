package installer

import (
	"context"
	"reflect"
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

func TestInstallStopsAfterFailure(t *testing.T) {
	actions := []Action{{Tool: config.Tool{Name: "one"}, Command: "one"}, {Tool: config.Tool{Name: "two"}, Command: "two"}}
	runner := &fakeRunner{failOn: "one"}
	results := Install(context.Background(), runner, actions)
	if len(results) != 1 || len(runner.commands) != 1 {
		t.Fatalf("installation devrait s'arrêter: results=%d commands=%d", len(results), len(runner.commands))
	}
	if !reflect.DeepEqual(runner.commands[0], []string{"one"}) {
		t.Fatalf("commande exécutée inattendue: %#v", runner.commands)
	}
}

type fakeRunner struct {
	commands [][]string
	failOn   string
}

func (r *fakeRunner) Run(_ context.Context, command string, args []string) error {
	r.commands = append(r.commands, append([]string{command}, args...))
	if command == r.failOn {
		return errCommandFailed{}
	}
	return nil
}

type errCommandFailed struct{}

func (errCommandFailed) Error() string { return "command failed" }
