package edit

import (
	"os"
	"path/filepath"
	"testing"
)

// A secret (sync.json's passphrase, autobackup.json's) is never on disk
// 0644 waiting for a caller to chmod it (#4133).
func TestWriteAtomicModeIsPrivateFromTheStart(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sync.json")
	if err := WriteAtomicMode(p, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(p); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("new: %v %v", fi.Mode(), err)
	}
	// an older file made 0644 is narrowed, not kept
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomicMode(p, []byte("{} "), 0o600); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(p); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("existing: %v %v", fi.Mode(), err)
	}
	// the plain write keeps what it did
	q := filepath.Join(dir, "plain.json")
	if err := WriteAtomic(q, []byte("{}")); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(q); err != nil || fi.Mode().Perm() != 0o644 {
		t.Fatalf("plain: %v %v", fi.Mode(), err)
	}
}
