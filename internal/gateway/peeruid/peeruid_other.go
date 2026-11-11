//go:build !darwin && !linux && !android && !windows

package peeruid

import "net/netip"

// sameUser: this platform has no reader for TCP socket owners, so the
// caller keeps what it did before.
func sameUser(peer, local netip.AddrPort) (bool, bool) {
	return false, false
}
