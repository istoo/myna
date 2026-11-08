package agent

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/istoo/myna/internal/catalog"
	"github.com/istoo/myna/internal/provider"
	"github.com/istoo/myna/internal/settings"
)

func TestSpell(t *testing.T) {
	opts := func(prefix string) func(map[string]string) []Option {
		return func(map[string]string) []Option {
			return []Option{
				{Value: "gpt-6"},
				{Value: prefix + "copilot/gpt-6-sol", Ref: "copilot/gpt-6-sol"},
			}
		}
	}
	get := func() string { return "" }
	bare := &Agent{Fields: []Field{{Key: "model", Get: get, Options: opts("")}}}
	prefixed := &Agent{Fields: []Field{{Key: "model", Get: get, Options: opts("myna/")}}}
	for _, c := range []struct {
		a       *Agent
		in, out string
		err     bool
	}{
		{bare, "copilot/gpt-6-sol", "copilot/gpt-6-sol", false},
		{bare, "myna/copilot/gpt-6-sol", "copilot/gpt-6-sol", false},
		{prefixed, "copilot/gpt-6-sol", "myna/copilot/gpt-6-sol", false},
		{prefixed, "myna/copilot/gpt-6-sol", "myna/copilot/gpt-6-sol", false},
		{bare, "gpt-6", "gpt-6", false},
		{bare, "some-unlisted-model", "some-unlisted-model", false},
		{bare, "", "", false},
		{bare, "myna/copilot/nope", "", true},
		{prefixed, "myna/copilot/nope", "", true},
	} {
		got, err := c.a.Spell("model", c.in)
		if got != c.out || (err != nil) != c.err {
			t.Errorf("%q: got %q, %v; want %q, err %v", c.in, got, err, c.out, c.err)
		}
	}
}

// #2376: a model of a remote myna that the agent's picker doesn't offer,
// kept from it by myna visible or not on the remote's list yet, was
// taken as the agent's own: Pi read remote/group/smart as its provider
// "remote" and model "group/smart", and was taken out of myna. It is
// refused, and one the picker offers, or the agent's own, is spelled as
// before.
func TestSpellRefusesMynaModelNotOffered(t *testing.T) {
	syncHome(t)
	if err := provider.Save(provider.Provider{ID: "remote", Name: "Remote myna", Key: "k", Chat: "http://127.0.0.1:1/v1", Models: []string{"group/smart", "codex/gpt-5.5"}}); err != nil {
		t.Fatal(err)
	}
	if err := settings.Save(settings.Settings{Visible: map[string][]string{"pi": {"relay"}, "codex": {"relay"}}}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { settings.Save(settings.Settings{}) })
	home := os.Getenv("HOME")
	for _, a := range []*Agent{pi(home), codex(home)} {
		for _, v := range []string{"remote/group/smart", "remote/codex/gpt-5.5", "remote/group/new-pool", "group/nope"} {
			if got, err := a.Spell(a.Fields[0].Key, v); err == nil {
				t.Errorf("%s: %s spelled %q, want refused", a.ID, v, got)
			}
		}
		want := "relay/glm-4.6"
		if a.ID == "pi" {
			want = "myna/" + want
		}
		if got, err := a.Spell(a.Fields[0].Key, "relay/glm-4.6"); err != nil || got != want {
			t.Errorf("%s: relay/glm-4.6 spelled %q, %v; want %q", a.ID, got, err, want)
		}
	}
	// a provider of the agent's own that myna hasn't is the agent's
	if got, err := pi(home).Spell("model", "my-own/some-model"); err != nil || got != "my-own/some-model" {
		t.Errorf("pi's own: %q, %v", got, err)
	}
	// shown it every model, the remote's are offered and spelled as myna's
	if err := settings.Save(settings.Settings{}); err != nil {
		t.Fatal(err)
	}
	if got, err := pi(home).Spell("model", "remote/group/smart"); err != nil || got != "myna/remote/group/smart" {
		t.Errorf("pi, shown every model: %q, %v", got, err)
	}
}

