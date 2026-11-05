package testenv

import "testing"

func TestOwnAddrReachable(t *testing.T) {
	if !OwnAddrReachable("127.0.0.1") {
		t.Fatal("loopback wasn't reached")
	}
	// TEST-NET-1, no interface of this computer's: nothing listens on it
	if OwnAddrReachable("192.0.2.1") {
		t.Fatal("TEST-NET-1 was reached")
	}
}
