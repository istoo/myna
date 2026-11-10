package agent

// Command Code (commandcode.ai) keeps its settings in ~/.commandcode/
// settings.json, the model as "provider/model", and takes providers of the
// user's own in ~/.commandcode/providers.json. owl adds itself there as
// the provider "owl", keyless, with the catalog as its models; a model
// through owl is "owl/<provider>/<model>", which Command Code splits
// at the first "/". Command Code still wants its own sign-in for these.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/istoo/owl/internal/edit"
)

// ccEfforts are the reasoning levels Command Code knows.
var ccEfforts = []string{"low", "medium", "high", "xhigh", "max"}

func commandCode(home string) *Agent { return commandCodeIn(here(home)) }

// commandCodeIn is Command Code at a place: this machine's home, or a WSL
// distro's (see wsl.go), its provider naming the gateway as the distro
// reaches it.
func commandCodeIn(at place) *Agent {
	ccProviderJSON := func() any { return ccProviderJSONAt(at.v1()) }
	dir := at.join(".commandcode")
	path := filepath.Join(dir, "settings.json")
	providers := filepath.Join(dir, "providers.json")
	get := func(k string) string { v, _ := edit.GetJSON(path, k); return v }
	dropOwl := func() error {
		if _, err := os.Stat(providers); err != nil {
			return nil
		}
		return edit.DelJSON(providers, "provider."+owlID)
	}
	return &Agent{
		ID: "commandcode", Name: "Command Code", Icon: "commandcode", Aliases: []string{"command-code", "cmd"}, Spelled: prefixed,
		UA:  []string{"command-code", "commandcode"},
		Bin: "command-code", Dir: dir, Path: path,
		ListOnly: true,
		Sync: func() error {
			return syncJSON(providers, "provider."+owlID, theirsKept(providers, "provider."+owlID, ccProviderJSON, "models"))
		},
		Notice: func() string {
			notes := []string{noticeCommandcode1.String()}
			if Running(`(^|/)(cmd|cmdc|command-code|commandcode)( |$)`) {
				notes = append(notes, noticeCommandcode2.String())
			}
			return strings.Join(notes, " ")
		},
		Check: func() string {
			if !usesOwl(get("model")) {
				return ""
			}
			if p := get("modelProvider"); p != owlID {
				return "Command Code's modelProvider (settings.json) is " + orDefault(p) + ", so it no longer asks owl"
			}
			return wiringOff("Command Code", providers, func(k string) (string, bool) { return edit.GetJSON(providers, "provider."+owlID+"."+k) },
				"baseURL", at.v1())
		},
		Fields: []Field{{
			Key: "model", Label: "model",
			Get: func() string { return get("model") },
			Set: func(v string) error {
				if v == "" {
					keys := []string{"model"}
					if get("modelProvider") == owlID {
						keys = append(keys, "modelProvider")
					}
					if err := edit.DelJSON(path, keys...); err != nil {
						return err
					}
					return dropOwl()
				}
				if ref, ok := strings.CutPrefix(v, owlID+"/"); ok && isOwl(ref) {
					if err := edit.SetJSON(providers, edit.KV{Path: "provider." + owlID, Value: theirsKept(providers, "provider."+owlID, ccProviderJSON, "models")()}); err != nil {
						return err
					}
					return edit.SetJSON(path, edit.KV{Path: "model", Value: v}, edit.KV{Path: "modelProvider", Value: owlID})
				}
				if get("modelProvider") == owlID {
					if err := edit.DelJSON(path, "modelProvider"); err != nil {
						return err
					}
				}
				if err := edit.SetJSON(path, edit.KV{Path: "model", Value: v}); err != nil {
					return err
				}
				return dropOwl()
			},
			Options: func(cur map[string]string) []Option {
				var out []Option
				if v := cur["model"]; v != "" && !usesOwl(v) {
					out = append(out, Option{Value: v, Icon: modelIcon("", v)})
				}
				return withVia(out, "commandcode", owlID+"/")
			},
		}, {
			// settings.json's reasoningEffort keeps an effort for each model
			// (Command Code's /effort saves it); this is the current model's
			Key: "effort", Label: "effort", Closed: true,
			Get: func() string { return ccEffortMap(path)[get("model")] },
			Set: func(v string) error {
				model := get("model")
				if model == "" {
					return fmt.Errorf("pick Command Code's model first; it keeps an effort for each model")
				}
				efforts := ccEffortMap(path)
				if v == "" {
					delete(efforts, model)
				} else {
					efforts[model] = v
				}
				if len(efforts) == 0 {
					return edit.DelJSON(path, "reasoningEffort")
				}
				// written whole, as model ids hold dots
				return edit.SetJSON(path, edit.KV{Path: "reasoningEffort", Value: efforts})
			},
			Options: func(cur map[string]string) []Option {
				if ref, ok := strings.CutPrefix(cur["model"], owlID+"/"); ok {
					for _, m := range sharedModels("commandcode") {
						if m.ID != ref {
							continue
						}
						var efforts []string
						for _, x := range ccEfforts {
							if slices.Contains(m.Efforts, x) {
								efforts = append(efforts, x)
							}
						}
						if len(efforts) > 0 {
							return static(efforts...)
						}
					}
				}
				return static(ccEfforts...)
			},
		}},
	}
}

// ccProviderJSONAt is owl's entry in providers.json, for a Command Code
// reaching the gateway's /v1 at v1. The key is false: the gateway takes
// any, and Command Code refuses one written out.
func ccProviderJSONAt(v1 string) any {
	ms := map[string]any{}
	for _, m := range sharedModels("commandcode") {
		e := map[string]any{"name": m.Name}
		var efforts []string
		for _, x := range m.Efforts {
			if slices.Contains(ccEfforts, x) {
				efforts = append(efforts, x)
			}
		}
		// a model that thinks with no levels owl knows (Trae CN's
		// kimi-k3-max, #2380) is marked reasoning alone, which Command
		// Code takes for low, medium and high (its BYOK reference); the
		// gateway fits them to the model
		switch {
		case len(efforts) > 0:
			e["reasoning"] = true
			e["reasoningEfforts"] = efforts
		case len(m.Efforts) == 0 && m.Reasoning:
			e["reasoning"] = true
		}
		ms[m.ID] = e
	}
	return map[string]any{"name": "owl", "api": "openai-completions", "baseURL": v1, "apiKey": false, "models": ms}
}

// ccEffortMap is settings.json's reasoningEffort, model id to effort.
func ccEffortMap(path string) map[string]string {
	out := map[string]string{}
	b, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	var c struct {
		Efforts map[string]any `json:"reasoningEffort"`
	}
	if json.Unmarshal(b, &c) != nil {
		return out
	}
	for k, v := range c.Efforts {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

// what commandcode says after a change (notice.go)
var (
	noticeCommandcode1 = newNotice("Command Code wants its own sign-in (cmd login) even for models through owl.")
	noticeCommandcode2 = newNotice("It reads its settings at start-up — restart open Command Code sessions to use this.")
)
