package gateway

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/istoo/owl/internal/settings"
)

// maskingSSE is a Chat upstream that keeps what it was sent and answers
// as a stream, the way every translation's upstream does.
type maskingSSE struct {
	mu     sync.Mutex
	bodies [][]byte
}

func (u *maskingSSE) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, _ := io.ReadAll(r.Body)
	u.mu.Lock()
	u.bodies = append(u.bodies, b)
	u.mu.Unlock()
	w.Header().Set("Content-Type", "text/event-stream")
	io.WriteString(w, sse(
		`data: {"id":"c1","object":"chat.completion.chunk","model":"m","choices":[{"index":0,"delta":{"role":"assistant","content":"OK"}}]}`,
		`data: {"id":"c1","object":"chat.completion.chunk","model":"m","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":2,"completion_tokens":1}}`,
		`data: [DONE]`))
}

func (u *maskingSSE) count() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return len(u.bodies)
}

func (u *maskingSSE) last() []byte {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.bodies[len(u.bodies)-1]
}

// A secret the fixture carries, as an agent's tool output would.
const fixtureSecret = "sk-proj-abcdEFGH1234ijklMNOP5678qrst"

// claudeCodeRequest is an Anthropic Messages request shaped like Claude
// Code's: a system prompt and tools, then turns of a tool call with its
// arguments and a tool result with the user's words. Every turn carries
// the secret, and the tool result is long, as a file's contents are.
func claudeCodeRequest(turns int) []byte {
	var b strings.Builder
	b.WriteString(`{"model":"pa/m","max_tokens":8192,"stream":true,"system":[{"type":"text","text":"You are Claude Code, Anthropic's official CLI. `)
	b.WriteString(strings.Repeat("Work carefully and read before you write. ", 60))
	b.WriteString(`"}],"tools":[{"name":"Read","description":"Read a file's contents","input_schema":{"type":"object","properties":{"file_path":{"type":"string"}}}}],"messages":[{"role":"user","content":[{"type":"text","text":"read the files I list"}]}`)
	body := strings.Repeat("read this file and tell me what it does\n", 150)
	for i := range turns {
		fmt.Fprintf(&b, `,{"role":"assistant","content":[{"type":"text","text":"It reads a file. %s"},{"type":"tool_use","id":"toolu_%04d","name":"Read","input":{"file_path":"/tmp/file%d.go","api_key":"%s"}}]}`,
			strings.Repeat("ok ", 40), i, i, fixtureSecret)
		fmt.Fprintf(&b, `,{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_%04d","content":"%s"},{"type":"text","text":"the key is %s, mail bob%04d@example.org"}]}`,
			i, strings.ReplaceAll(body, "\n", "\\n"), fixtureSecret, i)
	}
	b.WriteString(`]}`)
	return []byte(b.String())
}

