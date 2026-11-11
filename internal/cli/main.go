// Package cli is owl's command line: every command, and the app, tray
// and TUI it opens (the root's main.go only calls Main).
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/istoo/owl/internal/agent"
	"github.com/istoo/owl/internal/appdir"
	"github.com/istoo/owl/internal/backup"
	"github.com/istoo/owl/internal/browsermcp"
	"github.com/istoo/owl/internal/catalog"
	"github.com/istoo/owl/internal/claudebridge"
	"github.com/istoo/owl/internal/claudecode"
	"github.com/istoo/owl/internal/claudetui"
	"github.com/istoo/owl/internal/codexrelay"
	"github.com/istoo/owl/internal/davsync"
	"github.com/istoo/owl/internal/gateway"
	"github.com/istoo/owl/internal/imagemcp"
	"github.com/istoo/owl/internal/netproxy"
	"github.com/istoo/owl/internal/notices"
	"github.com/istoo/owl/internal/pluginrt"
	"github.com/istoo/owl/internal/proc"
	"github.com/istoo/owl/internal/profile"
	"github.com/istoo/owl/internal/provider"
	"github.com/istoo/owl/internal/sessions"
	"github.com/istoo/owl/internal/settings"
	"github.com/istoo/owl/internal/tui"
	"github.com/istoo/owl/internal/update"
)

// version is the root main package's, set at build time
// (-ldflags "-X main.version=…") and handed over by Main.
var version = "dev"

