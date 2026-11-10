package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/istoo/owl/internal/edit"
)

// OpenCode reads its config from $OPENCODE_CONFIG_DIR when that is set
// (OpenCode 2 only from there), so that is the file owl edits (#321).
func TestOpenCodeConfigDir(t *testing.T) {
	home, _ := codexHome(t, "", "")
	dir := filepath.Join(home, "elsewhere")
	t.Setenv("OPENCODE_CONFIG_DIR", dir)
	oc := opencode(home, filepath.Join(home, ".config"))
	if want := filepath.Join(dir, "opencode.json"); oc.Path != want {
		t.Fatalf("path = %s, want %s", oc.Path, want)
	}
	if err := oc.Apply("model", "owl/fake/m1"); err != nil {
		t.Fatal(err)
	}
	if v, _ := edit.GetJSON(filepath.Join(dir, "opencode.json"), "model"); v != "owl/fake/m1" {
		t.Fatalf("model = %q", v)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "opencode", "opencode.json")); err == nil {
		t.Fatal("wrote ~/.config/opencode, which OpenCode doesn't read with OPENCODE_CONFIG_DIR set")
	}
	// set but blank, as OpenCode takes it: the usual folder
	t.Setenv("OPENCODE_CONFIG_DIR", " ")
	if oc := opencode(home, filepath.Join(home, ".config")); oc.Path != filepath.Join(home, ".config", "opencode", "opencode.json") {
		t.Fatalf("blank: %s", oc.Path)
	}
}

// OpenChamber sends each new session its own default model ahead of
// OpenCode's: owl sets that one, in preferences.json (which wins) and
// its copy in settings.json, keeping the rest of both, and puts owl's
// provider in the OpenCode config OpenChamber's OpenCode reads.
func TestOpenChamberModel(t *testing.T) {
	home, _ := codexHome(t, "", "")
	ocDir := filepath.Join(home, "oc")
	t.Setenv("OPENCODE_CONFIG_DIR", ocDir)
	dir := filepath.Join(home, ".config", "openchamber")
	prefs, settings := filepath.Join(dir, "preferences.json"), filepath.Join(dir, "settings.json")
	writeFile(t, prefs, `{"version":1,"fields":{"themeId":{"value":"flexoki","updatedAt":5},"defaultModel":{"value":"anthropic/claude-sonnet-5","updatedAt":7},"defaultVariant":{"value":"high","updatedAt":7}}}`)
	writeFile(t, settings, `{"lastDirectory":"/work","themeId":"flexoki","defaultModel":"anthropic/claude-sonnet-5","defaultVariant":"high"}`)
	ch := openChamber(home, filepath.Join(home, ".config"))
	if !ch.Detected() || ch.Path != prefs {
		t.Fatalf("detected %v, path %s", ch.Detected(), ch.Path)
	}
	if v := ch.Field("model").Get(); v != "anthropic/claude-sonnet-5" {
		t.Fatalf("model = %q", v)
	}
	if err := ch.Apply("model", "owl/fake/m1"); err != nil {
		t.Fatal(err)
	}
	p, s := readFile(prefs), readFile(settings)
	if v, _ := edit.GetJSON(prefs, "fields.defaultModel.value"); v != "owl/fake/m1" {
		t.Fatalf("preferences: %s", p)
	}
	if v, _ := edit.GetJSON(prefs, "fields.defaultModel.updatedAt"); v == "7" || v == "" {
		t.Fatalf("stamp not renewed: %s", p)
	}
	if v, _ := edit.GetJSON(settings, "defaultModel"); v != "owl/fake/m1" {
		t.Fatalf("settings: %s", s)
	}
	if strings.Contains(p+s, "defaultVariant") || !strings.Contains(p, "flexoki") || !strings.Contains(s, `"/work"`) {
		t.Fatalf("other keys: %s\n%s", p, s)
	}
	cfg := filepath.Join(ocDir, "opencode.json")
	if _, ok := edit.GetJSON(cfg, "provider.owl.options.baseURL"); !ok {
		t.Fatalf("no owl provider in OpenCode's config: %s", readFile(cfg))
	}
	if v := ch.Field("model").Get(); v != "owl/fake/m1" {
		t.Fatalf("model = %q", v)
	}
	if d := ch.Drift(); d != nil {
		t.Fatalf("drift: %+v", d)
	}
	// OpenCode stepping off owl leaves the provider OpenChamber sends to
	oc := opencode(home, filepath.Join(home, ".config"))
	if err := oc.Apply("model", "owl/fake/m1"); err != nil {
		t.Fatal(err)
	}
	if err := oc.Apply("model", ""); err != nil {
		t.Fatal(err)
	}
	if _, ok := edit.GetJSON(cfg, "provider.owl"); !ok {
		t.Fatal("OpenCode took out the provider OpenChamber's model needs")
	}
	// the small model: an override of OpenChamber's, not its default
	if err := ch.Apply("small", "owl/fake/m1"); err != nil {
		t.Fatal(err)
	}
	if v, _ := edit.GetJSON(prefs, "fields.smallModelUseDefault.value"); v != "false" {
		t.Fatalf("small: %s", readFile(prefs))
	}
	if v := ch.Field("small").Get(); v != "owl/fake/m1" {
		t.Fatalf("small = %q", v)
	}
	// both cleared: OpenChamber is left to OpenCode's config, and owl's
	// provider goes with nothing left on it
	if err := ch.Apply("model", ""); err != nil {
		t.Fatal(err)
	}
	if err := ch.Apply("small", ""); err != nil {
		t.Fatal(err)
	}
	if p, s := readFile(prefs), readFile(settings); strings.Contains(p+s, "defaultModel") || strings.Contains(p+s, "smallModel") || !strings.Contains(p, "flexoki") {
		t.Fatalf("cleared: %s\n%s", p, s)
	}
	if _, ok := edit.GetJSON(cfg, "provider.owl"); ok {
		t.Fatalf("provider left behind: %s", readFile(cfg))
	}
}

