package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// A plugin or script written before the rename asks for /v1/myna/...: it
// reaches owl's own path, and every other path is left as it is.
func TestLegacyMynaPathsAreOwls(t *testing.T) {
	var seen []string
	h := legacyMyna(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.URL.Path)
	}))
	for _, p := range []string{"/v1/myna/quotas", "/v1/myna/usage/requests", "/v1/owl/quotas", "/v1/models", "/v1/mynaa/x"} {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", p, nil))
	}
	want := []string{"/v1/owl/quotas", "/v1/owl/usage/requests", "/v1/owl/quotas", "/v1/models", "/v1/mynaa/x"}
	for i := range want {
		if seen[i] != want[i] {
			t.Errorf("%s reached %s, want %s", want[i], seen[i], want[i])
		}
	}
}
