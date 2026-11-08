package suggest

import "testing"

func TestClose(t *testing.T) {
	ids := []string{"claude/claude-opus-5-5", "copilot/claude-opus-5.5", "deepseek/deepseek-chat", "b/gpt-5.5"}
	got := Close("claude-opus", ids, 5)
	if len(got) != 2 || got[0] != "claude/claude-opus-5-5" {
		t.Errorf("%v", got)
	}
	if got := Close("x/deepseek-chta", ids, 5); len(got) != 1 || got[0] != "deepseek/deepseek-chat" {
		t.Errorf("%v", got)
	}
	if got := Close("zzz", ids, 5); len(got) != 0 {
		t.Errorf("%v", got)
	}
}

// n bounds the suggestions: a wrong name that is like many others offers
// the closest few, not all of them.
func TestCloseKeepsAtMostN(t *testing.T) {
	ids := []string{"codex", "qoder", "zcode", "opencode", "openbot"}
	if got := Close("ode", ids, 3); len(got) != 3 {
		t.Errorf("Close(ode, 3) = %v, want 3", got)
	}
}