const usage = `owl — one place to pick every agent's model

  owl                          open the app: a window plus a menu bar icon
  owl tray                     start in the menu bar only
  owl panel                    open the menu bar icon's quick panel, or close it
  owl autostart [on|off]       open owl (in the menu bar) when you log in, or say whether it does
  owl tui                      the same thing, in the terminal (serves the gateway while open when no owl does)
  owl web [--addr host:port] [--lan] [--no-open] [--gateway] [--tls-cert <file> --tls-key <file>]
                                  the app's window in a browser, with the gateway (no desktop needed: WSL, a server over SSH)
                                  a new key each run; OWL_WEB_KEY (16+ letters, digits, - . _ ~) keeps one, signed in for 400 days
                                  behind a reverse proxy, OWL_WEB_URL=https://<the page there> prints the link through it
                                  --gateway: gateway mode, no Agents, Sessions or Library (on by itself with no agents here; Settings › General turns it off)
                                  --tls-cert/--tls-key: the page and the gateway over https:// to other computers
  owl ls                       list detected agents and their settings
  owl <agent>                  show one agent
  owl <agent> <model>          set an agent's model   e.g. owl claude deepseek/deepseek-chat
  owl <agent> <field>          show one field   e.g. owl codex effort
  owl <agent> <field> <value>  set another field   e.g. owl codex effort high
  owl <agent> default          take owl out: the agent back on what it had before
  owl <agent> <field> default  that field back to the agent's own default
  owl <command> --help         what a command or an agent takes; -h or --help anywhere does nothing else

  owl save <name>              snapshot every agent's settings as a profile
  owl use <name>               apply a profile
  owl profiles                 list profiles
  owl rename <old> <new>       rename a profile
  owl rm <name>                delete a profile

  owl backup [--no-keys] [--no-library] [file]    providers, keys, settings, profiles, agent models and the library in one file, sealed with a passphrase
  owl restore [--no-agents] [--no-library] <file> put a backup in on this machine
                                                     --sign-ins on backup: the subscriptions' and plugins' sign-ins too, sealed (off unless given); --no-sign-ins on restore leaves them out
  owl uninstall [--yes]                           disconnect every agent, putting back what each had, then remove owl's service and files
  owl webdav [on <address>|set k=v…|now|off]      the same, kept the same on every computer through a WebDAV folder (owl webdav help)
  owl s3 [on s3://<bucket>[/<prefix>]|set k=v…|now|off]   the same through an S3-compatible bucket: AWS, R2, B2, MinIO… (owl s3 help)
  owl github [on github://<owner>/<repo>[/<folder>]|set k=v…|now|off]   the same in an encrypted GitHub repository (owl github help)

  owl library [sync|instructions|mcp|skill]   the instructions, MCP servers and skills written into every agent (owl library help)

  owl providers                list your providers: host, key, models, who uses them
  owl presets                  the vendors owl knows: add one with just a key
  owl provider add <preset> <key>   e.g. owl provider add deepseek sk-…
  owl provider add <name> k=v…      a custom vendor (owl provider for the fields)
  owl provider key|models|test|rm <id>
  owl provider fallback <id> <provider/model>…   use these when it's out of quota or down
  owl import [-y] <link>       add the provider a owl://import?… link describes
  owl models [<agent>]         every model agents can pick, as provider/model; an agent's, and why others aren't
  owl model name <provider/model> <name>|--reset       the name a model goes by, everywhere
  owl model efforts <provider/model> <l>,<l>|--reset   the reasoning levels a model offers (owl model help)
  owl visible [<agent> <family|provider|group>,… | all | --only-picked | --show-new]
                                  which models an agent is shown: families (owl provider/group set <id> family=…),
                                  or only the models ticked for it, a new one off until it is ticked
  owl search [add <api> <key>|rm <api>]   Tavily, Brave, Exa, Firecrawl, You.com, DeepSeek, Doubao Search (Volcengine), Yuanbao Search (Tencent Cloud) or SearXNG for web search when no provider can search
  owl groups                   routing groups: several models agents pick as one, group/<id>
  owl group add <name> models=<m1>,<m2> [routing=smart|order|rotate|usage|pace|weight|free|fast|cheap|drain] [stays=auto|session|turn|off]
  owl group <id> | set <id> k=v… | rm <id>   show, change or remove one (owl group help for more)
  owl accounts [agent] [--json]  every subscription owl knows, with each one's allowance used and when it resets
  owl accounts add <agent>     sign in to one more Claude, ChatGPT or Google (Gemini CLI, Antigravity) subscription
  owl accounts add copilot [--host <name>.ghe.com]   one more Copilot account, on github.com or an enterprise's GHE.com
  owl accounts discovery [on|off]   read agents' sign-ins (default on; OWL_NO_ACCOUNTS=1 disables too)
  owl accounts import <codex|claude|antigravity|factory> <file>... [--yes]   bring in accounts from other tools' files (Codex's auth.json, Cockpit Tools, CLIProxyAPI, Sub2API, claude-swap); the tool a ChatGPT or Claude file came from is signed out of it
  owl accounts switch <agent> <email|name>   sign the agent in to another of them, by its email or the name you gave it
  owl accounts switch <claude|codex> next   to the next account in order with room, or the one with the most left
  owl accounts refresh         renew the saved ChatGPT and Claude sign-ins now (the gateway does it every 8 days)
  owl accounts checkin         WorkBuddy's daily check-in (签到) for each WorkBuddy account, now (Settings can do it daily)
  owl accounts project <gemini|antigravity> <email> <project>   the Google Cloud project a Google account's requests go to
  owl accounts alias <agent> <email> [<name>|--clear]   a name of your own for an account, shown on its card, in the menu bar and in owl quota
  owl accounts switch-only codex [on|off]   only switch Codex's own sign-in to an account with room; its requests go straight to OpenAI, never through owl
  owl accounts switch-only claude [on|off]  only switch Claude Code's own sign-in to an account with room; owl serves no requests on the Claude accounts
  owl claude-code [install [--yes]|remove]   the Claude Code a Claude subscription runs; install downloads Anthropic's own build, checked against its manifest, for a server or container without one
  owl plugin [add <package>|rm|update|on|off|login <provider>|logout <provider>]
                                  OpenCode provider plugins and pi packages: subscriptions signed in to, and served, through a plugin
  owl plugin move|migrate <subscription>   run a built-in subscription's accounts on its community plugin

  owl serve [--web[=host:port]] [--tls-cert <file> --tls-key <file>]
                                  run the gateway alone (the app runs it too); --web serves owl web's page too, from the same process (127.0.0.1:3430 unless told);
                                  with a certificate and key (PEM; OWL_TLS_CERT, OWL_TLS_KEY) other computers reach both over https://
  owl daemon [install|uninstall|start|stop|restart|status]
                                  owl serve as a service of yours, from boot or login and started again when it stops (systemd --user on Linux, launchd on the Mac)
  owl daemon install [--web] [--lan] [--addr host:port] [--no-web] [--tls-cert <file> --tls-key <file>|--no-tls]
                                  --web: the service serves owl web's page too, with a key kept across restarts; installed again, it keeps what it served
  owl daemon web [on|off] [--lan] [--addr host:port]
                                  the service's page: its link, or turn it on or off
  owl healthcheck              exit 0 when the gateway answers (a container's HEALTHCHECK)
  owl heap [<file>]            a Go heap profile of the running owl, with its memory numbers, for a report of it using too much memory
  owl cpu [<file>] [--seconds N]   a Go CPU profile of the running owl over 30s (5..120) while you use it, for a report of it using too much CPU
  owl goroutines [<file>]      every goroutine of the running owl, its stack and how long it waited, for a report of it stuck at something
  owl log [--path] [-n N] [-f]  where owl's logs are, to send with a report: the app's (app.log), the service's; -n prints the app log's last N lines, -f follows it
  owl gateway-key list|add <name> [--read-only]|rotate <id>|remove <id>   manage the keys clients use to call a shared gateway
  owl gateway-key limit <id> [list|off|reset [n]|remove <n>|[add] day|week|month|<N>d|<N>h --tokens N --cost USD --cache-reads]   a key's own limits, and what it used
  owl gateway-key models <id> [all|<provider>/<model>|<provider>/* ...]   the models a key may use, every one unless it names some
  owl gateway-key quotas <id> [on|off]   whether a key held to a budget, models or accounts is told the quotas of what it may use
  owl gateway-key access <id> [read-only|full]   read-only: the key reads quotas and usage (GET /, /v1/owl/quotas, /v1/owl/usage) and uses no model, MCP server or write
  owl gateway-key move <id> <place>   put a key at a place in the list (from 1), the order list and the app show them in
  owl gateway-key issue --ttl 1h [--name n] [--models a,b] [--accounts a,b] [--tokens N] [--cost USD] [--read-only] [--from <id>] [--json]
                                         a short-lived key (1m to 30 days) that stops working by itself; --from holds it to a key that issues keys; prints the key
  owl gateway-key issuer <id> [on|off]   let a key issue short-lived keys under itself (POST /v1/owl/keys with it), each held to its models, accounts, limits and life
  owl gateway-key oidc add --key <id> --repo owner/name [--ref refs/heads/main] [--environment name] [--claim name=value]… [--ttl 1h] [--issuer URL] [--audience aud]
                                         let a CI job's OIDC token (GitHub Actions by default) be traded for a key issued under <id> (POST /v1/owl/keys/oidc); oidc list | remove <rule id>
  owl mcp image                the image and video generation MCP server an agent is given from the library (stdio)
  owl tools [<name>]           the tools owl's MCP servers give agents (images, vision, web search, plugins'), with their parameters
  owl tool <name> [--<param> <value>…] [<words>] [-o <file|folder>] [--json]   call one through the running owl; an image or video it makes is saved
  owl usage [today|7d|30d|all] tokens and cost per agent, model and subscription account (30d)
  owl usage --csv [--account <name>] [today|7d|30d|all]   every request as CSV (or one account's): the model asked for, sent and served, tokens, cost, time, status, account
  owl usage [--csv [--account <name>]] --last <dur>|--days <n>   the same for a window ending now (24h, 90m, 1.5 days), not from a midnight
  owl sessions [--model <m>] [--folder <f>] [--json]   the latest Claude Code, Codex, OpenCode and Pi sessions, with what each cost
  owl sessions --days N|today|all [--model <m>] [--folder <f>] [--json]
                                  what every session spent, day by day, with the top models and folders (7 days)
  owl sessions read [on|off all|<agent>…]   which agents' session history owl may read (none until allowed)
  owl quota [<provider>] [--json]  what is left of every subscription, plan and key balance
  owl quota wait <provider|account> [--timeout <d>] [--quiet]
                                  block until that subscription (any of its accounts) or account has allowance again
  owl quota history [<provider|account>] [--days N] [--json]
                                  each window's readings over time, kept 45 days
  owl statusline [<provider|account>]   one line for Claude Code's status line: the session's model and each account's windows
  owl sync                     refresh the model catalog and vendor model lists
  owl agents                   list every supported agent
  owl settings [list|get <key>|set <key> <value>|reset <key>] [--json]
                                  every setting the window's Settings page saves, read and set as it saves them (owl settings help)
  owl update [check] [--proxy <url>] [--mirror <prefix>]
                                  install the newest release (check: only say if there is one); --proxy: an
                                  http(s):// or socks5:// proxy for it; --mirror: a GitHub download mirror put
                                  before the github.com URL (none unless given; still checked against github.com/istoo/owl's SHA-256)
  owl update mirror [<prefix>|off]  the mirror every update, the app's own too, is downloaded through
  owl update channel [stable|every]  which versions updates offer: stable (one a week, an urgent one at
                                  once) or every version as it comes out (the default)
  owl update auto [on|off] [download|notify] [30m|1h|6h|24h]
                                  whether the app looks for updates by itself, how often (6h), and whether
                                  it downloads one at once or only says it is out (notify: for metered networks)
  owl licenses                 the third-party notices of what owl is built with

agents: claude (cc), codex, gemini, opencode (oc), mimocode, deveco, pi, goose, cursor, zed, copilot, crush, aside
`

