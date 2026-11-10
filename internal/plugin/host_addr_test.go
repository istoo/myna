package plugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/istoo/owl/internal/settings"
)

// A plugin with a service of its own is told where owl's gateway is:
// Magiczerowxy/owl-chat's chat server asks OWL_ADDR, then 3425, so
// with the port set in Settings its model list was empty and every
// message failed (yuesh1_ on X, Magiczerowxy/owl-chat#1). The host is
// told Settings' port, started again when it moves, and a OWL_ADDR
// owl runs under, on loopback when it listens on every interface.
func TestHostToldTheGatewayAddress(t *testing.T) {
	sandbox(t)
	t.Setenv("OWL_ADDR", "")
	os.Unsetenv("OWL_ADDR")
	was := proxyLookNone
	proxyLookNone = 0
	t.Cleanup(func() { proxyLookNone = was })
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	out := filepath.Join(t.TempDir(), "addr")
	t.Setenv("ADDR_PROBE_OUT", out)
	dir := filepath.Join(t.TempDir(), "addr-probe")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"package.json": `{"name": "addr-probe", "version": "0.0.1", "type": "module", "main": "./index.mjs"}`,
		// as owl-chat's index.mjs, which starts its server on import
		"index.mjs": `import { writeFileSync } from "node:fs"
writeFileSync(process.env.ADDR_PROBE_OUT, process.env.OWL_ADDR ?? "unset")
export default { id: "addr-probe", server: async () => ({}) }
`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	setPort := func(p int) {
		t.Helper()
		s := settings.Load()
		s.Port = p
		if err := settings.Save(s); err != nil {
			t.Fatal(err)
		}
	}
	told := func() string {
		t.Helper()
		if err := Warm(ctx); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(out)
		if err != nil {
			t.Fatalf("the plugin didn't load: %v", err)
		}
		return string(b)
	}

	setPort(3591)
	if _, err := Add(ctx, dir); err != nil {
		t.Fatal(err)
	}
	if got := told(); got != "127.0.0.1:3591" {
		t.Fatalf("with Settings' port 3591 the plugin was told OWL_ADDR=%s", got)
	}
	// the port moves in Settings: the page opened next has a host told so
	setPort(3592)
	if got := told(); got != "127.0.0.1:3592" {
		t.Errorf("after the port moved to 3592 the plugin was told OWL_ADDR=%s", got)
	}
	// owl's own OWL_ADDR comes first, reached on loopback
	t.Setenv("OWL_ADDR", "0.0.0.0:4999")
	Restart()
	if got := told(); got != "127.0.0.1:4999" {
		t.Errorf("under OWL_ADDR=0.0.0.0:4999 the plugin was told OWL_ADDR=%s", got)
	}
}
