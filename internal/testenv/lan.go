package testenv

import (
	"net"
	"time"
)

// OwnAddrReachable is whether ip, an address of this computer, is reached
// by a plain dial from this process: a listener of its own is made on that
// address and dialled. macOS's Local Network privacy refuses a newly built
// binary — the dial times out while curl and bun, signed differently,
// reach it (#3, on this Mac on 11-05) — and a test that has a plugin or a
// helper process call back to that address can say nothing about myna while
// the OS refuses it. Such a test skips.
func OwnAddrReachable(ip string) bool {
	ln, err := net.Listen("tcp", net.JoinHostPort(ip, "0"))
	if err != nil {
		return false
	}
	defer ln.Close()
	c, err := net.DialTimeout("tcp", ln.Addr().String(), 2*time.Second)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

// OwnAddrSkip is why a test that must reach ip should skip, "" when it
// can: the sentence a skipped test says, kept out of the test so both
// outcomes can be checked (TestOwnAddrSkip).
func OwnAddrSkip(ip string) string { return ownAddrSkip(ip, OwnAddrReachable) }

// ownAddrSkip is OwnAddrSkip with the dial its caller's, so the skip itself
// is checked without the OS having to refuse one (TestOwnAddrSkip).
func ownAddrSkip(ip string, reach func(string) bool) string {
	if reach(ip) {
		return ""
	}
	return "this process can't reach this computer's own address " + ip +
		": macOS's Local Network privacy refuses a newly built binary, and curl/bun reach it (#3)"
}
