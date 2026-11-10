package appdir

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

// This build keeps its files in owl's own folders, but a user coming from
// upstream magpie has theirs there: the first start copies them over, once,
// so the sign-ins, providers and settings are all there. The copy goes to a
// temporary folder that is renamed into place, so a start that is killed
// part way leaves nothing half-written to be trusted next time.

// The folder the upstream build kept its files in, which this one copies
// from. Spelled out here rather than derived from owl's name: it is
// history, fixed.
const legacyUpName = "magpie"

func legacyConfigs(home string) []string {
	return []string{filepath.Join(home, ".config", legacyUpName)}
}

func legacyCaches(home string) []string {
	return []string{filepath.Join(home, ".cache", legacyUpName)}
}

// migrate copies src to dst once, when dst holds nothing yet. It reports
// what it did, "" when there was nothing to do.
func migrate(src, dst string) string {
	if src == "" || dst == "" || filepath.Clean(src) == filepath.Clean(dst) {
		return ""
	}
	if _, err := os.Stat(src); err != nil {
		return ""
	}
	if es, err := os.ReadDir(dst); err == nil && len(es) > 0 {
		return ""
	}
	tmp := dst + ".migrating"
	os.RemoveAll(tmp)
	if err := copyTree(src, tmp); err != nil {
		os.RemoveAll(tmp)
		log.Printf("appdir: couldn't copy %s to %s: %v", src, dst, err)
		return ""
	}
	if err := os.Rename(tmp, dst); err != nil {
		// dst may have been made by this process meanwhile
		if es, rerr := os.ReadDir(dst); rerr == nil && len(es) > 0 {
			os.RemoveAll(tmp)
			return ""
		}
		os.RemoveAll(dst)
		if err := os.Rename(tmp, dst); err != nil {
			os.RemoveAll(tmp)
			log.Printf("appdir: couldn't move %s into place: %v", dst, err)
			return ""
		}
	}
	return src
}

// copyTree copies src into the new folder dst, links and modes kept.
// Locks and sockets are runtime state, not the user's: they are left out.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o700)
		}
		to := filepath.Join(dst, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(to, 0o700)
		case d.Type()&os.ModeSymlink != 0:
			target, err := os.Readlink(p)
			if err != nil {
				return err
			}
			return os.Symlink(target, to)
		case d.Type().IsRegular():
			if skipRuntimeFile(d.Name()) {
				return nil
			}
			return copyFile(p, to)
		default:
			return nil // sockets and devices aren't files of owl's
		}
	})
}

// skipRuntimeFile says whether a file is one a running owl made rather
// than something the user would miss.
func skipRuntimeFile(name string) bool {
	switch filepath.Ext(name) {
	case ".lock", ".sock":
		return true
	}
	return false
}

func copyFile(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	fi, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fi.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("copy %s: %w", from, err)
	}
	return out.Close()
}
