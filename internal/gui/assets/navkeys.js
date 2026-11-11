// navkeys.js — the window's page keys, loaded by app.js once the window is
// up (the tray panel loads none of it). ⌘1 to ⌘9 (Ctrl off the Mac) open the
// pages in the order the sidebar lists them, each page's tooltip saying its
// own once the pointer is on it, and ↑/↓ (Home/End, or a letter) walk the
// list, Enter opening. A browser tab — owl web — keeps the digits for its
// own tabs and shows no shortcut.
const mac = /^Mac/.test(navigator.platform);
const isWeb = () => !!window.bootPrefs?.web;
// the pages the sidebar lists: not one switched off, plugin-replaced or hidden
const pageKeys = () => [...document.querySelectorAll("#nav button[data-view]")]
  .filter((b) => !b.hidden && !b.dataset.replaced && !featureHides(b));
const pageMod = (e) => (mac ? e.metaKey && !e.ctrlKey : e.ctrlKey && !e.metaKey) && !e.shiftKey && !e.altKey;
const nameOf = (b) => (b.querySelector("span[data-t]") || b.querySelector("span"))?.textContent?.trim() || "";

export function start() {
  const nav = $("#nav");
  // each page's shortcut, in its tooltip when the pointer comes to it
  if (!isWeb()) nav.addEventListener("pointerover", (e) => {
    const b = e.target.closest?.("button[data-view]");
    if (!b) return;
    const i = pageKeys().indexOf(b);
    if (i >= 0 && i < 9) b.title = `${nameOf(b)} (${mac ? "⌘" : "Ctrl+"}${i + 1})`;
    else b.removeAttribute("title");
  });
  // the pages from the keyboard too: ↑/↓ walk them, Home/End the ends, a
  // letter the next page beginning with it, Enter opens (the button's own)
  nav.addEventListener("keydown", (e) => {
    if (e.metaKey || e.ctrlKey || e.altKey || e.shiftKey) return;
    const bs = pageKeys().filter((b) => b.offsetParent);
    if (!bs.length) return;
    const i = bs.indexOf(document.activeElement);
    const go = (j) => { e.preventDefault(); bs[j].focus(); };
    if (e.key === "ArrowDown") return go(i < 0 ? 0 : Math.min(bs.length - 1, i + 1));
    if (e.key === "ArrowUp") return go(i < 0 ? bs.length - 1 : Math.max(0, i - 1));
    if (e.key === "Home") return go(0);
    if (e.key === "End") return go(bs.length - 1);
    if (e.key.length === 1 && e.key.trim()) {
      const k = e.key.toLowerCase();
      const from = i < 0 ? 0 : i + 1;
      for (let j = 0; j < bs.length; j++) {
        const n = (from + j) % bs.length;
        if (nameOf(bs[n]).toLowerCase().startsWith(k)) return go(n);
      }
    }
  });
  // a page's key opens it, the digits counting the pages the sidebar lists
  addEventListener("keydown", (e) => {
    if (isWeb() || !pageMod(e) || !/^Digit[1-9]$/.test(e.code) || !$("#modal").hidden) return;
    const b = pageKeys()[+e.code.slice(5) - 1];
    if (b) { e.preventDefault(); show(b.dataset.view); }
  });
}