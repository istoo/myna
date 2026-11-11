package gateway

import (
	"net/http"
	"os"

	"github.com/istoo/owl/internal/omarchy"
	"github.com/istoo/owl/internal/provider"
)

// siteOrigins are the pages that may read the Omarchy theme: github.com/istoo/owl,
// which on an Omarchy machine draws itself in that machine's theme
// (site/public/omarchy.js) and asks the owl running there which one it
// is; OWL_SITE_ORIGIN adds one (the site served locally to try it).
var siteOrigins = map[string]bool{"https://github.com/istoo/owl": true, "https://www.github.com/istoo/owl": true}

func siteOrigin(o string) bool {
	return siteOrigins[o] || o != "" && o == os.Getenv("OWL_SITE_ORIGIN")
}

// onOmarchy is whether this owl runs on Omarchy, asked once.
var onOmarchy = omarchy.Detect()

// omarchyTheme is Omarchy's current theme as owl's own windows draw it
// (name, mode, stamp, CSS variables: palette, font, corners, border), for
// github.com/istoo/owl's pages on the same machine. It answers this machine only,
// and the browser lets only github.com/istoo/owl read it; a owl elsewhere says
// 404.
func (s *Server) omarchyTheme(w http.ResponseWriter, r *http.Request) {
	if o := r.Header.Get("Origin"); siteOrigin(o) {
		w.Header().Set("Access-Control-Allow-Origin", o)
		w.Header().Set("Vary", "Origin")
		if r.Header.Get("Access-Control-Request-Private-Network") == "true" {
			w.Header().Set("Access-Control-Allow-Private-Network", "true")
		}
	}
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !local(r) {
		writeError(w, provider.Chat, http.StatusForbidden, "owl tells its Omarchy theme to this machine only")
		return
	}
	th, ok := omarchy.Theme{}, false
	if onOmarchy {
		th, ok = omarchy.Current()
	}
	if !ok {
		writeError(w, provider.Chat, http.StatusNotFound, "owl isn't running on Omarchy")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, th)
}
