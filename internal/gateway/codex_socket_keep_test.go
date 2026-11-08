package gateway

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/tidwall/gjson"

	"github.com/istoo/myna/internal/provider"
)

// keptMsg is one turn a fake ChatGPT backend was sent on a socket: which
// socket, the message, and the response it named (empty for the whole turn).
type keptMsg struct {
	conn  int
	msg   []byte
	delta string
}

// keepBackend speaks the ChatGPT backend's WebSocket the way the real one
// does for Codex's own client: a socket takes turn after turn, each message
// answered with its frames, and stays open until its other end closes it.
type keepBackend struct {
	mu    sync.Mutex
	msgs  []keptMsg
	conns int // handshakes
	open  int // handlers still running, one per open socket
	turns int // messages answered

	// answer is a socket turn's frames; the default is one message.
	answer func(conn, n int, msg []byte) []string
	// rejectDelta answers any delta previous_response_not_found, as a
	// backend that lost the response does.
	rejectDelta bool
	// endDelta closes the socket on a delta before any frame, as a stale
	// kept socket that was already gone does.
	endDelta bool
	// hold, when set, is waited on before a turn's frames are written:
	// a test keeps a turn in flight while another asks the same session.
	hold func(msg []byte)
}

func (f *keepBackend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "/backend-api/codex/responses") {
		http.NotFound(w, r)
		return
	}
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		io.Copy(io.Discard, r.Body)
		// a turn that went over HTTP: answered cut, so a test that wanted
		// the socket sees it
		w.Header()["Content-Type"] = nil
		io.WriteString(w, "event: response.created\ndata: {\"type\":\"response.created\",\"response\":{\"id\":\"http\"}}\n\n")
		return
	}
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer c.CloseNow()
	f.mu.Lock()
	f.conns++
	conn := f.conns
	f.open++
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		f.open--
		f.mu.Unlock()
	}()
	c.SetReadLimit(-1)
	for {
		_, data, err := c.Read(r.Context())
		if err != nil {
			return
		}
		msg := append([]byte(nil), data...)
		delta := gjson.GetBytes(msg, "previous_response_id").String()
		if f.hold != nil {
			f.hold(msg)
		}
		if f.endDelta && delta != "" {
			return
		}
		f.mu.Lock()
		f.msgs = append(f.msgs, keptMsg{conn: conn, msg: msg, delta: delta})
		f.turns++
		n := f.turns
		answer := f.answer
		reject := f.rejectDelta && delta != ""
		f.mu.Unlock()
		var frames []string
		switch {
		case reject:
			frames = []string{`{"type":"error","status":400,"error":{"type":"invalid_request_error","code":"previous_response_not_found","message":"Previous response with id '` + delta + `' not found."}}`}
		case answer != nil:
			frames = answer(conn, n, msg)
		default:
			frames = keepReply(conn, n)
		}
		for _, fr := range frames {
			if c.Write(r.Context(), websocket.MessageText, []byte(fr)) != nil {
				return
			}
		}
	}
}

// keepReply is a whole reply: one assistant message, then completed.
func keepReply(conn, n int) []string {
	id := fmt.Sprintf("resp_%d_%d", conn, n)
	item := fmt.Sprintf(`{"type":"message","id":"msg_%d_%d","role":"assistant","status":"completed","content":[{"type":"output_text","text":"answer %d"}]}`, conn, n, n)
	return []string{
		fmt.Sprintf(`{"type":"response.created","response":{"id":%q,"model":"gpt-5.5"}}`, id),
		fmt.Sprintf(`{"type":"response.output_item.done","output_index":0,"item":%s}`, item),
		fmt.Sprintf(`{"type":"response.completed","response":{"id":%q,"usage":{"input_tokens":1,"output_tokens":1}}}`, id),
	}
}

// keepWith is a reply whose one output item is item.
func keepWith(conn, n int, item string) []string {
	id := fmt.Sprintf("resp_%d_%d", conn, n)
	return []string{
		fmt.Sprintf(`{"type":"response.created","response":{"id":%q,"model":"gpt-5.5"}}`, id),
		fmt.Sprintf(`{"type":"response.output_item.done","output_index":0,"item":%s}`, item),
		fmt.Sprintf(`{"type":"response.completed","response":{"id":%q,"usage":{"input_tokens":1,"output_tokens":1}}}`, id),
	}
}

