// Run with Node's test runner and Playwright on the module path; see README.md.
// Settings stays where it was clicked (jen_celeus on X): its tabs used to
// be listed under its button at the sidebar's foot, so the foot grew upward
// and the button rose with it, out from under the pointer. Beside the rail
// its gear is at the right end of the status bar under the page, the
// gateway's address at its left, and its tabs are the page's own strip:
// the gear never moves, as Settings opens, as a tab is picked, or as
// another page opens, in a wide window, a narrow one, a short one and one
// shorter still.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(process.env.OWL_ASSETS || path.resolve(__dirname, "../assets"));

function serve(lang) {
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:${JSON.stringify(lang)},theme:"light",web:false,sideGateway:{url:"http://127.0.0.1:3425",running:true,boot:true}};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang, theme: "light" } });
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { url: "http://127.0.0.1:3425", running: true, window: true } });
    if (url.pathname === "/api/groups") return json({ groups: [] });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    await route.fulfill({ body: await fs.readFile(file), contentType });
  };
}

// every frame for a while, the furthest Settings' button got from where it was
const drift = (page, ms) => page.evaluate((ms) => new Promise((done) => {
  const b = document.querySelector("#prefs"), at = b.getBoundingClientRect();
  let most = 0;
  const end = performance.now() + ms;
  const look = () => {
    const r = b.getBoundingClientRect();
    most = Math.max(most, Math.abs(r.top - at.top), Math.abs(r.bottom - at.bottom), Math.abs(r.left - at.left), Math.abs(r.right - at.right));
    if (performance.now() < end) requestAnimationFrame(look); else done(most);
  };
  requestAnimationFrame(look);
}), ms);

const sizes = [
  { name: "wide", width: 1200, height: 760 },
  { name: "narrow", width: 760, height: 760, side: true },
  { name: "short", width: 1200, height: 460 },
  { name: "tiny", width: 1200, height: 340 },
];
const langs = ["en", "zh", "zh-TW", "ja", "de"];

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(`${engine}: Settings stays where it was clicked`, async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    for (const size of sizes) for (const lang of langs) await t.test(`${size.name} ${lang}`, async () => {
      const context = await browser.newContext({ viewport: { width: size.width, height: size.height } });
      if (size.side) await context.addInitScript(() => { try { localStorage.setItem("owl.navLayout", "side"); } catch {} });
      const page = await context.newPage();
      page.setDefaultTimeout(5000);
      page.setDefaultNavigationTimeout(60000);
      const errors = [];
      page.on("pageerror", (e) => errors.push(e.message));
      await page.route("**/*", serve(lang));
      await page.goto("http://owl.test/");
      await page.waitForSelector("#nav button.on");
      assert.ok(await page.evaluate(() => document.body.classList.contains("side-nav")), "the rail shows");
      const prefs = page.locator("#sideStatus #prefs");
      await page.locator("#sideStatus #sideGw").waitFor();
      const before = await prefs.boundingBox();
      // in the status bar at the window's foot, at its right end, the
      // gateway's address before it
      const at = await page.evaluate(() => {
        const bar = document.querySelector("#sideStatus").getBoundingClientRect(), g = document.querySelector("#sideGw").getBoundingClientRect(), p = document.querySelector("#prefs").getBoundingClientRect();
        return { bar: bar.bottom, vh: innerHeight, gw: g.right, pl: p.left, pr: p.right, br: bar.right };
      });
      assert.ok(Math.abs(at.bar - at.vh) < 1 && at.br - at.pr < 16 && at.gw <= at.pl, `the gear at the bar's right, the address before it: ${JSON.stringify(at)}`);

      // clicked, Settings opens and the gear stays under the pointer
      await page.mouse.click(before.x + before.width / 2, before.y + before.height / 2);
      const opening = await drift(page, 700);
      await page.locator("#view-settings:not([hidden])").waitFor();
      await page.locator("#setTabs button").first().waitFor();
      assert.equal(await page.locator(".nav-sub, .nav-s").count(), 0, "no list of tabs opens from it");
      const after = await prefs.boundingBox();
      assert.ok(opening < 1, `the gear moved ${opening}px while Settings opened`);
      assert.ok(Math.abs(after.y - before.y) < 1 && Math.abs(after.x - before.x) < 1, `the gear stays put: ${JSON.stringify(before)} then ${JSON.stringify(after)}`);
      assert.ok(await page.locator("#prefs").evaluate((b) => b.classList.contains("on")), "Settings is open");

      // a tab picked in the page's strip: the gear still where it was
      await page.locator("#setTab-network").click();
      await page.waitForFunction(() => document.querySelector("#setTab-network").classList.contains("on"));
      assert.ok((await drift(page, 300)) < 1, "picking a tab leaves the gear put");
      assert.ok(Math.abs((await prefs.boundingBox()).y - before.y) < 1);

      // another page: the gear never moved
      await page.locator('#nav button[data-view="agents"]').click();
      const closing = await drift(page, 600);
      assert.ok(closing < 1, `the gear moved ${closing}px as another page opened`);
      assert.ok(Math.abs((await prefs.boundingBox()).y - before.y) < 1, "the gear is where it began");
      assert.equal(await page.locator("#prefs").evaluate((b) => b.classList.contains("on")), false);

      assert.deepEqual(errors, []);
      await context.close();
    });
  });
}
