// Package peeruid tells whether the process on the other end of a loopback
// TCP connection runs as this computer user, so owl's gateway doesn't
// answer another user's process as if it were this one's.
//
// A TCP connection carries no peer credentials (unlike a unix socket): the
// kernel's own table of TCP sockets is read instead, and the socket whose
// local end is the peer's address is looked up for its owner. Each platform
// that can answer does so in its own file; where the table can't be read,
// or the platform has no reader, known is false and the caller keeps what
// it did before.
package peeruid

import "net/netip"

// Same reports whether the process whose TCP connection runs from peer to
// local is this user's. peer is the connection's remote address, local the
// address of owl's own end. known is false when this platform, or this
// connection, can't tell.
func Same(peer, local netip.AddrPort) (same, known bool) {
	return sameUser(peer, local)
}
