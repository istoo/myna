package autostart

import "testing"

// The Windows app installed itself from Downloads (update.Settle): the Run
// value opening the download opens the installed copy, with its arguments;
// one opening anything else, or written by hand without a path owl
// knows, is left.
func TestRepointed(t *testing.T) {
	from := `C:\Users\u\Downloads\owl-windows-amd64.exe`
	to := `C:\Users\u\AppData\Local\Programs\owl\owl.exe`
	for _, c := range []struct {
		v, want string
		ok      bool
	}{
		{`"` + from + `" tray`, `"` + to + `" tray`, true},
		{`"C:\USERS\U\DOWNLOADS\OWL-WINDOWS-AMD64.EXE" tray`, `"` + to + `" tray`, true},
		{`"` + from + `"`, `"` + to + `"`, true},
		{from + ` tray --x`, `"` + to + `" tray --x`, true},
		{`"C:\Tools\owl.exe" tray`, "", false},
		{`"` + from, "", false},
		{``, "", false},
	} {
		got, ok := repointed(c.v, from, to)
		if got != c.want || ok != c.ok {
			t.Errorf("repointed(%q) = %q, %v; want %q, %v", c.v, got, ok, c.want, c.ok)
		}
	}
}
