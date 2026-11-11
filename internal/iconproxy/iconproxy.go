// Package iconproxy names where owl downloads a remote picture from. A
// plugin's, an import link's or the library's icon sits on a host its author
// picked, and fetching it straight from there would hand that author the
// user's IP each time a page showed it. Upstream sent every such fetch
// through its own site; this build talks to no upstream service, so the
// picture is fetched from its host unless OWL_ICON_PROXY names a proxy of
// one's own.
package iconproxy

import (
	"net/url"
	"os"
)

// Base is the proxy's endpoint; empty fetches the picture from its host.
// OWL_ICON_PROXY replaces it.
const Base = ""

// URL is the address to fetch the picture at u through: u itself when no
// proxy is set.
func URL(u string) string {
	base := Base
	if v := os.Getenv("OWL_ICON_PROXY"); v != "" {
		base = v
	}
	if base == "" {
		return u
	}
	return base + "?url=" + url.QueryEscape(u)
}
