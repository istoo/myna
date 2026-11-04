package gateway

import (
	"context"
	"net"
	"net/http"
	"net/netip"

	"github.com/istoo/myna/internal/gateway/peeruid"
)

// A loopback request is this user's own only when the process on the other
// end of its connection runs as this user: another user on this computer
// can reach 127.0.0.1:3425 too, and was answered with no key, as the
// owner. The kernel's TCP table names the socket's owner (peeruid). The
// answer is taken once per connection, as it is accepted, so a request
// costs no table read.
type peerUserKey struct{}

// peerUserContext marks ctx with whether c's peer is this user. A peer
// that isn't on loopback is left unmarked: local() says no by address, and
// a WSL distro's connection, rewritten to loopback afterwards, is this
// computer's own.
func peerUserContext(ctx context.Context, c net.Conn) context.Context {
	peer, err := netip.ParseAddrPort(c.RemoteAddr().String())
	if err != nil || !peer.Addr().Unmap().IsLoopback() {
		return ctx
	}
	local, err := netip.ParseAddrPort(c.LocalAddr().String())
	if err != nil {
		return ctx
	}
	same, known := peeruid.Same(peer, local)
	if !known {
		return ctx
	}
	return context.WithValue(ctx, peerUserKey{}, same)
}

// sameOSUser: r's connection was made by this user, or nothing said it
// wasn't (a platform with no reader, or a request a test built).
func sameOSUser(r *http.Request) bool {
	same, known := r.Context().Value(peerUserKey{}).(bool)
	return !known || same
}
