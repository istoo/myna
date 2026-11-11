// The page's wheel coast runs a moment past the last step (the app's flings,
// 250ms), and a position read between two of its scrolls is not where the
// reader is: on a loaded Linux runner that read sat 5 to 13px from the one
// the click then moved from (issue #13). This waits for the scroller to come
// to rest — unchanged for eight frames and at least 250ms — with a 2s cap, so
// a run that never rests fails its own assertion instead of hanging here.
async function wheelRest(page, sel) {
  await page.evaluate((sel) => new Promise((done) => {
    const v = document.querySelector(sel) || document.scrollingElement;
    const t0 = performance.now();
    let last = v.scrollTop, still = 0;
    const step = (now) => {
      const at = v.scrollTop;
      still = at === last ? still + 1 : 0;
      last = at;
      if ((still >= 8 && now - t0 >= 250) || now - t0 > 2000) done();
      else requestAnimationFrame(step);
    };
    requestAnimationFrame(step);
  }), sel);
}
module.exports = { wheelRest };
