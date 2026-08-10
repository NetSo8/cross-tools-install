package tui

import (
	"testing"

	"cross-tools-install/internal/config"
	"cross-tools-install/internal/installer"
	"cross-tools-install/internal/platform"

	tea "github.com/charmbracelet/bubbletea"
)

func TestEnterWithNoSelectionDoesNothing(t *testing.T) {
	model := NewModel(nil, platform.Info{Name: "linux"}, nil, true)
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if command != nil || updated.(Model).state != "select" {
		t.Fatal("entrée ne devrait pas démarrer une installation vide")
	}
}

func TestEnterInstallsFullPack(t *testing.T) {
	actions := []installer.Action{{Tool: config.Tool{Name: "Git"}}, {Tool: config.Tool{Name: "CMake"}}}
	model := NewModel(actions, platform.Info{Name: "linux"}, nil, true)
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
