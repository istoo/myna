package shortcut

import "testing"

func TestTarget(t *testing.T) {
	for in, want := range map[string]string{
		`C:\Tools\owl-windows-amd64.exe`:          `C:\Tools\owl-windows-amd64.exe`,
		`C:\Tools\owl.exe.old`:                    `C:\Tools\owl.exe`,
		`C:\Tools\owl.exe.old-3`:                  `C:\Tools\owl.exe`,
		`C:\Tools\owl.exe.new`:                    `C:\Tools\owl.exe`,
		`C:\Tools\OWL.EXE.OLD`:                    `C:\Tools\OWL.EXE`,
		`C:\old\owl.exe`:                          `C:\old\owl.exe`,
		`C:\Tools\owl.exe.older`:                  `C:\Tools\owl.exe.older`,
		`/usr/local/bin/owl`:                      `/usr/local/bin/owl`,
		`C:\Users\me\Downloads\owl (1).exe.old-2`: `C:\Users\me\Downloads\owl (1).exe`,
	} {
		if got := Target(in); got != want {
			t.Errorf("Target(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStale(t *testing.T) {
	exe := "/tools/owl.exe"
	for _, c := range []struct {
		target, dir string
		stale       bool
	}{
		{"", "", true},                          // no shortcut yet
		{"/tools/owl.exe", "/tools", false},    // already this one
		{"/TOOLS/Owl.EXE", "/Tools/", false},   // the same, in another case
		{"/downloads/owl.exe", "/tools", true}, // the exe was moved
		{"/tools/owl.exe", "/downloads", true}, // starts in another folder
		{"/tools/owl.exe", "", true},           // in no folder
	} {
		if got := Stale(c.target, c.dir, exe); got != c.stale {
			t.Errorf("Stale(%q, %q) = %v, want %v", c.target, c.dir, got, c.stale)
		}
	}
}
