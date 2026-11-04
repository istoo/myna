package gateway

import (
	"context"
	"net/http"
	"strings"

	"github.com/istoo/myna/internal/provider"
)

// mynaUA is the User-Agent myna's requests go to a vendor with.
func mynaUA() string { return "myna/" + Version }

// userAgentFor is the User-Agent a request to p goes with, ua being the
// one the request came to myna with: the agent's own, as it sent it,
// where p is set to pass it on (provider.PassesUserAgent, #1851), and
// myna's where it isn't, where the agent sent none, and on myna's own
// requests — its classifier's, vision's, search's, the image tool's
// (mynasOwn) — which no agent sent. A User-Agent the user set among the
// provider's headers is put on after, when the request is signed, and
// stands over either.
func userAgentFor(p provider.Provider, ua string) string {
	if ua != "" && p.PassesUserAgent() && !mynasOwn(ua) {
		return ua
	}
	return mynaUA()
}

// mynasOwn is a User-Agent myna's own requests carry: another myna's
// (myna/…), the classifier's, vision's and search's (myna-router/1,
// myna-vision/1, myna-search/1), and the image tool's, which names the
// agent it draws for before its own (claude-code/2.1.0 myna-image/1)
// from what the agent told it, not from a User-Agent the agent sent.
func mynasOwn(ua string) bool {
	return strings.HasPrefix(ua, "myna/") || strings.HasPrefix(ua, "myna-") || strings.HasSuffix(ua, " "+DrawAgent)
}

// callerUA is the User-Agent the request ctx is for came with (withCaller),
// "" when there is none.
func callerUA(ctx context.Context) string {
	c, _ := ctx.Value(callerCtx{}).(caller)
	return c.ua
}

// stampUA puts on req, a request to p that isn't myna's own forwarded
// one (an embedding, a drawing, a video), the User-Agent forwardOnce's
// carry: myna's, or the agent's where p passes it on. A signed-in
// account's is left to its signing, as before, and another myna's is
// set where it is asked.
func stampUA(ctx context.Context, p provider.Provider, h http.Header) {
	if p.Account != nil || p.IsRemoteMyna() {
		return
	}
	h.Set("User-Agent", userAgentFor(p, callerUA(ctx)))
}
