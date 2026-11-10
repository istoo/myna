// Settings › About's rows, loaded when that tab is first drawn so they
// don't weigh on the window's core script (about.js's budget test). The
// window's own helpers come in through ctx: this file adds no state.
export function aboutRows(ctx) {
  // el, $, t and the window's other helpers are its own declarations,
  // reached as globals; the values app.js keeps in let or const come here
  const { about, s, keep, web, setTab, UPDATE_EVERY, GITHUB_SVG, gatewayMode } = ctx;
  const row = (name, sub, value, ...tools) => {
    const r = el("div", "row pref");
    const who = el("div", "who");
    who.append(el("div", "name", name));
    if (sub) who.append(el("div", "sub", sub));
    const val = el("div", "val");
    if (value) val.append(el("code", "", value));
    val.append(...tools);
    r.append(who, val);
    about.append(r);
    return r;
  };
  renderUpdate(row(t("Version"), "", s.version));
  // what changed in this version, and in one waiting; a build from source
  // has no notes
  if (/^v?\d+\.\d+\.\d+$/.test(s.version || "")) {
    const notes = el("button", "text", t("Open"));
    notes.onclick = () => openWhatsNew(notes);
    row(t("What's new"), t("The release notes since the last update"), "", notes).classList.add("whatsnew-row");
  }
  // the header's Update pill, kept away for good or for one version; the
  // version row above still says what is out and offers it
  const pill = row(t("Update reminder"), t("Shows in the title bar when a newer owl is out"),
    "", settingSwitch(t("Update reminder"), !s.noUpdatePill, (on) => savePrefs({ ...keep, noUpdatePill: !on }).then(renderUpdateBadge)));
  pill.classList.add("update-pill-row");
  if (s.updateSkip && !s.noUpdatePill) {
    api(updatePath("update")).then((u) => {
      if (!pill.isConnected || !u || u.latest !== s.updateSkip) return;
      pill.querySelector(".sub").textContent = t("Hidden for {v} until a newer version is out", { v: u.latest });
      const back = el("button", "text", t("Show again"));
      back.onclick = () => skipUpdate("");
      pill.querySelector(".val").prepend(back);
    }, () => {});
  }
  // whether owl asks for a newer version by itself, and how often;
  // off, only the version row's Check asks (#472)
  row(t("Automatic updates"), t("Checks for a newer owl by itself"),
    "", settingSwitch(t("Automatic updates"), !s.noAutoUpdate, (on) => savePrefs({ ...keep, noAutoUpdate: !on }))).classList.add("update-auto-row");
  // kept in place while off, dimmed, its height the same (a row or a line
  // taken away would shorten the page under the click), for when they are
  // turned on again
  row(t("Check every"), s.noAutoUpdate ? t("While automatic updates are on") : t("How often owl looks for a newer version"), "",
    segs(UPDATE_EVERY.map((m) => [m, m < 60 ? t("{n} min", { n: m }) : t("{n} h", { n: m / 60 })]), s.updateEvery || 360,
      (updateEvery) => savePrefs({ ...keep, updateEvery }))).classList.add("update-every-row", ...(s.noAutoUpdate ? ["off"] : []));
  // which versions an update offers (ssharpcc on X): every one as it comes
  // out, or Stable, one a week and an urgent one at once, chosen by the
  // update feed. It holds for the version row's Check too, so it stays lit
  // with automatic updates off.
  row(t("Update channel"), s.updateChannel === "stable"
    ? t("One version a week, out for three days first, and urgent fixes at once. What's new lists every change since yours")
    : t("Every version as soon as it is out"), "",
    segs([["every", t("Every version")], ["stable", t("Stable")]], s.updateChannel === "stable" ? "stable" : "every",
      (v) => savePrefs({ ...keep, updateChannel: v === "stable" ? "stable" : "" }))).classList.add("update-channel-row");
  // whether a newer version found is downloaded at once, or only said to
  // be out and downloaded on a click (#2296): a metered network or a
  // proxy billed by traffic. It holds for the version row's Check too.
  row(t("Download updates automatically"), s.noAutoDownload
    ? t("Off: a newer version is only shown, with its size; it downloads when you click Download update. For metered networks and proxies billed by traffic")
    : t("Downloads a newer version as soon as it is found. Turn off on a metered network or a proxy billed by traffic"), "",
    settingSwitch(t("Download updates automatically"), !s.noAutoDownload, (on) => savePrefs({ ...keep, noAutoDownload: !on }))).classList.add("update-download-row");
  row(t("Restart automatically"), s.noAutoUpdate ? t("While automatic updates are on") : t("Once an update is downloaded, restarts into it without cutting off an agent's request"), "",
    settingSwitch(t("Restart automatically"), !!s.autoRestart, (on) => savePrefs({ ...keep, autoRestart: on }))).classList.add("update-restart-row", ...(s.noAutoUpdate ? ["off"] : []));
  renderUpdateMirror(row(t("Download source"), "", ""), s);
  const open = el("button", "text", t("Open"));
  open.onclick = () => api("settings/reveal", {}).catch(statusErr);
  // portable (a data folder beside owl, #508): everything is in there
  row(t("Config folder"), s.portable ? t("Portable: everything owl keeps, in the data folder beside it") : t("providers, profiles and these settings"), s.dir, copyBtn(s.dir, t("Path")), open);
  // app.log (applog); owl web writes none
  if (s.log) {
    const o = el("button", "text", t("Open"));
    o.onclick = () => api("settings/log/reveal", {}).catch(statusErr);
    row(t("Log"), t("What owl has logged, to send with a bug report"), s.log, copyBtn(s.log, t("Path")), o).id = "logRow";
  }
  renderCLI(row(t("Command line"), t("Checking your shells…"), ""));
  // served to other machines (owl web), the one they reach (noahhrreion)
  const gw = web && s.publicURL || s.gateway;
  row(t("Gateway URL"), t("the address every agent is pointed at"), gw, copyBtn(gw, t("Gateway URL")));
  // Claude Code elsewhere (Docker): the host runs this curl
  if (web) {
    const cmd = "curl -fsS -m 1 -H 'Authorization: Bearer <gateway key>' --data-binary @- " + gw + "/v1/owl/statusline || true";
    row(t("Status line on another computer"), t("statusLine's command in Claude Code's settings.json there; a read-only gateway key is enough"), cmd, copyBtn(cmd, t("Command"))).id = "statuslineRemoteRow";
  }
  // OWL_PUBLIC_URL, where other machines reach it (#1551)
  if (s.publicURL && gw !== s.publicURL) row(t("Public address"), t("OWL_PUBLIC_URL: where other computers reach the gateway, with a gateway key"), s.publicURL, copyBtn(s.publicURL, t("Public address"))).id = "publicURLRow";
  const site = el("button", "text", t("Open"));
  site.onclick = () => api("open", { url: "https://github.com/istoo/owl" }).catch(() => {});
  row(t("Website"), t("downloads, docs and what's new"), "github.com/istoo/owl", site);
  const lic = el("button", "text", t("View"));
  lic.onclick = showNotices;
  row(t("Open-source licenses"), t("the notices of the code and data owl is built with"), "", lic);
  // issues and the code are on GitHub
  const repo = el("button", "github");
  repo.innerHTML = GITHUB_SVG;
  repo.append(el("span", "", "GitHub"));
  repo.title = "github.com/istoo/owl";
  repo.onclick = () => api("open", { url: "https://github.com/istoo/owl" }).catch(() => {});
  row(t("Community"), t("questions, ideas and feedback, on GitHub"), "", repo);
  // owl uninstall, from the app (Jevik_AI on X deleted owl's folders
  // by hand and Codex kept pointing at a gateway that was gone). Never in a
  // browser: owl web would be taken off the computer serving the page.
  if (!web) {
    const go = el("button", "text danger", t("Uninstall…"));
    go.onclick = () => uninstallOwl(go);
    row(t("Uninstall owl"), t("Disconnects every agent, putting back what each had, then removes owl's service and files"), "", go).id = "uninstallRow";
  }
  // gateway mode leaves out what is written into this computer's agents'
  // files, and the alerts its desktop would show
  for (const r of [$("#plainNamesSegs").parentElement, $("#usageAlertRow"), $("#balanceAlertRow"), $("#resetReminderRow"), $("#turnDoneRow")]) r.hidden = gatewayMode;
  // the parts' tabs, one gone whose rows are all hidden here
  setSetTab(setTab);
}