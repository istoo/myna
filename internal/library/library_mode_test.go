package library

import (
	"os"
	"path/filepath"
	"testing"
)

// library.json holds an MCP server's env and headers, its credentials:
// it is not left readable by another user, even in a folder owl made
// 0755 (#4131).
func TestLibraryFileIsPrivate(t *testing.T) {
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("USERPROFILE", h)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config"))
	l := &Library{}
	if err := l.save(); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path())
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Fatalf("library.json is %v, want 0600", perm)
	}
	di, err := os.Stat(filepath.Dir(path()))
	if err != nil {
		t.Fatal(err)
	}
	if perm := di.Mode().Perm(); perm != 0o700 {
		t.Fatalf("the settings folder is %v, want 0700", perm)
	}
}

// A project's repository can hold a symbolic link where owl writes
// (.gitignore -> ~/.zshrc): owl refuses rather than write through it
// (#4132).
func TestProjectWritesRefuseSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(target, []byte("kept\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, ".gitignore")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("no symbolic links here: %v", err)
	}
	p := &Project{Dir: dir, Placed: []string{"skill"}, MadeFiles: []string{"skill"}}
	if err := p.writeIgnore(); err == nil {
		t.Fatal("writeIgnore wrote through a symbolic link")
	}
	if b, _ := os.ReadFile(target); string(b) != "kept\n" {
		t.Fatalf("the link's target changed: %q", b)
	}
}
