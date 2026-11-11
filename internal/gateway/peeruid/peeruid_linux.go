//go:build linux || android

package peeruid

import (
	"bufio"
	"encoding/hex"
	"net/netip"
	"os"
	"strconv"
	"strings"
)

// /proc/net/tcp and /proc/net/tcp6 list the namespace's TCP sockets: the
// local and remote address and port, in hex, and the uid that owns the
// socket. The socket whose local end is the peer's address is the one to
// ask about.
func sameUser(peer, local netip.AddrPort) (bool, bool) {
	uid := uint32(os.Getuid())
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		same, known, err := lookIn(path, peer, local, uid)
		if err == nil && known {
			return same, true
		}
	}
	return false, false
}

// lookIn is sameUser over one of the tables. known is false when no row
// is the peer's connection.
func lookIn(path string, peer, local netip.AddrPort, uid uint32) (same, known bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		return false, false, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	first := true
	for sc.Scan() {
		if first { // the column names
			first = false
			continue
		}
		fields := strings.Fields(sc.Text())
		if len(fields) < 8 {
			continue
		}
		la, lp, ok1 := parseNet(fields[1])
		ra, rp, ok2 := parseNet(fields[2])
		if !ok1 || !ok2 {
			continue
		}
		if lp != peer.Port() || rp != local.Port() || la != peer.Addr().Unmap() || ra != local.Addr().Unmap() {
			continue
		}
		u, err := strconv.ParseUint(fields[7], 10, 32)
		if err != nil {
			return false, false, err
		}
		return uint32(u) == uid, true, nil
	}
	return false, false, sc.Err()
}

// parseNet reads "0100007F:1F90" or a 32-digit IPv6 form with its port.
// Each 32-bit word is written in host order, so its bytes come reversed;
// the port is plain hex.
func parseNet(s string) (netip.Addr, uint16, bool) {
	i := strings.IndexByte(s, ':')
	if i < 0 {
		return netip.Addr{}, 0, false
	}
	p, err := strconv.ParseUint(s[i+1:], 16, 16)
	if err != nil {
		return netip.Addr{}, 0, false
	}
	raw, err := hex.DecodeString(s[:i])
	if err != nil || len(raw) != 4 && len(raw) != 16 {
		return netip.Addr{}, 0, false
	}
	for i := 0; i < len(raw); i += 4 {
		raw[i], raw[i+1], raw[i+2], raw[i+3] = raw[i+3], raw[i+2], raw[i+1], raw[i]
	}
	a, ok := netip.AddrFromSlice(raw)
	if !ok {
		return netip.Addr{}, 0, false
	}
	return a.Unmap(), uint16(p), true
}
