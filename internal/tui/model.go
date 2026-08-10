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
	actions []installer.Action
	info    platform.Info
	runner  installer.Runner
	dryRun  bool
	state   string
	results []installer.Result
}

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A78BFA"))
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C8494"))
	okStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#34D399"))
	badStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FB7185"))
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#67E8F9"))
	panelStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#3F4654")).Padding(0, 1)
)

func NewModel(actions []installer.Action, info platform.Info, runner installer.Runner, dryRun bool) Model {
	return Model{actions: actions, info: info, runner: runner, dryRun: dryRun, state: "select"}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state == "installing" {
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
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
	actions := m.actions
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
	b.WriteString(titleStyle.Render("CROSS TOOLS"))
	b.WriteString(mutedStyle.Render("  /  ENVIRONMENT PACK"))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("Installation complète des outils compatibles avec la plateforme détectée."))
	b.WriteString("\n\n")

	summary := fmt.Sprintf("OS  %s\nGestionnaires  %s\nOutils du pack  %d", m.info.Name, strings.Join(m.info.Managers, ", "), len(m.actions))
	b.WriteString(panelStyle.Render(summary))
	b.WriteString("\n\n")
	if m.dryRun {
		b.WriteString(mutedStyle.Render("Mode simulation: aucune commande ne sera exécutée."))
		b.WriteString("\n\n")
	}
	b.WriteString(sectionStyle.Render("PACK DISPONIBLE"))
	b.WriteString("\n")
	b.WriteString(renderCategories(m.actions))
	b.WriteString("\n")
	switch m.state {
	case "select":
		b.WriteString(okStyle.Render("PRÊT"))
		b.WriteString("  ")
		b.WriteString(mutedStyle.Render("Entrée installe le pack complet   q quitte"))
	case "installing":
		b.WriteString(mutedStyle.Render("INSTALLATION EN COURS..."))
	case "finished":
		b.WriteString(renderResults(m.results, len(m.actions), m.dryRun))
	}
	return b.String()
}

func renderCategories(actions []installer.Action) string {
	var b strings.Builder
	seen := make(map[string]bool)
	for _, action := range actions {
		if seen[action.Tool.Category] {
			continue
		}
		seen[action.Tool.Category] = true
		b.WriteString("  ")
		b.WriteString(mutedStyle.Render(action.Tool.Category))
		b.WriteString("\n    ")
		var names []string
		for _, candidate := range actions {
			if candidate.Tool.Category != action.Tool.Category {
				continue
			}
			name := candidate.Tool.Name
			if candidate.Builtin {
				name += " (système)"
			}
			names = append(names, name)
		}
		b.WriteString(strings.Join(names, "  ·  "))
		b.WriteString("\n")
	}
	return b.String()
}

func renderResults(results []installer.Result, total int, dryRun bool) string {
	success, skipped, failed := 0, 0, 0
	var failure string
	for _, result := range results {
		if result.Err != nil {
			failed++
			failure = result.Action.Tool.Name + ": " + result.Err.Error()
		} else if result.Action.Builtin {
			skipped++
		} else {
			success++
		}
	}
	if dryRun {
		return "\n" + mutedStyle.Render(fmt.Sprintf("PLAN PRÊT  ·  %d commandes à exécuter", len(results))) + "\n" + mutedStyle.Render("Appuyez sur q pour quitter")
	}
	if failed > 0 {
		return "\n" + badStyle.Render("INSTALLATION INTERROMPUE") + "\n" + mutedStyle.Render(fmt.Sprintf("%d réussies  ·  %d système  ·  %d en échec  ·  %d non exécutées", success, skipped, failed, total-len(results))) + "\n" + badStyle.Render(indentError(failure)) + "\n" + mutedStyle.Render("Appuyez sur q pour quitter")
	}
	return "\n" + okStyle.Render("PACK INSTALLÉ") + "\n" + mutedStyle.Render(fmt.Sprintf("%d outils installés  ·  %d fournis par le système", success, skipped)) + "\n" + mutedStyle.Render("Appuyez sur q pour quitter")
}

func indentError(message string) string {
	return "  " + strings.ReplaceAll(message, "\n", "\n  ")
}
