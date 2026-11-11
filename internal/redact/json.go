package redact

import (
	"bytes"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

// kept are the keys whose strings are left alone: ids, names and kinds the
// vendor needs as they are, and what is sealed or encoded (a thinking
// block's signature, reasoning's encrypted content, an image's data). A
// tool is named by its name and, for an MCP tool, its server's: a
// Responses function_call's namespace, an mcp_call's server_label,
// Anthropic's mcp_tool_use server_name. Codex checks a namespace against
// ^[a-zA-Z0-9_-]+$, so one with a word masked in it was a 400 on every turn
// that replayed the call (#3143).
var kept = map[string]bool{
	"signature": true, "encrypted_content": true, "data": true, "thoughtSignature": true, "thought_signature": true,
	"model": true, "id": true, "tool_use_id": true, "call_id": true, "item_id": true, "type": true, "role": true,
	"name": true, "namespace": true, "server_label": true, "server_name": true, "previous_response_id": true, "prompt_cache_key": true, "media_type": true, "mime_type": true,
	"mimeType": true, "url": true, "image_url": true, "file_id": true, "reasoning_effort": true, "effort": true,
	"stop_reason": true, "finish_reason": true, "status": true, "object": true, "event": true,
}

// keepIn is keep for a request's string at path, with its ids kept too.
// In a tool call's arguments (args) the keys are the tool's, not the
// vendor's: a "signature", "name" or "user_id" there holds what the agent
// wrote, so only an encoded data: value is kept. In what the agent defines
// for the vendor to check (defs: its tools, their schemas and grammars, the
// output's schema, the request's settings) only a description is masked:
// the rest is read by the vendor as structure, and a word masked inside a
// Lark grammar's rule name ("pd" in update_hunk) left a grammar the vendor
// turned away with 400 "Invalid lark grammar" (#1911).
func keepIn(path []byte, key, s string, args, defs []string) bool {
	if under(path, args) {
		// the keys are the tool's, so only encoded bytes are kept, and
		// only ones that are data: URLs: "data:" on its own is a word
		// the agent or the user wrote, not an image
		return isDataURL(s)
	}
	if isDataURL(s) {
		return true
	}
	if under(path, defs) {
		return key != "description"
	}
	return kept[key] || isID(key)
}

// isDataURL says s is a data: URL carrying encoded bytes (an image, an
// audio clip), which no rule can read and which a vendor needs whole. A
// string that only begins with "data:" is someone's text.
func isDataURL(s string) bool {
	if !strings.HasPrefix(s, "data:") {
		return false
	}
	comma := strings.IndexByte(s, ',')
	return comma > 0 && strings.Contains(s[:comma], ";base64")
}

// isID says a key holds an identifier the vendor matches against another:
// a tool result's tool_call_id against its call's id, an approval's
// approval_request_id against its request. A vendor's ids are its own
// letters and digits, and GLM's call_ and 19 digits can read as a bank
// card number; masked on one side only, the two no longer match.
func isID(key string) bool {
	return key == "id" || key == "ids" || strings.HasSuffix(key, "_id") || strings.HasSuffix(key, "_ids") ||
		strings.HasSuffix(key, "Id") || strings.HasSuffix(key, "Ids")
}

// signatureKeys are the keys of what a vendor seals the content beside it
// with: Anthropic's thinking signature, Gemini's thought signature,
// Responses' encrypted reasoning.
var signatureKeys = []string{"signature", "thoughtSignature", "thought_signature", "encrypted_content"}

// scan finds three kinds of objects in body. signed are those a vendor
// sealed with a signature: what is in them the vendor wrote and checks, so
// it goes back exactly as the vendor wrote it. args are a tool call's
// arguments (Anthropic's tool_use input, Gemini's functionCall args), whose
// keys are the tool's own, as are those of Gemini's functionResponse
// response, the structured result the tool gave; nothing in them counts
// as sealed. defs are what the agent defines for the vendor rather than
// says to the model (definitions).
//
// It reads the request's structure without decoding it, and reads a
// value only of the keys it cares about: a request's whole body decoded
// into maps and slices cost 2.9 MB on every turn, its tool outputs
// unquoted again to look for a "type" and a "data".
func scan(body []byte) (signed, args, defs, textDocs []string) {
	found := false
	for _, k := range append(signatureKeys, "tool_use", "mcp_tool_use", "functionCall", "function_call", "functionResponse", "function_response", "document") {
		found = found || bytes.Contains(body, []byte(`"`+k+`"`))
	}
	for _, k := range defKeys {
		found = found || bytes.Contains(body, []byte(`"`+k+`"`))
	}
	if !found {
		return nil, nil, nil, nil
	}
	defs = definitionsOf(body)
	// obj is what the object being visited says it is, as its keys come
	type objFacts struct {
		typ                                       string
		sig, thought, data, input, args, response bool
	}
	var stack []objFacts
	w := walker{in: body, path: make([]byte, 0, 64), only: scanKeys}
	w.fn = func(path []byte, key, s string) string {
		if len(stack) > 0 {
			t := &stack[len(stack)-1]
			switch key {
			case "type":
				t.typ = s
			case "signature", "encrypted_content":
				if s != "" {
					t.sig = true
				}
			case "thoughtSignature", "thought_signature":
				if s != "" {
					t.sig, t.thought = true, true
				}
			}
		}
		return s
	}
	w.obj = func(path []byte, key string, open bool) {
		if open {
			stack = append(stack, objFacts{})
			return
		}
		if len(stack) == 0 {
			return
		}
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		// sealed(x): an object a vendor signed (sealed, in the types
		// that carry one)
		thinkingLike := f.typ == "thinking" || f.typ == "reasoning" || strings.HasPrefix(f.typ, "reasoning.")
		if f.thought || thinkingLike && f.sig {
			signed = append(signed, string(path))
			return
		}
		if f.typ == "text" && f.data {
			// Anthropic's document block carries its text in source.data
			// when source.type is "text": that is the user's document, not
			// an image's encoded bytes, so it is masked like any text.
			textDocs = append(textDocs, join(string(path), "data"))
		}
		switch {
		case f.input && (f.typ == "tool_use" || f.typ == "server_tool_use" || f.typ == "mcp_tool_use"):
			args = append(args, join(string(path), "input"))
		case f.args && (key == "functionCall" || key == "function_call"):
			args = append(args, join(string(path), "args"))
		case f.response && (key == "functionResponse" || key == "function_response"):
			args = append(args, join(string(path), "response"))
		}
	}
	w.member = func(path []byte, key string) {
		if len(stack) == 0 {
			return
		}
		t := &stack[len(stack)-1]
		switch key {
		case "data":
			t.data = true
		case "input":
			t.input = true
		case "args":
			t.args = true
		case "response":
			t.response = true
		}
	}
	i, good := w.value(w.ws(0), "")
	if !good || w.ws(i) != len(body) {
		return nil, nil, nil, nil
	}
	// what is under a tool's arguments is the tool's, never a vendor's
	// seal or a document: the walk reads it, where scan's visit of the
	// decoded request stepped over the arguments whole
	if len(args) > 0 {
		signed = slices.DeleteFunc(signed, func(s string) bool { return under([]byte(s), args) })
		textDocs = slices.DeleteFunc(textDocs, func(s string) bool { return under([]byte(s), args) })
	}
	return signed, args, defs, textDocs
}

// scanKeys are the keys scan reads a value of: a "type", and the keys of
// what a vendor seals. Every other string goes unread, so a request's tool
// outputs aren't unquoted a second time to look for them.
var scanKeys = func() map[string]bool {
	m := map[string]bool{"type": true}
	for _, k := range signatureKeys {
		m[k] = true
	}
	return m
}()

// objectMember is one member of an object: its key, and where its value
// starts and ends in the bytes.
type objectMember struct {
	key      string
	val, end int
}

// objectMembers are the members of the object at i, and where it ends.
func objectMembers(w *walker, i int) (ms []objectMember, end int, ok bool) {
	i = w.ws(i + 1)
	if i < len(w.in) && w.in[i] == '}' {
		return ms, i + 1, true
	}
	for {
		if i >= len(w.in) || w.in[i] != '"' {
			return nil, i, false
		}
		endK, esc, ok := w.str(i)
		if !ok {
			return nil, i, false
		}
		k, ok := w.stringAt(i, endK, esc)
		if !ok {
			return nil, i, false
		}
		i = w.ws(endK)
		if i >= len(w.in) || w.in[i] != ':' {
			return nil, i, false
		}
		v := w.ws(i + 1)
		ve, ok := w.skip(v)
		if !ok {
			return nil, i, false
		}
		ms = append(ms, objectMember{key: k, val: v, end: ve})
		i = w.ws(ve)
		if i < len(w.in) && w.in[i] == ',' {
			i = w.ws(i + 1)
			continue
		}
		if i < len(w.in) && w.in[i] == '}' {
			return ms, i + 1, true
		}
		return nil, i, false
	}
}

// definitionsOf is definitions for a request body, read without decoding
// it: the top level's defKeys (a text one only as an object), and the
// tools of the input items that carry them (defItems).
func definitionsOf(body []byte) []string {
	w := &walker{in: body}
	i := w.ws(0)
	if i >= len(body) || body[i] != '{' {
		return nil
	}
	ms, _, ok := objectMembers(w, i)
	if !ok {
		return nil
	}
	var defs []string
	for _, m := range ms {
		if m.key == "text" {
			if body[m.val] == '{' {
				defs = append(defs, "text")
			}
			continue
		}
		if slices.Contains(defKeys, m.key) {
			defs = append(defs, m.key)
		}
	}
	for _, m := range ms {
		if m.key != "input" || body[m.val] != '[' {
			continue
		}
		j := w.ws(m.val + 1)
		for n := 0; j < len(body) && body[j] != ']'; n++ {
			if body[j] == '{' {
				if ims, _, ok := objectMembers(w, j); ok {
					for _, im := range ims {
						if im.key != "type" || body[im.val] != '"' {
							continue
						}
						endT, escT, ok := w.str(im.val)
						if !ok {
							break
						}
						if t, ok := w.stringAt(im.val, endT, escT); ok && defItems[t] {
							defs = append(defs, join(join("input", strconv.Itoa(n)), "tools"))
						}
					}
				}
			}
			end, ok := w.skip(j)
			if !ok {
				break
			}
			j = w.ws(end)
			if j < len(body) && body[j] == ',' {
				j = w.ws(j + 1)
				continue
			}
			break
		}
	}
	return defs
}

// defKeys are the request's own keys whose values the agent defines for the
// vendor to read as structure, in every protocol an agent sends: the tools
// (Anthropic, Chat Completions, Responses, Gemini) and the choice among
// them, the output's schema (Chat's response_format, Responses' text,
// Anthropic's output_format and output_config, Gemini's generationConfig),
// Responses' reasoning and include, how the reply streams
// (stream_options, whose reasoning_summary_delivery the vendor checks is
// "sequential" or "sequential_cutoff", #3144), and the client's ids that
// no model reads (client_metadata: Codex's installation, session, thread,
// turn and window, as a JSON string in x-codex-turn-metadata, which the
// vendor and owl's own sessions match a turn by).
var defKeys = []string{
	"tools", "tool_choice", "functions", "function_call", "toolConfig", "tool_config",
	"response_format", "text", "reasoning", "include", "output_format", "output_config",
	"generationConfig", "generation_config", "stream_options", "client_metadata",
}

// defItems are the Responses input items that carry tools: Responses
// Lite's additional_tools, which Codex sends its tools in in place of the
// request's tools, and a tool search's output, the tools it found.
var defItems = map[string]bool{"additional_tools": true, "tool_search_output": true}

// definitions are the paths in a request v at which the agent defines
// rather than says: the request's own defKeys, and the tools of its
// defItems. Only where they sit in the protocol counts: a "tools" or
// "text" key deeper in, in a message or a tool call's arguments, is what
// the agent or the model wrote.
func definitions(v any) []string {
	m, _ := v.(map[string]any)
	if m == nil {
		return nil
	}
	var defs []string
	for _, k := range defKeys {
		if _, ok := m[k]; !ok {
			continue
		}
		if k == "text" {
			// Responses' text is the output's settings; a prompt's text
			// (a completion's, a Gemini contents part's) is a string
			if _, ok := m[k].(map[string]any); !ok {
				continue
			}
		}
		defs = append(defs, k)
	}
	items, _ := m["input"].([]any)
	for i, it := range items {
		if im, _ := it.(map[string]any); im != nil {
			if t, _ := im["type"].(string); defItems[t] {
				defs = append(defs, join(join("input", strconv.Itoa(i)), "tools"))
			}
		}
	}
	return defs
}

// under says path is in one of the objects at roots.
func under(path []byte, roots []string) bool {
	for _, r := range roots {
		if r == "" || len(path) == len(r) && pathHas(path, r) ||
			len(path) > len(r) && path[len(r)] == '.' && pathHas(path, r) {
			return true
		}
	}
	return false
}

// pathHas says the first len(r) bytes of path are r.
func pathHas(path []byte, r string) bool {
	if len(path) < len(r) {
		return false
	}
	for i := 0; i < len(r); i++ {
		if path[i] != r[i] {
			return false
		}
	}
	return true
}

// walk calls fn with every string value in the JSON document b — its path
// (keys and indexes joined by dots), the key it is under, and the value —
// and writes what fn returns in its place. Everything else is kept byte
// for byte, and b comes back as it is if nothing changed; if it isn't JSON,
// ok is false.
//
// The path is walk's own buffer, and the key and the value are b's own
// bytes when they need no unescaping: none of them may be kept past the
// call, and b must not be written while it is held. A caller that keeps
// one copies it (strings.Clone).
func walk(b []byte, fn func(path []byte, key, s string) string) (out []byte, ok bool) {
	return walkAll(b, nil, fn, nil, nil)
}

// walkRoots is walk for a change confined to the values at or in one of
// roots: a subtree that is neither under a root nor on the way to one is
// stepped over without reading its strings, so the strings of what isn't
// masked aren't unescaped either (MaskJSON's second pass over a request's
// definitions).
func walkRoots(b []byte, roots []string, fn func(path []byte, key, s string) string) (out []byte, ok bool) {
	return walkAll(b, roots, fn, nil, nil)
}

func walkAll(b []byte, roots []string, fn func(path []byte, key, s string) string, obj func(path []byte, key string, open bool), member func(path []byte, key string)) (out []byte, ok bool) {
	w := walker{in: b, fn: fn, obj: obj, member: member, roots: roots, path: make([]byte, 0, 64)}
	i, good := w.value(w.ws(0), "")
	if !good || w.ws(i) != len(b) {
		return b, false
	}
	if w.last == 0 {
		return b, true
	}
	w.out.Write(b[w.last:])
	out = w.out.Bytes()
	if cap(out)-len(out) > len(out)/8 {
		// placeholders longer than what they stand for grew it past the
		// room made for it: held as it is, a doubled buffer (#3383)
		out = bytes.Clone(out)
	}
	return out, true
}

type walker struct {
	in    []byte
	out   bytes.Buffer
	last  int
	path  []byte   // the path to the value being visited, built as the walk descends
	roots []string // walkRoots' roots, nil to visit everything
	fn    func(path []byte, key, s string) string
	obj   func(path []byte, key string, open bool)
	// member is told of each key an object has, with that object's path,
	// before the key's value is visited.
	member func(path []byte, key string)
	// only, when set, is the keys whose string values are unescaped and
	// handed to fn: a reader that wants a few of a request's fields isn't
	// read its whole text for them (scan).
	only map[string]bool
}

func (w *walker) ws(i int) int {
	for i < len(w.in) && (w.in[i] == ' ' || w.in[i] == '\t' || w.in[i] == '\n' || w.in[i] == '\r') {
		i++
	}
	return i
}

func join(path, k string) string {
	if path == "" {
		return k
	}
	return path + "." + k
}

// push adds k to the path, and returns what pop restores it to.
func (w *walker) push(k string) int {
	n := len(w.path)
	if n > 0 {
		w.path = append(w.path, '.')
	}
	w.path = append(w.path, k...)
	return n
}

// pushIndex adds an array's index to the path.
func (w *walker) pushIndex(n int) int {
	at := len(w.path)
	if at > 0 {
		w.path = append(w.path, '.')
	}
	w.path = strconv.AppendInt(w.path, int64(n), 10)
	return at
}

func (w *walker) pop(n int) { w.path = w.path[:n] }

// walkable says the value at the path may hold what walkRoots wants: it is
// under a root, or a root is under it.
func (w *walker) walkable() bool {
	for _, r := range w.roots {
		if r == "" {
			return true
		}
		if len(w.path) <= len(r) {
			if pathPrefixOf(w.path, r) {
				return true
			}
			continue
		}
		if w.path[len(r)] == '.' && pathHas(w.path, r) {
			return true
		}
	}
	return false
}

// pathPrefixOf says path is the first bytes of r.
func pathPrefixOf(path []byte, r string) bool {
	if len(path) > len(r) {
		return false
	}
	for i := range path {
		if path[i] != r[i] {
			return false
		}
	}
	return true
}

// skip is the end of the value at i, whose strings aren't read.
func (w *walker) skip(i int) (int, bool) {
	if i >= len(w.in) {
		return i, false
	}
	switch w.in[i] {
	case '"':
		end, _, ok := w.str(i)
		return end, ok
	case '{', '[':
		depth := 0
		for j := i; j < len(w.in); j++ {
			switch w.in[j] {
			case '"':
				end, _, ok := w.str(j)
				if !ok {
					return j, false
				}
				j = end - 1
			case '{', '[':
				depth++
			case '}', ']':
				depth--
				if depth == 0 {
					return j + 1, true
				}
			}
		}
		return i, false
	default:
		j := i
		for j < len(w.in) && !strings.ContainsRune(",]} \t\r\n", rune(w.in[j])) {
			j++
		}
		if j == i {
			return i, false
		}
		return j, true
	}
}

// str is the end of the string literal that starts at i.
func (w *walker) str(i int) (end int, escaped, ok bool) {
	for j := i + 1; j < len(w.in); j++ {
		switch w.in[j] {
		case '\\':
			escaped = true
			j++
		case '"':
			return j + 1, escaped, true
		}
	}
	return 0, false, false
}

// stringAt is the string literal from i to end: b's own bytes where it
// needs no unescaping, else a copy of its own.
func (w *walker) stringAt(i, end int, escaped bool) (string, bool) {
	if !escaped {
		return unsafe.String(&w.in[i+1], end-i-2), true
	}
	return unquoteJSONString(w.in[i:end])
}

func (w *walker) value(i int, key string) (int, bool) {
	if i >= len(w.in) {
		return i, false
	}
	if w.roots != nil && !w.walkable() {
		return w.skip(i)
	}
	switch c := w.in[i]; c {
	case '{':
		if w.obj != nil {
			w.obj(w.path, key, true)
		}
		i = w.ws(i + 1)
		if i < len(w.in) && w.in[i] == '}' {
			if w.obj != nil {
				w.obj(w.path, key, false)
			}
			return i + 1, true
		}
		for {
			if i >= len(w.in) || w.in[i] != '"' {
				return i, false
			}
			end, esc, ok := w.str(i)
			if !ok {
				return i, false
			}
			k, ok := w.stringAt(i, end, esc)
			if !ok {
				return i, false
			}
			i = w.ws(end)
			if i >= len(w.in) || w.in[i] != ':' {
				return i, false
			}
			if w.member != nil {
				w.member(w.path, k)
			}
			n := w.push(k)
			if i, ok = w.value(w.ws(i+1), k); !ok {
				return i, false
			}
			w.pop(n)
			i = w.ws(i)
			if i < len(w.in) && w.in[i] == ',' {
				i = w.ws(i + 1)
				continue
			}
			if i < len(w.in) && w.in[i] == '}' {
				if w.obj != nil {
					w.obj(w.path, key, false)
				}
				return i + 1, true
			}
			return i, false
		}
	case '[':
		i = w.ws(i + 1)
		if i < len(w.in) && w.in[i] == ']' {
			return i + 1, true
		}
		for n := 0; ; n++ {
			var ok bool
			// an array's strings are under the key the array is
			at := w.pushIndex(n)
			i, ok = w.value(i, key)
			w.pop(at)
			if !ok {
				return i, false
			}
			i = w.ws(i)
			if i < len(w.in) && w.in[i] == ',' {
				i = w.ws(i + 1)
				continue
			}
			if i < len(w.in) && w.in[i] == ']' {
				return i + 1, true
			}
			return i, false
		}
	case '"':
		end, esc, ok := w.str(i)
		if !ok {
			return i, false
		}
		if w.only != nil && !w.only[key] {
			return end, true
		}
		s, ok := w.stringAt(i, end, esc)
		if !ok {
			return i, false
		}
		if t := w.fn(w.path, key, s); t != s {
			if w.out.Cap() == 0 {
				// the masked body is about the body's size: grown by
				// doubling, its buffer was up to twice that, held as the
				// request's body while it streams (#3383)
				w.out.Grow(len(w.in) + len(w.in)/16 + 64)
			}
			w.out.Write(w.in[w.last:i])
			w.out.WriteByte('"')
			w.out.Write(appendJSONEscape(w.out.AvailableBuffer(), t))
			w.out.WriteByte('"')
			w.last = end
		}
		return end, true
	default:
		j := i
		for j < len(w.in) && !strings.ContainsRune(",]} \t\r\n", rune(w.in[j])) {
			j++
		}
		if j == i {
			return i, false
		}
		return j, true
	}
}

// bytesOf is s's bytes, for a reader that only reads them while it runs:
// a hash of a text, or the look for a value in it. s must not be written,
// and the bytes must not be kept past the call.
func bytesOf(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// unquoteJSONString is the string a valid JSON string literal holds, as
// encoding/json decodes it (unquoteBytes): invalid UTF-8 and lone
// surrogates become U+FFFD.
func unquoteJSONString(q []byte) (string, bool) {
	if len(q) < 2 || q[0] != '"' || q[len(q)-1] != '"' {
		return "", false
	}
	s := q[1 : len(q)-1]
	b := make([]byte, 0, len(s)+2*utf8.UTFMax)
	for r := 0; r < len(s); {
		switch c := s[r]; {
		case c == '\\':
			r++
			if r >= len(s) {
				return "", false
			}
			switch s[r] {
			default:
				return "", false
			case '"', '\\', '/', '\'':
				b = append(b, s[r])
				r++
			case 'b':
				b = append(b, '\b')
				r++
			case 'f':
				b = append(b, '\f')
				r++
			case 'n':
				b = append(b, '\n')
				r++
			case 'r':
				b = append(b, '\r')
				r++
			case 't':
				b = append(b, '\t')
				r++
			case 'u':
				r--
				rr := hex4(s[r:])
				if rr < 0 {
					return "", false
				}
				r += 6
				if utf16.IsSurrogate(rr) {
					if dec := utf16.DecodeRune(rr, hex4(s[r:])); dec != unicode.ReplacementChar {
						r += 6
						b = utf8.AppendRune(b, dec)
						break
					}
					rr = unicode.ReplacementChar
				}
				b = utf8.AppendRune(b, rr)
			}
		case c == '"', c < ' ':
			return "", false
		case c < utf8.RuneSelf:
			b = append(b, c)
			r++
		default:
			rr, size := utf8.DecodeRune(s[r:])
			r += size
			b = utf8.AppendRune(b, rr)
		}
	}
	if cap(b)-len(b) > len(b)/4 {
		// \u escapes came to much less than they were: a copy of its own
		// size, so a string kept long holds no room it doesn't use
		return string(b), true
	}
	return unsafe.String(unsafe.SliceData(b), len(b)), true
}

// hex4 is encoding/json's getu4: the rune of the \uXXXX at s, or -1.
func hex4(s []byte) rune {
	if len(s) < 6 || s[0] != '\\' || s[1] != 'u' {
		return -1
	}
	var r rune
	for _, c := range s[2:6] {
		switch {
		case c >= '0' && c <= '9':
			c -= '0'
		case c >= 'a' && c <= 'f':
			c = c - 'a' + 10
		case c >= 'A' && c <= 'F':
			c = c - 'A' + 10
		default:
			return -1
		}
		r = r*16 + rune(c)
	}
	return r
}

// jsonEscape is s as it goes between the quotes of a JSON string.
func jsonEscape(s string) string {
	return string(appendJSONEscape(make([]byte, 0, len(s)+8), s))
}

const escapeHex = "0123456789abcdef"

// appendJSONEscape appends s as the contents of a JSON string, as
// encoding/json writes them with SetEscapeHTML(false): a quote, a
// backslash and a control character escaped, U+2028 and U+2029 too, and
// each byte of invalid UTF-8 as U+FFFD. It doesn't make the calls
// encoding/json's encoder made for every value masked: a buffer, an
// encoder and a string of its own.
func appendJSONEscape(dst []byte, s string) []byte {
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if b >= 0x20 && b != '\\' && b != '"' {
				i++
				continue
			}
			dst = append(dst, s[start:i]...)
			switch b {
			case '\\', '"':
				dst = append(dst, '\\', b)
			case '\n':
				dst = append(dst, '\\', 'n')
			case '\r':
				dst = append(dst, '\\', 'r')
			case '\t':
				dst = append(dst, '\\', 't')
			case '\b':
				dst = append(dst, '\\', 'b')
			case '\f':
				dst = append(dst, '\\', 'f')
			default:
				dst = append(dst, '\\', 'u', '0', '0', escapeHex[b>>4], escapeHex[b&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case c == utf8.RuneError && size == 1:
			dst = append(dst, s[start:i]...)
			dst = append(dst, '\\', 'u', 'f', 'f', 'f', 'd') // six characters, as encoding/json writes it
			i++
			start = i
		case c == 0x2028 || c == 0x2029:
			dst = append(dst, s[start:i]...)
			dst = append(dst, '\\', 'u', '2', '0', '2', escapeHex[c&0xF])
			i += size
			start = i
		default:
			i += size
		}
	}
	return append(dst, s[start:]...)
}

// namedSecret says a string under a field named for a secret holds one,
// whatever the value looks like: the rules judge a value by its shape, the
// name by what a field so called carries (#2946) — {"api_key": "…"} with a
// value no rule knows, an AKIA under access_key_id, a bearer token under
// "Authorization". A short value, and one that stands for a variable
// (${PASS}, <token>, ****), are left alone. It is asked only of a tool
// call's arguments and a tool's structured result (args), whose keys are
// the tool's: a key the vendor's protocol names (Anthropic's
// mcp_servers[].authorization_token, a tool's own settings) is the
// vendor's to read, and masked by its name, it would no longer work.
func namedSecret(key, s string) bool {
	return key != "" && len(s) >= 8 && notAVariable(s) && SecretName(key)
}

// MaskJSON masks the strings of a request body, and says how many values
// it masked. A body that isn't JSON is masked as text.
//
// In a tool call's arguments and a tool's structured result (args), a
// field named for a secret has its value masked by its name, with secret
// masking on (#2946). Anywhere else a key is the protocol's, and a value
// is judged by its shape alone. What
// a vendor sealed with a signature (a thinking block, Gemini's
// thoughts, Responses' encrypted reasoning) is checked by the vendor
// against the text it wrote, so it goes back as written: the values
// masked elsewhere in the request become their placeholders again, which
// is what the vendor wrote where the agent reads the value (Restore), as
// do the secrets masked before (knownSpans), and nothing else in it is
// touched. Masked by the rules instead, a value the
// vendor wrote as a placeholder in other words around it ("the password
// is {{SECRET_…}}") went back as the value itself, and what the vendor
// wrote of its own that looks like an email or a phone number went as a
// placeholder: either way the text no longer matched its signature, and
// Anthropic answers 400 "Invalid `signature` in `thinking` block".
func MaskJSON(body []byte, o Options) ([]byte, int) {
	o.sent = true
	signed, args, defs, textDocs := scan(body)
	masked := func(path []byte, key, s string) bool {
		// a plain-text document's data is the user's own text, whatever
		// its key is named
		if under(path, textDocs) {
			return signed == nil || !under(path, signed)
		}
		return !keepIn(path, key, s, args, defs) && (signed == nil || !under(path, signed))
	}
	// What the agent defines (its tools' and schemas' descriptions) is
	// masked by the rules alone, as with the model off: it is the agent's
	// or a tool server's text, the same in every request, and no one's
	// conversation. Read by the model it was most of a request's text,
	// 52,577 of Claude Code 2.1.296's 67,532 bytes, and on a 4-core
	// server the first request waited out the gateway's 2 minutes while
	// the model read it (niuniu_8964 on X); on three real Claude Code
	// requests the model masked nothing there the rules didn't.
	defsO := o
	defsO.Model = nil
	if o.Model != nil {
		// the model is asked once, for all of the request's texts
		var texts []string
		if _, ok := walk(body, func(path []byte, key, s string) string {
			if masked(path, key, s) && !under(path, defs) {
				// kept past the walk, and it is the body's own bytes
				texts = append(texts, strings.Clone(s))
			}
			return s
		}); ok {
			o = withModel(o, texts)
		}
	}
	total := 0
	var seen []string // value, placeholder, value, placeholder…
	put := func(kind, v string) string {
		p := o.put()(kind, v)
		seen = append(seen, v, p)
		return p
	}
	// the definitions after the rest, so a value the model found in what
	// is said is masked in them too, as known
	out, ok := walk(body, func(path []byte, key, s string) string {
		if isDataURL(s) || signed != nil && under(path, signed) {
			return s
		}
		if o.Secrets && under(path, args) && namedSecret(key, s) {
			p := put("SECRET", s)
			total++
			return p
		}
		if !masked(path, key, s) || under(path, defs) {
			return s
		}
		t, n := mask(s, o, put, asJSON(key, ""))
		total += n
		return t
	})
	if ok && len(defs) > 0 {
		out, ok = walkRoots(out, defs, func(path []byte, key, s string) string {
			if !masked(path, key, s) || !under(path, defs) {
				return s
			}
			t, n := mask(s, defsO, put, asJSON(key, ""))
			total += n
			return t
		})
	}
	if !ok {
		t, n := Mask(string(body), o)
		return []byte(t), n
	}
	if len(signed) == 0 {
		return out, total
	}
	var remask *strings.Replacer
	if len(seen) > 0 {
		remask = replacerOf(seen)
	}
	out, _ = walk(out, func(path []byte, key, s string) string {
		if keepIn(path, key, s, args, defs) || !under(path, signed) {
			return s
		}
		t := s
		if remask != nil {
			if t = remask.Replace(s); t != s {
				total++
			}
		}
		// and a secret masked before, though nothing else in this request
		// has it now (the turn that brought it was compacted away): the
		// vendor wrote its placeholder there too. Not personal data: a
		// number or an address the vendor wrote of its own can be one
		// masked in another conversation (13800138000 is in many), and
		// masked, its text would no longer match its signature
		ko := Options{Secrets: o.Secrets, Route: o.Route}
		t, n := apply(t, sentSpans(sentKnown, t, ko, func() []span { return knownSpans(t, ko) }), o.put(), o)
		total += n
		return t
	})
	return out, total
}

// replacerOf swaps each value of pairs (value, placeholder, …) for its
// placeholder, the longest value first where two start at the same place.
func replacerOf(pairs []string) *strings.Replacer {
	type pair struct{ v, p string }
	var ps []pair
	have := map[string]bool{}
	for i := 0; i+1 < len(pairs); i += 2 {
		if !have[pairs[i]] {
			have[pairs[i]] = true
			ps = append(ps, pair{pairs[i], pairs[i+1]})
		}
	}
	sort.SliceStable(ps, func(i, j int) bool { return len(ps[i].v) > len(ps[j].v) })
	args := make([]string, 0, 2*len(ps))
	for _, p := range ps {
		args = append(args, p.v, p.p)
	}
	return strings.NewReplacer(args...)
}

// asJSON says a string under key holds JSON text of its own, so a value
// put back in it goes in escaped: a tool call's arguments, as a whole or
// streamed in pieces.
func asJSON(key, eventType string) bool {
	return key == "arguments" || key == "partial_json" || key == "delta" && strings.Contains(eventType, "arguments")
}

// typeOf is the type a JSON object says it is, wherever its "type" is
// among its keys. Read as the keys came, a Responses
// function_call_arguments.delta whose "delta" came before its "type", as
// a Go map writes them, had its value put back unescaped: a password
// with a quote in it ended the arguments' string there, and the arguments
// the agent put together were JSON no longer.
func typeOf(data []byte) string {
	typ := ""
	walk(data, func(path []byte, _, s string) string {
		if string(path) == "type" {
			typ = s
		}
		return s
	})
	return typ
}

// RestoreJSON puts the values back in a response body.
func RestoreJSON(body []byte) []byte { return restoreJSON(body, nil) }

// restoreJSON is RestoreJSON, counting in put as restore does.
func restoreJSON(body []byte, put map[string]int) []byte {
	if !heldIn(body) && !hasShapes(body) {
		return body
	}
	typ := typeOf(body)
	out, ok := walk(body, func(path []byte, key, s string) string {
		return restore(s, asJSON(key, typ), put)
	})
	if !ok {
		return []byte(restore(string(body), false, put))
	}
	return out
}

// heldIn says b may have a placeholder of a value held, in any of the
// ways Restore takes one back: a kind, _ and the 8 of its hash, in any
// case, with its braces or without. Looked for "{{" alone, a whole reply
// whose model wrote PERSON_kb32hzkg or {PERSON_kb32hzkg} kept it, while a
// stream had it put back.
func heldIn(b []byte) bool {
	mu.RLock()
	defer mu.RUnlock()
	if len(values) == 0 {
		return false
	}
	for i := 0; ; {
		j := bytes.IndexByte(b[i:], '_')
		if j < 0 {
			return false
		}
		i += j + 1
		z := i + 8
		if z > len(b) || z < len(b) && isWordByte(b[z]) || !base32Lower(b[i:z]) {
			continue
		}
		hash := strings.ToLower(string(b[i:z]))
		for k := range kinds {
			if n := len(k); i-1 >= n && strings.EqualFold(string(b[i-1-n:i-1]), k) {
				if _, ok := values["{{"+strings.ToUpper(k)+"_"+hash+"}}"]; ok {
					return true
				}
			}
		}
	}
}

// base32Lower says b is of the letters a placeholder's hash has, in
// either case.
func base32Lower(b []byte) bool {
	for _, c := range b {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '2' && c <= '7') {
			return false
		}
	}
	return true
}