var (
	bold  = lipgloss.NewStyle().Bold(true)
	muted = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#8B8F98", Dark: "#7C8290"})
	faint = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#C4C7CE", Dark: "#4A4F5A"})
	green = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#0F9D58", Dark: "#7EE2A8"})
)

// Main runs owl with os.Args, at the version the build was stamped with.
func Main(v string) {
	version = v
	// before anything reads or writes a file: no home, or a relative one,
	// would put the agents' configs and owl's keys under the working
	// folder. What needs no file still answers (owl version in a
	// container or a script without HOME): run makes owl's folders first.
	ignored, err := appdir.CheckEnv()
	if err != nil {
		if len(os.Args) > 1 {
			switch os.Args[1] {
			case "-v", "--version", "version":
				fmt.Println("owl", version)
				return
			case "-h", "--help", "help":
				fmt.Print(usage)
				return
			case "licenses":
				fmt.Print(notices.Text)
				return
			}
		}
		fmt.Fprintln(os.Stderr, "owl:", err)
		os.Exit(1)
	}
	slices.Sort(ignored)
	for _, v := range ignored {
		fmt.Fprintf(os.Stderr, "owl: ignoring %s: not an absolute path (the programs owl starts still get it)\n", v)
	}
	if provider.TookOpenedURL(os.Args[1:]) {
		// Claude Code, signing in for owl, handed over the page to open
		return
	}
	endProbesOnSignal()
	gateway.Version = version
	netproxy.Install()
	update.GUI = hasGUI
	err = run(os.Args[1:])
	proc.EndProbes() // a CLI still being asked something isn't left to init
	sessions.Saved() // the session index kept, for the next run
	if err != nil {
		// a command with exit codes of its own (quota wait) says which
		code := 1
		var e exitError
		if errors.As(err, &e) {
			code = e.code
		}
		if msg := err.Error(); msg != "" {
			fmt.Fprintln(os.Stderr, "owl:", msg)
		}
		os.Exit(code)
	}
}

// runTUI runs the TUI, which quits on Ctrl+C and SIGTERM itself once it
// has started; it asks CLIs first, and a signal then ends those.
func runTUI() error {
	return tuiRun(ownSignals)
}

// tuiRun is tui.Run; a var so tests can stand in for it.
var tuiRun = tui.Run

