// A press on a control in a sheet, once what is around it has settled.
//
// WebKitGTK lays a sheet's body out 10px too wide until something in it is
// first focused: the focus a press brings comes with the press, and the
// re-flow that follows re-wraps the row, the control moving a line down
// between the press and its release, so the click lands on another field
// (#13). Focus it first, wait for it to hold still, then press.
async function press(locator) {
  await locator.waitFor({ state: "visible" });
  await locator.focus();
  await locator.evaluate((el) => new Promise((done) => {
    const at = () => { const r = el.getBoundingClientRect(); return r.x + "," + r.y; };
    let last = at(), still = 0;
    const step = () => {
      const now = at();
      still = now === last ? still + 1 : 0;
      last = now;
      if (still >= 3 || !el.isConnected) done(); else requestAnimationFrame(step);
    };
    requestAnimationFrame(step);
  }));
  await locator.click();
}

module.exports = { press };
