// Run with Node's test runner and Playwright on the module path; see README.md.
// Usage with no calls in the period says what to do (docs/owl-design.md): the
// period's own line, why the page is empty, and the two ways to fill it —
// Connect an agent opens Agents, Add a provider opens Providers. The words
// are one line of text and a button each, never one run-on paragraph, in
// every language, at a phone's width too. No backend, the API is faked here.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");

function serve(lang) {
  const settings = { theme: "light", lang, quotaLeft: false, currency: "usd" };
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:true};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings });
    if (url.pathname === "/api/settings") return json(settings);
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true } });
    if (url.pathname === "/api/usage") return json({ period: url.searchParams.get("period") || "today", calls: 0, errors: 0, input: 0, output: 0, reasoning: 0, unpriced: 0, cost: 0, bucket: "day", series: [], agents: [], models: [], path: "" });
    if (url.pathname === "/api/usage/quotas") return json([]);
    if (url.pathname === "/api/groups") return json({ groups: [], models: [] });
    if (url.pathname === "/api/plugins") return json({ plugins: [] });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    try { await route.fulfill({ body: await fs.readFile(file), contentType }); } catch { await route.fulfill({ status: 404, body: "" }); }
  };
}

const words = {
  en: { title: "No calls today.", why: "Point an agent at a catalog model", connect: "Connect an agent", add: "Add a provider" },
  zh: { title: "今天没有调用。", why: "让某个 Agent 使用目录中的模型", connect: "接入一个 agent", add: "添加供应商" },
  "zh-TW": { title: "今天沒有呼叫。", why: "讓某個 Agent 使用目錄中的模型", connect: "接入一個 agent", add: "新增供應商" },
  ja: { title: "今日の呼び出しはありません。", why: "エージェントにカタログのモデルを", connect: "エージェントを接続", add: "プロバイダを追加" },
  de: { title: "Heute keine Aufrufe.", why: "Richten Sie einen Agenten auf ein Katalogmodell", connect: "Einen Agenten verbinden", add: "Anbieter hinzufügen" },
};

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(engine + ": Usage with no calls says what to do, and both ways work", async (t) => {
    assert(["chromium", "webkit"].includes(engine), "BROWSER must be chromium or webkit");
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    for (const [lang, w] of Object.entries(words)) {
      for (const width of [900, 390]) {
        await t.test(`${lang} ${width}`, async () => {
          const ctx = await browser.newContext({ viewport: { width, height: 760 } });
          const page = await ctx.newPage();
          const errors = [];
          page.on("pageerror", (e) => errors.push(e.message));
          await page.route("**/*", serve(lang));
          await page.goto("http://owl.test/?view=usage");
          const none = page.locator("#stats .none");
          await none.waitFor();
          // the period's line, then why, each a part of its own
          assert.equal((await none.locator("b").textContent()).trim(), w.title);
          assert.ok((await none.locator(".why").textContent()).includes(w.why), "why: " + (await none.locator(".why").textContent()));
          const [connect, add] = await page.locator("#stats .none-acts button").all();
          assert.equal((await connect.textContent()).trim(), w.connect);
          assert.equal((await add.textContent()).trim(), w.add);
          // the two buttons are a row, and inside the card
          const [c, a, card] = await Promise.all([connect.boundingBox(), add.boundingBox(), page.locator("#stats").boundingBox()]);
          assert.ok(Math.abs(c.y - a.y) < 2, "one row");
          assert.ok(c.x >= card.x && a.x + a.width <= card.x + card.width + 1, "inside the page's card");
          // each opens the page that fills it
          await connect.click();
          await page.waitForFunction(() => !document.querySelector("#view-agents")?.hidden);
          await page.goto("http://owl.test/?view=usage");
          await page.locator("#stats .none-acts .text.action").click();
          await page.waitForFunction(() => !document.querySelector("#view-providers")?.hidden);
          assert.deepEqual(errors, []);
          await ctx.close();
        });
      }
    }
  });
}