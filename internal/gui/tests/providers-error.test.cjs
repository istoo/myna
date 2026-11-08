// Run with Node's test runner and Playwright on the module path; see README.md.
// A Providers list that couldn't be read is not a skeleton forever: the
// page says why and offers Retry in the list's place, or over the list it
// kept from before (app.js renderProvidersError). The old list stays, and
// a good read after it puts the page back.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const LANGS = ["en", "zh"];
const WHY = "providers.json is not readable";

const provider = (name) => ({
  id: "relay", name, icon: "generic", chat: "https://relay.example.com/v1", responses: "", anthropic: "", catalog: "",
  models: [{ id: "model-a", name: "Model A", on: true }], agents: [], fallback: [], keyList: [],
  key: { set: true, masked: "sk-…one" }, balanceToken: { takes: false, set: false },
});
const list = (p) => ({ providers: [p], presets: [], excluded: [], gateway: { running: true, window: true } });

// fail: the number of fresh reads to refuse; kept: what ?last=1 answers
function serve(lang, reads) {
  const state = { agents: [], profiles: [], settings: { lang, theme: "light" } };
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data, status = 200) => route.fulfill({ json: data, status });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:true};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json(url.searchParams.get("last") === "1" ? null : state);
    if (url.pathname === "/api/providers") {
      if (url.searchParams.get("last") === "1") return json(reads.kept ? list(provider("Kept Relay")) : null);
      reads.asked++;
      if (reads.fail > 0) { reads.fail--; return json({ error: WHY }, 500); }
      return json(list(provider("Fresh Relay")));
    }
    if (url.pathname === "/api/groups") return json({ groups: [] });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    await route.fulfill({ body: await fs.readFile(file), contentType });
  };
}

async function open(engine, t, lang, reads) {
  const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
  t.after(() => browser.close());
  const page = await (await browser.newContext({ viewport: { width: 900, height: 700 }, reducedMotion: "reduce" })).newPage();
  page.setDefaultTimeout(5000);
  page.setDefaultNavigationTimeout(60000);
  const errors = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.route("**/*", serve(lang, reads));
  await page.goto("http://myna.test/?view=providers");
  return { page, errors };
}

const tr = (lang, key, vars = {}) => {
  const en = Object.entries(vars).reduce((s, [k, v]) => s.replace(`{${k}}`, v), key);
  if (lang === "en") return en;
  return key === "Couldn't read your providers" ? "无法读取供应商列表"
    : key === "Retry" ? "重试" : key === "Refresh failed: {why}" ? `刷新失败：${vars.why}` : en;
};

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const lang of LANGS) {
    test(`${engine} ${lang}: a first read that failed is a card with Retry, not a skeleton`, async (t) => {
      const reads = { fail: 99, asked: 0, done: 0, kept: false };
      const { page, errors } = await open(engine, t, lang, reads);
      const card = page.locator("#providers .empty-state.load-error");
      await card.waitFor();
      assert.match(await card.textContent(), new RegExp(tr(lang, "Couldn't read your providers") + ".*" + WHY));
      assert.equal(await page.locator("#providers .pv-sk-row").count(), 0, "skeletons left behind");
      assert.equal(await page.evaluate(() => document.querySelector("#view-providers").classList.contains("loading")), false);
      assert.equal(await page.locator("#view-providers").getAttribute("aria-busy"), null);
      const retry = card.getByRole("button", { name: tr(lang, "Retry"), exact: true });
      assert.equal(await retry.count(), 1);
      reads.fail = 0;
      await retry.click();
      await page.locator("#providers .row.provider:not(.pv-sk-row)").waitFor();
      assert.equal(await page.locator("#providers .empty-state.load-error").count(), 0, "the card stayed after a good read");
      assert.deepEqual(errors, []);
    });

    test(`${engine} ${lang}: a failed refresh keeps the list and says so above it`, async (t) => {
      const reads = { fail: 0, asked: 0, done: 0, kept: true };
      const { page, errors } = await open(engine, t, lang, reads);
      await page.locator("#providers .row.provider:not(.pv-sk-row)").waitFor();
      // the next read fails: a reload of the page reads it again
      reads.fail = 1;
      await page.evaluate(() => loadProviders());
      const line = page.locator("#providersError");
      await line.waitFor();
      assert.match(await line.textContent(), new RegExp(tr(lang, "Refresh failed: {why}", { why: WHY })));
      assert.equal(await page.locator("#providers .row.provider:not(.pv-sk-row)").count(), 1, "the list kept");
      assert.equal(await page.locator("#providers .empty-state.load-error").count(), 0, "the list was replaced by a card");
      assert.deepEqual(errors, []);
    });
  }
}