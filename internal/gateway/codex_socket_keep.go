package gateway

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/tidwall/gjson"
)

// A kept socket is a backend WebSocket left open after a turn ended with
// response.completed, under the session that turn belongs to. Codex's own
// client keeps one socket and sends each later turn on it as a delta after
// previous_response_id, so its messages stay small on a long conversation
// (codex-rs responses_websocket.rs, and the 1009 owl met: codex_socket.go).
// A turn that can't be matched to the kept turn is sent whole on the same
// socket, as before; a session whose kept socket is busy dials another and
// sends whole, since a delta only means anything on the socket that saw the
// response it follows.
//
// The socket is borrowed by one turn at a time: a concurrent turn of the
// same session (a subagent) takes none and dials its own, rather than
// waiting. A socket that ended with anything but response.completed, was
// read part way and closed, or failed is never kept.

// codexSocketIdle is how long a kept socket waits for its session's next
// turn before it is closed. Codex's backend keeps a connection for its own
// client's life; five minutes bounds the sockets a gateway holds for
// sessions that have stopped (a Codex thread left open overnight).
//
// debt: the fixed idle bound is this pool's ceiling. A per-session count or
// a share of live memory would let a busy machine hold more and a quiet one
// less; raise it there when an untended gateway's socket count matters.
var codexSocketIdle = 5 * time.Minute

// codexSocketMaxAge is how old a kept socket may be and still be kept. The
// backend caps a connection's life itself (the 1013 or
// websocket_connection_limit_reached it answers), and a socket past this
// could only fail its first write, so a dial is the cheaper choice.
var codexSocketMaxAge = 30 * time.Minute

// codexSocketKeepOff turns the kept sockets off, for a test that shows what
// the socket did before them: every turn dials its own and sends the whole
// body. Like codexSocketOff, it is a test's alone.
var codexSocketKeepOff atomic.Bool

// codexSocketCounts are the turns sent on a socket, by kind and bytes.
type codexSocketCounts struct {
	delta, full           atomic.Int64
	deltaBytes, fullBytes atomic.Int64
}

var codexSocketCount codexSocketCounts

// codexSocketDebug prints a kept socket's turn, as OWL_DEBUG does a call.
var codexSocketDebug atomic.Bool

// codexSocketUsed notes a turn written on a socket, sent bytes of whole.
func codexSocketUsed(host string, delta bool, sent, whole int) {
	if delta {
		codexSocketCount.delta.Add(1)
		codexSocketCount.deltaBytes.Add(int64(sent))
	} else {
		codexSocketCount.full.Add(1)
		codexSocketCount.fullBytes.Add(int64(sent))
	}
	if codexSocketDebug.Load() {
		kind := "whole"
		if delta {
			kind = "delta"
		}
		log.Printf("codex: %s turn on the WebSocket: %s, %d of %d bytes", host, kind, sent, whole)
	}
}

// codexKeptConn is the backend socket a session left open, the turn it
// answered last, and when it was dialled.
type codexKeptConn struct {
	c      *websocket.Conn
	turn   *codexSocketTurn
	opened time.Time
	timer  *time.Timer
	gen    uint64 // which idle timer may close it; under the pool's lock
}

// codexSocketPool is the kept sockets, at most one unborrowed per session.
type codexSocketPool struct {
	mu     sync.Mutex
	kept   map[string]*codexKeptConn
	closed bool
}

var codexKeptSockets = &codexSocketPool{}

// take borrows the session's kept socket, nil when there is none (or the
// gateway is ending, or it is older than codexSocketMaxAge).
func (p *codexSocketPool) take(key string) *codexKeptConn {
	p.mu.Lock()
	kc := p.kept[key]
	if kc != nil {
		delete(p.kept, key)
		if kc.timer != nil {
			kc.timer.Stop()
			kc.timer = nil
		}
	}
	closed := p.closed
	p.mu.Unlock()
	if kc == nil {
		return nil
	}
	if closed || time.Since(kc.opened) > codexSocketMaxAge {
		kc.c.CloseNow()
		return nil
	}
	return kc
}

// keep puts kc back for its session, closing it when the gateway is ending,
// it is too old, or another socket for the session is kept and newer is
// false (a turn that decided to use the session's other socket).
func (p *codexSocketPool) keep(key string, kc *codexKeptConn, newer bool) {
	if key == "" || time.Since(kc.opened) > codexSocketMaxAge {
		kc.c.CloseNow()
		return
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		kc.c.CloseNow()
		return
	}
	if p.kept == nil {
		p.kept = map[string]*codexKeptConn{}
	}
	old := p.kept[key]
	if old != nil && !newer {
		p.mu.Unlock()
		kc.c.CloseNow()
		return
	}
	if old != nil && old != kc && old.timer != nil {
		old.timer.Stop()
		old.timer = nil
	}
	p.kept[key] = kc
	kc.gen++
	gen := kc.gen
	kc.timer = time.AfterFunc(codexSocketIdle, func() { p.expire(key, kc, gen) })
	p.mu.Unlock()
	if old != nil && old != kc {
		old.c.CloseNow()
	}
}