// syncAgentsOnChange sets what a catalog change (a provider or group
// saved, a vendor's list fetched anew) does: the model lists agents keep
// in files of their own are brought up to date behind the save, which
// answers without waiting for every agent's files to be written (on
// Windows, half a second for each one another program holds open).
func syncAgentsOnChange() { catalog.Changed = agent.SyncCatalogSoon }

// startUp brings the files owl and its agents keep up to what this
// owl reads, before a command that may read them; a var so a test can
// tell it ran.
var startUp = func() {
	// started by an update, the owl it replaces goes first: the moves
	// below write the files it may still be writing
	update.AwaitPredecessor()
	makeDirs()
	settings.Migrate()
	// every built-in plugin starts on (plugins.yml), on a new install too
	if _, err := pluginrt.Settle(); err != nil {
		fmt.Fprintf(os.Stderr, "owl: plugins.yml: %v\n", err)
	}
	if _, err := pluginrt.Refresh(); err != nil {
		fmt.Fprintf(os.Stderr, "owl: plugins.yml: %v\n", err)
	}
	// a vendor's keys saved a provider each ("NVIDIA #001"…) are one
	// provider with those keys, before anything lists them
	if _, err := provider.FoldCopies(); err != nil {
		fmt.Fprintf(os.Stderr, "owl: providers.json: %v\n", err)
	}
	// the providers and settings read once for every agent's fields, which
	// the moves below look at (a write among them reads them again)
	release := provider.Hold()
	agent.RenameLegacy()
	agent.MoveCursorEfforts()
	agent.MoveAntigravityEfforts()
	agent.MoveOffAccountIDs()
	release()
}

