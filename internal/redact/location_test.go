package redact

import (
	"encoding/json"
	"strings"
	"testing"
)

// A string that only begins with "data:" is text someone wrote, not an
// image's encoded bytes: it is masked like any other text, and in a tool
// call's arguments too (#4130).
func TestDataPrefixedTextIsMasked(t *testing.T) {
	o := Options{Secrets: true}
	const key = "sk-ant-api03-abcdefghijklmnopqrstuvwxyz012345"
	bodies := []string{
		`{"model":"claude","messages":[{"role":"user","content":"data: my key is ` + key + `"}]}`,
		`{"model":"gpt-5.5","input":[{"type":"function_call","call_id":"c1","name":"run","arguments":"{\"cmd\":\"data: ` + key + `\"}"}]}`,
	}
	for _, body := range bodies {
		out, n := MaskJSON([]byte(body), o)
		if n == 0 || strings.Contains(string(out), key) {
			t.Errorf("masked %d: %s", n, out)
		}
	}
	// encoded bytes stay whole
	img := `data:image/png;base64,iVBORw0KGgoAAAANSUhEUg`
	body := `{"model":"claude","messages":[{"role":"user","content":[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"iVBORw0KGgoAAAANSUhEUg"}}]}]}`
	if out, _ := MaskJSON([]byte(body), o); strings.Contains(string(out), img) != strings.Contains(body, img) {
		t.Errorf("an image's data: URL was changed: %s", out)
	}
}

// Anthropic's mcp_tool_use names its arguments as tool_use does: a field
// called for a secret there holds what the user or the server wrote
// (#4130).
func TestMCPToolUseInputIsArguments(t *testing.T) {
	o := Options{Secrets: true}
	// Hunter2 has no shape a rule knows: only the field's name says it
	// is a secret, which is what being a tool call's arguments means
	const pass = "hunter2hunter2"
	body := `{"model":"claude","messages":[{"role":"assistant","content":[{"type":"mcp_tool_use","id":"t1","name":"db","server_name":"db","input":{"password":"` + pass + `","signature":"not-a-secret"}}]}]}`
	out, n := MaskJSON([]byte(body), o)
	if n == 0 || strings.Contains(string(out), pass) {
		t.Fatalf("masked %d: %s", n, out)
	}
	// the tool's own non-secret fields are the tool's, not the vendor's:
	// this one is not a signature block, so it is masked only by shape
	var v any
	if err := json.Unmarshal(out, &v); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
}

// A document whose source is text keeps the user's words in source.data:
// that is not an image's encoded bytes, and is masked (#4130).
func TestPlainTextDocumentIsMasked(t *testing.T) {
	o := Options{Secrets: true}
	const key = "sk-ant-api03-abcdefghijklmnopqrstuvwxyz012345"
	body := `{"model":"claude","messages":[{"role":"user","content":[{"type":"document","source":{"type":"text","media_type":"text/plain","data":"my key is ` + key + `"}}]}]}`
	out, n := MaskJSON([]byte(body), o)
	if n == 0 || strings.Contains(string(out), key) {
		t.Fatalf("masked %d: %s", n, out)
	}
	// an image document's base64 is still kept
	img := `{"model":"claude","messages":[{"role":"user","content":[{"type":"document","source":{"type":"base64","media_type":"image/png","data":"iVBORw0KGgoAAAANSUhEUg"}}]}]}`
	if got, _ := MaskJSON([]byte(img), o); strings.Contains(string(got), "iVBORw0KGgoAAAANSUhEUg") {
		return
	} else {
		t.Errorf("an image document's bytes were changed: %s", got)
	}
}
