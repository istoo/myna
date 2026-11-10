//go:build !windows

package agent

// desktopUnprotect: outside Windows owl opens nothing Claude Desktop
// sealed; what it holds stays unknown.
func desktopUnprotect([]byte) ([]byte, error) { return nil, errDesktopSealed }
