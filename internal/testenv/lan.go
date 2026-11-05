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
