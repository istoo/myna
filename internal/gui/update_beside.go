package gui

import (
	"context"
	"time"

	"github.com/istoo/owl/internal/gateway"
	"github.com/istoo/owl/internal/update"
)

// restartBeside answers the gateway's RestartPath: `owl update`, run in
// a terminal (gaubeebangeel on X: an agent this gateway serves, at 4am),
// put version in where this owl runs from. Before, the app ran on as
// the old version until someone quit and reopened it, which cut what the
// agents had in flight. It now restarts into it as Settings' AutoRestart
// does: at once when the restart cuts nothing short (on a Mac or Linux it
// hands the gateway over, so the agent's own requests go on), else once
// that is done (waitIdle). A owl that wasn't the one replaced (another
// copy) stays as it is.
func (u *updater) restartBeside(version string) gateway.RestartReply {
	u.mu.Lock()
	restart := u.autoRestart
	in := u.replaced()
	if in {
		u.followed = true
		// what the terminal put in is the version restarted into, as if
		// this one had downloaded it: the feed isn't asked first
		if version != "" && (u.latest == nil || update.Newer(version, u.latest.Version)) {
			u.latest, u.notesIn = &update.Release{Version: version}, ""
		}
		if u.state != "checking" && u.state != "downloading" {
			u.state = "ready"
		}
	}
	u.mu.Unlock()
	switch {
	case !in:
		return gateway.RestartReply{State: "current"}
	case restart == nil:
		return gateway.RestartReply{State: "unsupported"}
	}
	reply := gateway.RestartReply{Handover: handsOver()}
	if waiting, _ := u.waitingFor(); !waiting && u.idle() {
		reply.State = "restarting"
		go restart() // after the answer: the restart lets this request finish
		return reply
	}
	u.waitIdle(restart, false)
	b := restartBusy()
	reply.State, reply.Busy = "waiting", &b
	return reply
}

// replacedTick is how often followReplaced looks at what is on disk; tests
// shorten it.
var replacedTick = 5 * time.Second

// followReplaced restarts this owl into an update another owl put in
// where it runs from, as restartBeside does for owl update's ask
// (visamzh on X). The app and owl web run from the same program (the
// terminal's owl is the app's, linked), and each had its own updater:
// one restarting to update put the new version in, and the other ran on
// as the old one, its Update still offering the version already in, its
// page never reloaded. Neither tells the other, and the other may serve
// no gateway to be asked on, so each looks for itself, a stat every
// replacedTick, until ctx ends (the copy an update left draining looks no
// more, #4938). Only once for a change, so a wait the user called off
// isn't started again; never for what this one put in itself (installed),
// nor in a build of its own, which a rebuild in place would restart.
func (u *updater) followReplaced(ctx context.Context) {
	t := time.NewTicker(replacedTick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		if !update.Released(Version) {
			continue
		}
		u.mu.Lock()
		due := u.autoRestart != nil && !u.installed && !u.followed && u.replaced()
		u.mu.Unlock()
		if due {
			u.restartBeside("")
		}
	}
}
