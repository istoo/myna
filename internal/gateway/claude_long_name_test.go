package gateway

import "testing"

// A Claude model of the 5 generation that Claude Code doesn't know (Fable
// 5.2 on 2.1.316) is asked for by its "[1m]" name, which Claude Code runs
// to 1M: by its own id it held the model to 200K and turned a longer
// conversation away before Anthropic was asked, so Codex, told the model's
// 1M window and compacting at 272K, never compacted (magicalaciw18 on X).
// One it knows, and every 4 generation one, whose 1M a subscription
// without usage credits is refused a short turn on, keep their names.
func TestClaudeRunName(t *testing.T) {
	for model, want := range map[string]string{
		"claude-fable-5-2":     "claude-fable-5-2[1m]",
		"claude-opus-6":        "claude-opus-6[1m]",
		"claude-sonnet-5-7":    "claude-sonnet-5-7[1m]",
		"claude-fable-5-2[1m]": "claude-fable-5-2[1m]",
		"claude-fable-5-1":     "claude-fable-5-1",
		"claude-fable-5":       "claude-fable-5",
		"claude-opus-5-5":      "claude-opus-5-5",
		"claude-sonnet-5-5":    "claude-sonnet-5-5",
		"claude-haiku-5-5":     "claude-haiku-5-5",
		"claude-mythos-5-1":    "claude-mythos-5-1",
		"claude-sonnet-4-6":    "claude-sonnet-4-6",
		"claude-opus-4-8":      "claude-opus-4-8",
		"claude-haiku-4-5":     "claude-haiku-4-5",
		"claude-3-7-sonnet":    "claude-3-7-sonnet",
		"haiku":                "haiku",
		"m":                    "m",
	} {
		if got := claudeRunName(model); got != want {
			t.Errorf("claudeRunName(%q) = %q, want %q", model, got, want)
		}
	}
}
