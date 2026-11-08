package provider

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// A preset added without its key is refused with ErrNeedsKey, which the CLI
// turns into the command that adds it (TestAddWithoutKeySaysTheCommand).
// The words stay free of a command: the editor and the TUI show them, and
// ask for the key in a field of their own.
func TestMissingKeyIsErrNeedsKey(t *testing.T) {
	isolate(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	checked := 0
	for _, pr := range Presets() {
		p, err := FromPreset(pr.ID)
		if err != nil {
			t.Fatalf("%s: %v", pr.ID, err)
		}
		if pr.NoKey || pr.Endpoint != "" || p.IsVertex() {
			continue // a local server takes no key; a resource's address is asked first
		}
		_, err = Add(p)
		if err == nil {
			t.Fatalf("%s was added with no key", pr.ID)
		}
		if !strings.Contains(err.Error(), "needs an API key") {
			continue // refused for another reason before the key
		}
		checked++
		if !errors.Is(err, ErrNeedsKey) {
			t.Errorf("%s: %q isn't ErrNeedsKey", pr.ID, err)
		}
		if strings.Contains(err.Error(), "myna provider") {
			t.Errorf("%s: %q names a CLI command the editor would show", pr.ID, err)
		}
	}
	// the presets that ask for a key at all are most of them, so a change
	// that stops saying so everywhere can't pass on an empty loop
	if checked < 30 {
		t.Errorf("only %d presets were checked for the key", checked)
	}
}
