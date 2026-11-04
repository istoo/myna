//go:build windows

package peeruid

import (
	"errors"
	"net/netip"
	"unsafe"

	"golang.org/x/sys/windows"
)

// GetExtendedTcpTable (iphlpapi) lists every TCP connection with the
// process that owns it; that process's token names its user, which is
// compared with this one's.
const (
	tcpTableOwnerPidAll     = 5
	errorInsufficientBuffer = 122
)

var procGetExtendedTcpTable = windows.NewLazySystemDLL("iphlpapi.dll").NewProc("GetExtendedTcpTable")

func sameUser(peer, local netip.AddrPort) (bool, bool) {
	af := uint32(windows.AF_INET)
	if !peer.Addr().Is4() {
		af = uint32(windows.AF_INET6)
	}
	table, err := tcpTable(af)
	if err != nil {
		return false, false
	}
	pid, ok := winOwnerIn(table, af == uint32(windows.AF_INET6), peer, local)
	if !ok || pid == 0 {
		return false, false
	}
	return pidIsThisUser(pid)
}

// tcpTable is the table af's connections with their owning processes.
func tcpTable(af uint32) ([]byte, error) {
	var size uint32
	r, _, _ := procGetExtendedTcpTable.Call(0, uintptr(unsafe.Pointer(&size)), 0, uintptr(af), tcpTableOwnerPidAll, 0)
	if r != errorInsufficientBuffer && r != 0 {
		return nil, windows.Errno(r)
	}
	if size < 4 {
		return nil, errors.New("peeruid: the TCP table is empty")
	}
	buf := make([]byte, size)
	r, _, _ = procGetExtendedTcpTable.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), 0, uintptr(af), tcpTableOwnerPidAll, 0)
	if r != 0 {
		return nil, windows.Errno(r)
	}
	return buf, nil
}

// pidIsThisUser: the process pid runs as this user. known is false when
// its token can't be read (another user's process, typically).
func pidIsThisUser(pid uint32) (same, known bool) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return false, false
	}
	defer windows.CloseHandle(h)
	var tok windows.Token
	if err := windows.OpenProcessToken(h, windows.TOKEN_QUERY, &tok); err != nil {
		return false, false
	}
	defer tok.Close()
	them, err := tokenSID(tok)
	if err != nil {
		return false, false
	}
	// WSL's mirrored networking reaches Windows' loopback through a
	// service (a system SID), and that is this computer's own agent: it
	// keeps what was done before, as an unreadable owner does.
	if them.IsWellKnown(windows.WinLocalSystemSid) || them.IsWellKnown(windows.WinLocalServiceSid) || them.IsWellKnown(windows.WinNetworkServiceSid) {
		return false, false
	}
	me, err := tokenSID(windows.GetCurrentProcessToken())
	if err != nil {
		return false, false
	}
	return them.Equals(me), true
}

func tokenSID(t windows.Token) (*windows.SID, error) {
	var n uint32
	_ = windows.GetTokenInformation(t, windows.TokenUser, nil, 0, &n)
	if n == 0 {
		return nil, errors.New("peeruid: no token user")
	}
	buf := make([]byte, n)
	if err := windows.GetTokenInformation(t, windows.TokenUser, &buf[0], n, &n); err != nil {
		return nil, err
	}
	return (*windows.Tokenuser)(unsafe.Pointer(&buf[0])).User.Sid.Copy()
}
