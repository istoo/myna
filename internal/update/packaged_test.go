package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// pacman's local database as the AUR's owl-bin 0.1.2342-1 leaves it
// (its PKGBUILD installs /usr/bin/owl), beside a package that doesn't
// own owl.
func pacmanRoot(t *testing.T) string {
	root := t.TempDir()
	local := filepath.Join(root, "var/lib/pacman/local")
	writeFile(t, filepath.Join(local, "ALPM_DB_VERSION"), "9\n")
	writeFile(t, filepath.Join(local, "bash-5.3.3-2/files"), "%FILES%\nusr/\nusr/bin/\nusr/bin/bash\nusr/bin/sh\n\n%BACKUP%\netc/bash.bashrc\t0b5ab6a3\n")
	writeFile(t, filepath.Join(local, "bash-5.3.3-2/desc"), "%NAME%\nbash\n\n%VERSION%\n5.3.3-2\n")
	writeFile(t, filepath.Join(local, "owl-bin-0.1.2342-1/files"), "%FILES%\nusr/\nusr/bin/\nusr/bin/owl\nusr/share/\nusr/share/applications/\nusr/share/applications/owl.desktop\nusr/share/icons/hicolor/256x256/apps/owl.png\nusr/share/licenses/owl-bin/LICENSE\n\n")
	writeFile(t, filepath.Join(local, "owl-bin-0.1.2342-1/desc"), "%NAME%\nowl-bin\n\n%VERSION%\n0.1.2342-1\n\n%BASE%\nowl-bin\n")
	return root
}

func TestPackagedByPacman(t *testing.T) {
	root := pacmanRoot(t)
	p := packagedIn(root, "linux", "/usr/bin/owl")
	if p == nil || p.Manager != "pacman" || p.Package != "owl-bin" {
		t.Fatalf("/usr/bin/owl from owl-bin: %+v", p)
	}
	if c := p.Command(); c != "yay -S owl-bin" {
		t.Fatalf("command %q", c)
	}
	if s := p.Say(); !strings.Contains(s, "pacman") || !strings.Contains(s, "yay -S owl-bin") {
		t.Fatalf("said %q", s)
	}
	// /bin is /usr/bin on Arch: the same file
	if p := packagedIn(root, "linux", "/bin/owl"); p == nil || p.Package != "owl-bin" {
		t.Fatalf("/bin/owl: %+v", p)
	}
	// a copy in /usr/local, ~/.local/bin or /opt that no package lists is
	// the user's own, and updates itself
	for _, exe := range []string{"/usr/local/bin/owl", "/home/u/.local/bin/owl", "/opt/owl/owl", "/usr/lib/owl/owl"} {
		if p := packagedIn(root, "linux", exe); p != nil {
			t.Fatalf("%s: %+v", exe, p)
		}
	}
	// not Linux: pacman's files aren't read
	if p := packagedIn(root, "darwin", "/usr/bin/owl"); p != nil {
		t.Fatalf("darwin: %+v", p)
	}
}

// A package named otherwise is found too, its name from the folder when
// its desc can't be read.
func TestPackagedByPacmanOtherName(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "var/lib/pacman/local/ai-tools-2.0-3/files"), "%FILES%\nopt/ai/bin/owl\n")
	p := packagedIn(root, "linux", "/opt/ai/bin/owl")
	if p == nil || p.Package != "ai-tools" {
		t.Fatalf("%+v", p)
	}
}

// dpkg lists absolute paths, and on a merged /usr may list /bin for the
// /usr/bin the binary resolves to.
func TestPackagedByDpkg(t *testing.T) {
	root := t.TempDir()
	info := filepath.Join(root, "var/lib/dpkg/info")
	writeFile(t, filepath.Join(info, "coreutils.list"), "/.\n/bin\n/bin/ls\n")
	writeFile(t, filepath.Join(info, "owl:amd64.list"), "/.\n/usr\n/usr/bin\n/usr/bin/owl\n")
	p := packagedIn(root, "linux", "/usr/bin/owl")
	if p == nil || p.Manager != "dpkg" || p.Package != "owl" || p.Name() != "apt" || p.Command() != "sudo apt install --only-upgrade owl" {
		t.Fatalf("%+v", p)
	}
	if p := packagedIn(root, "linux", "/usr/bin/ls"); p == nil || p.Package != "coreutils" {
		t.Fatalf("merged /usr: %+v", p)
	}
	if p := packagedIn(root, "linux", "/usr/sbin/owl"); p != nil {
		t.Fatalf("not listed: %+v", p)
	}
}

func TestPackagedByRpm(t *testing.T) {
	root := t.TempDir()
	was := rpmOwner
	t.Cleanup(func() { rpmOwner = was })
	asked := ""
	rpmOwner = func(path string) string { asked = path; return "owl" }
	// no rpm database: rpm isn't asked
	if p := packagedIn(root, "linux", "/usr/bin/owl"); p != nil || asked != "" {
		t.Fatalf("no rpm here: %+v, asked %q", p, asked)
	}
	if err := os.MkdirAll(filepath.Join(root, "var/lib/rpm"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := packagedIn(root, "linux", "/usr/bin/owl")
	if p == nil || p.Manager != "rpm" || asked != "/usr/bin/owl" || p.Command() != "sudo dnf upgrade owl" {
		t.Fatalf("%+v, asked %q", p, asked)
	}
	rpmOwner = func(string) string { return "" }
	if p := packagedIn(root, "linux", "/usr/bin/owl"); p != nil {
		t.Fatalf("not owned: %+v", p)
	}
}

func TestPackagedByHomebrewAndNix(t *testing.T) {
	root := t.TempDir()
	for exe, want := range map[string]string{
		"/opt/homebrew/Cellar/owl/0.1.126/bin/owl":              "homebrew owl brew upgrade owl",
		"/home/linuxbrew/.linuxbrew/Cellar/owl/0.1.126/bin/owl": "homebrew owl brew upgrade owl",
		"/nix/store/abc123-owl-0.1.2342/bin/owl":                "nix  ",
	} {
		goos := "linux"
		if strings.HasPrefix(exe, "/opt/homebrew") {
			goos = "darwin"
		}
		p := packagedIn(root, goos, exe)
		if p == nil || p.Manager+" "+p.Package+" "+p.Command() != want {
			t.Fatalf("%s: %+v", exe, p)
		}
	}
	if p := packagedIn(root, "darwin", "/Applications/owl.app/Contents/MacOS/owl"); p != nil {
		t.Fatalf("the app: %+v", p)
	}
}
