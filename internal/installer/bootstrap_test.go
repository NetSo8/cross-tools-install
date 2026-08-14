package installer

import (
	"context"
	"testing"

	"cross-tools-install/internal/config"
	"cross-tools-install/internal/platform"
)

func TestBootstrapPlanForHomebrew(t *testing.T) {
	manifest := config.Manifest{Tools: []config.Tool{{
		Name: "Git", Category: "toolchain",
		Packages: map[string][]config.Package{"darwin": {{Manager: "brew", Name: "git"}}},
	}}}
	lookup := func(command string) (string, error) {
		if command == "bash" || command == "curl" {
			return "/usr/bin/" + command, nil
		}
		return "", bootstrapNotFound{}
	}
	actions := BootstrapPlan(manifest, "darwin", platform.Info{Commands: map[string]string{}}, lookup)
	if len(actions) != 1 || actions[0].Manager != "brew" {
		t.Fatalf("bootstrap inattendu: %#v", actions)
	}
	if FormatBootstrapCommand(actions[0]) == "" {
		t.Fatal("la commande de bootstrap ne devrait pas être vide")
	}
}

func TestInstallBootstrapStopsOnFailure(t *testing.T) {
	runner := &bootstrapRunner{}
	actions := []BootstrapAction{{Manager: "brew", Command: "bash"}, {Manager: "pip", Command: "python3"}}
	results := InstallBootstrap(context.Background(), runner, actions)
	if len(results) != 1 || len(runner.commands) != 1 {
		t.Fatalf("le bootstrap devrait s'arrêter après échec: %#v", results)
	}
}

func TestBootstrapPlanInstallsPipWithPacman(t *testing.T) {
	manifest := config.Manifest{Tools: []config.Tool{{
		Name: "Frida", Category: "reverse",
		Packages: map[string][]config.Package{"all": {{Manager: "pip", Name: "frida-tools"}}},
	}}}
	info := platform.Info{
		Name:     "linux",
		Managers: []string{"pacman"},
		Commands: map[string]string{"pacman": "pacman"},
	}
	lookup := func(command string) (string, error) {
		if command == "pacman" {
			return "/usr/bin/pacman", nil
		}
		return "", bootstrapNotFound{}
	}
	actions := BootstrapPlan(manifest, "linux", info, lookup)
	if len(actions) != 1 {
		t.Fatalf("bootstrap pip inattendu: %#v", actions)
	}
	if got := FormatBootstrapCommand(actions[0]); got != "sudo pacman -S --noconfirm python-pip" {
		t.Fatalf("commande pacman inattendue: %s", got)
	}
}

type bootstrapRunner struct {
	commands [][]string
}

func (r *bootstrapRunner) Run(_ context.Context, command string, args []string) error {
	r.commands = append(r.commands, append([]string{command}, args...))
	return errCommandFailed{}
}

type bootstrapNotFound struct{}

func (bootstrapNotFound) Error() string { return "not found" }
