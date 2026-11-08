// The header's priority+ tabs: pages that don't fit move under More, which
// reaches each; the open page stays in the row. Loaded when the window
// starts, so the core script keeps room for what every page needs.
function navPageButtons() {
  return [...$("#nav").querySelectorAll(":scope > button[data-view]")].filter((b) => !b.hidden && !b.dataset.replaced);
}
export function renderNavOver(mode) {
  const nav = $("#nav"), more = $("#navMore");
  if (!nav || !more) return;
  for (const b of nav.querySelectorAll(":scope > button[data-view]")) b.classList.remove("nav-over");
  more.hidden = true;
  if (mode !== "window" || sideNav()) return;
  if (phoneWeb()) {
    // a phone's browser: the tabs have a row of their own, which must need
    // no sideways scroll
    if (nav.scrollWidth <= nav.clientWidth + 1) return;
    more.hidden = false;
    const btns = navPageButtons();
    for (let i = btns.length - 1; i >= 0 && nav.scrollWidth > nav.clientWidth + 1; i--) {
      if (!btns[i].classList.contains("on")) btns[i].classList.add("nav-over");
    }
    return;
  }
  if (topFits()) return;
  more.hidden = false;
  const btns = navPageButtons();
  for (let i = btns.length - 1; i >= 0 && !topFits(); i--) {
    if (!btns[i].classList.contains("on")) btns[i].classList.add("nav-over");
  }
  // even the open page and More don't fit: fitTop's packed state takes the
  // brand away and shrinks the tabs; the open page is never the one hidden
}
export function openNavMore(view) {
  const more = $("#navMore");
  if (!more || more.hidden) return;
  if (more.classList.contains("open")) return closeProtoMenu();
  const opts = navPageButtons().filter((b) => b.classList.contains("nav-over"))
    .map((b) => ({ v: b.dataset.view, name: (b.querySelector("span[data-t]")?.textContent || b.textContent).trim(), literalName: true }));
  if (!opts.length) return;
  openProtoMenu(more, opts, view, (v) => { if (v && v !== view) show(v); }, t("Pages"), "nav-more-menu", "right");
}