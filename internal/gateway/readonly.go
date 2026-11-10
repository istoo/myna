package gateway

import (
	"log"
	"net/http"

	"github.com/istoo/owl/internal/access"
	"github.com/istoo/owl/internal/provider"
)

// A read-only gateway key (access.Key.ReadOnly) reads this owl's stats
// and uses nothing (Steven on Discord: an ESP32 desk display polling
// /v1/owl/quotas and /v1/owl/usage every 15s, its key kept in the
// device's flash, which had to be a key that could spend every account).
// It is answered the routes below alone, by the pattern the request is
// routed to, so a route added later is refused it until it is put here:
// every model route, count_tokens, a WebSocket's upgrade, every MCP
// server, a plugin's backend and every write are refused, whatever
// method or path spelling reaches them. TestReadOnlyKeyEveryRoute fails
// when a route is neither here nor in its list of refused ones.
//
// What it reads is what a key held to nothing reads: the usage summary,
// every account's quotas (it is no one else's to spend, so heldKey
// leaves it alone unless it is held to a budget, models or accounts as
// well) and the routing as it happens, redacted (route_events.go,
// joe21_eee's display, #3387). None of these routes tells a gateway
// key's secret.
var readOnlyRoutes = map[string]bool{
	"GET /{$}":                        true,
	"GET /v1":                         true,
	"GET /v1/{$}":                     true,
	"GET /v1/owl/quotas":             true,
	"GET /v1/owl/quotas/history":     true,
	"GET " + StatuslinePath:           true,
	"POST " + StatuslinePath:          true,
	"GET " + provider.RemoteCardsPath: true,
	"GET /v1/owl/usage":              true,
	"GET /v1/owl/usage/requests":     true,
	"GET /v1/owl/limit":              true,
	"GET /v1/owl/concurrency":        true,
	"GET " + routeSnapshotPath:        true,
	"GET " + routeEventsPath:          true,
	"GET " + OwnMCPPath:               true,
	"GET " + KeyPagePath:              true,
	"GET " + keyPageData:              true,
}

// ReadOnlyCode is the error code a read-only key is refused a route with.
const ReadOnlyCode = provider.KeyReadOnlyCode

// routeMux is a ServeMux that keeps the patterns it was given.
type routeMux struct {
	*http.ServeMux
	patterns []string
}

func (m *routeMux) HandleFunc(pattern string, h func(http.ResponseWriter, *http.Request)) {
	m.patterns = append(m.patterns, pattern)
	m.ServeMux.HandleFunc(pattern, h)
}

// readOnlyGuard refuses a read-only key every route of mux but
// readOnlyRoutes, before anything is read, reserved or sent.
func readOnlyGuard(mux *routeMux, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		who := access.Caller(r.Context())
		if who.ReadOnly {
			// a path that isn't clean is routed to a redirect under the
			// pattern of where it goes, and judged again when it gets there
			if _, pattern := mux.Handler(r); !readOnlyRoutes[pattern] {
				log.Printf("refused %s %s: gateway key %q is read-only", r.Method, r.URL.Path, who.KeyName)
				writeJSON(w, http.StatusForbidden, map[string]any{"error": map[string]any{
					"message": "this gateway key is read-only: it may read owl's quotas and usage (GET /v1/owl/quotas, /v1/owl/quotas/history, /v1/owl/usage, /v1/owl/usage/requests, /v1/owl/limit, /v1/owl/route-events) and use no model, MCP server or write. Use a key with full access for that (Gateway → Gateway keys, or owl gateway-key access <id> full)",
					"type":    "permission_error", "code": ReadOnlyCode}})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
