package tui

import (
	"strings"
	"testing"

	"cross-tools-install/internal/config"
	"cross-tools-install/internal/installer"
	"cross-tools-install/internal/platform"

	tea "github.com/charmbracelet/bubbletea"
)

func TestEnterWithNoSelectionDoesNothing(t *testing.T) {
	model := NewModel(nil, platform.Info{Name: "linux"}, true)
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if command != nil || updated.(Model).state != "select" {
		t.Fatal("entrée ne devrait pas démarrer une installation vide")
	}
}

func TestEnterInstallsFullPack(t *testing.T) {
	actions := []installer.Action{{Tool: config.Tool{Name: "Git"}}, {Tool: config.Tool{Name: "CMake"}}}
	model := NewModel(actions, platform.Info{Name: "linux"}, true)
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if command == nil || updated.(Model).state != "installing" {
		t.Fatal("entrée devrait lancer le pack complet")
	}
	message := command()
	finished, _ := updated.Update(message)
	if finished.(Model).state != "finished" || len(finished.(Model).results) != 2 {
		t.Fatal("le pack complet devrait contenir toutes les actions")
	}
}

func TestViewPresentsOneCompletePack(t *testing.T) {
	actions := []installer.Action{{Tool: config.Tool{Name: "Git", Category: "toolchain"}}}
	view := NewModel(actions, platform.Info{Name: "darwin", Managers: []string{"brew"}}, true).View()
	if !strings.Contains(view, "PACK DISPONIBLE") || !strings.Contains(view, "PRÊT") || !strings.Contains(view, "toolchain") {
		t.Fatalf("la vue devrait présenter clairement le pack: %s", view)
	}
	if strings.Contains(view, "naviguer") || strings.Contains(view, "sélectionner") {
		t.Fatalf("la vue ne devrait plus proposer de navigation individuelle: %s", view)
	}
}

func TestCommandFailureDoesNotStopPack(t *testing.T) {
	actions := []installer.Action{
		{Tool: config.Tool{Name: "one"}, Builtin: true},
		{Tool: config.Tool{Name: "two"}, Builtin: true},
	}
	model := NewModel(actions, platform.Info{Name: "darwin"}, false)
	model.state = "installing"
	updated, command := model.Update(commandFinishedMsg{index: 0, action: actions[0], err: errInstallFailed{}})
	if command == nil || updated.(Model).state != "installing" {
		t.Fatal("une erreur ne devrait pas arrêter le pack")
	}
	message := command()
	finished, _ := updated.Update(message)
	if finished.(Model).state != "finished" || len(finished.(Model).results) != 2 {
		t.Fatal("toutes les actions devraient être traitées")
	}
}

type errInstallFailed struct{}

func (errInstallFailed) Error() string { return "installation failed" }
