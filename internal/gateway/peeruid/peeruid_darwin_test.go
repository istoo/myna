//go:build darwin

package peeruid

import (
	"encoding/binary"
	"net/netip"
	"testing"
)

// pcblist builds a net.inet.tcp.pcblist holding one entry.
func pcblist(entry []byte) []byte {
	binary.LittleEndian.PutUint32(entry[0:4], uint32(len(entry))) // xt_len
	b := make([]byte, 24+len(entry))
	binary.LittleEndian.PutUint32(b[0:], 24)
	binary.LittleEndian.PutUint32(b[4:], 1)
	copy(b[24:], entry)
	return b
}

func TestSameInReadsTheOwner(t *testing.T) {
	e := make([]byte, xtcpcbSize)
	binary.BigEndian.PutUint16(e[pcOffLPort:], 60000)
	binary.BigEndian.PutUint16(e[pcOffFPort:], 3425)
	e[pcOffVFlag] = inpIPv4
	copy(e[pcOffLAddr:], []byte{127, 0, 0, 1})
	copy(e[pcOffFAddr:], []byte{127, 0, 0, 1})
	binary.LittleEndian.PutUint32(e[pcOffUID:], 501)
	b := pcblist(e)
	peer := netip.MustParseAddrPort("127.0.0.1:60000")
	local := netip.MustParseAddrPort("127.0.0.1:3425")

	if same, known := sameIn(b, peer, local, 501); !known || !same {
		t.Fatalf("the peer's own socket: same=%v known=%v, want true, true", same, known)
	}
	if same, known := sameIn(b, peer, local, 502); !known || same {
		t.Fatalf("a socket of another uid: same=%v known=%v, want false, true", same, known)
	}
	if _, known := sameIn(b, netip.MustParseAddrPort("127.0.0.1:60001"), local, 501); known {
		t.Fatal("a port the peer doesn't hold was taken for its connection")
	}
	if _, known := sameIn(b, netip.MustParseAddrPort("127.0.0.2:60000"), local, 501); known {
		t.Fatal("another address's socket was taken for the peer's")
	}
}

func TestSameInReadsAnIPv6Entry(t *testing.T) {
	e := make([]byte, xtcpcbSize)
	binary.BigEndian.PutUint16(e[pcOffLPort:], 60000)
	binary.BigEndian.PutUint16(e[pcOffFPort:], 3425)
	e[pcOffVFlag] = inpIPv6
	copy(e[pcOffLAddr6:], netip.MustParseAddr("::1").AsSlice())
	copy(e[pcOffFAddr6:], netip.MustParseAddr("::1").AsSlice())
	binary.LittleEndian.PutUint32(e[pcOffUID:], 501)
	b := pcblist(e)
	peer := netip.MustParseAddrPort("[::1]:60000")
	local := netip.MustParseAddrPort("[::1]:3425")
	if same, known := sameIn(b, peer, local, 501); !known || !same {
		t.Fatalf("an IPv6 peer's socket: same=%v known=%v, want true, true", same, known)
	}
}
