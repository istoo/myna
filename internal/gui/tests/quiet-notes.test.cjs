// A page's words on how a part of it works fold to their first line until
// asked for (app.js owlQuiet): Suercha007's thread on X, "一进去一股脑的全是
// AI 写的 notes，每个功能卡恨不得都直接写，信息量爆炸". Folded is one line,
// a button to the keyboard and to a screen reader, opened with a click,
// Enter or Space; a welcome (no agent switched on yet) and why nothing
// took a request are said whole. Both engines; the API is faked.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const now = new Date();
const opts = [{ value: "m1", label: "m1", icon: "generic", ref: "openai/m1" }];
const agent = (id, name, wired) => ({ id, name, path: "/fixture/" + id, icon: "generic", wired, version: "1.0.0", fields: [{ key: "model", label: "model", value: wired ? "m1" : "", options: opts }] });
const seat = { id: "openai", provider: "openai", name: "OpenAI", kind: "key", model: "m1" };
const routes = [
  { id: 2, seq: 2, time: now.toISOString(), agent: "codex", model: "openai/m1", provider: "openai", order: [seat],
    tries: [{ id: "openai", model: "m1", start: now.toISOString(), done: true, status: 200, ms: 900 }], done: true, status: 200, ms: 900 },
  { id: 1, seq: 1, time: new Date(now - 60e3).toISOString(), agent: "codex", model: "gone/m9", provider: "gone", order: [], tries: [],
    done: true, status: 404, error: "no provider serves gone/m9" },
];

function server(lang, wired) {
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    const p = url.pathname;
    if (p === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:false};` });
    if (p === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (p === "/api/state") return json({ agents: [agent("claude", "Claude Code", wired), agent("codex", "Codex", false)], profiles: [], settings: { lang, theme: "light", guideHidden: true } });
    if (p === "/api/providers") return json({ providers: [{ id: "openai", name: "OpenAI", models: [{ id: "m1", name: "m1", on: true }], key: { set: true } }], presets: [], excluded: [], gateway: { running: true, window: true } });
    if (p === "/api/gateway/trace") {
      if (url.searchParams.has("wait")) return new Promise(() => {});
      return json({ mine: true, now: now.toISOString(), seq: 2, totals: { requests: 2, rerouted: 0, errors: 1 }, routes });
    }
    if (p === "/api/gateway/history") return json({ cut: false, days: [], routes: [] });
    if (p === "/api/groups") return json({ groups: [] });
    if (p.startsWith("/api/")) return json({});
    const file = path.join(assets, p === "/" ? "index.html" : p);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    try { await route.fulfill({ body: await fs.readFile(file), contentType }); } catch { await route.fulfill({ status: 404, body: "" }); }
  };
}

const lineH = (loc) => loc.evaluate((e) => [e.getBoundingClientRect().height, parseFloat(getComputedStyle(e).lineHeight) || parseFloat(getComputedStyle(e).fontSize) * 1.5]);

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(`${engine}: a page's notes fold to a line until asked for`, { timeout: 180000 }, async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    for (const lang of ["en", "zh", "de"]) {
      await t.test(lang, async () => {
        // narrow enough that the Agents page's line runs to a second one
        // (the header's tabs' layout: the fold isn't the rail's alone)
        const page = await browser.newPage({ viewport: { width: 560, height: 760 }, reducedMotion: "reduce" });
        page.setDefaultTimeout(8000);
        page.setDefaultNavigationTimeout(60000);
        const errors = [];
        page.on("pageerror", (e) => errors.push(e.message));
        await page.route("**/*", server(lang, true));
        await page.goto("http://owl.test/?view=agents");
        const lead = page.locator("#agentsLead");
        await lead.waitFor();
        assert.equal(await lead.getAttribute("role"), "button");
        assert.equal(await lead.getAttribute("aria-expanded"), "false");
        let [h, line] = await lineH(lead);
        assert(h <= line * 1.2 + 1, `folded, one line: ${h} for a line of ${line}`);
        const full = await lead.evaluate((e) => e.scrollHeight);
        assert(full > h + 4, `the line runs on past the fold at this width (${full} > ${h}), or the test tests nothing`);
        await lead.click();
        assert.equal(await lead.getAttribute("aria-expanded"), "true");
        [h] = await lineH(lead);
        assert(h >= full - 1, `opened, it is whole: ${h} of ${full}`);
        // the keyboard: it is reached by Tab, and Enter and Space fold it
        await lead.focus();
        await page.keyboard.press("Enter");
        assert.equal(await lead.getAttribute("aria-expanded"), "false");
        await page.keyboard.press(" ");
        assert.equal(await lead.getAttribute("aria-expanded"), "true");

        // Routing: how it routes folds; why nothing took a request does not
        await page.goto("http://owl.test/?view=routing");
        const mode = page.locator(".rt-mode");
        await page.locator(".rt-req").nth(1).waitFor();
        await page.locator(".rt-req").nth(0).click();
        await page.waitForFunction(() => document.querySelector(".rt-mode")?.classList.contains("owl-quiet"));
        await page.locator(".rt-req").nth(1).click();
        await page.waitForFunction(() => /gone\/m9/.test(document.querySelector(".rt-mode")?.textContent || ""));
        assert.equal(await mode.getAttribute("role"), null, "why nothing took it is no button");
        assert.equal(await mode.evaluate((e) => e.classList.contains("owl-quiet")), false, "and it is said whole");
        await page.locator(".rt-req").nth(0).click();
        await page.waitForFunction(() => document.querySelector(".rt-mode")?.classList.contains("owl-quiet"));
        assert.equal(await mode.getAttribute("role"), "button", "folded again for the next request, once");
        assert.deepEqual(errors, []);
        await page.close();
      });
    }
    await t.test("a welcome is said whole", async () => {
      const page = await browser.newPage({ viewport: { width: 1040, height: 760 }, reducedMotion: "reduce" });
      await page.route("**/*", server("en", false));
      await page.goto("http://owl.test/?view=agents");
      const lead = page.locator("#agentsLead.welcome");
      await lead.waitFor();
      assert.equal(await lead.getAttribute("role"), null);
      assert.equal(await lead.evaluate((e) => e.classList.contains("owl-quiet")), false);
      await page.close();
    });
  });
}
