// Run with Node's test runner and Playwright on the module path; see README.md.
// The sidebar's Search (docs/owl-design.md): a button at its top opens the
// command palette, its shortcut shown beside the word; ⌘1 to ⌘9 (Ctrl off
// the Mac) open the pages in the sidebar's order, each page's tooltip
// saying its own; a browser tab (owl web) leaves the digits to the browser.
// No backend, the API is faked.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");

function server(lang, web) {
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:${web}};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang, theme: "light" } });
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true } });
    if (url.pathname === "/api/groups") return json({ groups: [], pools: [] });
    if (url.pathname === "/api/plugins") return json({ plugins: [] });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    try { await route.fulfill({ body: await fs.readFile(file), contentType }); } catch { await route.fulfill({ status: 404, body: "" }); }
  };
}

const MAC = process.platform === "darwin";
const mod = MAC ? "Meta" : "Control";
const word = { en: "Search", zh: "搜索", "zh-TW": "搜尋", ja: "検索", de: "Suchen" };

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(engine + ": the sidebar's Search and the pages' shortcuts", async (t) => {
    assert(["chromium", "webkit"].includes(engine), "BROWSER must be chromium or webkit");
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    for (const lang of Object.keys(word)) {
      await t.test(lang, async () => {
        const ctx = await browser.newContext({ viewport: { width: 1100, height: 760 } });
        const page = await ctx.newPage();
        const errors = [];
        page.on("pageerror", (e) => errors.push(e.message));
        await page.route("**/*", server(lang, false));
        await page.goto("http://owl.test/?view=agents");
        const find = page.locator("#sideFind");
        await find.waitFor();
        assert.equal((await find.locator("span").textContent()).trim(), word[lang], "the word is the language's");
        assert.equal(await find.locator("kbd").textContent(), MAC ? "⌘K" : "Ctrl K");
        const box = await find.boundingBox(), side = await page.locator("#side").boundingBox();
        assert.ok(box.x >= side.x && box.x + box.width <= side.x + side.width, "inside the sidebar");
        // it opens the palette, and Escape puts it away
        await find.click();
        await page.locator(".palette-dialog").waitFor();
        await page.keyboard.press("Escape");
        await page.locator(".palette-dialog").waitFor({ state: "detached" });

        // ⌘N opens the Nth page the sidebar lists
        const views = await page.locator("#nav button[data-view]").evaluateAll((bs) => bs.filter((b) => b.offsetParent).map((b) => b.dataset.view));
        assert.ok(views.length >= 3, "pages listed: " + views);
        for (const n of [3, 1, views.length > 9 ? 9 : views.length]) {
          await page.keyboard.press(`${mod}+Digit${n}`);
          await page.waitForFunction((v) => document.querySelector(`#nav button[data-view="${v}"]`)?.classList.contains("on"), views[n - 1]);
        }
        await page.keyboard.press(`${mod}+Digit2`);
        const second = page.locator(`#nav button[data-view="${views[1]}"]`);
        await page.waitForFunction((v) => document.querySelector(`#nav button[data-view="${v}"]`)?.classList.contains("on"), views[1]);
        // its tooltip says its shortcut
        await second.hover();
        assert.match(await second.getAttribute("title"), MAC ? / \(⌘2\)$/ : / \(Ctrl\+2\)$/);

        assert.deepEqual(errors, []);
        await ctx.close();
      });
    }
    await t.test("the pages from the keyboard", async () => {
      const ctx = await browser.newContext({ viewport: { width: 1100, height: 760 } });
      const page = await ctx.newPage();
      await page.route("**/*", server("en", false));
      await page.goto("http://owl.test/?view=agents");
      await page.locator("#sideFind").waitFor();
      const views = await page.locator("#nav button[data-view]").evaluateAll((bs) => bs.filter((b) => b.offsetParent).map((b) => b.dataset.view));
      const focused = () => page.evaluate(() => document.activeElement?.dataset?.view || "");
      const on = async (v) => page.locator(`#nav button[data-view="${v}"]`).focus();
      // ↓ and ↑ walk the pages, Home and End the ends
      await on(views[0]); await page.keyboard.press("ArrowDown");
      assert.equal(await focused(), views[1], "↓ moves to the next page");
      await page.keyboard.press("ArrowUp");
      assert.equal(await focused(), views[0], "↑ back");
      await page.keyboard.press("End");
      assert.equal(await focused(), views[views.length - 1], "End to the last");
      await page.keyboard.press("Home");
      assert.equal(await focused(), views[0], "Home to the first");
      // a letter jumps to the next page beginning with it
      await page.keyboard.press("u");
      const lit = await focused();
      assert.ok(lit.startsWith("u") || (await page.locator(`#nav button[data-view="${lit}"] span`).textContent()).toLowerCase().startsWith("u"), "a letter jumps to the page it begins: " + lit);
      // Enter opens the page focused
      await on(views[2]); await page.keyboard.press("Enter");
      await page.waitForFunction((v) => document.querySelector(`#nav button[data-view="${v}"]`)?.classList.contains("on"), views[2]);
      assert.equal(await page.evaluate(() => document.querySelector("#view-" + document.querySelector("#nav button.on")?.dataset.view)?.hidden === false), true, "the page opened");
      await ctx.close();
    });

    await t.test("a browser tab keeps ⌘N for its own tabs", async () => {
      const ctx = await browser.newContext({ viewport: { width: 1100, height: 760 } });
      const page = await ctx.newPage();
      await page.route("**/*", server("en", true));
      await page.goto("http://owl.test/?view=agents");
      await page.locator("#sideFind").waitFor();
      await page.keyboard.press(`${mod}+Digit3`);
      // nothing to wait on for something that doesn't happen: a round trip
      // through the page's own event loop is past any handler that ran
      await page.evaluate(() => new Promise((r) => setTimeout(r, 0)));
      assert.ok(await page.locator('#nav button[data-view="agents"]').evaluate((b) => b.classList.contains("on")), "still on Agents");
      await page.locator('#nav button[data-view="agents"]').hover();
      assert.equal(await page.locator('#nav button[data-view="agents"]').getAttribute("title"), null, "no shortcut said that it doesn't have");
      await ctx.close();
    });
  });
}
