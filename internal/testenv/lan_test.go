package testenv

import (
	"strings"
	"testing"
)

func TestOwnAddrReachable(t *testing.T) {
	if !OwnAddrReachable("127.0.0.1") {
		t.Fatal("loopback wasn't reached")
	}
	// TEST-NET-1, no interface of this computer's: nothing listens on it
	if OwnAddrReachable("192.0.2.1") {
		t.Fatal("TEST-NET-1 was reached")
	}
}

// Both outcomes of the skip, forced: the OS refusing the dial is what a
// skipped test meets, and no test can ask macOS's Local Network privacy to
// refuse one (on this Mac it does so for every newly built binary, #3), so
// the sentence is checked with a dial that answers either way.
func TestOwnAddrSkip(t *testing.T) {
	if why := ownAddrSkip("127.0.0.1", func(string) bool { return true }); why != "" {
		t.Fatalf("a reached address skipped: %s", why)
	}
	why := ownAddrSkip("192.0.2.1", func(string) bool { return false })
	if !strings.Contains(why, "192.0.2.1") || !strings.Contains(why, "Local Network privacy") {
		t.Fatalf("an unreached address skips with %q", why)
	}
}
