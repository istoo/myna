//go:build darwin && cgo

package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestDockFlapFrames writes the Dock tile's frames, cut from the icon of
// OWL_DOCK_APP (an installed owl.app), into OWL_DOCK_FRAMES, to
// look at by eye.
func TestDockFlapFrames(t *testing.T) {
	app, dir := os.Getenv("OWL_DOCK_APP"), os.Getenv("OWL_DOCK_FRAMES")
	if app == "" || dir == "" {
		t.Skip("set OWL_DOCK_APP and OWL_DOCK_FRAMES")
	}
	for f := 0; f <= 30; f++ {
		if !dockFramePNG(app, filepath.Join(dir, fmt.Sprintf("f%02d.png", f)), float64(f)/30) {
			t.Fatalf("frame %d not written", f)
		}
	}
}
