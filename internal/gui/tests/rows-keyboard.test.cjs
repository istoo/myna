// Run with Node's test runner and Playwright on the module path; see README.md.
// A provider's row and an agent's row are in Tab's reach and do with Enter or
// Space what a click does; their drag grips say they reorder, in every
// language, on Chromium and WebKit. No backend; the API is faked here.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");
const { navTo } = require("./reader.cjs");

const assets = path.resolve(__dirname, "../assets");
const LANGS = ["en", "zh"];

function server(lang) {
  const state = {
    agents: [{ id: "claude", name: "Claude Code", icon: "claude-color", path: "~/.claude/settings.json", fields: [], wired: false }],
    profiles: [], settings: { lang, theme: "light", guideHidden: true },
  };
  const providers = { providers: [{ id: "relay", name: "Relay", icon: "generic", chat: "https://relay.example.com/v1", responses: "", anthropic: "", catalog: "", models: [], agents: [], fallback: [], keyList: [], key: { set: true, masked: "sk-…one" }, balanceToken: { takes: false, set: false } }], presets: [], excluded: [], gateway: { running: true, window: true } };
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:false};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json(state);
    if (url.pathname === "/api/providers") return json(providers);
    if (url.pathname === "/api/groups") return json({ groups: [] });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    await route.fulfill({ body: await fs.readFile(file), contentType });
  };
}

const tr = (page, s) => page.evaluate((k) => t(k), s);

async function open(engine, t, lang) {
  const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
  t.after(() => browser.close());
  const page = await (await browser.newContext({ viewport: { width: 1000, height: 800 }, reducedMotion: "reduce" })).newPage();
  page.setDefaultTimeout(5000);
  page.setDefaultNavigationTimeout(60000);
  page.errors = [];
  page.on("pageerror", (e) => page.errors.push(e.message));
  await page.route("**/*", server(lang));
  await page.goto("http://owl.test/?view=providers");
  await page.locator("#providers .row.provider:not(.pv-sk-row)").first().waitFor();
  return page;
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const lang of LANGS) {
    test(`${engine} ${lang}: a provider's row opens on Enter and Space`, async (t) => {
      const page = await open(engine, t, lang);
      const row = page.locator("#providers .row.provider:not(.pv-sk-row)").first();
      assert.equal(await row.getAttribute("tabindex"), "0", "in Tab's reach");
      assert.equal(await row.getAttribute("role"), "button");
      assert.equal(await row.getAttribute("aria-label"), await tr(page, "Open {name}").then((s) => s.replace("{name}", "Relay")));
      await row.focus();
      await page.keyboard.press("Enter");
      await page.locator(".editor").waitFor();
      await page.keyboard.press("Escape");
      await page.locator(".editor").waitFor({ state: "detached" });
      await row.focus();
      await page.keyboard.press(" ");
      await page.locator(".editor").waitFor();
      await page.keyboard.press("Escape");
      await page.locator(".editor").waitFor({ state: "detached" });
      assert.deepEqual(page.errors, []);
      await page.context().close();
    });

    test(`${engine} ${lang}: an agent's row opens on Enter and Space, and its grip says what it does`, async (t) => {
      const page = await open(engine, t, lang);
      await navTo(page, "agents");
      const row = page.locator("#agents .row.agent").first();
      await row.waitFor();
      assert.equal(await row.getAttribute("tabindex"), "0", "in Tab's reach");
      assert.equal(await row.getAttribute("role"), "button");
      // both keys do what the pointer's click does: run the row's onclick
      await row.evaluate((r) => { r.dataset.clicks = "0"; r.onclick = () => { r.dataset.clicks = String(+r.dataset.clicks + 1); }; });
      await row.focus();
      await page.keyboard.press("Enter");
      assert.equal(await row.getAttribute("data-clicks"), "1", "Enter clicks it");
      await row.focus();
      await page.keyboard.press(" ");
      assert.equal(await row.getAttribute("data-clicks"), "2", "Space clicks it");
      const handle = row.locator(".ag-handle");
      assert.equal(await handle.getAttribute("aria-label"), await tr(page, "Drag to reorder"));
      if (lang === "zh") assert.equal(await handle.getAttribute("aria-label"), "拖动排序");
      assert.deepEqual(page.errors, []);
      await page.context().close();
    });
  }
}