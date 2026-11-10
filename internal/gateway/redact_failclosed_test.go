package gateway

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/istoo/owl/internal/settings"
)

// A settings.json that can't be read is unknown, not "off": masking stays
// on and a request is refused rather than sent as written (#4130).
func TestRedactionWantedWhenSettingsUnreadable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	dir := filepath.Dir(settings.Path())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	// no file: a new install, and masking is off by default
	if redactionWanted() {
		t.Fatal("a missing settings.json asks for masking")
	}
	if err := os.WriteFile(settings.Path(), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !redactionWanted() {
		t.Fatal("an unreadable settings.json read as masking off")
	}
}
