// A request's row says what the vendor said when it failed, and what owl
// itself took: saiksy on X ("api 报错了，只能看到状态码，要看具体报错内容，
// 隐藏的深") and Suercha007 ("看到了有个中转服务，会很慢吗"). A JSON error
// reads as its message, after who said it, the whole of it in the title;
// owl's own time is +N ms where the route says it (Route.Own), and nothing
// on a row from before it was kept. A metrics pick saved before "owl" was
// one leaves it showing; one that left it out keeps it hidden. No user
// state is read.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const now = new Date();
const seat = { id: "copilot", provider: "copilot", name: "GitHub Copilot", who: "octo", kind: "account", model: "gpt-5.5" };
const JSON_ERR = 'GitHub Copilot: 429 {"error":{"message":"Sorry, you have been \\"rate\\"-limited. Wait a moment.","code":"rate_limited"}}';
const PLAIN_ERR = "dial tcp: lookup api.example.com: no such host\nsecond line";
function req(id, extra = {}, tryExtra = {}) {
  const time = new Date(now.getTime() - (20 - id) * 60e3).toISOString();
  return { id, seq: id, time, agent: "codex", session: "s", model: "copilot/gpt-5.5", provider: "copilot",
    order: [seat], tries: [{ id: seat.id, model: seat.model, start: time, done: true, status: 200, ms: 1200, ...tryExtra }],
    done: true, status: 200, ms: 1200, tokens: 900, out: 100, ...extra };
}
const fixtures = [
  req(5, { own: 3, sent: 40 }),
  req(4, { own: 0, sent: 35 }),
  req(3, { status: 429, own: 2, sent: 30 }, { status: 429, fail: "rate", error: JSON_ERR }),
  req(2, { status: 502, own: 1, sent: 20 }, { status: 502, fail: "offline", error: PLAIN_ERR }),
  req(1), // kept before owl's own time was: no "owl" on it
];

function serve(lang) {
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:true};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [{ id: "codex", name: "Codex", fields: [] }], profiles: [], settings: { lang, theme: "light" } });
    if (url.pathname === "/api/gateway/trace") {
      if (url.searchParams.has("wait")) return new Promise(() => {});
      return json({ mine: true, now: now.toISOString(), seq: 5, totals: { requests: fixtures.length, rerouted: 0, errors: 2 }, routes: fixtures });
    }
    if (url.pathname === "/api/gateway/history") return json({ cut: false, days: [], routes: [] });
    if (url.pathname === "/api/groups") return json({ groups: [] });
    if (url.pathname === "/api/providers") return json({ providers: [], gateway: { running: true, window: true } });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    return route.fulfill({ body: await fs.readFile(file), contentType });
  };
}

async function open(browser, lang, saved) {
  const page = await browser.newPage({ viewport: { width: 1280, height: 900 }, reducedMotion: "reduce" });
  page.setDefaultTimeout(8000);
  page.setDefaultNavigationTimeout(60000);
  const errors = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.route("**/*", serve(lang));
  if (saved) await page.addInitScript((s) => { for (const [k, v] of Object.entries(s)) localStorage.setItem(k, JSON.stringify(v)); }, saved);
  await page.goto("http://owl.test/?view=routing");
  await page.locator(".rt-req").nth(fixtures.length - 1).waitFor();
  return { page, rows: page.locator(".rt-req"), errors };
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(`${engine}: a failed row says the vendor's words, every row owl's own time`, { timeout: 120000 }, async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    for (const lang of ["en", "zh"]) {
      await t.test(lang, async () => {
        const { page, rows, errors } = await open(browser, lang);
        const ms = (n) => (lang === "zh" ? `+${n} 毫秒` : `+${n} ms`);
        // owl's own time, +0 included; none on a row from before
        assert.equal(await rows.nth(0).locator(".own .v").textContent(), ms(3));
        assert.equal(await rows.nth(0).locator(".own .k").textContent(), "owl");
        assert.match(await rows.nth(0).locator(".own").getAttribute("title"), lang === "zh" ? /owl 自己花的时间/ : /owl itself took/);
        assert.equal(await rows.nth(1).locator(".own .v").textContent(), ms(0), "no time of owl's own is said as +0, not left out");
        assert.equal(await rows.nth(4).locator(".own").count(), 0, "a row kept before owl's own time has none");
        // the vendor's words on a failed row, one line, the whole in its title
        const json = rows.nth(2).locator(".rt-err");
        assert.equal(await json.textContent(), 'GitHub Copilot: Sorry, you have been "rate"-limited. Wait a moment.');
        assert.equal(await json.getAttribute("title"), JSON_ERR);
        assert(await json.isVisible(), "it shows in the row");
        const plain = rows.nth(3).locator(".rt-err");
        assert.equal(await plain.textContent(), "dial tcp: lookup api.example.com: no such host");
        assert.equal(await plain.getAttribute("title"), PLAIN_ERR);
        for (const i of [0, 1, 4]) assert.equal(await rows.nth(i).locator(".rt-err").count(), 0, `answered row ${i} says no error`);
        // it stays on its row: cut, not running over the numbers
        const [r, e] = await Promise.all([rows.nth(2).boundingBox(), json.boundingBox()]);
        assert(e.x + e.width <= r.x + r.width + 0.5, "the error is cut inside its row");
        assert.deepEqual(errors, []);
        await page.close();
      });
    }
    await t.test("a pick saved before owl's own time was one", async () => {
      const { page, rows } = await open(browser, "en", { "owl.routingMetrics": ["duration", "tokens"] });
      assert.equal(await rows.nth(0).locator(".own").count(), 1, "an option added since the pick shows");
      assert.equal(await rows.nth(0).locator(".ttft").count(), 0, "what the reader left out stays out");
      await page.close();
    });
    await t.test("a pick that left owl's own time out", async () => {
      const { page, rows } = await open(browser, "en", {
        "owl.routingMetrics": ["duration", "tokens"],
        "owl.routingMetricsKnown": ["duration", "ttft", "tokens", "cache", "speed", "cost", "own"],
      });
      assert.equal(await rows.nth(0).locator(".own").count(), 0, "the reader's leaving it out is kept");
      await page.close();
    });
  });
}
