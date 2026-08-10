package tui

import (
	"context"
	"fmt"
	"strings"

	"cross-tools-install/internal/installer"
	"cross-tools-install/internal/platform"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type installFinishedMsg struct {
	results []installer.Result
}

type Model struct {
	actions  []installer.Action
	selected []bool
	info     platform.Info
	runner   installer.Runner
	dryRun   bool
	cursor   int
	state    string
	results  []installer.Result
	height   int
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	badStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4672"))
	keyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FBA92C"))
)

func NewModel(actions []installer.Action, info platform.Info, runner installer.Runner, dryRun bool) Model {
	selected := make([]bool, len(actions))
	for i := range selected {
		selected[i] = true
	}
	return Model{actions: actions, selected: selected, info: info, runner: runner, dryRun: dryRun, state: "select"}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
	case tea.KeyMsg:
		if m.state == "installing" {
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.actions)-1 {
				m.cursor++
			}
		case " ", "space":
			if len(m.actions) > 0 {
				m.selected[m.cursor] = !m.selected[m.cursor]
			}
		case "a":
			for i := range m.selected {
				m.selected[i] = true
			}
		case "n":
			for i := range m.selected {
				m.selected[i] = false
			}
		case "enter":
			cmd := m.installCmd()
			if cmd != nil {
				m.state = "installing"
			}
			return m, cmd
		}
	case installFinishedMsg:
		m.results = msg.results
		m.state = "finished"
	}
	return m, nil
}

func (m Model) installCmd() tea.Cmd {
	actions := make([]installer.Action, 0, len(m.actions))
	for i, action := range m.actions {
		if m.selected[i] {
			actions = append(actions, action)
		}
	}
	if len(actions) == 0 {
		return nil
	}
	if m.dryRun {
		return func() tea.Msg {
			results := make([]installer.Result, len(actions))
			for i, action := range actions {
				results[i] = installer.Result{Action: action}
			}
			return installFinishedMsg{results: results}
		}
	}
	return func() tea.Msg {
		return installFinishedMsg{results: installer.Install(context.Background(), m.runner, actions)}
	}
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Cross Tools Install"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("OS: %s    Gestionnaires: %s\n", m.info.Name, strings.Join(m.info.Managers, ", ")))
	if m.dryRun {
		b.WriteString(mutedStyle.Render("Mode simulation: aucune commande ne sera exécutée."))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	start, end := 0, len(m.actions)
	if m.height > 0 {
		maxRows := m.height - 8
		if maxRows < 5 {
			maxRows = 5
		}
		if len(m.actions) > maxRows {
			start = m.cursor - maxRows/2
			if start < 0 {
				start = 0
			}
			if start+maxRows > len(m.actions) {
				start = len(m.actions) - maxRows
			}
			end = start + maxRows
		}
	}
	if start > 0 {
		b.WriteString(mutedStyle.Render("  ... plus haut ..."))
		b.WriteString("\n")
	}
	for i := start; i < end; i++ {
		action := m.actions[i]
		cursor := "  "
		if i == m.cursor && m.state != "finished" {
			cursor = keyStyle.Render("> ")
		}
		checked := "[ ]"
		if m.selected[i] {
			checked = "[x]"
		}
		line := fmt.Sprintf("%s%s %-27s %-20s", cursor, checked, action.Tool.Name, action.Tool.Category)
		if action.Unavailable {
			line += " " + mutedStyle.Render("gestionnaire absent")
		} else if action.Builtin {
			line += " " + mutedStyle.Render("fourni par le système")
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	if end < len(m.actions) {
		b.WriteString(mutedStyle.Render("  ... plus bas ..."))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	switch m.state {
	case "select":
		b.WriteString(mutedStyle.Render("j/k ou fleches: naviguer   espace: sélectionner   a: tout   n: rien   entrée: installer   q: quitter"))
	case "installing":
		b.WriteString(keyStyle.Render("Installation en cours..."))
	case "finished":
		b.WriteString(okStyle.Render("Traitement terminé. Appuyez sur q pour quitter."))
		for _, result := range m.results {
			b.WriteString("\n")
			if m.dryRun {
				b.WriteString(mutedStyle.Render(fmt.Sprintf("  [plan] %s", installer.FormatCommand(result.Action))))
			} else if result.Err != nil {
				b.WriteString(badStyle.Render(fmt.Sprintf("  [échec] %s: %v", result.Action.Tool.Name, result.Err)))
			} else if result.Action.Builtin || result.Action.Unavailable {
				b.WriteString(mutedStyle.Render(fmt.Sprintf("  [ignoré] %s", result.Action.Tool.Name)))
			} else {
				b.WriteString(okStyle.Render(fmt.Sprintf("  [ok] %s", result.Action.Tool.Name)))
			}
		}
	}
	return b.String()
}
