// pagehead.js — the Agents and Providers pages' head in the window's
// sidebar layout: the page's name, a line on where it stands and its own
// actions at the right, in place of the title bar's crumb
// (docs/owl-design.md, 11-11). The other pages have a toolbar of their own
// and keep the crumb. Loaded by navkeys.js, so the tray panel has none; the
// top tabs' layout (a narrow window, owl web on a phone) names the page in
// its tabs and draws none.
// An action moved into a head is the page's own element, its id kept, and
// goes back where it was when the head goes.
// A move takes the focus out of what moves (⌘K's Providers puts it in the
// search): it is given back, the caret where it was.
const home = new Map();
const moving = (e, go) => {
  const f = document.activeElement, had = f && e.contains(f), at = had && f.selectionStart != null ? [f.selectionStart, f.selectionEnd] : null;
  go();
  if (had && document.activeElement !== f) { f.focus({ preventScroll: true }); if (at) f.setSelectionRange(...at); }
};
const put = (into, e) => {
  if (!e) return;
  if (!home.has(e)) home.set(e, [e.parentNode, e.nextSibling]);
  if (e.parentNode !== into) moving(e, () => into.append(e));
};
const putBack = () => {
  for (const [e, [p, next]] of home) if (e.parentNode !== p) moving(e, () => p.insertBefore(e, next?.parentNode === p ? next : null));
  home.clear();
};
const titleOf = (v) => {
  const b = $(`#nav button[data-view="${v}"]`);
  return (b?.querySelector("span[data-t]") || b?.querySelector("span"))?.textContent?.trim() || "";
};
// a count in its words, the number bold: one piece of the line
const num = (n, s) => { const p = t(s, { n: "\u0000" }).split("\u0000"), e = el("span"); e.append(p[0], el("b", "", String(n)), p[1] || ""); return e; };

// today's calls on each agent's card, read while the Agents page is on
// screen, at most every 30s, and only with the Usage page there to read
let today = null, todayAt = 0, todayAsk = null;
function readToday() {
  const ub = $('#nav button[data-view="usage"]');
  if (!ub || ub.hidden || featureHides(ub) || todayAsk || Date.now() - todayAt < 30e3) return;
  let tz = "";
  try { tz = Intl.DateTimeFormat().resolvedOptions().timeZone || ""; } catch {}
  todayAsk = api("usage?period=today" + (tz ? "&tz=" + encodeURIComponent(tz) : ""))
    .then((u) => { today = u; todayAt = Date.now(); }, () => { todayAt = Date.now(); })
    .finally(() => { todayAsk = null; later(); });
}
function todayLines() {
  for (const r of document.querySelectorAll("#agents > .row.agent")) {
    const a = state.agents.find((x) => x.id === r.dataset.id);
    const u = today && a && (today.agents || []).find((g) => g.name === a.name);
    let line = r.querySelector(":scope > .ag-today");
    if (!u || !u.calls) { line?.remove(); continue; }
    if (!line) r.append(line = el("div", "ag-today"));
    const cost = fmtCost(u), key = [u.calls, u.input + u.output, cost].join();
    if (line.dataset.key === key) continue;
    line.dataset.key = key;
    line.replaceChildren(el("span", "", t("Today")), num(fmtN(u.calls), u.calls === 1 ? "{n} call" : "{n} calls"), num(fmtN(u.input + u.output), "{n} tokens"));
    if (cost) line.append(el("span", "cost", "≈" + cost));
  }
}

function status(v) {
  const s = [];
  if (v === "agents" && state?.agents?.length) {
    const lit = el("span", "ph-on");
    lit.append(el("i", "ph-dot"), num(state.agents.filter((a) => a.wired).length, "{n} connected"));
    s.push(lit, num(state.agents.length, "{n} installed"));
  }
  if (v === "providers" && typeof providers !== "undefined" && providers?.providers?.length) {
    const all = providers.providers, on = all.filter((p) => !p.off);
    const lit = el("span", "ph-on");
    lit.append(el("i", "ph-dot"), num(on.length, "{n} on"));
    s.push(lit);
    const signed = on.filter((p) => p.account).length;
    if (signed) s.push(num(signed, "{n} signed in"));
    s.push(num(on.reduce((n, p) => n + (p.models || []).filter((m) => m.on !== false).length, 0), "{n} models"));
  }
  return s;
}

let frame = 0;
const later = () => { if (!frame) frame = requestAnimationFrame(draw); };
function draw() {
  frame = 0;
  const v = typeof view === "string" ? view : "";
  const page = (v === "agents" || v === "providers") && $("#view-" + v);
  const on = sideNav() && !!page && mode === "window";
  for (const h of document.querySelectorAll("main.view > .ph")) if (!on || h.parentElement !== page) h.remove();
  document.body.classList.toggle("ph-on", on);
  // today's calls are the cards', in either layout
  if (v === "agents" && mode === "window") { readToday(); todayLines(); }
  if (!on) return putBack();
  let h = page.querySelector(":scope > .ph");
  if (!h) {
    h = el("header", "ph");
    const words = el("div", "ph-t");
    const name = el("div", "ph-name");
    name.setAttribute("aria-hidden", "true"); // #pageTitle says it
    words.append(name, el("div", "ph-s"));
    h.append(words, el("div", "ph-a"));
    page.prepend(h);
  }
  const name = h.querySelector(".ph-name"), title = titleOf(v);
  if (name.textContent !== title) name.textContent = title;
  const s = h.querySelector(".ph-s"), parts = status(v);
  const text = parts.map((p) => p.textContent).join("|");
  if (s.dataset.text !== text) { s.dataset.text = text; s.replaceChildren(...parts); }
  s.hidden = !parts.length;
  // the page's own actions, beside its name
  const acts = h.querySelector(".ph-a");
  if (v !== "providers") putBack();
  else { put(acts, $("#pvHead")); put(acts, $("#addProvider")); }
}

export function start() {
  // what the head says changes when the page, the layout or a list does
  // at once, not a frame later: a list drawn anew has its head and today's
  // calls before the window's scroll keeping measures it, so a line that
  // came late isn't a second redraw that holds the reader's place
  const watch = new MutationObserver((ms) => {
    if (!ms.some((m) => !m.target.closest?.(".ph"))) return;
    cancelAnimationFrame(frame);
    draw();
  });
  watch.observe(document.body, { attributes: true, attributeFilter: ["class"] });
  watch.observe($("#nav"), { attributes: true, subtree: true, attributeFilter: ["class", "hidden"] });
  for (const id of ["#agents", "#providers", "#offProviders"]) { const e = $(id); if (e) watch.observe(e, { childList: true }); }
  later();
}
