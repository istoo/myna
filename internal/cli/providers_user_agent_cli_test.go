package cli

import (
	"strings"
	"testing"

	"github.com/istoo/myna/internal/provider"
)

// user-agent=agent has a key+URL provider pass the agent's User-Agent on
// (#1851), user-agent=myna goes back to myna's, and a signed-in
// account, which goes as its own client, is refused it.
func TestProviderUserAgentPair(t *testing.T) {
	p := provider.Provider{ID: "plan", Name: "Plan", Chat: "https://plan.example/v1", Key: "k"}
	if err := applyPairs(&p, []string{"user-agent=agent"}); err != nil || !p.PassUserAgent {
		t.Fatalf("agent: %v %v", p.PassUserAgent, err)
	}
	if err := applyPairs(&p, []string{"user-agent=myna"}); err != nil || p.PassUserAgent {
		t.Fatalf("myna: %v %v", p.PassUserAgent, err)
	}
	if err := applyPairs(&p, []string{"user-agent=claude"}); err == nil || !strings.Contains(err.Error(), "agent|myna") {
		t.Fatalf("a made-up value: %v", err)
	}
	a := provider.Provider{ID: "zcode", Name: "ZCode", Account: &provider.Account{Agent: "zcode"}}
	if err := applyPairs(&a, []string{"user-agent=agent"}); err == nil || a.PassUserAgent {
		t.Fatalf("an account took it: %v", err)
	}
	if err := applyPairs(&a, []string{"user-agent=myna"}); err != nil {
		t.Fatalf("an account turning it off: %v", err)
	}
}

// compaction=native has a provider whose wire carries it be asked Codex's
// own compaction (#1099), compaction=myna goes back to myna's summary,
// and a provider without a Responses API is refused it.
func TestProviderCompactionPair(t *testing.T) {
	p := provider.Provider{ID: "relay", Name: "Relay", Responses: "https://relay.example/v1", Key: "k"}
	if err := applyPairs(&p, []string{"compaction=native"}); err != nil || !p.NativeCompaction {
		t.Fatalf("native: %v %v", p.NativeCompaction, err)
	}
	if err := applyPairs(&p, []string{"compaction=myna"}); err != nil || p.NativeCompaction {
		t.Fatalf("myna: %v %v", p.NativeCompaction, err)
	}
	if err := applyPairs(&p, []string{"compaction=sometimes"}); err == nil || !strings.Contains(err.Error(), "native|myna") {
		t.Fatalf("a made-up value: %v", err)
	}
	c := provider.Provider{ID: "chat", Name: "Chat", Chat: "https://chat.example/v1", Key: "k"}
	if err := applyPairs(&c, []string{"compaction=native"}); err == nil {
		t.Fatalf("a chat-only provider took it")
	}
	cp := provider.Provider{ID: "copilot", Name: "GitHub Copilot", Responses: "https://api.githubcopilot.com", Account: &provider.Account{Agent: "copilot"}}
	if err := applyPairs(&cp, []string{"compaction=native"}); err != nil || !cp.NativeCompaction {
		t.Fatalf("copilot: %v %v", cp.NativeCompaction, err)
	}
}
