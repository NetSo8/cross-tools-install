package platform

import "testing"

func TestDetectWith(t *testing.T) {
	available := map[string]bool{"apt-get": true, "pip3": true}
	info := DetectWith("linux", func(command string) (string, error) {
		if available[command] {
			return "/usr/bin/" + command, nil
		}
		return "", errNotFound{}
	})
	if len(info.Managers) != 2 || info.Managers[0] != "apt" || info.Managers[1] != "pip" {
		t.Fatalf("gestionnaires inattendus: %#v", info.Managers)
	}
	if info.Commands["pip"] != "pip3" {
		t.Fatalf("commande pip inattendue: %q", info.Commands["pip"])
	}
}

func TestSupported(t *testing.T) {
	for _, osName := range []string{"windows", "linux", "darwin"} {
		if !Supported(osName) {
			t.Errorf("%s devrait être supporté", osName)
		}
	}
	if Supported("freebsd") {
		t.Error("freebsd ne devrait pas être supporté")
	}
}

type errNotFound struct{}

func (errNotFound) Error() string { return "not found" }
