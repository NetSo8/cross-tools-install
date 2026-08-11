package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"cross-tools-install/internal/config"
	"cross-tools-install/internal/installer"
	"cross-tools-install/internal/platform"
	"cross-tools-install/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	manifestPath := flag.String("manifest", "tools.json", "chemin vers le manifeste JSON")
	list := flag.Bool("list", false, "afficher les outils disponibles et quitter")
	yes := flag.Bool("yes", false, "installer tous les outils sans TUI")
	dryRun := flag.Bool("dry-run", false, "afficher/examiner le plan sans exécuter de commande")
	bootstrap := flag.Bool("bootstrap", true, "installer automatiquement les gestionnaires manquants")
	osOverride := flag.String("os", "", "OS à simuler: windows, linux ou darwin")
	flag.Parse()

	manifest, err := config.Load(*manifestPath)
	if err != nil {
		fatal(err)
	}
	osName := *osOverride
	if osName == "" {
		osName = currentOS()
	}
	if !platform.Supported(osName) {
		fatal(fmt.Errorf("OS non supporté: %s", osName))
	}
	if *list {
		info := platform.DetectWith(osName, lookup)
		actions := installer.Plan(manifest, osName, info)
		printActions(actions, *dryRun)
		return
	}

	info := prepareManagers(manifest, osName, *bootstrap, *dryRun)
	actions := installer.Plan(manifest, osName, info)
	if *yes {
		runNonInteractive(actions, *dryRun)
		return
	}

	model := tui.NewModel(actions, info, *dryRun)
	if _, err := tea.NewProgram(model).Run(); err != nil {
		fatal(err)
	}
}

func currentOS() string { return platform.Detect().Name }

func lookup(command string) (string, error) { return platform.LookupCommand(command) }

func prepareManagers(manifest config.Manifest, osName string, enabled, dryRun bool) platform.Info {
	info := platform.DetectWith(osName, lookup)
	bootstrap := installer.BootstrapPlan(manifest, osName, info, lookup)
	if len(bootstrap) == 0 || !enabled {
		return info
	}
	if dryRun {
		fmt.Println("Gestionnaires à installer:")
		for _, action := range bootstrap {
			fmt.Printf("  %s: %s\n", action.Manager, installer.FormatBootstrapCommand(action))
		}
		return info
	}
	fmt.Println("Installation des gestionnaires manquants...")
	results := installer.InstallBootstrap(context.Background(), installer.ExecRunner{}, bootstrap)
	for _, result := range results {
		if result.Err != nil {
			fatal(fmt.Errorf("bootstrap %s: %w", result.Action.Manager, result.Err))
		}
		fmt.Printf("[ok] gestionnaire %s\n", result.Action.Manager)
	}
	updated := platform.DetectWith(osName, lookup)
	for _, action := range bootstrap {
		if !hasManager(updated, action.Manager) {
			fmt.Fprintf(os.Stderr, "[attention] %s a été exécuté mais reste introuvable dans PATH\n", action.Manager)
		}
	}
	return updated
}

func hasManager(info platform.Info, manager string) bool {
	for _, available := range info.Managers {
		if available == manager {
			return true
		}
	}
	return false
}

func printActions(actions []installer.Action, dryRun bool) {
	for _, action := range actions {
		if dryRun {
			fmt.Println(installer.FormatCommand(action))
			continue
		}
		fmt.Printf("%-28s %s\n", action.Tool.Name, installer.FormatCommand(action))
	}
}

func runNonInteractive(actions []installer.Action, dryRun bool) {
	if dryRun {
		printActions(actions, true)
		return
	}
	results := installer.Install(context.Background(), installer.ExecRunner{}, actions)
	failed := false
	for _, result := range results {
		if result.Action.Builtin {
			fmt.Printf("[skip] %s: fourni par le système\n", result.Action.Tool.Name)
			continue
		}
		if result.Action.Unavailable {
			fmt.Printf("[skip] %s: aucun gestionnaire disponible\n", result.Action.Tool.Name)
			continue
		}
		if result.Err != nil {
			fmt.Fprintf(os.Stderr, "[échec] %s: %v\n", result.Action.Tool.Name, result.Err)
			failed = true
			continue
		}
		fmt.Printf("[ok] %s\n", result.Action.Tool.Name)
	}
	if failed {
		os.Exit(1)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "Erreur:", err)
	os.Exit(1)
}
