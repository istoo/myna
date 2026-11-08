// Run with Node's test runner and Playwright on the module path; see README.md.
// A Settings row with two states is a switch: role=switch, its state in
// aria-checked, and a press or Enter/Space turns it over. A row with three
// or more states stays a segmented control. Chromium and WebKit; no backend,
// the API is faked.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");

function server(lang, bodies) {
  const settings = { lang, theme: "light", reduceMotion: false, colorblind: false, noBird: false, quitNoAsk: false, noUpdatePill: false };
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:${JSON.stringify(lang)},theme:"light",web:false};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings });
    if (url.pathname === "/api/settings" && route.request().method() === "POST") {
      bodies.push(route.request().postDataJSON());
      return json({ ...settings, ...route.request().postDataJSON() });
    }
    if (url.pathname === "/api/settings") return json(settings);
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true } });
    if (url.pathname === "/api/groups") return json({ groups: [] });
    if (url.pathname === "/api/plugins") return json({ plugins: [] });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    await route.fulfill({ body: await fs.readFile(file), contentType });
  };
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const lang of ["en", "zh"]) {
    test(`${engine} ${lang}: a two-state Settings row is a switch, three states stay a segmented control`, async (t) => {
      const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
      t.after(() => browser.close());
      const page = await (await browser.newContext({ viewport: { width: 1000, height: 800 }, reducedMotion: "reduce" })).newPage();
      page.setDefaultTimeout(6000);
      page.setDefaultNavigationTimeout(60000);
      const bodies = [], errors = [];
      page.on("pageerror", (e) => errors.push(e.message));
      await page.route("**/*", server(lang, bodies));
      await page.goto("http://myna.test/?view=settings&tab=general");
      await page.locator("#reduceMotionSegs [role=switch]").waitFor();
      // every two-state row the settings draw has its switch, its state and its name
      for (const [id, on, name] of [["reduceMotionSegs", "false", "Reduce motion"], ["colorblindSegs", "false", "Colorblind mode"], ["birdSegs", "true", "Loading bird"]]) {
        const sw = page.locator(`#${id} [role=switch]`);
        assert.equal(await sw.count(), 1, `${id}: one switch`);
        assert.equal(await sw.getAttribute("aria-checked"), on, `${id}: its state`);
        assert.equal(await sw.getAttribute("aria-label"), await page.evaluate((k) => t(k), name), `${id}: named`);
      }
      // three or more states are still the segmented control
      assert.equal(await page.locator("#themeSegs [role=switch]").count(), 0, "the theme stays segmented");
      assert.ok(await page.locator("#themeSegs .opt").count() >= 2, "the theme's options");
      assert.ok(await page.locator("#textSizeSegs .opt").count() >= 3, "the text sizes");
      // a press turns it over, and the save says so
      await page.locator("#reduceMotionSegs [role=switch]").click();
      await page.waitForFunction(() => document.querySelector("#reduceMotionSegs [role=switch]")?.getAttribute("aria-checked") === "true");
      assert.equal(bodies.at(-1)?.reduceMotion, true, "the save says reduceMotion is on");
      // so does the keyboard, on the control itself
      await page.locator("#reduceMotionSegs [role=switch]").focus();
      await page.keyboard.press("Space");
      await page.waitForFunction(() => document.querySelector("#reduceMotionSegs [role=switch]")?.getAttribute("aria-checked") === "false");
      assert.equal(bodies.at(-1).reduceMotion, false, "the save says reduceMotion is off");
      assert.deepEqual(errors, []);
      await page.context().close();
    });
  }
}