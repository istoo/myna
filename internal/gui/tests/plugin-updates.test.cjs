// Run with Node's test runner and Playwright on the module path; see README.md.
// Plugin updates: a plugin whose update waits for the reader (someone
// else's, or one pinned) puts a dot on Plugins, which goes once it's
// updated; a plugin owl updated by itself says so on its row. Plugins then
// opens on Installed, where the plugins with an update are named above the
// list. Every language, wide and narrow; the
// API is faked here.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");
const { navTo } = require("./reader.cjs");

const assets = path.resolve(__dirname, "../assets");

function server(lang) {
  const installed = [
    { spec: "opencode-copilot-auth", providers: ["GitHub Copilot"], version: "0.0.7", latest: "0.0.9" },
    { spec: "@magpie-community/opencode-zed-auth", providers: ["Zed"], version: "0.1.4", latest: "0.1.4",
      autoUpdated: { package: "@magpie-community/opencode-zed-auth", from: "0.1.3", to: "0.1.4", at: "2026-09-30T08:00:00Z" } },
  ];
  const waiting = () => installed.filter((e) => e.version !== e.latest).map((e) => ({ spec: e.spec, package: e.spec, version: e.version, latest: e.latest }));
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:false};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang, theme: "light" } });
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true }, plugins: [] });
    if (url.pathname === "/api/groups") return json({ groups: [], pools: [] });
    if (url.pathname === "/api/gateway/trace") return json({ routes: [] });
    if (url.pathname === "/api/plugins/updates") return json({ checked: "2026-10-01T00:00:00Z", waiting: waiting(), updated: [] });
    // the page asks for the market in parts (#488)
    if (url.pathname === "/api/plugins/market" || url.pathname === "/api/plugins" || url.pathname === "/api/plugins/listings") { const m = { listings: [], state: { bun: true, bunVersion: "1.3.0", plugins: installed } }; return json(url.pathname === "/api/plugins" ? m.state : url.pathname === "/api/plugins/listings" ? { listings: m.listings } : m); }
    if (url.pathname === "/api/plugins/upgrade") { installed[0].version = "0.0.9"; return json({}); }
    if (url.pathname.startsWith("/api/")) return json({});
    if (!/^\/[\w./-]*$/.test(url.pathname) || url.host !== "owl.test") return route.fulfill({ status: 404, body: "" });
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    try { await route.fulfill({ body: await fs.readFile(file), contentType }); } catch { await route.fulfill({ status: 404, body: "" }); }
  };
}

const L = {
  en: { bar: "A newer version of opencode-copilot-auth is out", installed: "Installed", update: "Update", auto: "Auto-updated", out: "An update for opencode-copilot-auth is out", from: /from v0\.1\.3 to v0\.1\.4/ },
  zh: { bar: "opencode-copilot-auth 有新版本", installed: "已安装", update: "更新", auto: "已自动更新", out: "opencode-copilot-auth 有新版本", from: /从 v0\.1\.3 更新到 v0\.1\.4/ },
  "zh-TW": { bar: "opencode-copilot-auth 有新版本", out: "opencode-copilot-auth 有新版本", installed: "已安裝" },
  ja: { bar: "opencode-copilot-auth の新しいバージョンがあります", out: "opencode-copilot-auth のアップデートが公開されました", installed: "インストール済み" },
  de: { bar: "Eine neue Version von opencode-copilot-auth ist da", out: "Ein Update für opencode-copilot-auth ist da", installed: "Installiert" },
};

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(engine + ": plugin updates", async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    for (const lang of Object.keys(L)) for (const width of [1024, 600]) {
      await t.test(lang + " " + width, async () => {
        const w = L[lang];
        const page = await (await browser.newContext({ viewport: { width, height: 820 } })).newPage();
        page.setDefaultTimeout(5000);
        page.setDefaultNavigationTimeout(60000);
        const errors = [];
        page.on("pageerror", (e) => errors.push(e.message));
        await page.route("**/*", server(lang));

        await page.goto("http://owl.test/?view=providers");
        const nav = page.locator('#nav button[data-view="plugins"]');
        await page.waitForFunction(() => document.querySelector('#nav button[data-view="plugins"]')?.classList.contains("has-dot"));
        assert.equal(await nav.getAttribute("title"), w.out);
        // the dot is drawn, small, inside the button: a tab's corner, or in
        // the rail at its icon's top right
        const side = await page.evaluate(() => document.body.classList.contains("side-nav"));
        assert.equal(side, width >= 1024, "a rail at " + width);
        const dot = await nav.evaluate((b) => { const s = getComputedStyle(b, "::after"); return { w: s.width, pos: s.position }; });
        assert.deepEqual(dot, { w: side ? "6px" : "5px", pos: "absolute" });
        if (side) {
          const [d, ic, bt] = await nav.evaluate((b) => {
            const s = getComputedStyle(b, "::after"), r = b.getBoundingClientRect(), i = b.querySelector(".nav-ic").getBoundingClientRect();
            const top = r.top + parseFloat(s.top), right = r.right - parseFloat(s.right);
            return [{ top, right, left: right - parseFloat(s.width) }, { top: i.top, right: i.right, left: i.left }, { top: r.top, right: r.right }];
          });
          assert.ok(d.top < ic.top + 6 && d.left >= ic.left && d.right <= bt.right, `at the icon's top right, in its row: ${JSON.stringify([d, ic, bt])}`);
        }

        await navTo(page, "plugins");
        // the dot, followed, lands on Installed (the page was last on
        // Discover), which names the plugins with an update first
        const view = page.locator("#view-plugins");
        const bar = view.locator(".pm-updates");
        await bar.waitFor();
        assert.equal(await bar.locator(".grow").textContent(), w.bar);
        assert.equal(await view.locator(".lib-tabs .opt.on").getAttribute("data-sub"), "installed");
        assert.ok(await bar.evaluate((b) => b.getBoundingClientRect().right <= b.closest("#view-plugins").getBoundingClientRect().right + 0.5), "the bar fits");
        assert.ok(await bar.evaluate((b) => b.compareDocumentPosition(b.parentElement.querySelector(".pm-list")) & Node.DOCUMENT_POSITION_FOLLOWING), "above the list");
        if (!w.update) { assert.deepEqual(errors, []); return; }
        const chip = view.locator(".pm-chip.soft", { hasText: w.auto });
        await chip.waitFor();
        assert.match(await chip.getAttribute("title"), w.from);

        // updating the waiting one takes the dot away
        await view.getByRole("button", { name: w.update, exact: true }).first().click();
        await page.waitForFunction(() => !document.querySelector('#nav button[data-view="plugins"]').classList.contains("has-dot"));
        assert.equal(await nav.getAttribute("title"), null);
        assert.equal(await bar.count(), 0);
        assert.deepEqual(errors, []);
      });
    }
  });
}
