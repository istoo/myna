// Run with Node's test runner and Playwright on the module path; see README.md.
// The window's polish (docs/owl-design.md, 2026-11-11), in both engines and
// every language:
// - the sidebar's icons are one set: a 20 grid, a 1.6 stroke, drawn at 16px,
//   Settings' gear too (it was a 24 grid's, at 15px);
// - a provider's row: the small switch (28×16) the Library and Plugins use,
//   and under a pointer the row's ground and its › say it opens;
// - a section's name and a tile's label in their own case at 12px, not in
//   small capitals;
// - the window at its narrowest (560): a usage row's cost stays on its card,
//   the breakdown before it giving way.
// No backend, the API is faked.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");

const provider = (id, name, host) => ({
  id, name, icon: "", preset: id, host, chat: `https://${host}/v1`, responses: "", anthropic: "", catalog: "", routing: "",
  models: [{ id: id + "-1", name: id + "-1", on: true }], agents: [], fallback: [], headers: {},
  keyList: [], key: { set: true, masked: "sk-…" + id.slice(0, 2) + "4f" }, balanceToken: { takes: false, set: false },
});
const providers = [provider("openai", "OpenAI", "api.openai.com"), provider("deepseek", "DeepSeek", "api.deepseek.com")];
// long breakdowns, so the narrow window has to give way somewhere
const g = (name, calls, cost) => ({ name, icon: "", calls, errors: 1, input: calls * 9100, output: calls * 1800, cache_read: calls * 30400, cache_write: calls * 410, reasoning: calls * 300, cost, unpriced: 0 });
const byAgent = [g("Claude Code", 214, 18.42), g("Codex", 96, 6.13)];
const byModel = [g("claude-opus-5-5", 171, 16.9), g("gpt-6.1-sol", 96, 6.13), g("claude-sonnet-5-5", 43, 1.52)];
const total = byAgent.reduce((a, b) => { for (const k of ["calls", "errors", "input", "output", "cache_read", "cache_write", "reasoning", "cost"]) a[k] = (a[k] || 0) + b[k]; return a; }, { unpriced: 0 });
const series = Array.from({ length: 24 }, (_, h) => ({ ...g("", h > 8 ? 10 : 0, h > 8 ? 0.5 : 0), label: String(h).padStart(2, "0") }));
const usage = { ...total, path: "/test/usage.jsonl", bucket: "hour", agents: byAgent, models: byModel, series, accounts: [], callerKeys: [] };

