package gateway

import (
	"net/http/httptest"
	"strings"
	"testing"
	"unsafe"

	"github.com/istoo/owl/internal/provider"
)

// A prompt's items are named — a tool, an MCP server, a file — and a route
// keeps its prompt for 60 requests while the body it was read from should
// go. A name is a string over the body's bytes when decodeJSON shares them
// with a held body, so an uncloned name kept the whole body alive under
// it: a live owl held 56 one-megabyte bodies, 86 MB, with its prompt text
// cache at 10 bodies and 9 MB, and no request in flight.
func TestPromptNamesHoldNoRequestBody(t *testing.T) {
	f := &fake{ctype: "application/json", reply: `{"id":"msg_1","type":"message","role":"assistant","model":"m1","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":300,"output_tokens":1}}`}
	setup(t, provider.Anthropic, f)
	s := New()
	h := s.Handler()
	body := `{"model":"fake/m1","max_tokens":10,` +
		`"tools":[{"name":"DeployTool","description":"d","input_schema":{"type":"object"}}],` +
		`"system":"You are the release agent.",` +
		`"messages":[{"role":"user","content":"` + strings.Repeat("x", 1<<20) + `"}]}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/v1/messages", strings.NewReader(body)))
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	s.trace.mu.Lock()
	bodies := append([]keptBody(nil), s.trace.bodies...)
	routes := append([]*Route(nil), s.trace.routes...)
	s.trace.mu.Unlock()
	if len(bodies) == 0 || len(routes) == 0 {
		t.Fatalf("kept %d bodies, %d routes", len(bodies), len(routes))
	}
	named := 0
	for _, kb := range bodies {
		lo := uintptr(unsafe.Pointer(unsafe.SliceData(kb.body)))
		hi := lo + uintptr(len(kb.body))
		for _, r := range routes {
			if r.Prompt == nil {
				continue
			}
			for _, part := range r.Prompt.Parts {
				for _, it := range part.Items {
					if it.Name == "" {
						continue
					}
					named++
					if p := uintptr(unsafe.Pointer(unsafe.StringData(it.Name))); p >= lo && p < hi {
						t.Errorf("%s item %q is a string over the request's body, which keeps it whole", part.Kind, it.Name)
					}
				}
			}
		}
	}
	if named == 0 {
		t.Fatal("no item was named: the test names nothing")
	}
}
