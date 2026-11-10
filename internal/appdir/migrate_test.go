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

// A user who came from upstream magpie still comes over.
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

// A folder owl already writes in is its own: nothing is copied over it.
func TestMigrateNeverOverOwl(t *testing.T) {
	home := root(t)
	cfg := filepath.Join(home, ".config")
	touch(t, filepath.Join(cfg, "magpie", "magpie.json"))
	held := filepath.Join(cfg, "owl")
	touch(t, filepath.Join(held, "settings.json"))

	if src := migrate(filepath.Join(cfg, "magpie"), held); src != "" {
		t.Fatalf("copied over an owl folder that had files, from %q", src)
	}
	if _, err := os.Stat(filepath.Join(held, "magpie.json")); err == nil {
		t.Fatal("magpie's file went into a folder that had owl's own")
	}
}

// The cache comes over the same way.
func TestMigrateCacheFromMagpie(t *testing.T) {
	home := root(t)
	cache := filepath.Join(home, ".cache")
	touch(t, filepath.Join(cache, "magpie", "models.json"))
	dst := filepath.Join(cache, "owl")

	if src := migrateFrom(t, home, dst, legacyCaches); src != filepath.Join(cache, "magpie") {
		t.Fatalf("migrated from %q", src)
	}
	if _, err := os.Stat(filepath.Join(dst, "models.json")); err != nil {
		t.Fatalf("owl's cache doesn't hold magpie's file: %v", err)
	}
}