function server(lang) {
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    const p = url.pathname;
    if (p === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:false};` });
    if (p === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (p === "/api/state") return json({ agents: [], profiles: [], settings: { lang, theme: "light", guideHidden: true } });
    if (p === "/api/providers") return json({ providers, presets: [], excluded: [], gateway: { running: true, window: true } });
    if (p === "/api/groups") return json({ groups: [], pools: [] });
    if (p === "/api/plugins") return json({ plugins: [] });
    if (p === "/api/usage") return json(usage);
    if (p === "/api/usage/quotas") return json([]);
    if (p.startsWith("/api/")) return json({});
    const file = path.join(assets, p === "/" ? "index.html" : p);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    try { await route.fulfill({ body: await fs.readFile(file), contentType }); } catch { await route.fulfill({ status: 404, body: "" }); }
  };
}

const LANGS = ["en", "zh", "zh-TW", "ja", "de"];

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(engine + ": the sidebar's icons, a provider's row, the sections' names, a narrow usage row", async (t) => {
    assert(["chromium", "webkit"].includes(engine), "BROWSER must be chromium or webkit");
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    for (const lang of LANGS) {
      await t.test(lang, async () => {
        const ctx = await browser.newContext({ viewport: { width: 1100, height: 760 } });
        t.after(() => ctx.close());
        const page = await ctx.newPage();
        const errors = [];
        page.on("pageerror", (e) => errors.push(e.message));
        await page.route("**/*", server(lang));
        await page.goto("http://owl.test/?view=providers");
        await page.locator("#providers .row.provider").first().waitFor();

        // one set of icons: the grid, the stroke and the size are the same
        const icons = await page.evaluate(() => [...document.querySelectorAll("#nav button[data-view] svg.nav-ic"), document.querySelector("#prefs svg")].map((s) => {
          const r = s.getBoundingClientRect();
          return { at: s.closest("button").dataset.view || s.closest("button").id, box: s.getAttribute("viewBox"), strokes: [...new Set([...s.querySelectorAll("[stroke-width]")].map((e) => e.getAttribute("stroke-width")))], w: Math.round(r.width), h: Math.round(r.height) };
        }));
        assert.ok(icons.length >= 10, "the nine pages and Settings");
        for (const i of icons) assert.deepEqual({ at: i.at, box: i.box, strokes: i.strokes, w: i.w, h: i.h }, { at: i.at, box: "0 0 20 20", strokes: ["1.6"], w: 16, h: 16 }, i.at);

        // a provider's switch is the small one
        const sw = await page.locator("#providers .row.provider .pswitch").first().boundingBox();
        assert.deepEqual([Math.round(sw.width), Math.round(sw.height)], [28, 16], "the small switch");
        // under a pointer the row takes the app's hover ground and its › moves toward where it goes
        const rows = page.locator("#providers .row.provider");
        const look = (n) => rows.nth(n).evaluate((r) => [getComputedStyle(r).backgroundColor, getComputedStyle(r.querySelector(".chev")).transform]);
        const pill = await page.evaluate(() => { const e = document.createElement("i"); e.style.background = "var(--pill)"; document.body.append(e); const c = getComputedStyle(e).backgroundColor; e.remove(); return c; });
        await rows.nth(1).hover();
        await page.waitForFunction((pill) => getComputedStyle(document.querySelectorAll("#providers .row.provider")[1]).backgroundColor === pill, pill);
        await page.waitForFunction(() => /^matrix\(1, 0, 0, 1, 2, 0\)$/.test(getComputedStyle(document.querySelectorAll("#providers .row.provider")[1].querySelector(".chev")).transform));
        const [hovered, rest] = [await look(1), await look(0)];
        assert.equal(hovered[0], pill, "the hovered row's ground");
        assert.notEqual(rest[0], pill, "the other row keeps its own");
        assert.match(hovered[1], /^matrix\(1, 0, 0, 1, 2, 0\)$/, "the › a step to the right");
        assert.equal(rest[1], "none");

        // a section's name and a tile's label: their own case, at 12px
        await page.goto("http://owl.test/?view=usage");
        await page.locator("#usageAgents .row.stat").first().waitFor();
        const names = await page.evaluate(() => [...document.querySelectorAll("#view-usage .row-head .label, #stats .kpi > span, #stats .kpi .kpi-pick > span")].filter((e) => e.offsetParent).map((e) => { const c = getComputedStyle(e); return [c.textTransform, c.fontSize]; }));
        assert.ok(names.length >= 3, "the tiles' labels and the sections' names are on the page");
        for (const n of names) assert.deepEqual(n, ["none", "12px"]);

        // the window at its narrowest: each row's cost is whole and on its card
        await page.setViewportSize({ width: 560, height: 760 });
        await page.waitForFunction(() => innerWidth === 560);
        // the rows are laid again as the page settles at its new width: wait
        // for every cost to be on its card, then say which one isn't
        const onCards = () => [...document.querySelectorAll("#view-usage .row.stat")].filter((r) => r.offsetParent && r.querySelector(".cost")?.textContent)
          .every((r) => r.querySelector(".cost").getBoundingClientRect().right <= r.closest(".list").getBoundingClientRect().right);
        await page.waitForFunction(onCards, null, { timeout: 5000 }).catch(() => {});
        const cut = await page.evaluate(() => [...document.querySelectorAll("#view-usage .row.stat")].filter((r) => r.offsetParent && r.querySelector(".cost")?.textContent).map((r) => {
          const card = r.closest(".list").getBoundingClientRect(), c = r.querySelector(".cost");
          return { name: r.querySelector(".name").textContent, out: Math.round(c.getBoundingClientRect().right - card.right), clipped: c.scrollWidth > c.clientWidth + 1 };
        }));
        assert.ok(cut.length >= 5, "the agents' and the models' rows");
        for (const c of cut) assert.deepEqual({ name: c.name, off: c.out > 0, clipped: c.clipped }, { name: c.name, off: false, clipped: false }, "a cost on its card");
        assert.deepEqual(errors, []);
      });
    }
  });
}
