package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// With myna Codex's provider, a model of Codex's own on the Agents page
// says what picking it does (#1269, shenghsi: "gpt-5.5 经 myna", and
// Codex was taken off myna the moment it was picked). With no ChatGPT
// account in myna to serve it, it goes straight to OpenAI and says so;
// with one, it is picked as myna serves it and Codex stays on myna.
func TestCodexProviderOwnModelSaysItsPath(t *testing.T) {
	ownOf := func(a *Agent) []Option {
		var out []Option
		for _, o := range a.Fields[0].Options(nil) {
			if o.Group == "OpenAI" && (o.Label == "gpt-a" || o.Value == "gpt-a") {
				out = append(out, o)
			}
		}
		return out
	}
	cache := []byte(`{"models":[{"slug":"gpt-a","display_name":"A","priority":1}]}`)

	t.Run("no ChatGPT account", func(t *testing.T) {
		home, read := codexHome(t, `{"OPENAI_API_KEY":"sk-x"}`, "")
		os.WriteFile(filepath.Join(home, ".codex", "models_cache.json"), cache, 0o644)
		cx := codex(home)
		if err := cx.Fields[0].Set("fake/m1"); err != nil {
			t.Fatal(err)
		}
		if cfg := read(); !strings.Contains(cfg, `model_provider = "myna"`) {
			t.Fatalf("myna isn't Codex's provider:\n%s", cfg)
		}
		own := ownOf(codex(home))
		if len(own) != 1 {
			t.Fatalf("Codex's own gpt-a: %+v", own)
		}
		if o := own[0]; o.Via || o.Direct != "OpenAI" || o.Value != "gpt-a" {
			t.Fatalf("picking gpt-a takes Codex off myna, yet it reads %+v", o)
		}
	})

	t.Run("ChatGPT account out of allowance", func(t *testing.T) {
		home, read := codexHome(t, `{"tokens":{"access_token":"x","id_token":"x.e30.x"}}`, "")
		os.WriteFile(filepath.Join(home, ".codex", "models_cache.json"), cache, 0o644)
		codexUsedUp = func() bool { return true }
		cx := codex(home)
		if err := cx.Fields[0].Set("fake/m1"); err != nil {
			t.Fatal(err)
		}
		if cfg := read(); !strings.Contains(cfg, `model_provider = "myna"`) {
			t.Fatalf("myna isn't Codex's provider:\n%s", cfg)
		}
		own := ownOf(codex(home))
		if len(own) != 1 || own[0].Value != "codex/gpt-a" || !own[0].Via {
			t.Fatalf("Codex's own gpt-a, served by myna: %+v", own)
		}
		if err := codex(home).Fields[0].Set(own[0].Value); err != nil {
			t.Fatal(err)
		}
		if cfg := read(); !strings.Contains(cfg, `model_provider = "myna"`) || !strings.Contains(cfg, `model = "codex/gpt-a"`) {
			t.Fatalf("picking it took Codex off myna:\n%s", cfg)
		}
	})
}