// A closed field (an effort level, a switch) takes only the values its
// picker offers: `myna codex effort highish` wrote a level Codex refuses at
// start-up, and every agent's closed field had the same hole. One whose
// values aren't known here (nothing wired, a file unreadable) is left as
// typed, and a model field is a model the agent may reach on its own.
func TestClosedFieldsTakeOnlyTheirOptions(t *testing.T) {
	syncHome(t)
	// Devin's list is asked of its CLI: never this machine's binary here
	was := providerDevinFamilies
	providerDevinFamilies = func(context.Context) ([]provider.DevinFamily, error) {
		return nil, errors.New("no Devin list in a test")
	}
	t.Cleanup(func() { providerDevinFamilies = was })

	checked := 0
	for _, a := range All() {
		for _, f := range a.Fields {
			if !f.Closed {
				continue
			}
			bad := "zz-bogus-" + f.Key
			got, err := a.Spell(f.Key, bad)
			opts := f.Options(a.Values())
			if len(opts) == 0 {
				// unknown here: left as typed, as for a model
				if err != nil || got != bad {
					t.Errorf("%s %s, values unknown: %q, %v; want it left as typed", a.ID, f.Key, got, err)
				}
				continue
			}
			checked++
			if err == nil || got != "" {
				t.Errorf("%s %s took %q, want it refused", a.ID, f.Key, got)
				continue
			}
			if !strings.Contains(err.Error(), bad) || !strings.Contains(err.Error(), "try: myna "+a.ID+" "+f.Key+" ") {
				t.Errorf("%s %s refused with %q, want the value, the ones it takes and an example command", a.ID, f.Key, err)
			}
			// each value it offers is taken, in any case; "" is the default
			for _, o := range opts {
				if o.Value == "" {
					continue
				}
				if got, err := a.Spell(f.Key, o.Value); err != nil || got != o.Value {
					t.Errorf("%s %s offered %q: got %q, %v", a.ID, f.Key, o.Value, got, err)
				}
				if got, err := a.Spell(f.Key, strings.ToUpper(o.Value)); err != nil || got != o.Value {
					t.Errorf("%s %s offered %q in another case: got %q, %v", a.ID, f.Key, o.Value, got, err)
				}
				break
			}
			if got, err := a.Spell(f.Key, ""); err != nil || got != "" {
				t.Errorf("%s %s default: got %q, %v", a.ID, f.Key, got, err)
			}
		}
	}
	// the fields whose values are known without wiring: the static levels
	// and the ones from a model the agent is on
	if checked < 15 {
		t.Fatalf("only %d closed fields had values to check", checked)
	}
}

// An effort whose options come from the model the agent is on: Devin's
// levels are its family's variants, so a level it doesn't list is refused.
func TestClosedEffortUsesTheModelsLevels(t *testing.T) {
	syncHome(t)
	was := providerDevinFamilies
	providerDevinFamilies = func(context.Context) ([]provider.DevinFamily, error) {
		return []provider.DevinFamily{{
			UID: "swe-2", Label: "SWE-2",
			Models: []catalog.Model{{ID: "swe-2-low"}, {ID: "swe-2-high"}},
		}}, nil
	}
	t.Cleanup(func() { providerDevinFamilies = was })

	a := devin(os.Getenv("HOME"), os.Getenv("XDG_CONFIG_HOME"))
	if err := a.Fields[0].Set("swe-2"); err != nil {
		t.Fatal(err)
	}
	if got, err := a.Spell("effort", "highish"); err == nil || got != "" {
		t.Errorf("highish: %q, %v; want refused", got, err)
	}
	for _, v := range []string{"low", "high"} {
		if got, err := a.Spell("effort", v); err != nil || got != v {
			t.Errorf("%s: %q, %v", v, got, err)
		}
	}
}

// For a model myna's catalog doesn't describe, Codex and Grok take the
// levels their own enums know: Codex none through xhigh (codexEffortsAll),
// Grok none through max (grokEfforts). A fallback shorter than that turned
// a level the agent takes into one myna refused.
func TestEffortFallbacksOfferEveryLevelTheAgentKnows(t *testing.T) {
	syncHome(t)
	home := os.Getenv("HOME")
	for _, c := range []struct {
		name string
		a    *Agent
		want []string
	}{
		{"codex", codex(home), codexEffortsAll},
		{"grok", grok(home), grokEfforts},
	} {
		f := c.a.Field("effort")
		if f == nil {
			t.Fatalf("%s: no effort field", c.name)
		}
		opts := f.Options(c.a.Values())
		for _, want := range c.want {
			if !slices.ContainsFunc(opts, func(o Option) bool { return o.Value == want }) {
				t.Errorf("%s offers %v, wanting %q", c.name, opts, want)
			}
			if got, err := c.a.Spell("effort", want); err != nil || got != want {
				t.Errorf("%s %q: %q, %v", c.name, want, got, err)
			}
		}
	}
}

// A stopped WSL distro's agent is a copy of the live one, its options the
// distro's: the closed fields go with it, so `myna codex@wsl:Ubuntu effort
// highish` is refused too.
func TestClosedFieldSurvivesTheWSLClone(t *testing.T) {
	syncHome(t)
	a := asleep(codex(os.Getenv("HOME")), wslKindOf("codex"), distro{Name: "Ubuntu"})
	if got, err := a.Spell("effort", "highish"); err == nil || got != "" {
		t.Errorf("highish: %q, %v; want refused", got, err)
	}
	if got, err := a.Spell("effort", "high"); err != nil || got != "high" {
		t.Errorf("high: %q, %v", got, err)
	}
}
