package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/istoo/owl/internal/agent"
	"github.com/istoo/owl/internal/provider"
)

// cliHome is groupsHome with every other folder an agent keeps files in
// (XDG_DATA_HOME, XDG_STATE_HOME, Windows' APPDATA) inside its home, so
// that homeFiles sees whatever a command writes, and with the gateway's
// address one nothing answers at. No agent runs: what a command prints
// doesn't gain "restart the Codex app" where Codex is open (and on Windows,
// where anything may be).
func cliHome(t *testing.T) {
	t.Helper()
	groupsHome(t)
	was := agent.Running
	agent.Running = func(...string) bool { return false }
	t.Cleanup(func() { agent.Running = was })
	home := os.Getenv("HOME")
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("OWL_ADDR", "127.0.0.1:1")
}

// homeFiles is every folder and file in the test's home, config and cache
// folders, with each file's contents. Not owl's cli-identity.json: who
// Cursor's and Devin's CLIs say is signed in, asked again behind a look a
// minute after the last answer and kept when it comes, which may be during
// a later command, or in a later test's home.
func homeFiles(t *testing.T) map[string]string {
	t.Helper()
	identities := filepath.Join(filepath.Dir(provider.Path()), "cli-identity.json")
	m := map[string]string{}
	for _, root := range []string{os.Getenv("HOME"), os.Getenv("XDG_CONFIG_HOME"), os.Getenv("XDG_CACHE_HOME")} {
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if p == identities {
				return nil
			}
			if d.IsDir() {
				m[p] = "(folder)"
				return nil
			}
			b, err := os.ReadFile(p)
			m[p] = string(b)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return m
}

// filesChanged names what differs between two homeFiles, "" for nothing.
func filesChanged(before, after map[string]string) string {
	var out []string
	for p, b := range before {
		if a, ok := after[p]; !ok {
			out = append(out, "removed "+p)
		} else if a != b {
			out = append(out, "changed "+p+":\n"+a)
		}
	}
	for p := range after {
		if _, ok := before[p]; !ok {
			out = append(out, "made "+p)
		}
	}
	sort.Strings(out)
	return strings.Join(out, "\n")
}

func writeHomeFile(t *testing.T, rel, s string) string {
	t.Helper()
	p := filepath.Join(os.Getenv("HOME"), rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// codexOnAdvanced is the reporter's machine: Codex's own config.toml, a
// routing group advanced, and Codex put on it with owl codex
// group/advanced. It gives the config's path.
func codexOnAdvanced(t *testing.T) string {
	t.Helper()
	cliHome(t)
	path := writeHomeFile(t, filepath.Join(".codex", "config.toml"),
		"model = \"gpt-5.5\"\nmodel_reasoning_effort = \"high\"\n\n[projects.\"/Users/me/work\"]\ntrust_level = \"trusted\"\n")
	for _, args := range [][]string{{"group", "add", "advanced", "models=a/m,b/gpt-5.5"}, {"codex", "group/advanced"}} {
		if _, err := printed(t, func() error { return run(args) }); err != nil {
			t.Fatalf("owl %s: %v", strings.Join(args, " "), err)
		}
	}
	if b, _ := os.ReadFile(path); !strings.Contains(string(b), `"group/advanced"`) || !strings.Contains(string(b), "trust_level") {
		t.Fatalf("Codex isn't on group/advanced:\n%s", b)
	}
	return path
}

// owl codex --help printed "✓ Codex model --help" and wrote model =
// "--help" into ~/.codex/config.toml, owl taken out with it (reported
// on 0.1.1082, Codex on group/advanced). --help, -h and help, wherever
// they come, now show how to set Codex, and any other word that starts
// with "-" is refused as a flag: neither writes a thing.
// owl codex model wrote model = "model" into Codex's config, owl
// taken out with it, and owl codex effort model = "effort" (reported
// with owl codex --help, on 0.1.1082). A field's name alone, as typed
// or as shown, now says what the field is set to, and a field's name is
// refused as a value: owl codex subagent effort ran the subagents on a
// model called effort. Neither writes a thing.
func TestAgentFieldNameShowsIt(t *testing.T) {
	codexOnAdvanced(t)
	before := homeFiles(t)
	for _, c := range []struct{ args, want string }{
		{"codex model", "Codex model group/advanced\n"},
		{"codex Model", "Codex model group/advanced\n"},
		{"codex effort", "Codex effort high\n"},
		{"codex EFFORT", "Codex effort high\n"},
		{"codex subagents", "Codex subagents default\n"},
		{"codex subagent_effort", "Codex subagent effort default\n"},
		{"codex default", ""},
	} {
		args := strings.Fields(c.args)
		if c.want == "" {
			// what tells a write: taking owl out still does
			if _, err := printed(t, func() error { return run(args) }); err != nil || filesChanged(before, homeFiles(t)) == "" {
				t.Fatalf("owl %s: %v, and wrote nothing", c.args, err)
			}
			continue
		}
		out, err := printed(t, func() error { return run(args) })
		if err != nil || out != c.want {
			t.Errorf("owl %s: %v %q, want %q", c.args, err, out, c.want)
		}
		if diff := filesChanged(before, homeFiles(t)); diff != "" {
			t.Fatalf("owl %s wrote:\n%s", c.args, diff)
		}
	}
	codexOnAdvanced(t)
	before = homeFiles(t)
	for _, c := range []struct{ args, want string }{
		{"codex subagent effort", `"subagent effort" is one field of Codex's: owl codex subagent_effort shows it`},
		{"codex subagents Effort", `"subagent effort" is one field of Codex's: owl codex subagent_effort shows it`},
		{"codex model effort", `"effort" is a field of Codex's, not a value for its model: owl codex effort shows it`},
		{"codex model Effort", `"Effort" is a field of Codex's, not a value for its model: owl codex effort shows it`},
		{"codex effort model", `"model" is a field of Codex's, not a value for its effort: owl codex model shows it`},
		{"codex model sign-in", `"sign-in" is a field of Codex's, not a value for its model: owl codex login shows it`},
	} {
		args := strings.Fields(c.args)
		out, err := printed(t, func() error { return run(args) })
		if err == nil || err.Error() != c.want || out != "" {
			t.Errorf("owl %s: %v %q, want %s", c.args, err, out, c.want)
		}
		if diff := filesChanged(before, homeFiles(t)); diff != "" {
			t.Fatalf("owl %s wrote:\n%s", c.args, diff)
		}
	}
	// a field's name in any case sets it too
	out, err := printed(t, func() error { return run([]string{"codex", "EFFORT", "low"}) })
	if err != nil || out != "✓ Codex effort low\n" || filesChanged(before, homeFiles(t)) == "" {
		t.Errorf("owl codex EFFORT low: %v %q", err, out)
	}
}

// Every agent's fields go by their names the same way: alone, each shows
// its field; after any field, each is refused. They were written as most
// agents' model, and connected Claude Desktop, Cursor Private Inference,
// Pencil, T3 Code, WorkBuddy and ZCode to owl. Claude Code's opus,
// sonnet, haiku and fable name its model as well as its tiers, and stay
// the model's (TestClaudeTierNamesSetTheModel).
func TestEveryAgentFieldNameWritesNothing(t *testing.T) {
	cliHome(t)
	writeHomeFile(t, filepath.Join(".claude", "settings.json"), "{\n  \"model\": \"opus\",\n  \"theme\": \"dark\"\n}\n")
	if _, err := printed(t, func() error { return run([]string{"version"}) }); err != nil {
		t.Fatal(err)
	}
	before := homeFiles(t)
	try := func(args []string, ok func(out string, err error) bool, want string) {
		t.Helper()
		out, err := printed(t, func() error { return run(args) })
		if !ok(out, err) {
			t.Errorf("owl %s: %v %q, want %s", strings.Join(args, " "), err, out, want)
		}
		if diff := filesChanged(before, homeFiles(t)); diff != "" {
			t.Errorf("owl %s wrote:\n%s", strings.Join(args, " "), diff)
			before = homeFiles(t)
		}
	}
	for _, a := range agent.All() {
		var names []string
		for _, f := range a.Fields {
			for _, n := range []string{f.Key, f.Label} {
				if !slices.Contains(names, n) && !slices.Contains(a.ModelAliases, n) {
					names = append(names, n)
				}
			}
		}
		for _, n := range names {
			f := a.Field(n)
			try([]string{a.ID, n}, func(out string, err error) bool {
				return err == nil && strings.HasPrefix(out, a.Name+" "+f.Label+" ") && strings.Count(out, "\n") == 1
			}, "its "+f.Label)
			refused := regexp.MustCompile(`^"[^"]+" is (a field|one field|a switch) of ` + regexp.QuoteMeta(a.Name) + `'s`)
			for _, g := range a.Fields {
				try([]string{a.ID, g.Key, n}, func(out string, err error) bool {
					return err != nil && refused.MatchString(err.Error()) && out == ""
				}, "it refused")
			}
		}
	}
}

// Claude Code's tiers are named as its model's aliases are: owl claude
// opus sets its model to opus, as the docs have it, where another field's
// name alone shows that field; owl claude help says so.
func TestClaudeTierNamesSetTheModel(t *testing.T) {
	cliHome(t)
	path := writeHomeFile(t, filepath.Join(".claude", "settings.json"), "{\n  \"model\": \"claude-opus-4-5\",\n  \"theme\": \"dark\"\n}\n")
	for _, args := range [][]string{{"claude", "opus"}, {"claude", "sonnet"}, {"claude", "haiku"}, {"claude", "fable"}, {"claude", "model", "opus"}} {
		want := args[len(args)-1]
		out, err := printed(t, func() error { return run(args) })
		if b, _ := os.ReadFile(path); err != nil || out != "✓ Claude Code model "+want+"\n" || !strings.Contains(string(b), `"model": "`+want+`"`) {
			t.Errorf("owl %s: %v %q\n%s", strings.Join(args, " "), err, out, b)
		}
	}
	out, err := printed(t, func() error { return run([]string{"claude", "help"}) })
	if err != nil || !strings.Contains(out, "\n  opus, sonnet, haiku and fable alone name the model: owl claude opus sets its model to opus\n") {
		t.Errorf("owl claude help: %v\n%s", err, out)
	}
	// in another case too, where Claude Code takes its aliases as they are
	// spelled: the tier's field isn't shown for it
	out, err = printed(t, func() error { return run([]string{"claude", "Opus"}) })
	if err == nil || !strings.HasPrefix(err.Error(), `"Opus" is no model of Claude Code's`) || out != "" {
		t.Errorf("owl claude Opus: %v %q, want it refused as a model", err, out)
	}
}

// Claude Code's ultracode is a switch: given as its model or its effort,
// the refusal says how to turn it on, as those fields' own errors did
// before a field's name was refused ahead of them.
func TestClaudeUltracodeNamedAsAValue(t *testing.T) {
	cliHome(t)
	path := writeHomeFile(t, filepath.Join(".claude", "settings.json"), "{\n  \"model\": \"opus\",\n  \"theme\": \"dark\"\n}\n")
	if _, err := printed(t, func() error { return run([]string{"version"}) }); err != nil {
		t.Fatal(err)
	}
	before := homeFiles(t)
	for _, args := range [][]string{{"claude", "model", "ultracode"}, {"claude", "effort", "ultracode"}, {"claude", "effort", "Ultracode"}} {
		out, err := printed(t, func() error { return run(args) })
		if err == nil || !strings.HasSuffix(err.Error(), ": owl claude ultracode on") || out != "" {
			t.Errorf("owl %s: %v %q, want it to say owl claude ultracode on", strings.Join(args, " "), err, out)
		}
		if diff := filesChanged(before, homeFiles(t)); diff != "" {
			t.Fatalf("owl %s wrote:\n%s", strings.Join(args, " "), diff)
		}
	}
	if _, err := printed(t, func() error { return run([]string{"claude", "ultracode", "on"}) }); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); !strings.Contains(string(b), `"ultracode": true`) {
		t.Errorf("owl claude ultracode on wrote:\n%s", b)
	}
}

// owl codex DEFAULT wrote model = "DEFAULT" into Codex's config, owl
// taken out with it, and owl codex effort Default wrote
// model_reasoning_effort = "Default". default is owl's word in any
// case, as help and a field's name are, and does what default does.
func TestAgentDefaultInAnyCase(t *testing.T) {
	path := codexOnAdvanced(t)
	for _, c := range []struct {
		args, want string
		gone       *regexp.Regexp
	}{
		{"codex effort Default", "✓ Codex effort default\n", regexp.MustCompile(`(?m)^model_reasoning_effort =`)},
		{"codex model DEFAULT", "✓ Codex model default\n", regexp.MustCompile(`(?m)^model =`)},
	} {
		out, err := printed(t, func() error { return run(strings.Fields(c.args)) })
		b, _ := os.ReadFile(path)
		if err != nil || out != c.want || c.gone.Match(b) {
			t.Errorf("owl %s: %v %q, want %q\n%s", c.args, err, out, c.want, b)
		}
	}
	for _, word := range []string{"DEFAULT", "Default"} {
		path := codexOnAdvanced(t)
		out, err := printed(t, func() error { return run([]string{"codex", word}) })
		b, _ := os.ReadFile(path)
		if err != nil || out != "✓ Codex model gpt-5.5\n  disconnected from owl, back to what it had before\n" ||
			!strings.Contains(string(b), "model = \"gpt-5.5\"\nmodel_reasoning_effort = \"high\"\n") || strings.Contains(string(b), "model_provider = ") {
			t.Errorf("owl codex %s: %v %q, want Codex back on gpt-5.5\n%s", word, err, out, b)
		}
	}
}

// owl claude sonnet, run by an agent that took it for a query of the
// sonnet tier, put Claude Code back on the DeepSeek endpoint it had before
// owl, which refused Sonnet with a 400, and printed only "✓ Claude Code
// model sonnet" (#2000, marshalys). An alias set while Claude Code is on
// owl now says it took owl out, and how to connect it again; help
// says so beside the aliases.
func TestClaudeAliasSaysItTookOwlOut(t *testing.T) {
	for _, alias := range []string{"opus", "sonnet", "haiku", "fable"} {
		cliHome(t)
		path := writeHomeFile(t, filepath.Join(".claude", "settings.json"),
			"{\n  \"model\": \"deepseek-chat\",\n  \"env\": {\n    \"ANTHROPIC_BASE_URL\": \"https://deepseek.invalid/anthropic\",\n    \"ANTHROPIC_AUTH_TOKEN\": \"sk-own\"\n  }\n}\n")
		for _, args := range [][]string{{"group", "add", "advanced", "models=a/m"}, {"claude", "group/advanced"}} {
			if _, err := printed(t, func() error { return run(args) }); err != nil {
				t.Fatalf("owl %s: %v", strings.Join(args, " "), err)
			}
		}
		if b, _ := os.ReadFile(path); strings.Contains(string(b), "deepseek.invalid") {
			t.Fatalf("Claude Code isn't on owl:\n%s", b)
		}
		out, err := printed(t, func() error { return run([]string{"claude", alias}) })
		b, _ := os.ReadFile(path)
		if err != nil || !strings.HasPrefix(out, "✓ Claude Code model "+alias+"\n  disconnected from owl: "+alias+" is Claude Code's own") ||
			!strings.Contains(out, "owl claude <model>") || !strings.Contains(string(b), "deepseek.invalid") {
			t.Errorf("owl claude %s: %v %q, want it to say owl is out\n%s", alias, err, out, b)
		}
		// set again, off owl already: nothing to say
		out, err = printed(t, func() error { return run([]string{"claude", alias}) })
		if err != nil || strings.Contains(out, "disconnected") {
			t.Errorf("owl claude %s, off owl: %v %q", alias, err, out)
		}
	}
	out, err := printed(t, func() error { return run([]string{"claude", "help"}) })
	if err != nil || !strings.Contains(out, "they are Claude Code's own models, not owl's: setting one takes owl out") {
		t.Errorf("owl claude help: %v\n%s", err, out)
	}
}

// `owl codex effort highish` printed "✓ Codex effort highish" and wrote
// model_reasoning_effort = "highish" into config.toml, a level Codex
// refuses at start-up. A closed field refuses a value its picker doesn't
// offer and writes nothing; the levels Codex takes, and default, still work.
func TestUnknownEffortWritesNothing(t *testing.T) {
	path := codexOnAdvanced(t)
	before := homeFiles(t)
	was, _ := os.ReadFile(path)

	out, err := printed(t, func() error { return run([]string{"codex", "effort", "highish"}) })
	if err == nil || out != "" {
		t.Fatalf("owl codex effort highish: %v %q, want refused", err, out)
	}
	for _, want := range []string{"highish", "one of", "try: owl codex effort"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q doesn't name %q", err, want)
		}
	}
	if diff := filesChanged(before, homeFiles(t)); diff != "" {
		t.Fatalf("owl codex effort highish wrote:\n%s", diff)
	}
	if now, _ := os.ReadFile(path); string(now) != string(was) {
		t.Fatalf("config.toml changed:\n%s", now)
	}

	out, err = printed(t, func() error { return run([]string{"codex", "effort", "high"}) })
	if err != nil || !strings.Contains(out, "✓ Codex effort high") {
		t.Errorf("owl codex effort high: %v %q", err, out)
	}
	out, err = printed(t, func() error { return run([]string{"codex", "effort", "default"}) })
	if err != nil || !strings.Contains(out, "✓ Codex effort default") {
		t.Errorf("owl codex effort default: %v %q", err, out)
	}
}

// Every closed field refuses a value its picker doesn't offer, through the
// CLI as the user reaches it, and leaves every file as it was. A field
// whose values aren't known in this sandbox (nothing wired, no model picked)
// is the agent's own to refuse, as in internal/agent's test; Devin's list
// is asked of its CLI, and its own test stands in for that one.
func TestEveryClosedFieldRefusesAnUnknownValue(t *testing.T) {
	cliHome(t)
	// start-up's own files (the plugins list, the state folders) settle first
	if _, err := printed(t, func() error { return run([]string{"version"}) }); err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, a := range agent.All() {
		if a.WSL != "" || a.ID == "devin" {
			continue
		}
		for _, f := range a.Fields {
			if !f.Closed || len(f.Options(a.Values())) == 0 {
				continue
			}
			checked++
			before := homeFiles(t)
			out, err := printed(t, func() error { return run([]string{a.ID, f.Key, "zz-bogus-closed"}) })
			if err == nil || out != "" {
				t.Errorf("owl %s %s zz-bogus-closed: %v %q, want refused", a.ID, f.Key, err, out)
			}
			if diff := filesChanged(before, homeFiles(t)); diff != "" {
				t.Errorf("owl %s %s zz-bogus-closed wrote:\n%s", a.ID, f.Key, diff)
			}
		}
	}
	if checked < 15 {
		t.Fatalf("only %d closed fields had values to check", checked)
	}
}