func run(args []string) error {
	// a command that saved a provider or a group ends with the agents'
	// files written, which its save no longer waits for; and a test's run
	// leaves catalog.Changed as it found it
	was := catalog.Changed
	defer func() {
		agent.SyncCatalogSettled()
		catalog.Changed = was
	}()
	// what a shell's agy runs (agy_cli.go): every argument after run is
	// agy's, --help too, and nothing is moved or started first
	if ok, err := agyCmd(args); ok {
		return err
	}
	// before anything is read, moved or started: --help on any command
	// prints its usage and does nothing else
	if helpAsked(args) {
		fmt.Println(strings.TrimRight(helpFor(args), "\n"))
		return nil
	}
	if len(args) > 0 && args[0] == "healthcheck" {
		return healthcheck() // every few seconds in a container: nothing else
	}
	if len(args) > 0 && args[0] == "heap" {
		// the running owl's heap profile: asks it, and moves nothing here
		return heapCmd(args[1:])
	}
	if len(args) > 0 && args[0] == "cpu" {
		// the running owl's CPU profile: asks it, and moves nothing here
		return cpuCmd(args[1:])
	}
	if len(args) > 0 && args[0] == "goroutines" {
		// the running owl's goroutines: asks it, and moves nothing here
		return goroutinesCmd(args[1:])
	}
	if len(args) > 0 && (args[0] == "log" || args[0] == "logs") {
		// where owl's logs are, and the app log's last lines: reads
		// files, and moves nothing
		return logCmd(args[1:])
	}
	if len(args) == 2 && args[0] == agent.DryRunArg {
		// an agent disconnected on a copy of its files under a temporary
		// home, for the Agents page to show what disconnecting changes
		// (agent.DisconnectPreview)
		return agent.DryRun(args[1])
	}
	if len(args) > 1 && (args[0] == "accounts" || args[0] == "account") && args[1] == "discovery" {
		return accountsDiscoveryCmd(args[2:])
	}
	// what an agent starts for each session or message talks to the
	// running gateway and nothing else: the moves below list every
	// provider and account, which starts the plugin host and asks claude,
	// cursor-agent and devin who is signed in — five processes and a
	// Claude /usage for each Claude Code session's image MCP (#3318)
	if len(args) > 0 {
		switch args[0] {
		case "mcp":
			if len(args) > 1 && args[1] == "browser" {
				return browsermcp.Run(args[2:])
			}
			if len(args) > 1 && args[1] == "search" {
				return browsermcp.RunSearch(args[2:])
			}
			if len(args) > 1 && args[1] == "vision" {
				return browsermcp.RunVision(args[2:])
			}
			return imagemcp.Run(args[1:])
		case "claude-mcp-helper": // internal: stdio MCP subprocess spawned by Claude Code
			return claudebridge.RunMCP(args[1:])
		case claudetui.Command: // internal: Claude Code's interactive mode for the bridge
			// it ends its Claude Code and removes its files on SIGINT,
			// SIGTERM and SIGHUP itself: owl dying of the signal first
			// left them behind
			leaveSignals()
			return claudetui.Run(args[1:])
		case "statusline":
			return statuslineCmd(args)
		case "tools":
			return toolsCmd(args[1:])
		case "tool":
			return toolCmd(args[1:])
		case codexrelay.Command: // internal: what a quitting owl leaves for Codex's open threads
			return codexrelay.Main(args[1:], gateway.CodexRelay())
		case "uninstall":
			// what the app starts as it quits from Settings › Uninstall
			// owl: nothing is made again before its folders go
			if len(args) > 1 && args[1] == "--data-after" {
				return uninstallData(args[2:])
			}
		}
	}
	// owl serve beside another owl serving the gateway (the app, a
	// service, the copy an update left) waits for the port before it moves
	// or starts anything, rather than doing it all to fail to bind (#4938)
	if len(args) > 0 && args[0] == "serve" && plainServe(args[1:]) {
		awaitGatewayFree(gatewayFreeEvery)
	}
	startUp()
	// a provider added, edited or removed, or a list fetched anew, reaches
	// the model lists agents keep in files of their own, behind the save
	// that changed it (agent.SyncCatalogSoon)
	syncAgentsOnChange()
	// a model Claude Code names that owl doesn't serve goes to the one
	// it is set to use for that tier
	gateway.StandIn = agent.StandIn
	// and its auto mode classifier, on a sonnet tier that follows a main
	// model /model has moved, to the model it moved to
	gateway.ClassifierStandIn = agent.ClassifierStandIn
	// a Codex that reaches owl for account failover alone is handed
	// only its own models (#1385)
	gateway.CodexOwnOnly = agent.CodexOwnOnly
	// what owl writes into an agent's TOML, YAML or dotenv config says
	// so in a comment, for whoever reads it once owl is gone
	agent.MarkWritten()
	// the Codex app's titles, descriptions and summaries, which it asks
	// for on a model of its own, on the model of owl's picked beside
	// its ChatGPT sign-in rather than on that account (atom_Ray on X)
	gateway.CodexSideModel = agent.CodexSideModel
	gateway.CodexSideSet = agent.CodexSideSet
	// the list the Codex app reads from a file beside the ChatGPT sign-in,
	// its picker showing owl's models only so (laddieyao on Discord),
	// written again when the backend's list changes
	agent.CodexDesktopList = gateway.CodexDesktopCatalog
	gateway.CodexDesktopChanged = agent.SyncCatalogSoon
	// the setup kept the same on every computer, by whichever serves
	gateway.WhileServing = append(gateway.WhileServing, davsync.Run)
	// and the setup backed up into a folder here by itself (jwaitq)
	gateway.WhileServing = append(gateway.WhileServing, backup.RunAuto)
	// and dsh's patch lists, which dsh reads live: a route left behind by
	// something else writing the file fails every session there until
	// owl writes its own list again
	gateway.WhileServing = append(gateway.WhileServing, agent.KeepDshWired)
	// and Cursor Private Inference's variables, which the Mac's launchd
	// forgets at a restart, for the gateway's address now
	gateway.WhileServing = append(gateway.WhileServing, agent.KeepCursorLocalEnv)
	// and Claude Science's daemon, which a restart of the computer ends,
	// started on the gateway again while it is connected
	gateway.WhileServing = append(gateway.WhileServing, agent.KeepClaudeScience)
	// and the Claude Code owl downloaded for a machine with none (a
	// server's, a container's), at Claude Code's newest release: its runs
	// turn its own updater off
	gateway.WhileServing = append(gateway.WhileServing, claudecode.KeepCurrent)
	// and the request archive, when it is on, goes to the bucket sync is to
	gateway.ArchiveBucket = func() (gateway.Putter, bool) {
		if b, ok := davsync.S3Bucket(); ok {
			return b, true
		}
		return nil, false
	}
	if len(args) == 0 {
		if hasGUI {
			return runGUI(true, "")
		}
		gateway.Serves = "tui"
		return runTUI()
	}
	// a owl:// link the system handed over (Windows, Linux): the app
	// opens it for the user to confirm
	if strings.HasPrefix(strings.ToLower(args[0]), "owl:") {
		if !hasGUI {
			return importCmd(args)
		}
		return runGUI(false, args[0])
	}
	// how this process was started, for the Routing page of another owl on
	// this computer to name the one that serves the gateway (#2606)
	switch args[0] {
	case "tui", "web", "serve", "tray":
		gateway.Serves = args[0]
	case "app", "gui", "panel", "-Embedding":
		gateway.Serves = "app"
	}
	switch args[0] {
	case "tui":
		return runTUI()
	case "web":
		return webCmd(args[1:])
	case "app", "gui":
		// `owl gui settings`: the window on that tab, as a restart to
		// update from it comes back (update.RelaunchArgs)
		if len(args) > 1 {
			return runWindow(args[1])
		}
		return runGUI(true, "")
	case "tray":
		return runGUI(false, "")
	case "-Embedding":
		// Windows starting owl for a click on one of its notifications
		// (a usage alert, #368) left in the Action Center after it quit:
		// the window, on the Usage page
		return runWindow("usage")
	case "panel":
		return runPanel()
	case "autostart":
		return autostartCmd(args[1:])
	case "daemon":
		return daemonCmd(args[1:])
	case "-h", "--help", "help":
		fmt.Print(usage)
		return nil
	case "-v", "--version", "version":
		fmt.Println("owl", version)
		return nil
	case "licenses":
		fmt.Print(notices.Text)
		return nil
	case "ls", "list":
		// in the order the app lists them; those hidden there come last, dimmed
		shown, hidden := settings.Arrange(settings.Load(), agent.Detected(), func(a *agent.Agent) string { return a.ID })
		return list(append(shown, hidden...), true, len(shown))
	case "agents":
		return list(agent.All(), false, -1)
	case "sync":
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := catalog.Sync(ctx); err != nil {
			return err
		}
		fmt.Println(green.Render("✓"), "catalog saved to", catalog.CachePath())
		refreshLive(ctx)
		return nil
	case "save", "use", "rm", "rename", "profiles":
		return profiles(args)
	case "import":
		return importCmd(args[1:])
	case "providers":
		return providers()
	case "presets":
		return presets()
	case "provider":
		return providerCmd(args)
	case "models":
		return models(args[1:])
	case "model":
		return modelCmd(args[1:])
	case "visible":
		return visibleCmd(args[1:])
	case "search":
		return searchCmd(args[1:])
	case "groups":
		return groups()
	case "group":
		return groupCmd(args)
	case "serve":
		return serveCmd(args[1:])
	case "gateway-key":
		return gatewayKeys(args)
	case "accounts", "account":
		return accountsCmd(args)
	case "usage":
		return usageCmd(args)
	case "sessions":
		return sessionsCmd(args)
	case "quota", "quotas":
		return quotaCmd(args)
	case "update":
		return updateCmd(args)
	case "library", "lib":
		return libraryCmd(args)
	case "backup":
		return backupCmd(args[1:])
	case "restore":
		return restoreCmd(args[1:])
	case "uninstall":
		return uninstallCmd(args[1:])
	case "webdav", "dav":
		return webdavCmd(args[1:])
	case "plugin", "plugins":
		return pluginCmd(args)
	case "s3":
		return s3Cmd(args[1:])
	case "github":
		return githubCmd(args[1:])
	case "claude-code":
		return claudeCodeCmd(args[1:])
	case "settings":
		return settingsCmd(args[1:])
	}

	// the agent of a plugin that is gone (deleted on disk) is no agent
	// here, but what owl stashed of its file puts it back (#3810)
	if len(args) == 2 && strings.EqualFold(args[1], "default") {
		if g, ok, err := agent.RestoreGone(args[0]); ok {
			return goneDefault(args[0], g, err)
		}
	}
	a, err := agent.Find(args[0])
	if err != nil {
		return err
	}
	// no agent form takes a flag: one is never set as a model or a value
	// (`owl snow --verbose` would take owl out of Snow, #1457)
	if f := flagArg(args[1:]); f != "" {
		return fmt.Errorf("unknown flag %s: nothing was changed\n\n%s", f, strings.TrimRight(agentHelp(a), "\n"))
	}
	if len(a.Fields) == 0 && a.Import != nil {
		// `owl cindy`: it takes owl through its own link, confirmed there
		if len(args) > 1 {
			link := a.Import()
			if link == "" {
				// Lody takes only a gateway on this computer's loopback
				return fmt.Errorf("%s takes owl only at a loopback address, and the gateway is at %s: nothing was opened", a.Name, gateway.URL())
			}
			// handed over only when something here opens it, or nothing
			// opens and nothing says so (sungt, 10-29)
			openApp := func(u string) error { return appLink(u, func(u string) error { openInBrowser(u); return nil }) }
			handOver := openApp
			if a.ID == "lody" {
				// Lody on Linux is run with the link (agent.OpenLody)
				handOver = func(u string) error { return agent.OpenLody(u, openApp) }
			}
			if err := handOver(link); err != nil {
				return fmt.Errorf("%w. Open %s once or install it again, then run this again", err, a.Name)
			}
			fmt.Println(green.Render("✓"), bold.Render(a.Name), muted.Render("opened to add owl — confirm it there"))
			fmt.Println(muted.Render("  " + link))
			return nil
		}
	}
	switch len(args) {
	case 1:
		return list([]*agent.Agent{a}, true, -1)
	case 2:
		// default is owl's word in any case, as an agent's name and a
		// field's are: owl codex DEFAULT wrote model = "DEFAULT"
		if strings.EqualFold(args[1], "default") {
			if a.Wired() {
				return disconnect(a)
			}
			return set(a, a.Fields[0].Key, "")
		}
		// `owl codex model`: a field's name alone asks what it is set to,
		// where it was set as the model (model = "model"). Claude Code's
		// opus is a model's name as well as a tier's, and stays the model
		if f := a.Field(args[1]); f != nil && !modelAlias(a, args[1]) {
			return showField(a, f)
		}
		// `owl codex xhigh`: a bare value that belongs to a non-model field
		// (effort levels, for instance) is routed there; anything else is a model.
		if f := fieldForValue(a, args[1]); f != nil {
			return set(a, f.Key, args[1])
		}
		return set(a, a.Fields[0].Key, args[1])
	case 3:
		value := args[2]
		if strings.EqualFold(value, "default") {
			value = ""
		}
		return set(a, args[1], value)
	}
	return fmt.Errorf("too many arguments\n\n%s", usage)
}

