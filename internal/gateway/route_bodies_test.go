package gateway

import (
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/istoo/owl/internal/provider"
)

// A route the trace keeps holds no request body: 60 one-megabyte requests
// left 147 MB live after a collection, 31 with the two things that kept
// them — a route's toolScan held the agent's context, which carries the
// request's holds, its body among them (each Route kept 115 MB of the 147
// across 60 routes), and the model both a route and the Recent-calls ring
// keep was a string over the body's bytes, so every byte of it stayed.
func TestKeptRoutesHoldNoRequestBody(t *testing.T) {
	f := &fake{ctype: "application/json", reply: `{"id":"msg_1","type":"message","role":"assistant","model":"m1","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":300,"output_tokens":1}}`}
	setup(t, provider.Anthropic, f)
	s := New()
	h := s.Handler()
	huge := strings.Repeat("x", 1<<20)
	live := func() uint64 {
		var m runtime.MemStats
		runtime.GC()
		runtime.GC()
		runtime.ReadMemStats(&m)
		return m.HeapAlloc
	}
	ask := func() {
		body := `{"model":"fake/m1","max_tokens":10,"messages":[{"role":"user","content":"` + huge + `"}]}`
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("POST", "/v1/messages", strings.NewReader(body)))
		if rec.Code != 200 {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
	}
	ask() // whatever the first request sets up is not the ring's
	before := live()
	// the trace keeps 60 routes, and each one past 60 drops the oldest:
	// the ring is full at the end
	for i := 0; i < traceKeep; i++ {
		ask()
	}
	grew := int64(live()) - int64(before)
	// room above what is measured: a loaded runner adds noise to a total
	// and -race keeps its own; the bounds are still well under the 147 MB
	// this cost while a route held a body
	bound := int64(45 << 20)
	if raceOn {
		bound = 70 << 20
	}
	t.Logf("%d one-megabyte requests left %d MB live", traceKeep, grew>>20)
	if grew > bound {
		t.Errorf("%d one-megabyte requests kept %d MB live, want at most %d: a route holds a request body again",
			traceKeep, grew>>20, bound>>20)
	}
}
