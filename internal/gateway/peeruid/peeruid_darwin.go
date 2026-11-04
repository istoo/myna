//go:build darwin

package peeruid

import (
	"encoding/binary"
	"net/netip"
	"os"

	"golang.org/x/sys/unix"
)

// net.inet.tcp.pcblist is the kernel's table of TCP sockets: struct xtcpcb
// entries. A 64-bit process is given the 32-bit layout here (pack(4),
// 32-bit pointers), which carries the owning socket's uid where the newer
// xtcpcb64 doesn't (the xt_len tells them apart).
const (
	xtcpcbSize  = 524
	pcOffVFlag  = 80
	pcOffFPort  = 20
	pcOffLPort  = 22
	pcOffFAddr  = 96 // in_addr_4in6's address; the 16-byte union starts at 84
	pcOffLAddr  = 112
	pcOffFAddr6 = 84
	pcOffLAddr6 = 100
	pcOffUID    = 512 // xt_socket (416) + so_uid (96)
	inpIPv4     = 0x1
	inpIPv6     = 0x2
)

func sameUser(peer, local netip.AddrPort) (bool, bool) {
	b, err := unix.SysctlRaw("net.inet.tcp.pcblist")
	if err != nil {
		return false, false
	}
	return sameIn(b, peer, local, uint32(os.Getuid()))
}

// sameIn is sameUser over a pcblist read: the socket whose local end is
// peer and whose remote end is local belongs to uid.
func sameIn(b []byte, peer, local netip.AddrPort, uid uint32) (same, known bool) {
	if len(b) < 8 {
		return false, false
	}
	off := int(binary.LittleEndian.Uint32(b[0:4]))
	count := binary.LittleEndian.Uint32(b[4:8])
	for i := uint32(0); i < count && off+4 <= len(b); i++ {
		n := int(binary.LittleEndian.Uint32(b[off : off+4]))
		if n <= 0 || off+n > len(b) {
			return false, false
		}
		if n == xtcpcbSize && owns(b[off:off+n], peer, local) {
			return binary.LittleEndian.Uint32(b[off+pcOffUID:]) == uid, true
		}
		off += n
	}
	return false, false
}

// owns: e is the socket whose connection runs from peer to local.
func owns(e []byte, peer, local netip.AddrPort) bool {
	if binary.BigEndian.Uint16(e[pcOffLPort:]) != peer.Port() || binary.BigEndian.Uint16(e[pcOffFPort:]) != local.Port() {
		return false
	}
	switch v := e[pcOffVFlag]; {
	case v&inpIPv4 != 0:
		return addr4(e[pcOffLAddr:], peer.Addr()) && addr4(e[pcOffFAddr:], local.Addr())
	case v&inpIPv6 != 0:
		return addr16(e[pcOffLAddr6:], peer.Addr()) && addr16(e[pcOffFAddr6:], local.Addr())
	}
	return false
}

func addr4(b []byte, a netip.Addr) bool {
	a = a.Unmap()
	return a.Is4() && a.As4() == [4]byte(b)
}

func addr16(b []byte, a netip.Addr) bool {
	return a.As16() == [16]byte(b)
}
