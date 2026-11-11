package gateway

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// TestAnotherUsersLoopbackRequestIsNotLocal: a loopback request whose
// connection belongs to another process than this user's is not local, so
// it is answered as from another machine; this user's is, as before.
func TestAnotherUsersLoopbackRequestIsNotLocal(t *testing.T) {
	r := httptest.NewRequest("GET", "http://127.0.0.1:3425/api/state", nil)
	r.RemoteAddr = "127.0.0.1:12345"
	if !local(r) {
		t.Fatal("a loopback request with no owner known was not taken as local")
	}
	if local(r.WithContext(context.WithValue(r.Context(), peerUserKey{}, false))) {
		t.Fatal("another user's loopback request was taken as local")
	}
	if !local(r.WithContext(context.WithValue(r.Context(), peerUserKey{}, true))) {
		t.Fatal("this user's loopback request was not taken as local")
	}
}

// TestPeerUserContextOnARealConnection: the gateway's server marks a
// connection this process made as this user's, through ConnContext.
func TestPeerUserContextOnARealConnection(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{ConnContext: peerUserContext, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, strconv.FormatBool(sameOSUser(r)))
	})}
	go srv.Serve(ln)
	defer srv.Close()
	resp, err := http.Get("http://" + ln.Addr().String() + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "true" {
		t.Fatalf("this user's own connection was marked %q, want true", body)
	}
}
