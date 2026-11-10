package update

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func writeZip(t *testing.T, path string, names ...string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	z := zip.NewWriter(f)
	for _, n := range names {
		w, err := z.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte("x"))
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()
}

// The bundle is unpacked before its signature is checked: an entry that
// leaves the folder is refused first (#4134).
func TestZipSafeRefusesEscapingEntry(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.zip")
	writeZip(t, good, "owl.app/Contents/Info.plist", "owl.app/Contents/MacOS/owl")
	if err := zipSafe(good); err != nil {
		t.Fatalf("a plain bundle: %v", err)
	}
	for i, name := range []string{"../evil", "/tmp/evil", "owl.app/../../evil", `..\evil`} {
		p := filepath.Join(dir, "bad"+string(rune('a'+i))+".zip")
		writeZip(t, p, name)
		if err := zipSafe(p); err == nil {
			t.Errorf("%q was accepted", name)
		}
	}
}
