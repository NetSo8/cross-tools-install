package installer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"cross-tools-install/internal/config"
	"cross-tools-install/internal/platform"
)

type Action struct {
	Tool        config.Tool
	Package     config.Package
	Manager     string
	Command     string
	Args        []string
	Builtin     bool
	Unavailable bool
}

type Result struct {
	Action Action
	Err    error
}

type BootstrapAction struct {
	Manager string
	Command string
	Args    []string
	Hint    string
}

type BootstrapResult struct {
	Action BootstrapAction
	Err    error
}

type Runner interface {
	Run(context.Context, string, []string) error
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, command string, args []string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdin = os.Stdin
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdout)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
	if err := cmd.Run(); err != nil {
		output := strings.TrimSpace(stdout.String() + "\n" + stderr.String())
		if output != "" {
			return fmt.Errorf("%w: %s", err, output)
		}
		return err
	}
	return nil
}

func Plan(manifest config.Manifest, osName string, info platform.Info) []Action {
	actions := make([]Action, 0, len(manifest.Tools))
	for _, tool := range manifest.Tools {
		packages := tool.PackagesFor(osName)
		selected := Action{Tool: tool, Unavailable: true}
		for _, pkg := range packages {
			if pkg.Manager == "builtin" {
				selected = Action{Tool: tool, Package: pkg, Manager: pkg.Manager, Builtin: true}
				break
			}
			if contains(info.Managers, pkg.Manager) {
				selected = actionFor(tool, pkg, osName, info)
				break
			}
		}
		if selected.Unavailable {
			continue
		}
		actions = append(actions, selected)
	}
	return actions
}

func ExpectedCount(manifest config.Manifest, osName string) int {
	count := 0
	for _, tool := range manifest.Tools {
		if len(tool.PackagesFor(osName)) > 0 {
			count++
		}
	}
	return count
}

// BootstrapPlan returns only manager installers that can run without another
// package manager already being available on the current platform.
func BootstrapPlan(manifest config.Manifest, osName string, info platform.Info, lookup platform.Lookup) []BootstrapAction {
	needed := make([]BootstrapAction, 0)
	seen := make(map[string]bool)
	for _, tool := range manifest.Tools {
		packages := tool.PackagesFor(osName)
		if hasUsablePackage(packages, info) {
			continue
		}
		for _, pkg := range packages {
			if contains(info.Managers, pkg.Manager) || seen[pkg.Manager] {
				continue
			}
			if action, ok := bootstrapAction(osName, pkg.Manager, lookup); ok {
				needed = append(needed, action)
				seen[pkg.Manager] = true
				break
			}
		}
	}
	return needed
}

func InstallBootstrap(ctx context.Context, runner Runner, actions []BootstrapAction) []BootstrapResult {
	results := make([]BootstrapResult, 0, len(actions))
	for _, action := range actions {
		err := runner.Run(ctx, action.Command, action.Args)
		results = append(results, BootstrapResult{Action: action, Err: err})
		if err != nil {
			break
		}
	}
	return results
}

func FormatBootstrapCommand(action BootstrapAction) string {
	parts := append([]string{action.Command}, action.Args...)
	for i, part := range parts {
		if strings.ContainsAny(part, " \t\n&|;()") {
			parts[i] = "'" + strings.ReplaceAll(part, "'", "'\\''") + "'"
		}
	}
	return strings.Join(parts, " ")
}

func hasUsablePackage(packages []config.Package, info platform.Info) bool {
	for _, pkg := range packages {
		if pkg.Manager == "builtin" || contains(info.Managers, pkg.Manager) {
			return true
		}
	}
	return false
}

