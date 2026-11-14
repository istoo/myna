package gateway

import (
	"context"
	"net/http/httptest"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/istoo/owl/internal/proc"
	"github.com/istoo/owl/internal/provider"
	"github.com/istoo/owl/internal/settings"
)

// A route the trace keeps holds no request body: 60 one-megabyte requests
// left 147 MB live after a collection, 31 with the two things that kept
// them — a route's toolScan held the agent's context, which carries the
// request's holds, its body among them (each Route kept 115 MB of the 147
// across 60 routes), and the model both a route and the Recent-calls ring
// keep was a string over the body's bytes, so every byte of it stayed.
func TestKeptRoutesHoldNoRequestBody(t *testing.T) {
	if alone(t) {
		return
	}
	f := &fake{ctype: "application/json", reply: `{"id":"msg_1","type":"message","role":"assistant","model":"m1","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":300,"output_tokens":1}}`}
	setup(t, provider.Anthropic, f)
	s := New()
	h := s.Handler()
	routeGrowth(t, func(huge string) {
		body := `{"model":"fake/m1","max_tokens":10,"messages":[{"role":"user","content":"` + huge + `"}]}`
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("POST", "/v1/messages", strings.NewReader(body)))
		if rec.Code != 200 {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
	})
}

// routeBodiesChild is set in the process alone starts for a test.
const routeBodiesChild = "OWL_TEST_ROUTE_BODIES"

// alone runs t again in a process of its own, with nothing else of the
// package's in it, and reports whether it did (the caller returns then).
// The live heap a test measures is the whole process's: beside the
// package's other tests it held what they had left running, and in the
// package's order the same fills moved by 70, 58 and -84 MB.
func alone(t *testing.T) bool {
	t.Helper()
	if os.Getenv(routeBodiesChild) != "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := proc.CommandContext(ctx, os.Args[0], "-test.run=^"+t.Name()+"$", "-test.timeout=9m", "-test.v")
	cmd.Env = append(os.Environ(), routeBodiesChild+"=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s in a process of its own: %v\n%s", t.Name(), err, out)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "grew the heap") {
			t.Log(strings.TrimSpace(line))
		}
	}
	return true
}

// routeGrowth fills the trace's routes with requests of a one-megabyte
// body, then replaces every one of them with a request of four, and fails
// when that grew the live heap by what kept routes holding their bodies
// would. A route that holds its body (or a copy of it) grows it by 3 MB a
// route, 180 MB at least; one that holds none grows it only by what keeps a
// few bodies on purpose (the prompt text's last promptBodies, Recent
// calls' first 256 KiB of each), ~30 MB, which doesn't scale with the
// routes. A difference of two fills, not a total against a bound: what else
// a loaded runner's process holds at the moment moved the total by ~25 MB
// (one fill of 1 MB left 49 MB on ubuntu CI at 880db0b90, 18 to 26 here),
// over a bound that was 20 MB above it; it is measured alone.
func routeGrowth(t *testing.T, ask func(huge string)) {
	t.Helper()
	live := func() uint64 {
		var m runtime.MemStats
		runtime.GC()
		runtime.GC()
		runtime.ReadMemStats(&m)
		return m.HeapAlloc
	}
	small, big := strings.Repeat("x", 1<<20), strings.Repeat("x", 4<<20)
	// the trace keeps traceKeep routes, each one past them dropping the
	// oldest: full of small ones, then every one of them a big one
	for i := 0; i <= traceKeep; i++ {
		ask(small)
	}
	before := live()
	for i := 0; i < traceKeep; i++ {
		ask(big)
	}
	grew := int64(live()) - int64(before)
	bound := int64(90 << 20)
	if raceOn {
		bound = 110 << 20
	}
	t.Logf("%d routes of 1 MB replaced by %d of 4 MB grew the heap by %d MB", traceKeep, traceKeep, grew>>20)
	if grew > bound {
		t.Errorf("replacing %d routes of 1 MB by %d of 4 MB grew the heap by %d MB, want at most %d: a route holds a request body again",
			traceKeep, traceKeep, grew>>20, bound>>20)
	}
}

// The same, for a request as Claude Code sends one: its effort, a service
// tier, a value masking swaps for a placeholder (so the vendor is told of
// placeholders, written into a copy of the body) and Chat at the other end
// (so it is translated). effortOf handed back the level as the body had
// it, a string over the body's bytes, and every route kept its body whole
// in its tries' efforts: a live owl held 58 bodies, 224 MB, from the
// placeholder note's copy alone, and 60 one-megabyte requests here 191 MB.
func TestKeptRoutesHoldNoTranslatedBody(t *testing.T) {
	if alone(t) {
		return
	}
	for _, stream := range []bool{false, true} {
		t.Run(map[bool]string{false: "whole", true: "streamed"}[stream], func(t *testing.T) {
			f := &fake{reply: sse(`data: {"id":"c1","object":"chat.completion.chunk","model":"m1","choices":[{"index":0,"delta":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":300,"completion_tokens":1}}`, `data: [DONE]`)}
			setup(t, provider.Chat, f)
			if err := settings.Save(settings.Settings{Redact: true, RedactWords: []string{"百度"}}); err != nil {
				t.Fatal(err)
			}
			s := New()
			h := s.Handler()
			ask := func(huge string) {
				body := `{"model":"fake/m1","max_tokens":10,"stream":` + strconv.FormatBool(stream) +
					`,"thinking":{"type":"adaptive"},"output_config":{"effort":"high"},"service_tier":"auto",` +
					`"messages":[{"role":"user","content":"百度 ` + huge + `"}]}`
				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, httptest.NewRequest("POST", "/v1/messages", strings.NewReader(body)))
				if rec.Code != 200 {
					t.Fatalf("%d %s", rec.Code, rec.Body)
				}
			}
			ask("x")
			if !strings.Contains(string(f.got), "placeholder") {
				t.Fatalf("the vendor wasn't told of placeholders, so the note's copy isn't tested: %.300s", f.got)
			}
			routeGrowth(t, ask)
		})
	}
}
