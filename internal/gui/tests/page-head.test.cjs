// Run with Node's test runner and Playwright on the module path; see README.md.
// A page's head (pagehead.js, docs/owl-design.md 11-11), in both engines and
// every language:
// - in the sidebar's layout the Agents and Providers pages open under their
//   name and a line on where they stand, and the title bar's crumb gives
//   way to it; a page with a toolbar of its own keeps the crumb;
// - Providers' search, Hide accounts and Add provider sit in its head, the
//   same elements, and go back under the list in the top tabs' layout, the
//   search keeping the focus and the caret as it moves;
// - the Agents page's agents in use are cards side by side, each saying
//   today's calls, and a card dragged onto another's place takes it.
// No backend, the API is faked.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const opts = [{ value: "openai-1", label: "openai-1", icon: "generic", ref: "openai/openai-1" }, { value: "deepseek-1", label: "deepseek-1", icon: "generic", ref: "deepseek/deepseek-1" }];
const agent = (id, name, wired) => ({ id, name, path: "/fixture/" + id, icon: "generic", wired, version: "1.0.0", fields: [{ key: "model", label: "model", value: wired ? "openai-1" : "", options: opts }] });
const provider = (id, name, host, extra = {}) => ({
  id, name, icon: "", preset: id, host, chat: `https://${host}/v1`, responses: "", anthropic: "", catalog: "", routing: "",
  models: [{ id: id + "-1", name: id + "-1", on: true }, { id: id + "-2", name: id + "-2", on: true }], agents: [], fallback: [], headers: {},
  keyList: [], key: { set: true, masked: "sk-…4f" }, balanceToken: { takes: false, set: false }, ...extra,
});
const g = (name, calls, cost) => ({ name, icon: "", calls, errors: 0, input: calls * 1000, output: calls * 100, cache_read: 0, cache_write: 0, reasoning: 0, cost, unpriced: 0 });

