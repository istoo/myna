package library

// myna's own MCP servers on whichever computer reads the library (#1439).
// InstallServer gives myna-image this myna's binary by full path, and
// that path is right only here: a library carried to another computer (a
// WebDAV or S3 sync, a backup put back) came with C:\Users\…\Downloads\
// myna-windows-amd64.exe on a Mac, or a Mac's app on Windows, and a
// myna moved out of Downloads left its old path behind. Agents were
// given a program that isn't there, and the page said it can't start.
//
// So a server that is myna's own — `mcp …` run by a program named
// myna — and whose program isn't on this computer is pointed at this
// myna as the library is read (heal), and a carried library names
// myna's own as plain "myna" (Collect), which each computer reads as
// its own. Two computers then carry the same library, and a sync doesn't
// write one's path over the other's each time. A myna program that is
// there (another build kept beside this one) is the user's pick and stays;
// a server run by any other program is never touched.

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/istoo/myna/internal/update"
)

// selfExe is this myna by a path that stays put; a var for tests.
var selfExe = update.Program

// ownServer is whether s is myna's own: `mcp …` run by a myna binary,
// this computer's or another's.
func ownServer(s *Server) bool {
	return !s.Remote() && len(s.Args) > 0 && s.Args[0] == "mcp" && mynaBinary(s.Command)
}

// mynaBinary is whether cmd names a myna program, by a path of any
// system's: myna, myna.exe, myna-windows-amd64.exe, the AppImage.
func mynaBinary(cmd string) bool {
	base := strings.ToLower(cmd[strings.LastIndexAny(cmd, `/\`)+1:])
	base = strings.TrimSuffix(base, ".exe")
	return base == mynaCommand || strings.HasPrefix(base, mynaCommand+"-")
}

// runsHere is whether cmd is a program by full path on this computer.
func runsHere(cmd string) bool {
	if !filepath.IsAbs(cmd) {
		return false
	}
	fi, err := os.Stat(cmd)
	return err == nil && !fi.IsDir()
}

// heal points myna's own servers whose program isn't here at this myna.
func (l *Library) heal() {
	exe := ""
	for _, s := range l.MCP {
		if s == nil || !ownServer(s) || runsHere(s.Command) {
			continue
		}
		if exe == "" {
			e, err := selfExe()
			if err != nil || e == "" {
				return
			}
			exe = e
		}
		s.Command = exe
	}
}

// carried is s as another computer is given it: myna's own by name.
func carried(s *Server) *Server {
	c := *s
	if ownServer(s) {
		c.Command = mynaCommand
	}
	return &c
}
