package library

import (
	"testing"

	"github.com/istoo/owl/internal/agent"
)

// SSNQT on X: OpenBot could not be picked for skills in the Library. It
// reads them only from ~/.agents/skills (home-skills.ts, HOME_SKILL_FOLDER).
// Its MCP servers are rows in its SQLite database, which owl doesn't write,
// so it has no MCP place.
func TestOpenBotSkillsPlace(t *testing.T) {
	tg := targetOf(&agent.Agent{ID: "openbot", Dir: t.TempDir()})
	if tg == nil {
		t.Fatal("no Library target")
	}
	if tg.Skills != sharedSkillsDir() {
		t.Errorf("skills at %q, want %q", tg.Skills, sharedSkillsDir())
	}
	if tg.MCP != nil {
		t.Errorf("an MCP place at %q: OpenBot's servers are rows in its database", tg.MCP.Path)
	}
}