func set(a *agent.Agent, key, value string) error {
	f := a.Field(key)
	if f == nil {
		var keys []string
		for _, f := range a.Fields {
			keys = append(keys, f.Key)
		}
		return fmt.Errorf("%s has no field %q (fields: %s)", a.Name, key, strings.Join(keys, ", "))
	}
	// a field's name is no value: owl codex subagent effort ran Codex's
	// subagents on a model called effort. Claude Code's opus is a model's
	// name as well as a tier's
	if g := a.Field(value); value != "" && g != nil && !modelAlias(a, value) {
		return fieldAsValue(a, f, g, value)
	}
	value, err := a.Spell(f.Key, value)
	if err != nil {
		return err
	}
	before := f.Get()
	wired := a.Wired()
	// a owl model picked on an agent not connected yet connects it
	// first, as the Agents page's picker does (#1900): the agent is wired
	// in, and Disconnect puts back what it had before
	if err := a.Pick(f.Key, value); err != nil {
		return err
	}
	// what the config reads now, not what was asked: an agent may name the
	// model under a provider of its own (OpenCode's owl-relay/…), and a
	// value it had already is said to be so
	now := f.Get()
	shown := now
	if value == "" || now == "" {
		shown = muted.Render("default")
	}
	if now == before {
		shown += " " + muted.Render("(unchanged)")
	}
	fmt.Println(green.Render("✓"), bold.Render(a.Name), muted.Render(f.Label), shown)
	// a value that is the agent's own, not one of owl's, takes owl
	// out: said, as Disconnect is, with the way back (#2000: owl claude
	// sonnet put Claude Code back on its DeepSeek endpoint, which refused
	// Sonnet, and only "✓ Claude Code model sonnet" was printed)
	if wired && now != "" && !a.Wired() {
		fmt.Println(muted.Render(fmt.Sprintf("  disconnected from owl: %s is %s's own, sent to the endpoint it had before owl; owl %s <model> with one of owl's models connects it again", now, a.Name, a.ID)))
	}
	if a.Notice != nil && (!f.Live || !wired && a.Wired()) {
		if n := a.Notice(); n != "" {
			fmt.Println(muted.Render("  ↻ " + n))
		}
	}
	return nil
}