// expire closes a kept socket after its idle time, unless it was borrowed
// or kept again since that timer was set.
func (p *codexSocketPool) expire(key string, kc *codexKeptConn, gen uint64) {
	p.mu.Lock()
	if p.kept[key] != kc || kc.gen != gen {
		p.mu.Unlock()
		return
	}
	delete(p.kept, key)
	kc.timer = nil
	p.mu.Unlock()
	kc.c.CloseNow()
}

// closeAll closes every kept socket, as the gateway's end does.
func (p *codexSocketPool) closeAll() {
	p.mu.Lock()
	p.closed = true
	all := make([]*codexKeptConn, 0, len(p.kept))
	for k, kc := range p.kept {
		if kc.timer != nil {
			kc.timer.Stop()
			kc.timer = nil
		}
		all = append(all, kc)
		delete(p.kept, k)
	}
	p.mu.Unlock()
	for _, kc := range all {
		kc.c.CloseNow()
	}
}

// reset empties the pool and lets it keep again, for a test: a gateway
// ends once, and closeAll is for good.
func (p *codexSocketPool) reset() {
	p.closeAll()
	p.mu.Lock()
	p.closed = false
	p.mu.Unlock()
}

// size is how many sockets the pool keeps, for a test.
func (p *codexSocketPool) size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.kept)
}

// codexSocketTurn is the turn a kept socket answered: the whole message it
// was sent, the reply's id and the output items, in output_item.done order.
type codexSocketTurn struct {
	msg []byte
	id  string
	out []codexOutItem
}

// codexOutItem is what a later turn is matched on, no more: a reasoning
// item's fields aren't compared, and a message's text is a digest.
type codexOutItem struct {
	typ     string
	callID  string
	id      string
	role    string
	text    [32]byte
	hasText bool
}

// codexOutItemOf is item as the next turn is matched against it.
func codexOutItemOf(item []byte) codexOutItem {
	it := codexOutItem{
		typ:    gjson.GetBytes(item, "type").String(),
		callID: gjson.GetBytes(item, "call_id").String(),
		id:     gjson.GetBytes(item, "id").String(),
		role:   gjson.GetBytes(item, "role").String(),
	}
	if it.typ == "message" {
		it.text, it.hasText = codexMessageText(item)
	}
	return it
}

// codexMessageText is the digest of a message item's text parts, and false
// when it has none. Only the text is hashed: Codex sends the item back with
// its annotations, status and id as it likes.
func codexMessageText(item []byte) ([32]byte, bool) {
	var zero [32]byte
	content := gjson.GetBytes(item, "content")
	if !content.IsArray() {
		return zero, false
	}
	h := sha256.New()
	found := false
	content.ForEach(func(_, part gjson.Result) bool {
		for _, k := range []string{"text", "refusal"} {
			if v := part.Get(k); v.Type == gjson.String {
				h.Write([]byte(v.String()))
				h.Write([]byte{0})
				found = true
			}
		}
		return true
	})
	if !found {
		return zero, false
	}
	var sum [32]byte
	copy(sum[:], h.Sum(nil))
	return sum, true
}

// codexDelta is msg, a response.create message, as the delta that follows
// prev's response, and true when this turn is prev's exactly: the fields
// but input and client_metadata are the same, its input starts with prev's
// input, and then hold prev's output items. Anything else is false, and
// the whole message is sent.
func codexDelta(prev *codexSocketTurn, msg []byte) ([]byte, bool) {
	if prev == nil || prev.id == "" || len(prev.msg) == 0 {
		return nil, false
	}
	var was, now map[string]json.RawMessage
	if json.Unmarshal(prev.msg, &was) != nil || json.Unmarshal(msg, &now) != nil {
		return nil, false
	}
	wasRest, ok := codexTurnRest(was)
	if !ok {
		return nil, false
	}
	nowRest, ok := codexTurnRest(now)
	if !ok || !bytes.Equal(wasRest, nowRest) {
		return nil, false
	}
	wasIn := codexInputItems(was["input"])
	nowIn := codexInputItems(now["input"])
	if len(nowIn) < len(wasIn)+len(prev.out) {
		return nil, false
	}
	for i := range wasIn {
		if !codexSameJSON(wasIn[i], nowIn[i]) {
			return nil, false
		}
	}
	if !codexOutputMatches(prev.out, nowIn[len(wasIn):len(wasIn)+len(prev.out)]) {
		return nil, false
	}
	fresh := nowIn[len(wasIn)+len(prev.out):]
	if fresh == nil {
		fresh = []json.RawMessage{}
	}
	in, err := json.Marshal(fresh)
	if err != nil {
		return nil, false
	}
	now["input"] = in
	id, err := json.Marshal(prev.id)
	if err != nil {
		return nil, false
	}
	now["previous_response_id"] = id
	out, err := json.Marshal(now)
	if err != nil {
		return nil, false
	}
	return out, true
}

