package agent

import (
	"strings"
	"testing"
)

// An unknown agent name offers the few names nearest it and the commands
// that list the agents; it no longer prints every name owl supports (73
// at 10-28).
func TestUnknownAgentSuggestsAFew(t *testing.T) {
	_, err := Find("clade")
	if err == nil {
		t.Fatal(`Find("clade") found an agent`)
	}
	msg := err.Error()
	if !strings.Contains(msg, "did you mean claude") {
		t.Errorf("no suggestion for clade: %s", msg)
	}
	if n := strings.Count(msg, ", "); n > 2 {
		// three suggestions have at most two commas; the full list has 72
		t.Errorf("%d commas in the message, want at most 3 suggestions: %s", n, msg)
	}
	if !strings.Contains(msg, "owl ls") {
		t.Errorf("the message doesn't point at owl ls: %s", msg)
	}
	if strings.Contains(msg, "zoo-code") {
		t.Errorf("the message prints the full list again: %s", msg)
	}
}

// A name like nothing else still points at the lists, without a
// suggestion line that would be made up.
func TestUnknownAgentWithNoSuggestion(t *testing.T) {
	_, err := Find("frobnicate")
	if err == nil {
		t.Fatal(`Find("frobnicate") found an agent`)
	}
	msg := err.Error()
	if strings.Contains(msg, "did you mean") {
		t.Errorf("a made-up suggestion: %s", msg)
	}
	if !strings.Contains(msg, "owl ls") || !strings.Contains(msg, "owl agents") {
		t.Errorf("the message doesn't point at the lists: %s", msg)
	}
}
