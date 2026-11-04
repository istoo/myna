package peeruid

import (
	"encoding/binary"
	"net/netip"
	"testing"
)

// A TCP_TABLE_OWNER_PID_ALL row as Windows lays it out: the address bytes
// as they are, the port's network-order bytes at the start of its DWORD.
func TestWinOwnerInIPv4(t *testing.T) {
	row := make([]byte, winIPv4Row)
	binary.LittleEndian.PutUint32(row[0:], 2) // MIB_TCP_STATE_ESTAB
	copy(row[4:8], []byte{127, 0, 0, 1})
	binary.BigEndian.PutUint16(row[8:], 8080)
	copy(row[12:16], []byte{127, 0, 0, 1})
	binary.BigEndian.PutUint16(row[16:], 3425)
	binary.LittleEndian.PutUint32(row[20:], 1234)
	table := make([]byte, 4+len(row))
	binary.LittleEndian.PutUint32(table[0:], 1)
	copy(table[4:], row)

	peer := netip.MustParseAddrPort("127.0.0.1:8080")
	local := netip.MustParseAddrPort("127.0.0.1:3425")
	if pid, ok := winOwnerIn(table, false, peer, local); !ok || pid != 1234 {
		t.Fatalf("pid=%d ok=%v, want 1234, true", pid, ok)
	}
	if _, ok := winOwnerIn(table, false, netip.MustParseAddrPort("127.0.0.1:8081"), local); ok {
		t.Fatal("a port the peer doesn't hold was taken for its connection")
	}
	if _, ok := winOwnerIn(table, false, peer, netip.MustParseAddrPort("127.0.0.1:3426")); ok {
		t.Fatal("a connection to another local port was taken for the peer's")
	}
	if _, ok := winOwnerIn(table, false, netip.MustParseAddrPort("127.0.0.2:8080"), local); ok {
		t.Fatal("another address's socket was taken for the peer's")
	}
}

func TestWinOwnerInIPv6(t *testing.T) {
	row := make([]byte, winIPv6Row)
	copy(row[0:16], netip.MustParseAddr("::1").AsSlice())
	binary.BigEndian.PutUint16(row[20:], 8080)
	copy(row[24:40], netip.MustParseAddr("::1").AsSlice())
	binary.BigEndian.PutUint16(row[44:], 3425)
	binary.LittleEndian.PutUint32(row[48:], 2)
	binary.LittleEndian.PutUint32(row[52:], 5678)
	table := make([]byte, 4+len(row))
	binary.LittleEndian.PutUint32(table[0:], 1)
	copy(table[4:], row)

	peer := netip.MustParseAddrPort("[::1]:8080")
	local := netip.MustParseAddrPort("[::1]:3425")
	if pid, ok := winOwnerIn(table, true, peer, local); !ok || pid != 5678 {
		t.Fatalf("pid=%d ok=%v, want 5678, true", pid, ok)
	}
}
