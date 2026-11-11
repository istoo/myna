package gui

import (
	"errors"
	"net/http"

	"github.com/istoo/owl/internal/uninstall"
)

// what Settings › Uninstall owl asks of the system; the tests put
// fakes in
var (
	uninstallSystem = uninstall.System
	uninstallAgents = uninstall.Agents
	uninstallData   = uninstall.DataAfterExit
)

// uninstallRoutes: Settings › Uninstall owl, the app's `owl
// uninstall`. The app does it in two steps. POST /api/uninstall removes
// owl's service, login item and shortcuts and disconnects every agent,
// putting back what each had, and answers with what it did, so the page
// can say what failed or that the Codex app needs a restart. POST
// /api/uninstall/finish then starts the removal of owl's files once the
// app has quit (WebView2 holds its folder while the app runs), and quits.
// Never from a browser (owl web): that would take owl off the
// computer it serves from, not the one the page is on.
func uninstallRoutes(mux *http.ServeMux, w Windows) {
	web := isWeb(w)
	refuse := func(rw http.ResponseWriter) bool {
		if web {
			http.Error(rw, "owl is uninstalled from the computer it runs on, with owl uninstall there", http.StatusForbidden)
		}
		return web
	}
	mux.HandleFunc("POST /api/uninstall", func(rw http.ResponseWriter, r *http.Request) {
		if refuse(rw) {
			return
		}
		rep := &uninstall.Report{}
		// the service first: one still serving would put owl back
		uninstallSystem(rep)
		uninstallAgents(r.Context(), rep)
		writeJSON(rw, struct {
			*uninstall.Report
			Failed bool     `json:"failed"`
			Says   []string `json:"says"`
		}{rep, rep.Failed(), rep.Says()})
	})
	mux.HandleFunc("POST /api/uninstall/finish", func(rw http.ResponseWriter, r *http.Request) {
		if refuse(rw) {
			return
		}
		if err := uninstallData(); err != nil {
			fail(rw, errors.New("owl's files can't be removed after it quits: "+err.Error()))
			return
		}
		rw.WriteHeader(http.StatusNoContent)
		if f, ok := rw.(http.Flusher); ok {
			f.Flush()
		}
		go w.Quit()
	})
}