// modelAlias says whether a word names the agent's model, in any case,
// though a field goes by it too: Claude Code's opus, a tier's name as well
func modelAlias(a *agent.Agent, w string) bool {
	return slices.ContainsFunc(a.ModelAliases, func(m string) bool { return strings.EqualFold(m, w) })
}

// fieldAsValue refuses field g's name given as field f's value, for what
// was meant: two words that are one field's label (owl codex subagent
// effort), a switch, which is turned on (Claude Code's ultracode), or the
// field, which its name alone shows.
func fieldAsValue(a *agent.Agent, f, g *agent.Field, value string) error {
	for _, name := range []string{f.Key + " " + value, f.Key + "_" + value} {
		if h := a.Field(name); h != nil {
			return fmt.Errorf("%q is one field of %s's: owl %s %s shows it", h.Label, a.Name, a.ID, h.Key)
		}
	}
	if g.Options != nil {
		if o := g.Options(a.Values()); len(o) == 1 && o[0].Value == "on" {
			return fmt.Errorf("%q is a switch of %s's, not a value for its %s: owl %s %s on", value, a.Name, f.Label, a.ID, g.Key)
		}
	}
	return fmt.Errorf("%q is a field of %s's, not a value for its %s: owl %s %s shows it", value, a.Name, f.Label, a.ID, g.Key)
}

// showField is `owl <agent> <field>`: what the field is set to, as set
// prints it
func showField(a *agent.Agent, f *agent.Field) error {
	v := f.Get()
	if v == "" {
		v = muted.Render("default")
	}
	fmt.Println(bold.Render(a.Name), muted.Render(f.Label), v)
	return nil
}

// disconnect is `owl <agent> default` on an agent owl is wired into:
// the Agents page's Disconnect, which puts back what the user had before
// owl — Claude Code's own model, its endpoint — where a field's default
// leaves the agent as installed (__jingling on X: owl claude default
// took the model they had set away with owl's)
func disconnect(a *agent.Agent) error {
	before := a.Values()
	if err := a.Disconnect(); err != nil {
		return err
	}
	now := a.Values()
	for _, f := range a.Fields {
		if before[f.Key] == now[f.Key] {
			continue
		}
		shown := now[f.Key]
		if shown == "" {
			shown = muted.Render("default")
		}
		fmt.Println(green.Render("✓"), bold.Render(a.Name), muted.Render(f.Label), shown)
	}
	fmt.Println(muted.Render("  disconnected from owl, back to what it had before"))
	if a.Notice != nil {
		if n := a.Notice(); n != "" {
			fmt.Println(muted.Render("  ↻ " + n))
		}
	}
	return nil
}

// goneDefault says what `owl <id> default` did for the agent of a
// plugin that is gone.
func goneDefault(id string, g agent.Gone, err error) error {
	for _, p := range g.Put {
		fmt.Println(green.Render("✓"), bold.Render(id), muted.Render(p))
	}
	for _, p := range g.Off {
		fmt.Println(green.Render("✓"), bold.Render(id), muted.Render(p+": owl was out of it already"))
	}
	if len(g.Put) > 0 {
		fmt.Println(muted.Render("  its plugin is gone; disconnected from owl, back to what it had before"))
	}
	if g.PluginOff != "" {
		fmt.Println(green.Render("✓"), bold.Render(g.PluginOff), muted.Render("is not on owl: its plugin is off, and owl put back what it had when the plugin was turned off"))
	}
	if err != nil {
		return fmt.Errorf("%s's plugin is gone, and owl couldn't put its config back: %w", id, err)
	}
	return nil
}

func fieldForValue(a *agent.Agent, v string) *agent.Field {
	vals := a.Values()
	// the agent's suffix after a model (omp's ":max") aside: a role on the
	// same model at that level offers it as typed, and would take it. A list
	// of models no picker offers: it is for the model
	if a.SplitSuffix != nil {
		m, _, one := a.SplitSuffix(v)
		if !one {
			return nil
		}
		v = m
	}
	// a model stays with the model, even where other fields offer it too
	// (Claude Code's opus/sonnet/haiku/fable)
	for _, o := range a.Fields[0].Options(vals) {
		if o.Value == v {
			return nil
		}
	}
	for i := 1; i < len(a.Fields); i++ {
		for _, o := range a.Fields[i].Options(vals) {
			if o.Value == v {
				return &a.Fields[i]
			}
		}
	}
	return nil
}

