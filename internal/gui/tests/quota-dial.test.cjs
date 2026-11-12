// Run with Node's test runner and Playwright on the module path; see README.md.
// The tray panel's allowance windows. They were rings, kept round on a 1x
// screen (#1735, beansiayo: on an external 1080p display a ring looked
// slightly oval). Since 11-12 each is a bar on a line of its own under the
// account: its name, the bar and the share at its end, the bar as high
// along its length and across most of the card, the share inside the card,
// the name not cut. Chromium and WebKit, English and Chinese.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const at = (h) => new Date(Date.now() + h * 36e5).toISOString();
const quotas = [
  { provider: "claude", name: "Claude Code", icon: "claude-color", plan: "Max 5x",
    windows: [{ name: "5 hours", used: 30, resetsAt: at(2) }, { name: "7 days", used: 95, resetsAt: at(90) }] },
];

function serve(lang) {
  const settings = { theme: "light", lang, tray: "panel", quotaLeft: false, currency: "usd" };
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:false};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings });
    if (url.pathname === "/api/settings") return json(settings);
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true } });
    if (url.pathname === "/api/usage/quotas") return json(quotas);
    if (url.pathname === "/api/groups") return json({ groups: [], models: [] });
    if (url.pathname === "/api/plugins") return json({ plugins: [] });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    try { await route.fulfill({ body: await fs.readFile(file), contentType }); } catch { await route.fulfill({ status: 404, body: "" }); }
  };
}

// the panel's windows: a bar each, across the card, as high as it is
// everywhere along it, and the share at its end
for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const lang of ["en", "zh"]) test(`${engine} ${lang}: the panel's usage windows are bars`, async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    const page = await (await browser.newContext({ viewport: { width: 440, height: 640 }, reducedMotion: "reduce" })).newPage();
    page.setDefaultTimeout(5000);
    page.setDefaultNavigationTimeout(60000);
    const errors = [];
    page.on("pageerror", (e) => errors.push(e.message));
    await page.route("**/*", serve(lang));
    await page.goto("http://owl.test/?mode=panel");
    await page.locator('#ptabs [data-ptab="usage"]').click();
    const dials = page.locator("#panelQuota .pq-dial");
    await dials.first().waitFor();
    const got = await dials.evaluateAll((ds) => ds.map((d) => {
      const b = d.getBoundingClientRect(), card = d.closest(".pq-card").getBoundingClientRect(), share = d.querySelector("b").getBoundingClientRect();
      const name = d.parentElement.querySelector(".pq-rn");
      return { w: b.width, h: b.height, wide: b.width > card.width / 3, share: share.left >= b.right - 0.5 && share.right <= card.right, named: name.scrollWidth <= name.clientWidth };
    }));
    assert.equal(got.length, 2);
    for (const g of got) assert(g.h === 6 && g.wide && g.share && g.named, JSON.stringify(got));
    assert.deepEqual(errors, []);
  });
}
