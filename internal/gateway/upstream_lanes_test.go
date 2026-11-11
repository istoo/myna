package gateway

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// Every request to a vendor shared one HTTP/2 connection, so while one
// agent's turn of a few MB was still going up, every other turn and every
// small request to that vendor waited behind its bytes in the same
// socket: on a ~1MB/s uplink a small request took 2.2-4.1s instead of
// 0.8-1.2s on a connection of its own, and two large turns sent together
// both ended at ~20s. A large body goes out on a connection no other
// large body is using, and a small request on one apart from them.
func TestLargeBodiesGoOutOnConnectionsOfTheirOwn(t *testing.T) {
	fresh(t)
	const n = 3 // two large bodies and a small one, all in flight at once
	var mu sync.Mutex
	from := map[string]string{} // what each request was, by the connection it came on
	arrived := make(chan struct{}, n)
	all := make(chan struct{})
	vendor := h2Vendor(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		from[string(b[:8])] = r.RemoteAddr
		mu.Unlock()
		arrived <- struct{}{}
		select {
		case <-all: // every request is in flight before any is answered
		case <-r.Context().Done():
		}
		io.WriteString(w, r.Proto)
	})
	quickPings(t, vendor) // trusts its certificate; a live vendor answers the pings
	s := New()
	t.Cleanup(s.client.CloseIdleConnections)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	send := func(tag string, size int) error {
		body := []byte(tag + strings.Repeat("a", size))
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, vendor.URL+"/v1/messages", bytes.NewReader(body))
		if err != nil {
			return err
		}
		res, err := s.client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		proto, err := io.ReadAll(res.Body)
		if err == nil && string(proto) != "HTTP/2.0" {
			err = fmt.Errorf("%s went over %s, want HTTP/2.0, which keeps its ping", tag, proto)
		}
		return err
	}
	errs := make(chan error, n)
	go func() { errs <- send("large-1:", 256<<10) }()
	go func() { errs <- send("large-2:", 256<<10) }()
	go func() { errs <- send("small-1:", 100) }()
	for range n {
		select {
		case <-arrived:
		case <-ctx.Done():
			t.Fatal("not every request reached the vendor")
		}
	}
	close(all)
	for range n {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	conns := map[string][]string{}
	for tag, addr := range from {
		conns[addr] = append(conns[addr], tag)
	}
	if len(conns) != n {
		t.Fatalf("%d requests in flight together went out on %d connections, want one each: %v", n, len(conns), conns)
	}
	// one alone, after them, goes out on a connection already open
	if err := send("large-3:", 256<<10); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if _, ok := conns[from["large-3:"]]; !ok {
		t.Errorf("a large body sent alone dialled a new connection (%s); want one of %v", from["large-3:"], conns)
	}
}
