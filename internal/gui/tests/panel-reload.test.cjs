// Run with Node's test runner and Playwright on the module path; see README.md.
// The panel's fit is a notice to the window, one at every redraw: when the
// page is reloaded or closed with one in flight, the fetch fails, and with
// no one to tell its rejection was left unhandled — WebKit reported it as a
// page error ("TypeError: Load failed") across the reload, which made
// update-hide red on Linux CI (#24). A reload with the fit held must leave
// the page with no error. No backend: the API is faked here.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const update = { state: "available", current: "0.1.400", latest: "0.1.401", url: "https://github.com/istoo/owl/releases/tag/v0.1.401" };

function serve(held) {
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: 'window.bootPrefs = {lang:"en",theme:"light",web:true};' });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang: "en", theme: "light" } });
    if (url.pathname === "/api/settings") return json({ theme: "light", lang: "en", tray: "panel", currency: "usd" });
    if (url.pathname === "/api/update" || url.pathname === "/api/update/check") return json(update);
    // the fit is where it is slow on a busy machine: in flight at the reload
    if (url.pathname === "/api/window/fit") await held;
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    try { await route.fulfill({ body: await fs.readFile(file), contentType }); } catch { await route.fulfill({ status: 404, body: "" }); }
  };
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(`${engine}: a reload with the panel's fit in flight leaves no page error`, async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    const page = await browser.newPage({ viewport: { width: 420, height: 560 }, reducedMotion: "reduce" });
    page.setDefaultTimeout(5000);
    page.setDefaultNavigationTimeout(60000);
    const errors = [];
    page.on("pageerror", (e) => errors.push(e.message));
    let give;
    const held = new Promise((r) => { give = r; });
    await page.route("**/*", serve(held));
    await page.goto("http://owl.test/?mode=panel");
    await page.locator(".row.agent, #agents .empty-state").first().waitFor();
    await page.waitForTimeout(400);
    await page.reload();
    await page.waitForTimeout(400);
    give();
    await page.waitForTimeout(600);
    assert.deepEqual(errors, []);
  });
}