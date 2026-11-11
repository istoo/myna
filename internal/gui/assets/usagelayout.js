// usagelayout.js: where the packed Usage cards stand as one is dragged
// (dragCards' layout, app.js usageHandle). It reads app.js's packing
// (usagePlace, usagePacked, USAGE_GAP), and is imported once the window is up,
// so the core scripts carry none of it.
export function usageLayout(rects) {
  const list = $("#subscriptionUsage");
  const n = usagePacked?.n;
  if (!n || !list?.classList.contains("packed")) return null;
  // each card's own height, as packUsage measures the cards it draws anew
  // on the drop: the last card of a column is stretched to the tallest's
  // end, and packed on that height the others landed 18 to 98px from where
  // the drag showed them (the Mac's fonts, where the columns end apart).
  // Measured with the list held at its height, so the page above isn't
  // scrolled back by a list made short for a moment.
  const cards = [...list.children], kept = cards.map((c) => [c.style.gridArea, c.style.minHeight]), listMin = list.style.minHeight;
  list.style.minHeight = list.getBoundingClientRect().height + "px";
  for (const c of cards) { c.style.gridArea = "auto"; c.style.minHeight = ""; }
  const hs = cards.map((c) => c.getBoundingClientRect().height);
  cards.forEach((c, i) => { c.style.gridArea = kept[i][0]; c.style.minHeight = kept[i][1]; });
  list.style.minHeight = listMin;
  const now = usagePlace(n, hs).at;
  const left = [];
  now.forEach(([c], i) => { left[c] ??= rects[i].left; });
  if (left.length < n || left.includes(undefined)) return null;
  const top0 = rects[0].top - now[0][1];
  return (order) => {
    const at = [], { at: placed, ends } = usagePlace(n, order.map((i) => hs[i]));
    // the last card of a column runs to the tallest one's end, as packUsage
    // lays it once dropped: shown so while dragged, it doesn't grow on the drop
    const tall = Math.max(...ends) - USAGE_GAP, last = new Map(placed.map(([c], k) => [c, k]));
    placed.forEach(([c, top, h], k) => { at[order[k]] = { left: left[c], top: top0 + top, height: last.get(c) === k ? tall - top : h }; });
    return at;
  };
}