func bootstrapAction(osName, manager string, lookup platform.Lookup) (BootstrapAction, bool) {
	switch {
	case osName == "darwin" && manager == "brew":
		if !commandAvailable(lookup, "bash") || !commandAvailable(lookup, "curl") {
			return BootstrapAction{}, false
		}
		return BootstrapAction{
			Manager: "brew",
			Command: "bash",
			Args:    []string{"-c", "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"},
			Hint:    "Homebrew sera installé avec le script officiel.",
		}, true
	case osName == "windows" && manager == "scoop":
		command := "powershell"
		if !commandAvailable(lookup, command) {
			command = "pwsh"
		}
		if !commandAvailable(lookup, command) {
			return BootstrapAction{}, false
		}
		return BootstrapAction{
			Manager: "scoop",
			Command: command,
			Args:    []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", "irm get.scoop.sh | iex"},
			Hint:    "Scoop sera installé pour fournir les paquets Windows.",
		}, true
	case manager == "pip":
		for _, command := range []string{"python3", "python", "py"} {
			if commandAvailable(lookup, command) {
				return BootstrapAction{
					Manager: "pip",
					Command: command,
					Args:    []string{"-m", "ensurepip", "--upgrade"},
					Hint:    "pip sera activé avec ensurepip.",
				}, true
			}
		}
	}
	return BootstrapAction{}, false
}

func commandAvailable(lookup platform.Lookup, command string) bool {
	_, err := lookup(command)
	return err == nil
}

func actionFor(tool config.Tool, pkg config.Package, osName string, info platform.Info) Action {
	command := info.Commands[pkg.Manager]
	args := append([]string(nil), pkg.Options...)
	switch pkg.Manager {
	case "brew":
		args = append([]string{"install"}, args...)
		args = append(args, pkg.Name)
	case "scoop":
		args = append([]string{"install"}, args...)
		args = append(args, pkg.Name)
	case "winget":
		args = append([]string{"install", "--id", pkg.Name, "--exact", "--source", "winget", "--accept-source-agreements", "--accept-package-agreements", "--disable-interactivity"}, args...)
	case "apt":
		command = "sudo"
		args = append([]string{"apt-get", "install", "-y"}, args...)
		args = append(args, pkg.Name)
	case "dnf":
		command = "sudo"
		args = append([]string{"dnf", "install", "-y"}, args...)
		args = append(args, pkg.Name)
	case "pacman":
		command = "sudo"
		args = append([]string{"pacman", "-S", "--noconfirm"}, args...)
		args = append(args, pkg.Name)
	case "snap":
		command = "sudo"
		args = append([]string{"snap", "install"}, args...)
		args = append(args, pkg.Name)
	case "pip":
		args = append([]string{"install"}, args...)
		args = append(args, pkg.Name)
	case "xcode-select":
		command = info.Commands[pkg.Manager]
		args = []string{"--install"}
	case "script":
		command = pkg.Command
		args = append([]string(nil), pkg.Args...)
	}
	return Action{Tool: tool, Package: pkg, Manager: pkg.Manager, Command: command, Args: args}
}

func Install(ctx context.Context, runner Runner, actions []Action) []Result {
	results := make([]Result, 0, len(actions))
	for _, action := range actions {
		if action.Builtin || action.Unavailable {
			results = append(results, Result{Action: action})
			continue
		}
		err := runner.Run(ctx, action.Command, action.Args)
		if err != nil && alreadyInstalled(action, err) {
			err = nil
		}
		results = append(results, Result{Action: action, Err: err})
	}
	return results
}

func alreadyInstalled(action Action, err error) bool {
	if action.Manager != "winget" {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "Found an existing package already installed") && strings.Contains(message, "No available upgrade found")
}

func FormatCommand(action Action) string {
	if action.Builtin {
		return fmt.Sprintf("%s (déjà fourni par le système)", action.Tool.Name)
	}
	if action.Unavailable {
		return fmt.Sprintf("%s (aucun gestionnaire disponible)", action.Tool.Name)
	}
	parts := append([]string{action.Command}, action.Args...)
	for i, part := range parts {
		if strings.ContainsAny(part, " \t\n&|;()") {
			parts[i] = "'" + strings.ReplaceAll(part, "'", "'\\''") + "'"
		}
	}
	return strings.Join(parts, " ")
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
