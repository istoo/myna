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

func TestOwnAddrSkip(t *testing.T) {
	if why := OwnAddrSkip("127.0.0.1"); why != "" {
		t.Fatalf("loopback skipped: %s", why)
	}
	why := OwnAddrSkip("192.0.2.1")
	if !strings.Contains(why, "192.0.2.1") || !strings.Contains(why, "Local Network privacy") {
		t.Fatalf("an unreachable address skips with %q", why)
	}
}
