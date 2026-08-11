package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Lookup func(string) (string, error)

type Info struct {
	Name      string
	Managers  []string
	Privilege string
	Commands  map[string]string
}

func Detect() Info {
	return DetectWith(runtime.GOOS, LookupCommand)
}

// LookupCommand also checks standard install locations that may not have
// been added to the parent process PATH by a freshly installed manager.
func LookupCommand(command string) (string, error) {
	if path, err := exec.LookPath(command); err == nil {
		return path, nil
	}
	for _, path := range knownPaths(runtime.GOOS, command) {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("%s introuvable", command)
}

func knownPaths(goos, command string) []string {
	switch {
	case goos == "darwin" && command == "brew":
		return []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"}
	case goos == "windows" && (command == "scoop" || command == "scoop.cmd"):
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		return []string{
			filepath.Join(home, "scoop", "shims", "scoop.cmd"),
			filepath.Join(home, "scoop", "shims", "scoop.exe"),
		}
	case goos == "windows" && (command == "winget" || command == "winget.exe"):
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		return []string{filepath.Join(home, "AppData", "Local", "Microsoft", "WindowsApps", "winget.exe")}
	default:
		return nil
	}
}

func DetectWith(goos string, lookup Lookup) Info {
	info := Info{Name: goos, Commands: make(map[string]string)}
	switch goos {
	case "windows":
		if !info.add(lookup, "scoop", "scoop.cmd") {
			info.add(lookup, "scoop", "scoop")
		}
		if !info.add(lookup, "winget", "winget.exe") {
			info.add(lookup, "winget", "winget")
		}
		info.add(lookup, "pip", "pip")
	case "linux":
		info.Privilege = "sudo"
		info.add(lookup, "script", "bash")
		info.add(lookup, "apt", "apt-get")
		info.add(lookup, "dnf", "dnf")
		info.add(lookup, "pacman", "pacman")
		info.add(lookup, "snap", "snap")
		if !info.add(lookup, "pip", "pip3") {
			info.add(lookup, "pip", "pip")
		}
	case "darwin":
		info.add(lookup, "brew", "brew")
		if info.Commands["brew"] != "" {
			info.add(lookup, "script", "bash")
		}
		info.add(lookup, "xcode-select", "xcode-select")
		if !info.add(lookup, "pip", "pip3") {
			info.add(lookup, "pip", "pip")
		}
	default:
		info.add(lookup, "pip", "pip3")
	}
	return info
}

func RefreshManagerPath(goos, manager string) {
	commands := []string{manager}
	if manager == "scoop" {
		commands = append(commands, "scoop.cmd")
	}
	if manager == "winget" {
		commands = append(commands, "winget.exe")
	}
	for _, command := range commands {
		for _, path := range knownPaths(goos, command) {
			if _, err := os.Stat(path); err != nil {
				continue
			}
			directory := filepath.Dir(path)
			pathEntries := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
			for _, entry := range pathEntries {
				if entry == directory {
					return
				}
			}
			_ = os.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
			return
		}
	}
}

func (i *Info) add(lookup Lookup, manager, command string) bool {
	if _, err := lookup(command); err != nil {
		return false
	}
	if i.Commands[manager] == "" {
		i.Managers = append(i.Managers, manager)
		i.Commands[manager] = command
	}
	return true
}

func Supported(goos string) bool {
	return goos == "windows" || goos == "linux" || goos == "darwin"
}

func (i Info) String() string {
	if i.Name == "" {
		return "OS inconnu"
	}
	return fmt.Sprintf("%s (%d gestionnaires détectés)", i.Name, len(i.Managers))
}