// codexTurnRest is the message's fields a later turn must repeat exactly:
// everything but the input, the turn's own metadata and the response it
// names, as canonical JSON.
func codexTurnRest(m map[string]json.RawMessage) ([]byte, bool) {
	rest := make(map[string]json.RawMessage, len(m))
	for k, v := range m {
		switch k {
		case "type", "input", "client_metadata", "previous_response_id":
			continue
		}
		rest[k] = v
	}
	raw, err := json.Marshal(rest)
	if err != nil {
		return nil, false
	}
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return nil, false
	}
	b, err := json.Marshal(v)
	return b, err == nil
}

// codexInputItems is a request's input as items: a list as it is, text as
// its one item, absent as none.
func codexInputItems(raw json.RawMessage) []json.RawMessage {
	t := bytes.TrimSpace(raw)
	if len(t) == 0 || bytes.Equal(t, []byte("null")) {
		return nil
	}
	if t[0] == '[' {
		var items []json.RawMessage
		if json.Unmarshal(t, &items) != nil {
			return nil
		}
		return items
	}
	return []json.RawMessage{append(json.RawMessage(nil), t...)}
}

// codexSameJSON says whether a and b are the same JSON: the same bytes, or
// the same value with each one's keys and numbers canonical.
func codexSameJSON(a, b json.RawMessage) bool {
	a, b = bytes.TrimSpace(a), bytes.TrimSpace(b)
	if bytes.Equal(a, b) {
		return true
	}
	var va, vb any
	if json.Unmarshal(a, &va) != nil || json.Unmarshal(b, &vb) != nil {
		return false
	}
	ba, errA := json.Marshal(va)
	bb, errB := json.Marshal(vb)
	return errA == nil && errB == nil && bytes.Equal(ba, bb)
}

// codexOutputMatches says whether items at the head of a later turn's new
// input are prev's output: the same type, a call's call_id, an assistant
// message's text and role. A reasoning item is matched on its type alone,
// since Codex sends it back without fields it was given.
func codexOutputMatches(prev []codexOutItem, items []json.RawMessage) bool {
	if len(prev) != len(items) {
		return false
	}
	for i, p := range prev {
		it := items[i]
		typ := gjson.GetBytes(it, "type").String()
		if typ != p.typ {
			return false
		}
		if typ == "reasoning" {
			continue
		}
		if call := gjson.GetBytes(it, "call_id").String(); call != p.callID {
			return false
		}
		if typ != "message" {
			continue
		}
		if p.role != "" {
			if role := gjson.GetBytes(it, "role").String(); role != p.role {
				return false
			}
		}
		text, ok := codexMessageText(it)
		if ok != p.hasText || (ok && text != p.text) {
			return false
		}
	}
	return true
}

// codexSessionKey is the session a turn belongs to, as a key for the pool:
// the backend, the sign-in, and the conversation's id. The sign-in is
// hashed with the rest, never kept in the clear. "" when the turn names no
// session, which is sent and closed as it always was.
func codexSessionKey(req *http.Request, body []byte) string {
	session := ""
	for _, h := range []string{"session-id", "session_id", "thread-id"} {
		if v := strings.TrimSpace(req.Header.Get(h)); v != "" {
			session = v
			break
		}
	}
	if session == "" {
		session = strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String())
	}
	if session == "" {
		return ""
	}
	h := sha256.New()
	for _, part := range []string{
		req.URL.Host,
		req.Header.Get("Authorization"),
		req.Header.Get("Chatgpt-Account-Id"),
		session,
	} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// codexPreviousGone says whether frame is the backend saying it doesn't
// know the response a delta follows.
func codexPreviousGone(frame []byte) bool {
	for _, path := range []string{"error.code", "code"} {
		if gjson.GetBytes(frame, path).String() == "previous_response_not_found" {
			return true
		}
	}
	return false
}
