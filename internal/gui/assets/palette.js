// The command palette (⌘K, Ctrl+K). app.js imports this when it is first
// asked for, so it weighs nothing on the core script. It searches the
// pages, Settings' rows, providers, agents and plugin pages; the arrows
// and Enter run what is picked, Escape puts it away. Settings' own search
// does the matching for its rows, so the same words find them there.
export function openPalette() {
  const modal = $("#modal");
  if (!modal.hidden) return;
  const box = el("div", "palette-dialog");
  const head = el("div", "ehead palette-head");
  head.append(el("b", "", t("Commands")));
  const q = el("input", "palette-q");
  q.type = "search";
  q.autocomplete = "off";
  q.spellcheck = false;
  q.placeholder = t("Search pages, settings, providers and agents");
  q.setAttribute("aria-label", q.placeholder);
  q.setAttribute("role", "combobox");
  q.setAttribute("aria-expanded", "true");
  q.setAttribute("aria-controls", "paletteList");
  q.setAttribute("aria-autocomplete", "list");
  const list = el("div", "palette-list");
  list.id = "paletteList";
  list.setAttribute("role", "listbox");
  list.setAttribute("aria-label", t("Commands"));
  box.append(head, q, list);

  const entries = () => {
    const out = [], seen = new Set();
    for (const b of document.querySelectorAll("#nav button[data-view]")) {
      if (b.hidden || b.dataset.replaced || seen.has(b.dataset.view)) continue;
      seen.add(b.dataset.view);
      const name = (b.querySelector("span[data-t]") || b.querySelector("span"))?.textContent?.trim() || b.dataset.view;
      out.push({ name, kind: "Page", run: () => show(b.dataset.view) });
    }
    out.push({ name: t("Settings"), kind: "Page", run: () => show("settings") });
    if (providers) {
      for (const p of providers.providers) {
        out.push({ name: p.name, kind: "Provider", words: [p.id, p.host || "", ...(p.models || []).slice(0, 12).map((m) => m.id)],
          run: () => openProvider(p.id) });
      }
    }
    if (state) {
      for (const a of state.agents) {
        out.push({ name: a.name, kind: "Agent", words: [a.id, a.path || ""], run: () => show("agents").then(() => {
          const r = document.querySelector(`#agents .row.agent[data-id="${CSS.escape(a.id)}"]`);
          if (!r) return;
          r.focus({ preventScroll: true });
          r.scrollIntoView({ block: "center" });
        }) });
      }
    }
    for (const v of owl.pages()) {
      if (seen.has(v)) continue;
      const part = owl.page(v);
      if (!part?.title) continue;
      seen.add(v);
      out.push({ name: t(part.title), kind: "Plugin page", run: () => show(v) });
    }
    // Settings' rows carry their section and part in the words they match
    for (const r of document.querySelectorAll("#view-settings .row.pref")) {
      const name = r.querySelector(".name")?.textContent?.trim();
      // a row hidden in its own part (from another OS, or a plugin off);
      // the Settings view being closed is not the row's doing
      const hid = r.closest("[hidden]");
      if (!name || (hid && hid.id !== "view-settings") || featureHides(r)) continue;
      out.push({ name, kind: "Setting", words: [r.textContent], run: () => show("settings").then(() => {
        const search = $("#setQ");
        if (!search) return;
        search.value = name;
        searchSettings(name);
        requestAnimationFrame(() => {
          const hit = [...document.querySelectorAll("#view-settings .row.pref")].find((x) => x.textContent.includes(name));
          hit?.scrollIntoView({ block: "center" });
        });
      }) });
    }
    return out;
  };
  let all = entries();
  // the providers' list may not have been read yet (the page is another):
  // read it, and let the results in when it is
  if (!providers) loadProviders().then(() => { all = entries(); draw(); }, () => {});
  let items = [], at = 0;
  const draw = () => {
    const words = q.value.trim().toLowerCase().split(/\s+/).filter(Boolean);
    items = all.filter((it) => words.every((w) => (it.name + " " + (it.words || []).join(" ")).toLowerCase().includes(w))).slice(0, 40);
    at = 0;
    list.replaceChildren();
    items.forEach((it, i) => {
      const b = el("button", "palette-item");
      b.type = "button";
      b.id = "paletteItem" + i;
      b.setAttribute("role", "option");
      b.append(el("span", "palette-name", it.name), el("span", "palette-kind", t(it.kind)));
      b.onmouseenter = () => { at = i; paint(); };
      b.onclick = (e) => { e.stopPropagation(); go(it); };
      list.append(b);
    });
    if (!items.length) list.append(el("div", "palette-none", t("No matches")));
    paint();
  };
  const paint = () => {
    [...list.children].forEach((b, i) => {
      b.classList.toggle("on", i === at);
      if (b.getAttribute("role") === "option") b.setAttribute("aria-selected", String(i === at));
    });
    q.setAttribute("aria-activedescendant", items.length ? "paletteItem" + at : "");
    list.children[at]?.scrollIntoView({ block: "nearest" });
  };
  const go = (it) => {
    if (!it) return;
    // the page opens once the dialog has gone: show() skips closing a
    // dialog already on its way out, and one left open takes "/" with it
    closeModal().then(() => it.run());
  };
  q.oninput = draw;
  q.onkeydown = (e) => {
    if (e.key === "ArrowDown" || (e.ctrlKey && e.key === "n")) { e.preventDefault(); at = Math.min(items.length - 1, at + 1); paint(); }
    else if (e.key === "ArrowUp" || (e.ctrlKey && e.key === "p")) { e.preventDefault(); at = Math.max(0, at - 1); paint(); }
    else if (e.key === "Home") { e.preventDefault(); at = 0; paint(); }
    else if (e.key === "End") { e.preventDefault(); at = Math.max(0, items.length - 1); paint(); }
    else if (e.key === "Enter") { e.preventDefault(); go(items[at]); }
  };
  draw();
  openModal(box);
}