// A build of OpenChamber from before preferences.json reads settings.json
// alone, and a newer one seeds preferences.json from it: with no
// preferences.json owl writes settings.json and makes none. One it
// can't read, OpenChamber doesn't write either, nor does owl.
func TestOpenChamberSettingsOnly(t *testing.T) {
	home, _ := codexHome(t, "", "")
	dir := filepath.Join(home, "chamber")
	t.Setenv("OPENCHAMBER_DATA_DIR", dir)
	settings := filepath.Join(dir, "settings.json")
	writeFile(t, settings, `{"themeId":"flexoki"}`)
	ch := openChamber(home, filepath.Join(home, ".config"))
	if ch.Dir != dir {
		t.Fatalf("dir = %s", ch.Dir)
	}
	if err := ch.Apply("model", "owl/fake/m1"); err != nil {
		t.Fatal(err)
	}
	if v, _ := edit.GetJSON(settings, "defaultModel"); v != "owl/fake/m1" || !strings.Contains(readFile(settings), "flexoki") {
		t.Fatalf("settings: %s", readFile(settings))
	}
	if _, err := os.Stat(filepath.Join(dir, "preferences.json")); err == nil {
		t.Fatal("made a preferences.json")
	}
	if _, ok := edit.GetJSON(filepath.Join(home, ".config", "opencode", "opencode.json"), "provider.owl"); !ok {
		t.Fatal("no owl provider in OpenCode's config")
	}
	bad := `{"fields":[]}`
	writeFile(t, filepath.Join(dir, "preferences.json"), bad)
	if err := ch.Apply("model", ""); err == nil {
		t.Fatal("wrote beside a preferences.json OpenChamber can't read")
	}
	if readFile(filepath.Join(dir, "preferences.json")) != bad {
		t.Fatal("preferences.json changed")
	}
}

// OpenCode on owl/… with a model its copy of owl's list doesn't have
// (a list from before the provider had it) is still on owl, and
// disconnecting takes it off: it read as not connected, and Disconnect left
// it on owl's provider.
func TestOpenCodeOnAModelNotInItsList(t *testing.T) {
	home, _ := codexHome(t, "", "")
	oc := opencode(home, filepath.Join(home, ".config"))
	if err := oc.Apply("model", "owl/fake/m1"); err != nil {
		t.Fatal(err)
	}
	if err := edit.SetJSON(oc.Path, edit.KV{Path: "model", Value: "owl/fake/newer"}); err != nil {
		t.Fatal(err)
	}
	if !oc.Wired() {
		t.Fatal("not connected")
	}
	if err := oc.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if v, _ := edit.GetJSON(oc.Path, "model"); strings.HasPrefix(v, "owl/") {
		t.Fatalf("still on owl: %s", readFile(oc.Path))
	}
	if _, ok := edit.GetJSON(oc.Path, "provider.owl"); ok {
		t.Fatalf("owl's provider left: %s", readFile(oc.Path))
	}
}