func (f *keepBackend) seen() (msgs []keptMsg, conns int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]keptMsg(nil), f.msgs...), f.conns
}

func (f *keepBackend) openSockets() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.open
}

// dump lists the turns a fake backend was sent, for a failure.
func dump(msgs []keptMsg) string {
	var b strings.Builder
	for i, m := range msgs {
		fmt.Fprintf(&b, "#%d socket %d, delta %q, input %d items, model %q\n",
			i, m.conn, m.delta, len(gjson.GetBytes(m.msg, "input").Array()), gjson.GetBytes(m.msg, "model").String())
	}
	return b.String()
}

// keepOn serves the socket and empties the pool between tests.
func keepOn(t *testing.T) {
	t.Helper()
	codexSocketKeepOff.Store(false)
	socketOn(t)
	codexKeptSockets.reset()
	codexSocketCount.delta.Store(0)
	codexSocketCount.full.Store(0)
	codexSocketCount.deltaBytes.Store(0)
	codexSocketCount.fullBytes.Store(0)
	t.Cleanup(func() {
		codexKeptSockets.reset()
		codexSocketKeepOff.Store(false)
	})
}

// waitState waits for cond, a state a test is about, and fails if it never
// holds: never a fixed sleep, and room on both sides.
func waitState(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("waited 10s for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

const (
	keepUser1 = `{"type":"message","role":"user","content":[{"type":"input_text","text":"one"}]}`
	keepUser2 = `{"type":"message","role":"user","content":[{"type":"input_text","text":"two"}]}`
	keepUser3 = `{"type":"message","role":"user","content":[{"type":"input_text","text":"three"}]}`
)

// keepAnswer1 and answerTwoItem are the assistant items keepReply sends for
// the first two turns, as Codex sends them back in the next turn.
const (
	keepAnswer1   = `{"type":"message","id":"msg_1_1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"answer 1"}]}`
	answerTwoItem = `{"type":"message","id":"msg_1_2","role":"assistant","status":"completed","content":[{"type":"output_text","text":"answer 2"}]}`
)

// keepTurn1 is the session's first turn.
const keepTurn1 = `{"model":"gpt-5.5","stream":true,"prompt_cache_key":"sess-keep","instructions":"brief","input":[` + keepUser1 + `]}`

// keepTurn2 is the second turn over HTTP: the whole conversation, the
// first reply's item and the new input.
func keepTurn2(answer string) string {
	return `{"model":"gpt-5.5","stream":true,"prompt_cache_key":"sess-keep","instructions":"brief","input":[` +
		keepUser1 + `,` + answer + `,` + keepUser2 + `]}`
}

// keepDo is one turn through the gateway, as Codex sends it.
func keepDo(t *testing.T, body string) string {
	t.Helper()
	code, out := codexPost(t, body)
	if code != http.StatusOK {
		t.Fatalf("%d %s", code, out)
	}
	return out
}

// A session's second turn on the kept socket is the delta Codex's own
// client sends: previous_response_id names the first reply, and input is
// only what came after it, byte for byte.
func TestCodexKeptSocketSecondTurnIsADelta(t *testing.T) {
	keepOn(t)
	f := &keepBackend{}
	chatgpt(t, f.ServeHTTP)

	if body := keepDo(t, keepTurn1); !strings.Contains(body, "answer 1") || !strings.Contains(body, "response.completed") {
		t.Fatalf("first turn: %s", body)
	}
	waitState(t, "the first turn kept its socket", func() bool { return codexKeptSockets.size() == 1 })
	if body := keepDo(t, keepTurn2(keepAnswer1)); !strings.Contains(body, "answer 2") {
		t.Fatalf("second turn: %s", body)
	}
	msgs, conns := f.seen()
	if conns != 1 || len(msgs) != 2 {
		t.Fatalf("%d sockets, %d turns; want the second turn on the kept one", conns, len(msgs))
	}
	second := msgs[1].msg
	if got := gjson.GetBytes(second, "previous_response_id").String(); got != "resp_1_1" {
		t.Errorf("previous_response_id %q, want resp_1_1: %s", got, second)
	}
	want := `[` + keepUser2 + `]`
	if got := gjson.GetBytes(second, "input").Raw; got != want {
		t.Errorf("input %s, want only the new item %s", got, want)
	}
	if got := gjson.GetBytes(second, "model").String(); got != "gpt-5.5" {
		t.Errorf("model %q", got)
	}

	// the second reply is the next baseline: a third turn follows it
	thirdBody := `{"model":"gpt-5.5","stream":true,"prompt_cache_key":"sess-keep","instructions":"brief","input":[` +
		keepUser1 + `,` + keepAnswer1 + `,` + keepUser2 + `,` + answerTwoItem + `,` + keepUser3 + `]}`
	if body := keepDo(t, thirdBody); !strings.Contains(body, "answer 3") {
		t.Fatalf("third turn: %s", body)
	}
	msgs, conns = f.seen()
	if conns != 1 || len(msgs) != 3 {
		t.Fatalf("%d sockets, %d turns; want all three on the kept socket", conns, len(msgs))
	}
	third := msgs[2].msg
	if got := gjson.GetBytes(third, "previous_response_id").String(); got != "resp_1_2" {
		t.Errorf("third turn's previous_response_id %q, want resp_1_2", got)
	}
	if got, want := gjson.GetBytes(third, "input").Raw, `[`+keepUser3+`]`; got != want {
		t.Errorf("third turn input %s, want only the new item %s", got, want)
	}
	if n := codexSocketCount.delta.Load(); n != 2 {
		t.Errorf("%d delta turns, want 2", n)
	}
}

// With the kept sockets off, the same two turns do what the socket did
// before them: two sockets, the whole body each. It is the injection that
// shows TestCodexKeptSocketSecondTurnIsADelta fails without the feature.
func TestCodexKeptSocketOffSendsWholeAgain(t *testing.T) {
	keepOn(t)
	codexSocketKeepOff.Store(true)
	f := &keepBackend{}
	chatgpt(t, f.ServeHTTP)

	keepDo(t, keepTurn1)
	keepDo(t, keepTurn2(keepAnswer1))
	msgs, conns := f.seen()
	if conns != 2 || len(msgs) != 2 {
		t.Fatalf("%d sockets, %d turns; want one socket per turn", conns, len(msgs))
	}
	for i, m := range msgs {
		if m.delta != "" {
			t.Errorf("turn %d named %q; want the whole body", i, m.delta)
		}
	}
	// the first turn holds one item, the second all three
	if n := len(gjson.GetBytes(msgs[0].msg, "input").Array()); n != 1 {
		t.Errorf("first turn input has %d items, want 1", n)
	}
	if n := len(gjson.GetBytes(msgs[1].msg, "input").Array()); n != 3 {
		t.Errorf("second turn input has %d items, want 3", n)
	}
}

// A turn whose other fields changed is the whole turn, still on the
// session's socket.
func TestCodexKeptSocketChangedFieldsGoWhole(t *testing.T) {
	keepOn(t)
	f := &keepBackend{}
	chatgpt(t, f.ServeHTTP)
	keepDo(t, keepTurn1)
	waitState(t, "the first turn kept its socket", func() bool { return codexKeptSockets.size() == 1 })

	keepDo(t, strings.Replace(keepTurn2(keepAnswer1), `"instructions":"brief"`, `"instructions":"verbose"`, 1))
	msgs, conns := f.seen()
	if conns != 1 || len(msgs) != 2 {
		t.Fatalf("%d sockets, %d turns", conns, len(msgs))
	}
	if msgs[1].delta != "" {
		t.Errorf("changed instructions still named %q", msgs[1].delta)
	}
	if n := len(gjson.GetBytes(msgs[1].msg, "input").Array()); n != 3 {
		t.Errorf("whole turn input has %d items, want 3", n)
	}
}

// An output item that isn't the one the reply said, a different call or a
// changed text, means the whole turn.
func TestCodexKeptSocketOutputMismatchGoesWhole(t *testing.T) {
	call := `{"type":"function_call","id":"fc_1","call_id":"call_1","name":"shell","arguments":"{}"}`
	for _, tc := range []struct {
		name  string
		reply func(conn, n int, msg []byte) []string
		turn  string
	}{
		{
			name:  "a call id changed",
			reply: func(conn, n int, msg []byte) []string { return keepWith(conn, n, call) },
			turn: `{"model":"gpt-5.5","stream":true,"prompt_cache_key":"sess-keep","instructions":"brief","input":[` +
				keepUser1 + `,{"type":"function_call","id":"fc_1","call_id":"call_2","name":"shell","arguments":"{}"},` +
				`{"type":"function_call_output","call_id":"call_2","output":"ok"},` + keepUser2 + `]}`,
		},
		{
			name:  "an output item dropped",
			reply: nil,
			turn: `{"model":"gpt-5.5","stream":true,"prompt_cache_key":"sess-keep","instructions":"brief","input":[` +
				keepUser1 + `,` + keepUser2 + `]}`,
		},
		{
			name:  "the message text changed",
			reply: nil,
			turn:  keepTurn2(`{"type":"message","id":"msg_1_1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"something else"}]}`),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			keepOn(t)
			f := &keepBackend{answer: tc.reply}
			chatgpt(t, f.ServeHTTP)
			keepDo(t, keepTurn1)
			waitState(t, "the first turn kept its socket", func() bool { return codexKeptSockets.size() == 1 })
			keepDo(t, tc.turn)
			msgs, conns := f.seen()
			if conns != 1 || len(msgs) != 2 {
				t.Fatalf("%d sockets, %d turns", conns, len(msgs))
			}
			if msgs[1].delta != "" {
				t.Errorf("mismatched output still named %q", msgs[1].delta)
			}
		})
	}
}

// A backend that no longer knows the response a delta follows is sent the
// whole turn on a new socket, and the caller sees only that reply.
func TestCodexKeptSocketPreviousResponseNotFoundGoesWhole(t *testing.T) {
	keepOn(t)
	f := &keepBackend{rejectDelta: true}
	chatgpt(t, f.ServeHTTP)
	keepDo(t, keepTurn1)
	waitState(t, "the first turn kept its socket", func() bool { return codexKeptSockets.size() == 1 })

	body := keepDo(t, keepTurn2(keepAnswer1))
	if strings.Contains(body, "previous_response_not_found") {
		t.Errorf("the refused delta reached the caller: %s", body)
	}
	if !strings.Contains(body, "response.completed") || !strings.Contains(body, "answer 3") {
		t.Errorf("the caller didn't get the whole reply: %s", body)
	}
	msgs, conns := f.seen()
	if conns != 2 || len(msgs) != 3 {
		t.Fatalf("%d sockets, %d turns; want full, delta, then full on a new socket", conns, len(msgs))
	}
	if msgs[1].delta == "" || msgs[2].delta != "" {
		t.Errorf("turns were %q, %q; want a delta then the whole", msgs[1].delta, msgs[2].delta)
	}
	if n := len(gjson.GetBytes(msgs[2].msg, "input").Array()); n != 3 {
		t.Errorf("the retry's input has %d items, want 3", n)
	}
}

// A kept socket that ends before the delta's reply is replaced by a new
// one with the whole turn.
func TestCodexKeptSocketEndedBeforeReplyGoesWhole(t *testing.T) {
	keepOn(t)
	f := &keepBackend{endDelta: true}
	chatgpt(t, f.ServeHTTP)
	keepDo(t, keepTurn1)
	waitState(t, "the first turn kept its socket", func() bool { return codexKeptSockets.size() == 1 })

	body := keepDo(t, keepTurn2(keepAnswer1))
	if !strings.Contains(body, "response.completed") {
		t.Fatalf("the caller's reply was cut: %s", body)
	}
	msgs, conns := f.seen()
	if conns != 2 || len(msgs) != 2 {
		t.Fatalf("%d sockets, %d turns; want the whole turn on a new socket", conns, len(msgs))
	}
	if msgs[1].delta != "" {
		t.Errorf("the new socket was sent a delta: %q", msgs[1].delta)
	}
}

// A turn the session started while its kept socket was busy dials its own
// and sends whole; both succeed. It waits on the first turn's state, not a
// duration, and runs under -race.
func TestCodexKeptSocketBusySessionDialsAnother(t *testing.T) {
	keepOn(t)
	deltaSeen := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	f := &keepBackend{hold: func(msg []byte) {
		if gjson.GetBytes(msg, "previous_response_id").String() != "" {
			once.Do(func() { close(deltaSeen) })
			<-release
		}
	}}
	chatgpt(t, f.ServeHTTP)
	keepDo(t, keepTurn1)
	waitState(t, "the first turn kept its socket", func() bool { return codexKeptSockets.size() == 1 })

	type result struct {
		code int
		body string
	}
	firstDone := make(chan result, 1)
	go func() {
		code, body := codexPost(t, keepTurn2(keepAnswer1))
		firstDone <- result{code, body}
	}()
	<-deltaSeen // the session's kept socket is taken by the first turn
	secondCode, secondBody := codexPost(t, keepTurn2(keepAnswer1))
	close(release)
	first := <-firstDone

	if first.code != 200 || secondCode != 200 ||
		!strings.Contains(first.body, "response.completed") || !strings.Contains(secondBody, "response.completed") {
		t.Fatalf("a concurrent turn failed:\n%d %s\n%d %s", first.code, first.body, secondCode, secondBody)
	}
	msgs, conns := f.seen()
	// the first turn, then the busy session's whole turn on its own socket,
	// then the kept socket's delta once the first turn was let go
	if conns != 2 || len(msgs) != 3 {
		t.Fatalf("%d sockets, %d turns; want a second socket for the busy session:\n%s", conns, len(msgs), dump(msgs))
	}
	var deltas, wholes int
	for _, m := range msgs[1:] {
		if m.delta != "" {
			deltas++
			if m.conn != 1 {
				t.Errorf("a delta went on socket %d, want the kept socket 1", m.conn)
			}
		} else {
			wholes++
			if m.conn != 2 {
				t.Errorf("the whole turn went on socket %d, want the new socket 2", m.conn)
			}
		}
	}
	if deltas != 1 || wholes != 1 {
		t.Errorf("%d deltas, %d wholes; want one of each", deltas, wholes)
	}
}

// A reply that failed or a turn the caller closed before its end keeps
// nothing: the next turn dials a new socket and sends the whole body.
func TestCodexKeptSocketFailedOrClosedNotKept(t *testing.T) {
	t.Run("failed reply", func(t *testing.T) {
		keepOn(t)
		f := &keepBackend{answer: func(conn, n int, msg []byte) []string {
			return []string{
				fmt.Sprintf(`{"type":"response.created","response":{"id":"resp_%d_%d"}}`, conn, n),
				`{"type":"response.failed","response":{"id":"resp_1_1","status":"failed","error":{"message":"no"}}}`,
			}
		}}
		chatgpt(t, f.ServeHTTP)
		// straight to the transport: the gateway asks a failed reply again
		// within its own turn, which is its behavior, not this one's
		res := keepTransportPost(t, keepTurn1)
		if b, _ := io.ReadAll(res.Body); !strings.Contains(string(b), "response.failed") {
			t.Fatalf("first turn: %s", b)
		}
		res.Body.Close()
		waitState(t, "the failed turn left nothing kept", func() bool { return codexKeptSockets.size() == 0 })
		res = keepTransportPost(t, keepTurn2(keepAnswer1))
		io.ReadAll(res.Body)
		res.Body.Close()
		msgs, conns := f.seen()
		if conns != 2 || len(msgs) != 2 {
			t.Fatalf("%d sockets, %d turns; want a new socket:\n%s", conns, len(msgs), dump(msgs))
		}
		if msgs[1].delta != "" {
			t.Errorf("the turn after a failure was a delta: %q", msgs[1].delta)
		}
	})
	t.Run("closed before the reply's end", func(t *testing.T) {
		keepOn(t)
		f := &keepBackend{}
		chatgpt(t, f.ServeHTTP)
		res := keepTransportPost(t, keepTurn1)
		buf := make([]byte, 1)
		if _, err := res.Body.Read(buf); err != nil {
			t.Fatalf("first read: %v", err)
		}
		res.Body.Close()
		waitState(t, "the cut turn left nothing kept", func() bool { return codexKeptSockets.size() == 0 })
		keepDo(t, keepTurn2(keepAnswer1))
		msgs, conns := f.seen()
		if conns != 2 || len(msgs) != 2 {
			t.Fatalf("%d sockets, %d turns; want a new socket:\n%s", conns, len(msgs), dump(msgs))
		}
		if msgs[1].delta != "" {
			t.Errorf("the turn after a cut was a delta: %q", msgs[1].delta)
		}
	})
}

// keepTransportPost is one turn straight to the ChatGPT backend's transport,
// as the gateway's own client sends it, so a test can stop reading part way.
func keepTransportPost(t *testing.T, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("POST", provider.CodexBase+"/responses", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer chatgpt-token")
	req.Header.Set("chatgpt-account-id", "acct-1")
	req.Header.Set("Content-Type", "application/json")
	res, err := New().client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// A kept socket is closed after its idle time, and at the gateway's end.
func TestCodexKeptSocketIdleAndShutdownClose(t *testing.T) {
	t.Run("idle", func(t *testing.T) {
		keepOn(t)
		was := codexSocketIdle
		codexSocketIdle = 50 * time.Millisecond
		t.Cleanup(func() { codexSocketIdle = was })
		f := &keepBackend{}
		chatgpt(t, f.ServeHTTP)
		keepDo(t, keepTurn1)
		waitState(t, "the first turn kept its socket", func() bool { return codexKeptSockets.size() == 1 })
		waitState(t, "the idle socket to close", func() bool {
			return codexKeptSockets.size() == 0 && f.openSockets() == 0
		})
	})
	t.Run("shutdown", func(t *testing.T) {
		keepOn(t)
		f := &keepBackend{}
		chatgpt(t, f.ServeHTTP)
		keepDo(t, keepTurn1)
		waitState(t, "the first turn kept its socket", func() bool { return codexKeptSockets.size() == 1 })
		codexKeptSockets.closeAll()
		waitState(t, "the gateway's end to close the socket", func() bool {
			return codexKeptSockets.size() == 0 && f.openSockets() == 0
		})
	})
}

// A turn with no session id is sent and closed as it always was: one socket
// a turn, the whole body, and nothing kept.
func TestCodexKeptSocketNoSessionIDUnchanged(t *testing.T) {
	keepOn(t)
	f := &keepBackend{}
	chatgpt(t, f.ServeHTTP)
	body := `{"model":"gpt-5.5","stream":true,"input":[` + keepUser1 + `]}`
	code, out := codexPost(t, body)
	if code != 200 || !strings.Contains(out, "response.completed") {
		t.Fatalf("%d %s", code, out)
	}
	waitState(t, "the turn to close its socket", func() bool {
		return codexKeptSockets.size() == 0 && f.openSockets() == 0
	})
	code, out = codexPost(t, body)
	if code != 200 || !strings.Contains(out, "response.completed") {
		t.Fatalf("%d %s", code, out)
	}
	msgs, conns := f.seen()
	if conns != 2 || len(msgs) != 2 {
		t.Fatalf("%d sockets, %d turns; want one socket a turn", conns, len(msgs))
	}
	for i, m := range msgs {
		if m.delta != "" {
			t.Errorf("turn %d was a delta", i)
		}
	}
	if n := codexSocketCount.delta.Load(); n != 0 {
		t.Errorf("%d delta turns without a session", n)
	}
}

// The counters tell a delta from a whole turn. It is also the one line
// MYNA_DEBUG logs, checked here through the same helper.
func TestCodexKeptSocketCounts(t *testing.T) {
	keepOn(t)
	f := &keepBackend{}
	chatgpt(t, f.ServeHTTP)
	keepDo(t, keepTurn1)
	keepDo(t, keepTurn2(keepAnswer1))
	if d, w := codexSocketCount.delta.Load(), codexSocketCount.full.Load(); d != 1 || w != 1 {
		t.Errorf("%d delta, %d whole; want one of each", d, w)
	}
	// the delta is smaller than the whole second turn it stands for
	if b := codexSocketCount.deltaBytes.Load(); b <= 0 || b >= int64(len(keepTurn2(keepAnswer1))) {
		t.Errorf("delta %d bytes, whole turn %d", b, len(keepTurn2(keepAnswer1)))
	}
	if b := codexSocketCount.fullBytes.Load(); b <= 0 {
		t.Errorf("%d whole bytes", b)
	}
	if _, conns := f.seen(); conns != 1 {
		t.Errorf("%d sockets", conns)
	}
}
