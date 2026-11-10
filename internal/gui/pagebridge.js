// The page's side of a plugin page's messages to owl's window
// (docs/subsystems/plugin-runtime.md, P4). A plugin's page loads it with
// <script src="_owl/bridge.js"></script>; it is served beside every
// page's own files.
//
//   await owl.ready            the window's theme, once it has answered
//   owl.theme                  { dark, lang, tokens: { bg, fg, … } }
//   owl.onTheme(fn)            fn(theme) on each change of it
//   await owl.options.get()    the plugin's own options (plugins.json)
//   await owl.options.set(o)   sets them, and answers with them
//   await owl.read(name, p)    a read its package.json's
//                                 owl.permissions declares
//   owl.context                where a panel is (P5): { provider: {
//                                 id, name } } in a provider's editor,
//                                 else {}; set once owl.ready is
//   owl.service               the address of the page's local service
//                                 (package.json's owl.page.service)
//                                 through owl, set once owl.ready
//                                 is, on a page a browser opened through
//                                 owl web; "" where 127.0.0.1 is the
//                                 service's own computer (the app's
//                                 window) or owl is older
//   owl.onAction(fn)           fn({ id, provider | agent: { id, name } })
//                                 when one of its owl.actions is picked
//                                 in a row's menu (P6); one picked before
//                                 fn was given waits for it
//   owl.visible               false while the reader is on another of
//                                 the window's pages
//   owl.onVisibility(fn)      fn(visible) as the reader leaves the
//                                 page and comes back, to pause what only
//                                 matters while it is seen; a page left
//                                 10 minutes is let go, and loads again
//                                 when next opened, unless package.json
//                                 says owl.page.keepAlive
//
// A panel (owl.panels) is as tall as what it draws: the window is told
// its height as it changes.
//
// What the page leaves unstyled takes owl's look: the bridge adds a
// stylesheet of no weight (:where()) for the font, colours, headings,
// buttons, fields and tables, and the theme's colours are the page's
// --owl-<token> variables.
//
// The page can't reach the network or owl's API itself: everything goes
// through the window, which answers only what the package declared. An
// error the page doesn't catch shows the window's "This page stopped
// working" in its place.
(() => {
  const waiting = new Map();
  let next = 0;
  const themed = [];
  const watched = [];
  let acting = null;
  const acted = [];
  function call(method, params) {
    const id = ++next;
    return new Promise((resolve, reject) => {
      waiting.set(id, { resolve, reject });
      parent.postMessage({ owl: 1, id, method, params }, "*");
    });
  }
  function paint(theme) {
    const root = document.documentElement;
    root.dataset.theme = theme.dark ? "dark" : "light";
    // the frame's canvas matches the window's, not a white sheet in dark
    root.style.colorScheme = theme.dark ? "dark" : "light";
    root.lang = theme.lang || "en";
    for (const [k, v] of Object.entries(theme.tokens || {})) root.style.setProperty("--owl-" + k, v);
  }
  addEventListener("message", (e) => {
    if (e.source !== parent) return;
    const m = e.data;
    if (!m || m.owl !== 1) return;
    if (m.event === "theme") {
      api.theme = m.data;
      paint(m.data);
      for (const fn of themed) { try { fn(m.data); } catch (err) { report(err); } }
      return;
    }
    if (m.event === "visibility") {
      api.visible = !!m.data?.visible;
      for (const fn of watched) { try { fn(api.visible); } catch (err) { report(err); } }
      return;
    }
    if (m.event === "action") {
      if (acting) act(m.data); else acted.push(m.data);
      return;
    }
    const w = waiting.get(m.id);
    if (!w) return;
    waiting.delete(m.id);
    if (m.error) w.reject(new Error(m.error)); else w.resolve(m.result);
  });
  function report(err) {
    parent.postMessage({ owl: 1, method: "error", params: { message: String(err?.message || err) } }, "*");
  }
  // a script's error reaches a sandboxed page only as "Script error." (its
  // files are another origin's to it); the window then says "The page hit
  // an error" in the reader's language, and the console has the rest
  //
  // The size watch below can be told its own "ResizeObserver loop
  // completed with undelivered notifications" when the window resizes the
  // frame in the same frame of drawing; that is no error of the page's,
  // and WebKit hands it to a sandboxed page as a bare "Script error." with
  // no file, so one that comes while the watch has just run is let go
  let measuring = false;
  addEventListener("error", (e) => {
    if (/ResizeObserver loop/.test(e.message) || (measuring && !e.error && !e.filename)) { e.preventDefault(); return; }
    report(e.error || (e.message === "Script error." ? "" : e.message));
  });
  addEventListener("unhandledrejection", (e) => report(e.reason));
  function act(data) {
    try { Promise.resolve(acting(data)).catch(report); } catch (err) { report(err); }
  }
  const api = {
    theme: null,
    context: {},
    service: "",
    visible: true,
    onTheme(fn) { themed.push(fn); },
    onVisibility(fn) { watched.push(fn); },
    onAction(fn) {
      acting = fn;
      for (const d of acted.splice(0)) act(d);
    },
    options: {
      get: () => call("options.get"),
      set: (options) => call("options.set", { options }),
    },
    read: (name, params) => call("read", { name, params }),
  };
  api.ready = call("hello").then(({ context, service, visible, ...theme }) => {
    api.theme = theme;
    if (visible === false) api.visible = false;
    api.context = context || {};
    api.service = "";
    try { if (service) api.service = new URL(service, location.href).href; } catch { /* none */ }
    paint(theme);
    return theme;
  });
  // how tall the page draws, for a panel's frame; a page's frame fills its
  // view, and the window doesn't heed it there
  let told = -1;
  const size = () => {
    const h = Math.ceil(document.documentElement.getBoundingClientRect().height);
    if (h === told) return;
    told = h;
    parent.postMessage({ owl: 1, method: "size", params: { height: h } }, "*");
  };
  const watch = () => {
    new ResizeObserver(() => {
      if (!measuring) { measuring = true; setTimeout(() => { measuring = false; }); }
      size();
    }).observe(document.documentElement);
    size();
  };
  if (document.readyState === "loading") addEventListener("DOMContentLoaded", watch); else watch();
  // owl's look for what the page leaves unstyled: its font, colours,
  // headings, buttons, fields and tables, so a page with no stylesheet of
  // its own reads as part of the window, not a browser's default sheet.
  // Every rule is :where(), of no weight: any rule of the page's wins.
  const base = document.createElement("style");
  base.dataset.owl = "base";
  base.textContent = `
:where(html) { background: var(--owl-bg, Canvas); color: var(--owl-fg, CanvasText); font: 13px/1.45 var(--owl-font, -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif); -webkit-font-smoothing: antialiased; }
:where(body) { margin: 0; padding: 20px 24px; }
:where(h1, h2, h3, h4) { font-weight: 600; margin: 0 0 8px; letter-spacing: -.01em; }
:where(h1) { font-size: 20px; } :where(h2) { font-size: 15px; } :where(h3, h4) { font-size: 13px; }
:where(p) { margin: 0 0 8px; }
:where(small, .muted) { color: var(--owl-muted, GrayText); }
:where(a) { color: var(--owl-accent, LinkText); text-decoration: none; }
:where(a:hover) { text-decoration: underline; }
:where(button, input, textarea, select) { font: inherit; color: inherit; }
:where(button) { background: var(--owl-card, ButtonFace); border: 1px solid var(--owl-line, ButtonBorder); border-radius: 6px; padding: 3px 10px; }
:where(button:hover:not(:disabled)) { border-color: var(--owl-muted, ButtonBorder); }
:where(button[aria-pressed="true"], button[aria-selected="true"]) { background: var(--owl-accent-soft, Highlight); border-color: transparent; color: var(--owl-accent, HighlightText); }
:where(button:disabled) { opacity: .5; }
:where(input, textarea, select) { background: var(--owl-card, Field); border: 1px solid var(--owl-line, ButtonBorder); border-radius: 6px; padding: 4px 8px; }
:where(:focus-visible) { outline: 2px solid var(--owl-accent, Highlight); outline-offset: 1px; }
:where(table) { border-collapse: collapse; width: 100%; }
:where(th) { text-align: left; font-weight: 500; font-size: 12px; color: var(--owl-muted, GrayText); }
:where(th, td) { padding: 6px 8px; border-bottom: 1px solid var(--owl-line, ButtonBorder); }
:where(code, kbd, pre) { font-family: var(--owl-code, ui-monospace, Menlo, monospace); font-size: 12px; }
:where(pre) { background: var(--owl-card, Canvas); border: 1px solid var(--owl-line, ButtonBorder); border-radius: 8px; padding: 8px 10px; overflow: auto; }
:where(hr) { border: 0; border-top: 1px solid var(--owl-line, ButtonBorder); margin: 12px 0; }
:where(::selection) { background: var(--owl-accent-soft, Highlight); }
`;
  (document.head || document.documentElement).prepend(base);
  window.owl = api;
})();