function server(lang, arranged) {
  const state = { agents: [agent("claude", "Claude Code", true), agent("codex", "Codex", true), agent("gemini", "Gemini CLI", true), agent("pi", "Pi", false)], profiles: [], settings: { lang, theme: "light", guideHidden: true } };
  const providers = [provider("openai", "OpenAI", "api.openai.com"), provider("deepseek", "DeepSeek", "api.deepseek.com"), provider("off", "Off", "off.example", { off: true })];
  const usage = { ...g("", 310, 24.55), agents: [g("Claude Code", 214, 18.42), g("Codex", 96, 6.13)], models: [], series: [], accounts: [], callerKeys: [], bucket: "hour", path: "/test/usage.jsonl" };
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    const p = url.pathname;
    if (p === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:false};` });
    if (p === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (p === "/api/state") return json(state);
    if (p === "/api/agents/arrange") {
      const body = route.request().postDataJSON();
      arranged.push(body.order);
      state.settings = { ...state.settings, agentOrder: body.order };
      return json(state);
    }
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
  test(engine + ": a page's head, Providers' actions in it, the Agents page's cards", { timeout: 240000 }, async (t) => {
    assert(["chromium", "webkit"].includes(engine), "BROWSER must be chromium or webkit");
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    for (const lang of LANGS) {
      await t.test(lang, async () => {
        const ctx = await browser.newContext({ viewport: { width: 1280, height: 800 }, reducedMotion: "reduce" });
        t.after(() => ctx.close());
        const page = await ctx.newPage();
        const errors = [], arranged = [];
        page.on("pageerror", (e) => errors.push(e.message));
        await page.route("**/*", server(lang, arranged));
        await page.goto("http://owl.test/?view=providers");
        await page.locator("#providers .row.provider").first().waitFor();

        // the head: the page's name as the sidebar says it, what is on, the
        // crumb out of sight
        const head = page.locator("#view-providers > .ph");
        await head.waitFor();
        const navName = await page.locator('#nav button[data-view="providers"] span[data-t]').textContent();
        assert.equal(await head.locator(".ph-name").textContent(), navName);
        assert.equal(await page.locator("#crumb").evaluate((c) => getComputedStyle(c).visibility), "hidden");
        // two on of three, four models on them: the numbers in the line
        assert.deepEqual(await head.locator(".ph-s b").allTextContents(), ["2", "4"]);
        // its actions in the head, the page's own elements, at its right
        for (const id of ["#pvQ", "#pvMask", "#addProvider"]) assert.equal(await page.locator(`#view-providers > .ph .ph-a ${id}`).count(), 1, id);
        const [h, add] = [await head.boundingBox(), await page.locator("#addProvider").boundingBox()];
        assert.ok(add.x + add.width > h.x + h.width - 2 && add.y >= h.y && add.y + add.height <= h.y + h.height + 1, "Add provider at the head's right");
        await page.locator("#pvQ").fill("deep");
        await page.waitForFunction(() => [...document.querySelectorAll("#providers .row.provider")].filter((r) => r.offsetParent).length === 1);
        await page.locator("#pvQ").fill("");
        await page.locator("#addProvider").click();
        await page.locator("#addSheet:not([hidden]), #modal:not([hidden])").first().waitFor();
        await page.keyboard.press("Escape");

        // a page with a toolbar of its own keeps the crumb, and has no head
        await page.locator('#nav button[data-view="usage"]').click();
        await page.waitForFunction(() => !document.querySelector("main.view > .ph") && getComputedStyle(document.querySelector("#crumb")).visibility === "visible");
        await page.locator('#nav button[data-view="providers"]').click();
        await page.locator("#view-providers > .ph .ph-a #addProvider").waitFor();

        // the top tabs' layout: no head, the actions back under the list
        await page.setViewportSize({ width: 900, height: 800 });
        await page.waitForFunction(() => !document.body.classList.contains("side-nav") && !document.querySelector("main.view > .ph"));
        assert.equal(await page.locator("#view-providers .after-list #addProvider").count(), 1, "Add provider back under the list");
        assert.equal(await page.locator("#view-providers > #pvHead").count(), 1, "the search back above the list");
        // the search keeps the focus and the caret as it moves into the head
        await page.locator("#pvQ").focus();
        await page.keyboard.type("op");
        await page.keyboard.press("ArrowLeft");
        await page.setViewportSize({ width: 1280, height: 800 });
        await page.locator("#view-providers > .ph .ph-a #addProvider").waitFor();
        assert.deepEqual(await page.evaluate(() => [document.activeElement?.id, document.activeElement?.value, document.activeElement?.selectionStart]), ["pvQ", "op", 1], "the search's focus and caret");
        await page.locator("#pvQ").fill("");

        // the Agents page: three in use, two cards to a row, today's calls on
        // the two that made any
        await page.locator('#nav button[data-view="agents"]').click();
        await page.locator("#view-agents > .ph").waitFor();
        assert.deepEqual(await page.locator("#view-agents > .ph .ph-s b").allTextContents(), ["3", "4"]);
        const cards = page.locator("#agents > .row.agent");
        await page.waitForFunction(() => document.querySelectorAll("#agents > .row.agent > .ag-today").length === 2);
        const boxes = await cards.evaluateAll((rs) => rs.map((r) => { const b = r.getBoundingClientRect(); return { id: r.dataset.id, x: b.x, y: b.y, w: b.width, today: r.querySelector(".ag-today")?.textContent || "" }; }));
        assert.deepEqual(boxes.map((b) => b.id), ["claude", "codex", "gemini"]);
        assert.ok(Math.abs(boxes[0].y - boxes[1].y) < 1 && boxes[1].x > boxes[0].x + boxes[0].w, "the first two side by side");
        assert.ok(boxes[2].y > boxes[0].y, "the third under them");
        assert.match(boxes[0].today, /214/);
        assert.match(boxes[0].today, /18\.42/);
        assert.equal(boxes[2].today, "", "no calls, no line");
        // each card's model picker across it
        if (process.env.SHOT) await page.screenshot({ path: "/tmp/owlx/ph-" + lang + ".png" });
        const pick = await cards.first().evaluate((r) => { const f = r.querySelector(".field"); return [f.getBoundingClientRect().width, r.getBoundingClientRect().width]; });
        assert.ok(pick[0] > pick[1] * 0.8, "the model across the card");

        // the second card dragged onto the first's place takes it
        const grip = async (i) => { const b = await cards.nth(i).locator(".ag-handle").boundingBox(); return { x: b.x + b.width / 2, y: b.y + b.height / 2 }; };
        const [a, b] = [await grip(0), await grip(1)];
        await page.mouse.move(b.x, b.y);
        await page.mouse.down();
        await page.mouse.move(a.x, a.y, { steps: 14 });
        await page.locator("#agents > .row.dragging").waitFor();
        await page.mouse.up();
        await page.waitForFunction(() => document.querySelector("#agents > .row.agent")?.dataset.id === "codex");
        assert.deepEqual(arranged.at(-1)?.slice(0, 2), ["codex", "claude"]);
        assert.deepEqual(errors, []);
      });
    }
  });
}
