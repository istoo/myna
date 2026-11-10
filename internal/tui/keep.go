package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/istoo/owl/internal/agent"
	"github.com/istoo/owl/internal/appdir"
	"github.com/istoo/owl/internal/autostart"
	"github.com/istoo/owl/internal/daemon"
	"github.com/istoo/owl/internal/proc"
)

// On Linux with no desktop session (a WSL distro, a server over SSH)
// nothing opens owl at login, so once owl tui quit, the agents it had
// connected asked a gateway that was gone: Codex in WSL "连接会失败，没自动
// 启动owl的网关" (zaidu259434 on X, with owl installed in WSL and
// pointed at Windows' owl as a remote one). As it quits having served
// the gateway with an agent connected, the terminal hands the gateway on:
// to owl's service (owl daemon), installed now where systemd runs it
// and the user never removed it, which starts it with the system or the
// user's login from then on; else to a `owl serve` of its own, in the
// background until the system stops.

// keeper is what keepGateway asks and does; a var for the tests.
type keeper struct {
	goos        string
	autostartOn func() bool
	status      func() (daemon.Status, error)
	removed     func() bool
	start       func() error
	install     func() error
	linger      func() string
	background  func() (pid int, log string, err error)
}

var keep = keeper{
	goos:        runtime.GOOS,
	autostartOn: func() bool { return autostart.Enabled() != autostart.Off },
	status:      daemon.Get,
	removed:     daemon.Removed,
	start:       daemon.Start,
	install:     func() error { return daemon.Install(daemon.Options{}) },
	linger:      daemon.Linger,
	background:  serveInBackground,
}

// keepGateway hands the gateway on as the terminal quits having served it
// (see above), and is what to say of it.
func keepGateway(connected bool) string {
	k := keep
	if k.goos != "linux" || !connected {
		return gatewayGoneNote()
	}
	if k.autostartOn() {
		return atLoginNote()
	}
	st, err := k.status()
	switch {
	case err == nil && st.Installed:
		if err := k.start(); err == nil {
			return "owl: the gateway goes on in owl's service (owl daemon status)"
		}
	case err == nil && !k.removed():
		if err := k.install(); err == nil {
			msg := "owl: the gateway goes on in owl's service, which now starts with this system or your login, so agents connected to owl reach it with owl tui closed; owl daemon uninstall removes it"
			if l := k.linger(); l != "" {
				msg += " (" + l + ")"
			}
			return msg
		}
	}
	pid, log, err := k.background()
	if err != nil {
		return gatewayGoneNote() + " (couldn't start owl serve in the background: " + err.Error() + ")"
	}
	msg := fmt.Sprintf("owl: the gateway goes on in the background (owl serve, pid %d, log %s) until this system stops", pid, log)
	if wslHere() {
		msg += "; for it to start with WSL, turn systemd on with [boot] systemd=true in /etc/wsl.conf and run owl daemon install"
	} else {
		msg += "; owl daemon install starts it with the system"
	}
	return msg
}

// atLoginNote is said where owl opens at login, so the terminal left
// the gateway to the app. On Linux that can be a package's own autostart
// entry on a box no desktop logs into (NixOS-WSL, a server over SSH),
// which nothing ever starts (#3756): the note says the switch that has the
// terminal hand the gateway on instead.
func atLoginNote() string {
	return gatewayGoneNote() + "; owl opens at login and serves it then. Where no desktop logs in here, `owl autostart off` has owl tui hand it on as it quits"
}

// BackgroundLog is the log of a owl serve that owl tui left
// running as it quit (serveInBackground).
func BackgroundLog() string { return filepath.Join(appdir.Cache(), "serve.log") }

// serveInBackground starts `owl serve` on its own, outliving this
// process and its terminal, its output in a log in owl's cache.
func serveInBackground() (int, string, error) {
	exe, err := os.Executable()
	if err != nil {
		return 0, "", err
	}
	log := BackgroundLog()
	if err := os.MkdirAll(filepath.Dir(log), 0o700); err != nil {
		return 0, "", err
	}
	f, err := os.OpenFile(log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return 0, "", err
	}
	defer f.Close()
	cmd := proc.Command(exe, "serve")
	cmd.Stdout, cmd.Stderr = f, f
	detach(cmd)
	// reaped if it ends while this process runs: a released one stayed
	// <defunct> until it quit
	if err := proc.StartReaped(cmd); err != nil {
		return 0, "", err
	}
	return cmd.Process.Pid, log, nil
}

// wslHere: this Linux is a WSL distro.
func wslHere() bool {
	b, _ := os.ReadFile("/proc/version")
	return strings.Contains(strings.ToLower(string(b)), "microsoft")
}

// connected says one of agents is connected to owl.
func connected(agents []*agent.Agent) bool {
	for _, a := range agents {
		if a.Routed != nil && a.Routed() {
			return true
		}
	}
	return false
}
