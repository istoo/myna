package autostart

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// On, the system's record names this program and the tray; off, it's gone,
// and turning it off again is no error.
func TestSet(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the Run key is the real user's")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", home+"/.config")
	// Linux's system folders, a package's entry in them: the test's own
	t.Setenv("XDG_CONFIG_DIRS", home+"/etc/xdg")
	t.Setenv("APPIMAGE", "/opt/my apps/owl.AppImage")
	if Enabled() != Off {
		t.Fatal("on before it was turned on")
	}
	if err := Set(true); err != nil {
		t.Fatal(err)
	}
	if Enabled() == Off {
		t.Fatal("not on after Set(true)")
	}
	path := record()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	arg := Arg
	if runtime.GOOS == "android" {
		arg = "serve"
	}
	for _, want := range []string{"/opt/my apps/owl.AppImage", arg} {
		if !strings.Contains(string(b), want) {
			t.Errorf("%s lacks %q:\n%s", path, want, b)
		}
	}
	if err := Set(false); err != nil || Enabled() != Off {
		t.Fatalf("still on after Set(false): %v", err)
	}
	if err := Set(false); err != nil {
		t.Fatal(err)
	}
}

// This fork wrote its entry as myna before it was owl: Refresh takes the
// old one out, so two of them don't open at login and fight for one port.
func TestRefreshDropsMynaEntry(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the Run key is the real user's")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	legacy := filepath.Join(home, ".config", "autostart", "myna.desktop")
	if runtime.GOOS == "darwin" {
		legacy = filepath.Join(home, "Library", "LaunchAgents", "com.istoo.myna.plist")
	}
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Refresh(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("myna's entry is still there: %v", err)
	}
}
