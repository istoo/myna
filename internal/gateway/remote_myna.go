package gateway

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"unicode"

	"github.com/istoo/myna/internal/usage"
)

// A request passed on to another computer's myna (a Remote myna
// provider) goes as myna's own, with myna's User-Agent and the other
// myna's key, so that myna knew it only as "myna" (Jorben on Discord).
// It now names the agent it came from (AgentHeader) and the computer
// (ViaHeader), and the other myna keeps them in its usage, as "Claude
// Code · via <computer>". They are a label and nothing more: what myna
// does with the request — the models an agent is shown, its stand-ins, a
// routing group's rules — still goes by who sent it, and they are taken
// only from a myna's request (its User-Agent), one a client on the
// network could send only with the key anyway, and could name itself any
// agent by its User-Agent as well. No vendor is ever sent them: they come
// off every request as it arrives, and go on only to a remote myna.
const (
	AgentHeader = "X-Magpie-Agent"
	ViaHeader   = "X-Magpie-Via"
)

// caller is who a request is recorded as: its agent, and the computer it
// was passed on from, "" for this one. session and native are the session
// the caller named (sessionOf's X-Magpie-Session, and the agent's own
// header nativeSessionOf takes, by its name), passed on to a remote myna
// with the request (#672).
type caller struct {
	agent, via string
	session    string
	native     [2]string // header, value
	// compact is DSH's compaction mark as it came, passed on with the
	// session so the other myna records the call as one (#2505)
	compact string
	// ua is the User-Agent the request came with, as it came: the one a
	// provider that passes the agent's on is sent (userAgentFor)
	ua string
}

type callerCtx struct{}

// withCaller notes on each request who it is recorded as, and takes the
// headers that said so off it.
func withCaller(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), callerCtx{}, callerFrom(r))
		if searchingFrom(r) {
			// another myna's search: only the model itself searches
			ctx = context.WithValue(ctx, searchingKey{}, true)
		}
		r.Header.Del(AgentHeader)
		r.Header.Del(ViaHeader)
		r.Header.Del(SearchingHeader)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// callerFrom reads who r is recorded as from its headers, before withCaller
// takes myna's own off: the one rule for every place that names a
// request's agent (usage, sessions, the routing trace, the masking log,
// the requests in flight), so another myna's request is shown as the
// agent that made it there, never as "myna" (linkaww on Discord: the
// masking log named myna where usage named Pi).
func callerFrom(r *http.Request) caller {
	c := caller{agent: agentOf(r), session: boundedSession(r.Header.Get(SessionHeader)), ua: r.Header.Get("User-Agent")}
	if dshCompacts(r.Header) {
		c.compact = boundedSession(r.Header.Get(dshCompactHeader))
	}
	for _, h := range sessionHeaders {
		if v := boundedSession(r.Header.Get(h)); v != "" {
			c.native = [2]string{h, v}
			break
		}
	}
	if strings.HasPrefix(r.Header.Get("User-Agent"), "myna/") {
		if a := label(r.Header.Get(AgentHeader)); a != "" {
			c.agent = usage.AgentOf(a)
			c.via = label(r.Header.Get(ViaHeader))
			if c.via == "" {
				c.via, _, _ = net.SplitHostPort(r.RemoteAddr)
			}
		}
	}
	return c
}

// callerOf is who the request is recorded as.
func callerOf(r *http.Request) caller {
	if c, ok := r.Context().Value(callerCtx{}).(caller); ok {
		return c
	}
	return caller{agent: agentOf(r)}
}

// label is a header's name for an agent or a computer, if it is a fit one
// to show: short, one word of printable characters.
func label(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || len(v) > 64 || strings.IndexFunc(v, func(r rune) bool { return !unicode.IsPrint(r) || unicode.IsSpace(r) }) >= 0 {
		return ""
	}
	return v
}

// boundedSession is a session header's value as myna keeps it: trimmed,
// at most 128 bytes, as sessionOf takes it.
func boundedSession(v string) string {
	v = strings.TrimSpace(v)
	return v[:min(len(v), 128)]
}

// passOnCaller names, on a request to a remote myna, the agent it is for
// and the computer it comes from: the one it was passed on from, else this.
// The session the caller named goes on too, as it named it — myna's
// X-Magpie-Session and the agent's own header (session_id, …) — so the
// other myna records the call under it and keeps a conversation on its
// account by it rather than by its first message (#672). Only a remote
// myna is sent them; a vendor never is. It is asked, too, to name the
// vendor's model in its reply, as it did before a reply named the model
// asked for: this myna records what served the call from it (served,
// swapped, routed; TestRemoteMynaGroupNotSwapped), and gives its own
// agent the name it asked for itself.
func passOnCaller(ctx context.Context, req *http.Request) {
	req.Header.Set(responseModelHeader, "vendor")
	c, ok := ctx.Value(callerCtx{}).(caller)
	if !ok {
		return
	}
	if c.session != "" {
		req.Header.Set(SessionHeader, c.session)
	}
	if c.native[0] != "" {
		req.Header.Set(c.native[0], c.native[1])
	}
	if c.compact != "" {
		req.Header.Set(dshCompactHeader, c.compact)
	}
	if c.agent == "" {
		return
	}
	req.Header.Set(AgentHeader, c.agent)
	if c.via == "" {
		c.via = hostName()
	}
	if c.via != "" {
		req.Header.Set(ViaHeader, c.via)
	}
}

// hostName is this computer's name as people know it (MacBook-Pro, not
// MacBook-Pro.local).
var hostName = sync.OnceValue(func() string {
	h, _ := os.Hostname()
	h, _, _ = strings.Cut(h, ".")
	return label(h)
})
