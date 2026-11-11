// pmorder.js — the Plugins page's Discover: which kind of plugin it shows
// and in what order, loaded by plugins.js when the page is first drawn.
// The kinds are the market's sections, each with how many it lists; the
// order is the market's own or npm's downloads last week, most first (a
// plugin npm hasn't counted last). Both are kept for the next visit.
const NAMES = { pv: "Subscriptions", mw: "Gateway middleware", ag: "Agents", tl: "Tools for agents", pg: "Pages" };
const kept = (k) => { try { return localStorage.getItem(k) || ""; } catch { return ""; } };
const keep = (k, v) => { try { localStorage.setItem(k, v); } catch {} };
let kind = kept("owl.pluginKind"), sort = kept("owl.pluginSort");

// lists: kind → its listings, in the market's order. Back: the strip to
// draw above them, and the lists to draw, a kind not picked left empty.
// forAgents: the page opened at its Agents (Install another agent), which
// a pick of another kind would hide.
export function mk(lists, redraw, forAgents) {
  const kinds = Object.keys(NAMES).filter((k) => lists[k]?.length);
  if (!kinds.includes(kind) || forAgents) kind = "";
  const bar = el("div", "pm-ghbar pm-mkbar");
  if (kinds.length > 1) {
    const all = kinds.reduce((n, k) => n + lists[k].length, 0);
    const pick = segs([["", t("All") + " · " + all], ...kinds.map((k) => [k, t(NAMES[k]) + " · " + lists[k].length])], kind, (k) => { kind = k; keep("owl.pluginKind", k); redraw(); });
    pick.classList.add("pm-mkkinds");
    pick.setAttribute("role", "group");
    pick.setAttribute("aria-label", t("Kind of plugin"));
    bar.append(pick);
  }
  const order = segs([["", t("Suggested")], ["dl", t("Most downloaded")]], sort, (s) => { sort = s; keep("owl.pluginSort", s); redraw(); });
  order.classList.add("pm-mksort");
  order.setAttribute("role", "group");
  order.setAttribute("aria-label", t("Order"));
  order.title = t("Most downloaded: by npm's downloads last week");
  bar.append(el("span", "grow"), order);
  const out = {};
  for (const k of Object.keys(lists)) {
    const l = kind && kind !== k ? [] : lists[k];
    out[k] = sort === "dl" ? [...l].sort((a, b) => (b.npm?.weekly || 0) - (a.npm?.weekly || 0)) : l;
  }
  return { bar, lists: out };
}
