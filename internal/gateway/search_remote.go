package gateway

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/istoo/owl/internal/catalog"
	"github.com/istoo/owl/internal/provider"
)

// A Remote owl searches the web for the models it serves as it would
// for its own agents: by the model's provider, when that searches by
// itself, else with its own searcher. Its list says which of the two for
// each model (web_search: "native" or "owl"), and a web search offered
// to one of them goes to it as it was offered rather than being run here
// with this owl's searcher (Player on Discord: remoteOwl/glm-5.3's
// searches went to another model). Only a model it searches by itself,
// natively, is one this owl may search with for its other models: a
// search owl runs for another model is marked (SearchingHeader), and
// the other owl runs no searcher of its own for it, so two owls
// that each name the other never ask each other in turn.

// SearchingHeader marks a request a owl made to search for another
// model, to a remote owl, which then searches only by the model itself.
const SearchingHeader = "X-Magpie-Searching"

// Web search as a Remote owl's list says it.
const (
	searchNative = "native"
	searchOwl   = "owl"
)

// remoteSearch is how a Remote owl says it searches the web for model,
// "" for another provider or a model it doesn't.
func remoteSearch(p provider.Provider, model string) string {
	if !p.IsRemoteOwl() {
		return ""
	}
	if i := slices.IndexFunc(p.Available(), func(m catalog.Model) bool { return m.ID == model }); i >= 0 {
		return p.Available()[i].WebSearch
	}
	return ""
}

// remoteSearchesAny is whether a Remote owl searches natively for any
// of its models, which makes it one this owl may search with.
func remoteSearchesAny(p provider.Provider) bool {
	return p.IsRemoteOwl() && slices.ContainsFunc(p.Available(), func(m catalog.Model) bool { return m.WebSearch == searchNative })
}

// searchableModel is a model of p that owl may search with: a Gemini
// on a Google sign-in, one a Remote owl searches natively for, one of
// searchModels or a plugin marks that searches by itself (partSearcher),
// any other
// provider's.
func searchableModel(p provider.Provider, model string) bool {
	switch {
	case googleAccount(p), geminiKeySearches(p):
		return searchModel(model)
	case p.IsRemoteOwl():
		return remoteSearch(p, model) == searchNative
	case partSearcher(p):
		return searchesAlone(p, model)
	}
	return true
}

// webSearchOf is what this owl's list tells another owl of how it
// searches for e (remoteSearch): owl is whether it has a searcher.
func webSearchOf(e provider.Entry, owl bool) string {
	p := e.Provider
	if e.Group == "" {
		if p.IsRemoteOwl() && remoteSearch(p, e.Model) == searchNative || codeAssistSearches(p, provider.CodeAssist, e.Model) {
			return searchNative
		}
		for _, proto := range p.Speaks() {
			if searchesItself(p, proto) || modelSearches(p, proto, e.Model) {
				return searchNative
			}
		}
	}
	if owl || searchesOwn(p) {
		return searchOwl
	}
	return ""
}

// markSearching marks a request to a remote owl made for a search.
func markSearching(ctx context.Context, req *http.Request) {
	if searching(ctx) {
		req.Header.Set(SearchingHeader, "1")
	}
}

// searchingFrom is whether a request is a search another owl made.
func searchingFrom(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get("User-Agent"), "owl/") && r.Header.Get(SearchingHeader) != ""
}
