//go:build darwin && cgo && !nogui

package gui

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// owl out of the Dock, activated by its quick panel, opens its window
// and joins the Dock (Settings › Show in Dock › while the window is open): it leads
// Command-Tab's order, so one Command-Tab away comes back to it (#1381). It
// joined while active and stayed last, behind every app, though in front.
// The order is the Mac's front-to-back list, which `lsappinfo` prints.
func TestDockJoinLeadsCommandTab(t *testing.T) {
	out, err := runAppKit(t, 30*time.Second, []string{"OWL_TEST_DOCK_ORDER=1"})
	if strings.Contains(string(out), "dock order: skip") {
		t.Skipf("this session can't keep an app in front:\n%s", out)
	}
	if err != nil || !strings.Contains(string(out), "dock order: ok") {
		t.Fatalf("dock order: %v\n%s", err, out)
	}
}

// dockVerdict is what a check of owl's place in Command-Tab's order came
// to: "" when it is first; "skip" when another app took the front while the
// check ran (one the user was using), where the order says nothing about
// owl's own, as #1651 is about owl front and not first; and why when owl
// is front but not first. Both outcomes are checked without a session of
// any kind (TestDockVerdict).
func dockVerdict(first, front bool) string {
	if first {
		return ""
	}
	if !front {
		return "skip"
	}
	return "owl isn't first in Command-Tab's order"
}

// TestDockVerdict checks each of the three outcomes, including the skip a
// session that can't keep an app in front gets.
func TestDockVerdict(t *testing.T) {
	for _, c := range []struct {
		name         string
		first, front bool
		want         string
	}{
		{"first", true, true, ""},
		{"another app took the front", false, false, "skip"},
		{"front but not first", false, true, "owl isn't first in Command-Tab's order"},
	} {
		if got := dockVerdict(c.first, c.front); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

// lsFirst says whether pid's app leads the front-to-back order, and whether
// it is the front app at all.
func lsFirst(pid int) (first, front bool, out string) {
	me, err := exec.Command("lsappinfo", "find", "pid="+strconv.Itoa(pid)).Output()
	asn := strings.TrimSpace(string(me))
	if i := strings.Index(asn, "-\""); i > 0 {
		asn = asn[:i]
	}
	if err != nil || asn == "" {
		return false, false, fmt.Sprintf("no ASN for pid %d: %v", pid, err)
	}
	f, _ := exec.Command("lsappinfo", "front").Output()
	list, _ := exec.Command("lsappinfo", "visibleProcessList").Output()
	front = strings.HasPrefix(strings.TrimSpace(string(f)), asn)
	first = strings.HasPrefix(strings.TrimSpace(string(list)), asn+"-")
	return first, front, fmt.Sprintf("me %s, front %s, order %.300s", asn, strings.TrimSpace(string(f)), list)
}

func init() {
	if os.Getenv("OWL_TEST_DOCK_ORDER") != "1" {
		return
	}
	app := application.New(application.Options{
		Name: "owl dock order regression",
		Mac:  application.MacOptions{ActivationPolicy: application.ActivationPolicyAccessory},
		Assets: application.AssetOptions{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<p>owl</p>`)
		})},
	})
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		go func() {
			defer app.Quit()
			// The quick panel's Focus activates owl out of the Dock, and
			// its Open (openMain) joins the Dock, then shows the window. The
			// Mac makes owl the front app at once, but NSApp says it is
			// active only once the main thread has taken the activation in,
			// which on a busy Mac can be after the join (#1651). Both are
			// done in one turn of the main thread here, so the join always
			// comes before NSApp knows.
			skip := application.InvokeSyncWithResult(func() bool {
				activateApp()
				deadline := time.Now().Add(5 * time.Second)
				for _, front, _ := lsFirst(os.Getpid()); !front; _, front, _ = lsFirst(os.Getpid()) {
					if time.Now().After(deadline) {
						return true
					}
					time.Sleep(100 * time.Millisecond)
				}
				setDock(true, true)
				w := app.Window.NewWithOptions(application.WebviewWindowOptions{Name: "main", Width: 300, Height: 200})
				w.Show()
				w.Focus()
				return false
			})
			if skip {
				fmt.Println("dock order: skip")
				return
			}
			var first bool
			var got string
			for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
				if first, _, got = lsFirst(os.Getpid()); first {
					break
				}
			}
			_, front, _ := lsFirst(os.Getpid())
			switch v := dockVerdict(first, front); v {
			case "":
				fmt.Println("dock order: ok")
			case "skip":
				fmt.Println("dock order: skip")
			default:
				fmt.Printf("dock order: %s: %s\n", v, got)
			}
		}()
	})
	if err := app.Run(); err != nil {
		panic(err)
	}
	os.Exit(0)
}
