package peeruid

import (
	"encoding/binary"
	"net/netip"
)

// The rows of GetExtendedTcpTable's TCP_TABLE_OWNER_PID_ALL (Windows).
// The table starts with its row count; an IPv4 row is six DWORDs, an IPv6
// row carries 16-byte addresses. Ports are in network byte order in the
// low 16 bits of their DWORD, which is why they are read big-endian.
const (
	winIPv4Row = 24
	winIPv6Row = 56
)

// winOwnerIn finds the process whose TCP connection runs from peer to
// local. v6 says which table it is.
func winOwnerIn(t []byte, v6 bool, peer, local netip.AddrPort) (uint32, bool) {
	if len(t) < 4 {
		return 0, false
	}
	n := int(binary.LittleEndian.Uint32(t))
	rowSize := winIPv4Row
	if v6 {
		rowSize = winIPv6Row
	}
	for i := 0; i < n; i++ {
		o := 4 + i*rowSize
		if o+rowSize > len(t) {
			return 0, false
		}
		row := t[o : o+rowSize]
		var laddr, raddr []byte
		var lport, rport uint16
		var pid uint32
		if !v6 {
			laddr, raddr = row[4:8], row[12:16]
			lport, rport = binary.BigEndian.Uint16(row[8:10]), binary.BigEndian.Uint16(row[16:18])
			pid = binary.LittleEndian.Uint32(row[20:24])
		} else {
			laddr, raddr = row[0:16], row[24:40]
			lport, rport = binary.BigEndian.Uint16(row[20:22]), binary.BigEndian.Uint16(row[44:46])
			pid = binary.LittleEndian.Uint32(row[52:56])
		}
		if lport != peer.Port() || rport != local.Port() {
			continue
		}
		if addrEq(laddr, peer.Addr()) && addrEq(raddr, local.Addr()) {
			return pid, true
		}
	}
	return 0, false
}

func addrEq(b []byte, a netip.Addr) bool {
	if len(b) == 4 {
		a = a.Unmap()
		return a.Is4() && a.As4() == [4]byte(b)
	}
	return a.As16() == [16]byte(b)
}
