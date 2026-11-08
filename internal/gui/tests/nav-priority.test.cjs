// Run with Node's test runner and Playwright on the module path; see README.md.
// A narrow window keeps every page reachable: the tabs that don't fit move
// under the More button at the row's end (priority+, app.js renderNavOver),
// which opens the app's own menu with the arrows, Enter and Escape. Wide
// enough for every tab, More is gone and the row is as it was.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const LANGS = ["en", "zh"];

function serve(lang, web) {
  const state = { agents: [], profiles: [], settings: { lang, theme: "light" } };
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:${!!web}};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json(state);
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true } });
    if (url.pathname === "/api/groups") return json({ groups: [] });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    await route.fulfill({ body: await fs.readFile(file), contentType });
  };
}

async function open(engine, t, lang, width, web) {
  const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
  t.after(() => browser.close());
  const page = await (await browser.newContext({ viewport: { width, height: 760 }, reducedMotion: "reduce" })).newPage();
  page.setDefaultTimeout(5000);
  page.setDefaultNavigationTimeout(60000);
  const errors = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.route("**/*", serve(lang, web));
  await page.goto("http://myna.test/");
  await page.waitForSelector("#nav button.on");
  await page.waitForFunction(() => !document.querySelector("#view-agents").classList.contains("loading"));
  return { page, errors };
}

const visiblePages = (page) => page.evaluate(() => [...document.querySelectorAll("#nav > button[data-view]")]
  .filter((b) => !b.hidden && !b.dataset.replaced && !b.classList.contains("nav-over")).map((b) => b.dataset.view));
const overPages = (page) => page.evaluate(() => [...document.querySelectorAll("#nav > button[data-view].nav-over")].map((b) => b.dataset.view));

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const lang of LANGS) {
    test(`${engine} ${lang}: 390px keeps every page, the ones that don't fit under More`, async (t) => {
      const { page, errors } = await open(engine, t, lang, 390, false);
      const more = page.locator("#navMore");
      assert.equal(await more.isVisible(), true, "More is not in the row");
      assert.equal(await more.getAttribute("aria-haspopup"), "menu");
      const visible = await visiblePages(page), over = await overPages(page);
      assert.ok(over.length, "every tab fit at 390px: nothing went under More");
      assert.ok(!over.includes("agents"), "the open page stayed in the row");
      // no sideways scroll left to reach a page
      assert.equal(await page.evaluate(() => { const n = document.querySelector("#nav"); return n.scrollWidth <= n.clientWidth + 1; }), true);

      // a click opens the menu with every page that didn't fit
      await more.click();
      const menu = page.locator(".proto-menu.nav-more-menu");
      await menu.waitFor();
      const shown = await menu.locator(".pm-item .pm-name").allTextContents();
      assert.equal(shown.length, over.length, `the menu named ${shown.length} of ${over.length}`);
      // Escape puts it away
      await page.keyboard.press("Escape");
      await menu.waitFor({ state: "detached" });
      // ArrowDown on More opens it too, and the arrows and Enter pick
      await more.focus();
      await page.keyboard.press("ArrowDown");
      await menu.waitFor();
      const items = await menu.locator(".pm-item").count();
      assert.ok(items > 1);
      await page.keyboard.press("ArrowDown");
      await page.keyboard.press("Enter");
      await page.waitForFunction(() => document.querySelector("#nav button.on")?.dataset.view !== "agents");
      assert.deepEqual(errors, []);
    });

    test(`${engine} ${lang}: wide enough for every tab, More is gone`, async (t) => {
      const { page, errors } = await open(engine, t, lang, 1280, false);
      assert.equal(await page.locator("#navMore").isHidden(), true);
      assert.deepEqual(await overPages(page), []);
      const pages = await page.evaluate(() => [...document.querySelectorAll("#nav > button[data-view]")].filter((b) => !b.hidden && !b.dataset.replaced).length);
      assert.ok(pages >= 6);
      assert.deepEqual(errors, []);
    });

    test(`${engine} ${lang}: a phone's browser row uses More too, not a sideways scroll`, async (t) => {
      const { page, errors } = await open(engine, t, lang, 390, true);
      await page.waitForFunction(() => document.querySelector("#navMore") && !document.querySelector("#navMore").hidden);
      assert.equal(await page.evaluate(() => { const n = document.querySelector("#nav"); return n.scrollWidth <= n.clientWidth + 1; }), true);
      assert.ok((await overPages(page)).length > 0);
      assert.deepEqual(errors, []);
    });
  }
}