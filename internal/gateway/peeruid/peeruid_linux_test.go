//go:build linux || android

package peeruid

import (
	"net"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// The child mode of TestSameRefusesAnotherUser: dial, stay open, leave.
func TestMain(m *testing.M) {
	if addr := os.Getenv("OWL_PEERUID_DIAL"); addr != "" {
		c, err := net.Dial("tcp", addr)
		if err != nil {
			os.Exit(1)
		}
		time.Sleep(3 * time.Second)
		c.Close()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// A row of /proc/net/tcp from a Linux 5.15 aarch64 machine: 127.0.0.1,
// an established connection from port 52074 to 8001, owned by uid 0.
const realRow = `  38: 0100007F:CB6A 0100007F:1F41 01 00000000:00000000 00:00000000 00000000     0        0 80170 1 ffff00011d258000 20 4 26 10 -1`

func TestLookInFindsTheOwner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tcp")
	body := "  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode\n" + realRow + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	peer := netip.MustParseAddrPort("127.0.0.1:52074")
	local := netip.MustParseAddrPort("127.0.0.1:8001")
	if same, known, err := lookIn(path, peer, local, 0); err != nil || !known || !same {
		t.Fatalf("the peer's own row: same=%v known=%v err=%v, want true, true, nil", same, known, err)
	}
	if same, known, err := lookIn(path, peer, local, 501); err != nil || !known || same {
		t.Fatalf("another uid's row: same=%v known=%v err=%v, want false, true, nil", same, known, err)
	}
	if _, known, _ := lookIn(path, netip.MustParseAddrPort("127.0.0.1:52075"), local, 0); known {
		t.Fatal("a port the peer doesn't hold was taken for its connection")
	}
}

func TestParseNetIPv6(t *testing.T) {
	// ::1, port 631, from the same machine's /proc/net/tcp6.
	a, p, ok := parseNet("00000000000000000000000001000000:0277")
	if !ok || a != netip.MustParseAddr("::1") || p != 631 {
		t.Fatalf("parsed %v:%d ok=%v, want ::1:631", a, p, ok)
	}
}

// TestSameRefusesAnotherUser: a process running as another user, dialing
// this one on loopback, is not this user's. It starts the dialer as uid
// 65534, which needs root; without root it skips.
func TestSameRefusesAnotherUser(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("needs root to run the dialer as another user")
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "OWL_PEERUID_DIAL="+ln.Addr().String())
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 65534, Gid: 65534}}
	if err := cmd.Start(); err != nil {
		t.Skipf("can't start a process as another user: %v", err)
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()
	if err := ln.(*net.TCPListener).SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatal(err)
	}
	c, err := ln.Accept()
	if err != nil {
		t.Fatalf("the other user's dialer didn't connect: %v", err)
	}
	defer c.Close()
	peer, err := netip.ParseAddrPort(c.RemoteAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	local, err := netip.ParseAddrPort(c.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	same, known := Same(peer, local)
	if !known {
		t.Fatal("the other user's connection has no owner in the TCP table")
	}
	if same {
		t.Fatal("another user's process was taken for this one's")
	}
}
