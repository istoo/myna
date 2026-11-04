package peeruid

import (
	"net"
	"net/netip"
	"runtime"
	"testing"
)

// TestSameOwnConnection: a loopback connection this process made is this
// user's, and the platform can say so.
func TestSameOwnConnection(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			close(accepted)
			return
		}
		accepted <- c
	}()
	cli, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	srv, ok := <-accepted
	if !ok {
		t.Fatal("accept failed")
	}
	defer srv.Close()
	peer, err := netip.ParseAddrPort(srv.RemoteAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	local, err := netip.ParseAddrPort(srv.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	same, known := Same(peer, local)
	switch runtime.GOOS {
	case "darwin", "linux", "android", "windows":
		if !known {
			t.Fatalf("%s: a connection this test made has no owner in the TCP table", runtime.GOOS)
		}
		if !same {
			t.Fatal("a connection this test made is not this user's")
		}
	default:
		if known {
			t.Fatalf("%s: peeruid claims to know an owner it has no reader for", runtime.GOOS)
		}
	}
}
