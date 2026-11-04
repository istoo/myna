package shortcut

import "testing"

func TestTarget(t *testing.T) {
	for in, want := range map[string]string{
		`C:\Tools\myna-windows-amd64.exe`:          `C:\Tools\myna-windows-amd64.exe`,
		`C:\Tools\myna.exe.old`:                    `C:\Tools\myna.exe`,
		`C:\Tools\myna.exe.old-3`:                  `C:\Tools\myna.exe`,
		`C:\Tools\myna.exe.new`:                    `C:\Tools\myna.exe`,
		`C:\Tools\MYNA.EXE.OLD`:                    `C:\Tools\MYNA.EXE`,
		`C:\old\myna.exe`:                          `C:\old\myna.exe`,
		`C:\Tools\myna.exe.older`:                  `C:\Tools\myna.exe.older`,
		`/usr/local/bin/myna`:                      `/usr/local/bin/myna`,
		`C:\Users\me\Downloads\myna (1).exe.old-2`: `C:\Users\me\Downloads\myna (1).exe`,
	} {
		if got := Target(in); got != want {
			t.Errorf("Target(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStale(t *testing.T) {
	exe := "/tools/myna.exe"
	for _, c := range []struct {
		target, dir string
		stale       bool
	}{
		{"", "", true},                            // no shortcut yet
		{"/tools/myna.exe", "/tools", false},    // already this one
		{"/TOOLS/Myna.EXE", "/Tools/", false},   // the same, in another case
		{"/downloads/myna.exe", "/tools", true}, // the exe was moved
		{"/tools/myna.exe", "/downloads", true}, // starts in another folder
		{"/tools/myna.exe", "", true},           // in no folder
	} {
		if got := Stale(c.target, c.dir, exe); got != c.stale {
			t.Errorf("Stale(%q, %q) = %v, want %v", c.target, c.dir, got, c.stale)
		}
	}
}
