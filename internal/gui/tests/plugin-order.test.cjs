// Run with Node's test runner and Playwright on the module path; see README.md.
// Discover's kinds and order (pmorder.js): a strip names each kind the
// market lists with how many, and a pick shows that kind alone; the order is
// the market's own or npm's downloads last week, most first, a plugin npm
// hasn't counted last. Both are kept for the next visit. Every language,
// both engines, at a narrow width too; the API is faked here.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const state = { bun: true, bunVersion: "1.3.0", plugins: [] };
const pv = (name, weekly) => ({ package: "@magpie-community/opencode-" + name.toLowerCase() + "-auth", name, community: true, providers: [name.toLowerCase()], summary: { en: name + " plan" }, npm: { version: "0.1.0", weekly } });
const listings = [
  pv("Alpha", 5), pv("Beta", 500), pv("Gamma", 0),
  { package: "@magpie-community/middleware-trim", name: "Trim", kind: "middleware", community: true, summary: { en: "Trims" }, npm: { version: "0.1.0", weekly: 9 } },
  { package: "@magpie-community/tools-calc", name: "Calc tools", kind: "tools", community: true, summary: { en: "Exact arithmetic" }, npm: { version: "0.1.0" } },
  { package: "@magpie-community/page-glance", name: "Glance page", kind: "page", community: true, summary: { en: "Today's usage" }, npm: { version: "0.1.0", weekly: 3 } },
];

function server(lang) {
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:false};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang, theme: "light" } });
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true }, plugins: [] });
    if (url.pathname === "/api/groups") return json({ groups: [], pools: [] });
    if (url.pathname === "/api/plugins") return json(state);
    if (url.pathname === "/api/plugins/listings") return json({ listings });
    if (url.pathname === "/api/plugins/market") return json({ listings, state });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    try { await route.fulfill({ body: await fs.readFile(file), contentType }); } catch { await route.fulfill({ status: 404, body: "" }); }
  };
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(engine + ": Discover's kinds to pick, and its order by downloads", { timeout: 300000 }, async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    for (const lang of ["en", "zh", "zh-TW", "ja", "de"]) for (const width of [980, 420]) {
      await t.test(lang + " " + width, async () => {
        const ctx = await browser.newContext({ viewport: { width, height: 820 } });
        t.after(() => ctx.close());
        const page = await ctx.newPage();
        // the strip comes with pmorder.js, loaded as the page draws: a
        // loaded machine may take a few seconds over it
        page.setDefaultTimeout(15000);
        page.setDefaultNavigationTimeout(60000);
        const errors = [];
        page.on("pageerror", (e) => errors.push(e.message));
        await page.route("**/*", server(lang));
        await page.goto("http://owl.test/?view=plugins");
        const view = page.locator("#view-plugins");
        await view.locator(".lib-tabs .opt").nth(1).click();
        const kinds = view.locator(".pm-mkbar .pm-mkkinds .opt");
        const sorts = view.locator(".pm-mkbar .pm-mksort .opt");
        await kinds.first().waitFor();
        const names = (i) => view.locator(".pm-sec:not(.pm-ghsec)").nth(i).locator(".pm-card .pm-name b").allInnerTexts();
        const sections = () => view.locator(".pm-sec:not(.pm-ghsec)").count();

        // every kind the market lists, each with its count, All first
        const counts = (await kinds.allInnerTexts()).map((s) => s.replace(/\s+/g, " ").trim().split(" · ")[1]);
        assert.deepEqual(counts, ["6", "3", "1", "1", "1"]);
        assert.equal(await kinds.first().evaluate((b) => b.classList.contains("on")), true, "All, at first");
        assert.equal(await sections(), 4);

        // the market's own order, then npm's downloads, most first, the
        // uncounted last
        assert.deepEqual(await names(0), ["Alpha", "Beta", "Gamma"]);
        await sorts.nth(1).click();
        await page.waitForFunction(() => document.querySelector(".pm-sec .pm-card .pm-name b")?.textContent === "Beta");
        assert.deepEqual(await names(0), ["Beta", "Alpha", "Gamma"]);
        assert.equal(await sorts.nth(1).evaluate((b) => b.classList.contains("on")), true);

        // a kind alone
        await kinds.nth(3).click();
        await page.waitForFunction(() => document.querySelectorAll(".pm-sec:not(.pm-ghsec)").length === 1);
        assert.deepEqual(await names(0), ["Calc tools"]);

        // kept for the next visit
        await page.reload();
        await view.locator(".lib-tabs .opt").nth(1).waitFor();
        await page.locator(".pm-mkbar .pm-mkkinds .opt.on").waitFor();
        assert.equal(await kinds.nth(3).evaluate((b) => b.classList.contains("on")), true, "the kind kept");
        assert.equal(await sorts.nth(1).evaluate((b) => b.classList.contains("on")), true, "the order kept");
        assert.deepEqual(await names(0), ["Calc tools"]);

        // all again, still by downloads
        await kinds.first().click();
        await page.waitForFunction(() => document.querySelectorAll(".pm-sec:not(.pm-ghsec)").length === 4);
        assert.deepEqual(await names(0), ["Beta", "Alpha", "Gamma"]);

        const over = await page.evaluate(() => document.scrollingElement.scrollWidth - document.scrollingElement.clientWidth);
        assert.ok(over <= 0, "page scrolls sideways by " + over);
        assert.deepEqual(errors, []);
      });
    }
  });
}
