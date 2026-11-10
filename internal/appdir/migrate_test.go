package appdir

import (
	"os"
	"path/filepath"
	"testing"
)

// migrateFrom runs what Config and Cache do at a first start: the first of
// the folders an older build kept its files in that holds anything, copied
// into owl's.
func migrateFrom(t *testing.T, home, dst string, legacy func(string) []string) string {
	t.Helper()
	for _, from := range legacy(home) {
		if src := migrate(from, dst); src != "" {
			return src
		}
	}
	return ""
}

// This fork was myna before it was owl: its sign-ins and settings are in
// ~/.config/myna, and the first start as owl brings them over.
func TestMigrateFromMyna(t *testing.T) {
	home := root(t)
	cfg := filepath.Join(home, ".config")
	touch(t, filepath.Join(cfg, "myna", "providers.json"))
	dst := filepath.Join(cfg, "owl")

	if src := migrateFrom(t, home, dst, legacyConfigs); src != filepath.Join(cfg, "myna") {
		t.Fatalf("migrated from %q", src)
	}
	if _, err := os.Stat(filepath.Join(dst, "providers.json")); err != nil {
		t.Fatalf("owl's folder doesn't hold myna's file: %v", err)
	}
}

// A user who came from upstream magpie and never ran myna still comes over.
func TestMigrateFromMagpie(t *testing.T) {
	home := root(t)
	cfg := filepath.Join(home, ".config")
	touch(t, filepath.Join(cfg, "magpie", "providers.json"))
	dst := filepath.Join(cfg, "owl")

	if src := migrateFrom(t, home, dst, legacyConfigs); src != filepath.Join(cfg, "magpie") {
		t.Fatalf("migrated from %q", src)
	}
	if _, err := os.Stat(filepath.Join(dst, "providers.json")); err != nil {
		t.Fatalf("owl's folder doesn't hold magpie's file: %v", err)
	}
}

// A folder holding both copies myna's, the newer, and never puts one
// over what owl already has.
func TestMigrateMynaBeforeMagpieAndNeverOverOwl(t *testing.T) {
	home := root(t)
	cfg := filepath.Join(home, ".config")
	touch(t, filepath.Join(cfg, "myna", "myna.json"))
	touch(t, filepath.Join(cfg, "magpie", "magpie.json"))
	dst := filepath.Join(cfg, "owl")

	if src := migrateFrom(t, home, dst, legacyConfigs); src != filepath.Join(cfg, "myna") {
		t.Fatalf("migrated from %q", src)
	}
	if _, err := os.Stat(filepath.Join(dst, "magpie.json")); err == nil {
		t.Fatal("magpie's files came over beside myna's")
	}

	// a folder owl already writes in is its own, whatever myna's holds
	held := filepath.Join(cfg, "owl-held")
	touch(t, filepath.Join(held, "settings.json"))
	if src := migrate(filepath.Join(cfg, "myna"), held); src != "" {
		t.Fatalf("copied over an owl folder that had files, from %q", src)
	}
	if _, err := os.Stat(filepath.Join(held, "myna.json")); err == nil {
		t.Fatal("myna's file went into a folder that had owl's own")
	}
}

// The cache comes over the same way, myna's before magpie's.
func TestMigrateCacheFromMyna(t *testing.T) {
	home := root(t)
	cache := filepath.Join(home, ".cache")
	touch(t, filepath.Join(cache, "myna", "models.json"))
	dst := filepath.Join(cache, "owl")

	if src := migrateFrom(t, home, dst, legacyCaches); src != filepath.Join(cache, "myna") {
		t.Fatalf("migrated from %q", src)
	}
	if _, err := os.Stat(filepath.Join(dst, "models.json")); err != nil {
		t.Fatalf("owl's cache doesn't hold myna's file: %v", err)
	}
}
