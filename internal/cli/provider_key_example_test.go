package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/istoo/myna/internal/provider"
)

// myna provider add <preset> without its key names the command that adds
// it, with the preset's own id, rather than only saying a key is needed.
func TestAddWithoutKeySaysTheCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	checked := 0
	for _, pr := range provider.Presets() {
		p, err := provider.FromPreset(pr.ID)
		if err != nil || pr.NoKey || pr.Endpoint != "" || p.IsVertex() {
			continue // a local server takes no key; a resource's address is asked first
		}
		err = addProvider([]string{pr.ID})
		if err == nil {
			t.Fatalf("%s was added with no key", pr.ID)
		}
		if !strings.Contains(err.Error(), "needs an API key") {
			continue // refused for another reason before the key
		}
		checked++
		if want := "myna provider add " + pr.ID + " <key>"; !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %q doesn't say %q", pr.ID, err, want)
		}
	}
	// most presets ask for a key, so a loop that checks none can't pass
	if checked < 30 {
		t.Errorf("only %d presets were checked for the key", checked)
	}
}
