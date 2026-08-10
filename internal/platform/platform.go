package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
		if _, err := exec.LookPath(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("%s introuvable", command)
}

func knownPaths(goos, command string) []string {
	switch {
	case goos == "darwin" && command == "brew":
		return []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"}
	case goos == "windows" && command == "scoop":
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		return []string{
			filepath.Join(home, "scoop", "shims", "scoop.cmd"),
			filepath.Join(home, "scoop", "shims", "scoop.exe"),
		}
	default:
		return nil
	}
}

func DetectWith(goos string, lookup Lookup) Info {
	info := Info{Name: goos, Commands: make(map[string]string)}
	switch goos {
	case "windows":
		info.add(lookup, "scoop", "scoop")
		info.add(lookup, "winget", "winget")
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
