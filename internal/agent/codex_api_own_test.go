package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// On myna API, one of Codex's own models picked as its default keeps
// myna as Codex's provider, myna's models in its picker: the model goes
// as myna serves it on the ChatGPT account. Picking it used to take myna
// out, and every model of myna's left Codex's picker with it (#701). Back
// on ChatGPT, the model is Codex's own again, straight to OpenAI.
func TestCodexAPIKeepsMynaOnOwnModel(t *testing.T) {
	home, read := codexHome(t, `{"tokens":{"access_token":"x","id_token":"x.e30.x"}}`, "model = \"gpt-a\"\n")
	dir := filepath.Join(home, ".codex")
	os.WriteFile(filepath.Join(dir, "models_cache.json"), []byte(`{"models":[
		{"slug":"gpt-a","display_name":"A","priority":1},
		{"slug":"gpt-b","display_name":"B","priority":2}]}`), 0o644)
	catalogPath := filepath.Join(dir, "myna-models.json")
	wired := func(model string) {
		t.Helper()
		cfg := read()
		if !strings.Contains(cfg, `model_provider = "myna"`) || !strings.Contains(cfg, "model_catalog_json") ||
			!strings.Contains(cfg, "openai_base_url") || !strings.Contains(cfg, `model = "`+model+`"`) {
			t.Fatalf("not wired on %s:\n%s", model, cfg)
		}
		b, err := os.ReadFile(catalogPath)
		if err != nil || !strings.Contains(string(b), `"fake/m1"`) {
			t.Fatalf("myna's models not in Codex's catalog (%v):\n%s", err, b)
		}
	}
	cx := codex(home)
	login, model := cx.Field("login"), cx.Fields[0]

	// myna API chosen on Codex's own model wires myna in at once
	if err := login.Set("api"); err != nil {
		t.Fatal(err)
	}
	wired("codex/gpt-a")

	// a myna model, then Codex's own again: myna stays
	if err := model.Set("fake/m1"); err != nil {
		t.Fatal(err)
	}
	wired("fake/m1")
	if err := model.Set("gpt-b"); err != nil {
		t.Fatal(err)
	}
	wired("codex/gpt-b")

	// the picker offers Codex's own as they are set, under their names
	found := false
	for _, o := range model.Options(nil) {
		if o.Group == "OpenAI" && o.Value == "codex/gpt-b" && o.Label == "gpt-b" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Codex's own models in the picker: %+v", model.Options(nil))
	}
	if got := model.Get(); got != "codex/gpt-b" {
		t.Fatalf("model = %q", got)
	}

	// one myna doesn't serve is refused, the config left as it was
	before := read()
	if err := model.Set("gpt-zzz"); err == nil {
		t.Fatal("a model myna doesn't serve was taken on myna API")
	}
	if read() != before {
		t.Fatalf("a refused model changed the config:\n%s", read())
	}

	// back on ChatGPT: Codex's own model, straight to OpenAI
	if err := login.Set(""); err != nil {
		t.Fatal(err)
	}
	cfg := read()
	if !strings.Contains(cfg, `model = "gpt-b"`) || strings.Contains(cfg, "model_provider =") ||
		strings.Contains(cfg, "openai_base_url") || strings.Contains(cfg, "model_catalog_json") {
		t.Fatalf("back on ChatGPT:\n%s", cfg)
	}
	if err := model.Set("gpt-a"); err != nil {
		t.Fatal(err)
	}
	if cfg := read(); !strings.Contains(cfg, `model = "gpt-a"`) || strings.Contains(cfg, "openai_base_url") {
		t.Fatalf("own model on ChatGPT:\n%s", cfg)
	}
}

// A Codex an older myna left on its own model, unwired, while its sign-in
// is myna API (#701's config) is wired again at the next sync.
func TestCodexAPISyncWiresOwnModel(t *testing.T) {
	home, read := codexHome(t, `{"tokens":{"access_token":"x","id_token":"x.e30.x"}}`, "model = \"gpt-a\"\n")
	dir := filepath.Join(home, ".codex")
	os.WriteFile(filepath.Join(dir, "models_cache.json"), []byte(`{"models":[{"slug":"gpt-a","display_name":"A","priority":1}]}`), 0o644)
	cx := codex(home)
	// the choice alone, as the old code kept it on Codex's own model
	stash(map[string]string{here(home).key("codex.login"): "api"})
	if err := cx.Sync(); err != nil {
		t.Fatal(err)
	}
	if cfg := read(); !strings.Contains(cfg, `model_provider = "myna"`) || !strings.Contains(cfg, `model = "codex/gpt-a"`) {
		t.Fatalf("after sync:\n%s", cfg)
	}
}
