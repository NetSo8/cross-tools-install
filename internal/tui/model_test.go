package tui

import (
	"testing"

	"cross-tools-install/internal/config"
	"cross-tools-install/internal/installer"
	"cross-tools-install/internal/platform"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSpaceTogglesSelection(t *testing.T) {
	actions := []installer.Action{{Tool: config.Tool{Name: "Git"}}}
	model := NewModel(actions, platform.Info{Name: "linux"}, nil, true)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	if updated.(Model).selected[0] {
		t.Fatal("la sélection devrait être désactivée")
	}
}

func TestEnterWithNoSelectionDoesNothing(t *testing.T) {
	actions := []installer.Action{{Tool: config.Tool{Name: "Git"}}}
	model := NewModel(actions, platform.Info{Name: "linux"}, nil, true)
	model.selected[0] = false
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if command != nil || updated.(Model).state != "select" {
		t.Fatal("entrée ne devrait pas démarrer une installation vide")
	}
}
