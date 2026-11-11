// A Providers list that couldn't be read: a card in the list's place, or a
// line over the list kept from before. Loaded when the page needs it, so
// the core script keeps room for what every page needs.
function card(why) {
  const c = el("div", "empty-state load-error");
  c.setAttribute("role", "alert");
  c.append(el("b", "", t("Couldn't read your providers")), el("span", "why", why));
  const again = el("button", "text primary", t("Retry"));
  again.type = "button";
  again.onclick = retryProviders;
  c.append(again);
  return c;
}
export function drawProvidersError(why, kept) {
  const page = $("#view-providers");
  page.classList.remove("loading");
  page.removeAttribute("aria-busy");
  const host = $("#providersError");
  if (kept) {
    // the list read before stays, under a line saying this read failed
    renderProviders();
    host.replaceChildren(el("span", "load-error-why", t("Refresh failed: {why}", { why })));
    const again = el("button", "text", t("Retry"));
    again.type = "button";
    again.onclick = retryProviders;
    host.append(again);
    host.hidden = false;
    return;
  }
  host.hidden = true;
  host.replaceChildren();
  const list = $("#providers");
  list.hidden = false;
  list.replaceChildren(card(why));
  $("#offHead").hidden = $("#offProviders").hidden = true;
  $("#excluded").replaceChildren();
  $("#pvHead").hidden = true;
  $("#pvNone").hidden = true;
}