// list prints the agents; those from dimFrom on (when not -1) are the ones
// hidden in the app, and are dimmed.
func list(agents []*agent.Agent, detectedOnly bool, dimFrom int) error {
	if len(agents) == 0 {
		return fmt.Errorf("no supported agents found on this machine")
	}
	type row struct{ name, vals, path string }
	var rows []row
	nameW, valW := 0, 0
	for i, a := range agents {
		r := row{name: a.Name, path: tilde(a.Path)}
		if !detectedOnly && !a.Detected() {
			r.name = faint.Render(a.Name)
			r.vals = faint.Render("not detected")
			r.path = ""
		} else if a.CLIMissing() {
			// its folder left by an uninstall (#2173): what it holds is
			// read by nothing
			r.name = faint.Render(a.Name)
			r.vals = faint.Render("CLI not found")
		} else {
			vals := a.Values()
			dim := dimFrom >= 0 && i >= dimFrom
			label, value := muted, lipgloss.NewStyle()
			if dim {
				label, value = faint, faint
			}
			var parts []string
			for _, f := range a.Fields {
				v := vals[f.Key]
				if v == "" && f.Quiet {
					continue
				}
				if v == "" {
					v = faint.Render("—")
				} else {
					v = value.Render(v)
				}
				if f.Label == "model" {
					parts = append(parts, v)
				} else {
					parts = append(parts, label.Render(f.Label)+" "+v)
				}
			}
			r.name = bold.Render(a.Name)
			if dim {
				r.name = faint.Render(a.Name) + " " + faint.Render("hidden")
			}
			r.vals = strings.Join(parts, label.Render("  ·  "))
			// through owl for account failover while not connected (#1385)
			if said := a.FailoverSaid(); said != "" {
				r.vals += label.Render("  ·  " + said)
			}
			if a.Import != nil {
				// unknown reads as not added here: the line only offers the
				// command, never says owl is missing
				added := false
				if a.Added != nil {
					added, _ = a.Added()
				}
				if added {
					r.vals = value.Render("owl added")
				} else {
					r.vals = label.Render("owl "+a.ID+" add") + faint.Render("  to add owl")
				}
			}
		}
		nameW = max(nameW, lipgloss.Width(r.name))
		valW = max(valW, lipgloss.Width(r.vals))
		rows = append(rows, r)
	}
	for _, r := range rows {
		fmt.Printf("  %s  %s  %s\n", pad(r.name, nameW), pad(r.vals, valW), faint.Render(r.path))
	}
	return nil
}

func tilde(p string) string { return appdir.Tilde(p) }

func profiles(args []string) error {
	switch args[0] {
	case "profiles":
		ps, err := profile.Load()
		if err != nil {
			return err
		}
		if len(ps) == 0 {
			fmt.Println(muted.Render("no profiles yet · owl save <name>"))
			return nil
		}
		detected := agent.Detected()
		for _, n := range profile.Names(ps) {
			// ✓ the one the setup is now (superhtoooth on X)
			mark := " "
			if profile.InUse(ps[n], detected, nil) {
				mark = green.Render("✓")
			}
			fmt.Printf("%s %s  %s\n", mark, bold.Render(n), muted.Render(profile.LongSummary(ps[n])))
		}
		return nil
	case "save":
		if len(args) < 2 {
			return fmt.Errorf("usage: owl save <name>")
		}
		p, err := profile.Snapshot()
		if err != nil {
			return err
		}
		if err := profile.Save(args[1], p); err != nil {
			return err
		}
		fmt.Println(green.Render("✓"), "saved profile", bold.Render(args[1]))
		if p.Library != nil {
			fmt.Println(" ", muted.Render("with the "+p.Library.Summary()))
		}
		return nil
	case "use":
		if len(args) < 2 {
			return fmt.Errorf("usage: owl use <name>")
		}
		ps, err := profile.Load()
		if err != nil {
			return err
		}
		p, ok := ps[args[1]]
		if !ok {
			return fmt.Errorf("no profile named %q", args[1])
		}
		a, err := profile.Apply(p)
		if err != nil {
			return err
		}
		fmt.Println(green.Render("✓"), "applied", bold.Render(args[1]), muted.Render(fmt.Sprintf("(%d changed)", a.Changed)))
		for _, line := range profile.Report(a) {
			fmt.Println(" ", muted.Render(line))
		}
		return nil
	case "rm":
		if len(args) < 2 {
			return fmt.Errorf("usage: owl rm <name>")
		}
		if err := profile.Delete(args[1]); err != nil {
			return err
		}
		fmt.Println(green.Render("✓"), "deleted profile", bold.Render(args[1]))
		return nil
	case "rename":
		// a flag is never taken for a name to write (owl snow --help
		// wrote "--help" into a profile)
		if len(args) != 3 || strings.HasPrefix(args[2], "-") {
			return fmt.Errorf("usage: owl rename <old> <new>")
		}
		if err := profile.Rename(args[1], args[2]); err != nil {
			return err
		}
		fmt.Println(green.Render("✓"), "renamed profile", bold.Render(args[1]), "to", bold.Render(args[2]))
		return nil
	}
	return nil
}

func pad(s string, w int) string {
	if n := w - lipgloss.Width(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}