// postWithCtx serves body at path as a request whose context can be
// cancelled, so holdBody shares the request's bytes (a test request's
// context has no Done otherwise).
func postWithCtx(t *testing.T, s *Server, path string, body []byte) (int, string) {
	t.Helper()
	ctx, done := context.WithCancel(context.Background())
	defer done()
	rec := httptest.NewRecorder()
	// the bytes as a client sends them: the reader holds its own copy, the
	// same in every measurement here
	req := httptest.NewRequest("POST", path, strings.NewReader(string(body))).WithContext(ctx)
	s.Handler().ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

// oneRequestAllocs reports what one of n requests allocates, after a
// warm-up that fills what masking remembers of a text sent before.
func oneRequestAllocs(t *testing.T, n int, f func()) (bytes, mallocs uint64) {
	t.Helper()
	f()
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	for range n {
		f()
	}
	runtime.ReadMemStats(&after)
	return (after.TotalAlloc - before.TotalAlloc) / uint64(n), (after.Mallocs - before.Mallocs) / uint64(n)
}

// A Claude Code request masked and translated to a Chat upstream
// allocates at most this much. Masking walked the body three to five
// times, each string and path a copy, and every reader decoded the request
// again: a 1 MB conversation cost 33.6 MB and ~80k allocations a request,
// on a gateway that had allocated 395 GB in 6.5 hours.
func TestMaskedTranslatedRequestAlloc(t *testing.T) {
	fresh(t)
	if err := settings.Save(settings.Settings{Redact: true}); err != nil {
		t.Fatal(err)
	}
	up := &maskingSSE{}
	serveOn(t, "pa", "ka", []string{"m"}, up)
	s := New()
	body := claudeCodeRequest(150)
	if len(body) < 900_000 || len(body) > 1_200_000 {
		t.Fatalf("the fixture is %d bytes, want about 1 MB", len(body))
	}
	req := func() {
		code, reply := postWithCtx(t, s, "/v1/messages", body)
		if code != 200 {
			t.Fatalf("%d %s", code, reply)
		}
	}
	req()
	if up.count() != 1 {
		t.Fatalf("the upstream was asked %d times", up.count())
	}
	// the body went masked: the fixture's secret did not reach the vendor
	if sent := string(up.last()); strings.Contains(sent, fixtureSecret) {
		t.Fatalf("a secret reached the vendor: %.300s", sent)
	}
	per, mallocs := oneRequestAllocs(t, 4, req)
	// room above what is measured, as a loaded runner adds noise to a
	// total and -race counts its own allocations; the bounds are still
	// well under the 33.6 MB and 79.8k allocations a request cost before
	// the request was parsed and masked once
	bound, mallocBound := uint64(26<<20), uint64(55_000)
	if raceOn {
		bound, mallocBound = 45<<20, 60_000
	}
	t.Logf("%d-byte request: %d bytes and %d allocations a request", len(body), per, mallocs)
	if per > bound {
		t.Errorf("a %d-byte masked request allocated %d bytes, want at most %d", len(body), per, bound)
	}
	if mallocs > mallocBound {
		t.Errorf("a %d-byte masked request made %d allocations, want at most %d", len(body), mallocs, mallocBound)
	}
}

// placeholderShape is a placeholder masking writes, as the vendor sees it:
// its kind and the 8 of its keyed hash. The hash is made with a key each
// run makes its own, so a golden holds the kind and the text around it.
var placeholderShape = regexp.MustCompile(`\{\{([A-Za-z][A-Za-z0-9_]*)_[A-Za-z2-7]{8}\}\}`)

// cacheKeyShape is the per-run key a Chat request carries to a vendor's
// prompt cache: a UUID made when the request is, not a value the request
// or masking decides.
var cacheKeyShape = regexp.MustCompile(`"prompt_cache_key":"[0-9a-f-]{36}"`)

// normalized replaces each placeholder with its kind, so what the vendor
// was sent can be compared across runs, and the per-run cache key with a
// fixed one.
func normalized(b []byte) []byte {
	b = placeholderShape.ReplaceAllFunc(b, func(m []byte) []byte {
		body := m[2 : len(m)-2]
		i := bytes.LastIndexByte(body, '_')
		out := append([]byte("{{"), body[:i]...)
		return append(out, '}', '}')
	})
	return cacheKeyShape.ReplaceAll(b, []byte(`"prompt_cache_key":"key"`))
}

// TestMaskedTranslatedBodyIsStable is the behavior's own check: the bytes
// a Chat upstream is sent for the same masked request are the golden's,
// byte for byte (each placeholder as its kind, its hash being keyed per
// run). The golden was recorded before the request was parsed and masked
// once.
func TestMaskedTranslatedBodyIsStable(t *testing.T) {
	fresh(t)
	if err := settings.Save(settings.Settings{Redact: true}); err != nil {
		t.Fatal(err)
	}
	up := &maskingSSE{}
	serveOn(t, "pa", "ka", []string{"m"}, up)
	s := New()
	body := claudeCodeRequest(4)
	code, reply := postWithCtx(t, s, "/v1/messages", body)
	if code != 200 || !strings.Contains(reply, "OK") {
		t.Fatalf("%d %s", code, reply)
	}
	sent := up.last()
	n := normalized(sent)
	sum := sha256.Sum256(n)
	got := fmt.Sprintf("%x %d %d", sum, len(n), len(sent))
	if got != maskedGolden {
		t.Fatalf("the upstream was sent %d bytes, normalized %d, sha256 %x; want %s", len(sent), len(n), sum, maskedGolden)
	}
}

// maskedGolden is the sha256 and length of the normalized Chat body an
// upstream was sent for claudeCodeRequest(4) under masking, recorded on
// the code before the request was parsed and masked once, and the length
// of the body itself.
const maskedGolden = "4c107bdcfbb4aaa6bdc6657dfdba1b5ff98188dda949594449d888d6d39dec57 29649 29754"
