// Run with Node's test runner and Playwright on the module path; see README.md.
// The light theme's quiet words (--muted and --faint) are at least 4.5:1
// against every ground they sit on: the page, a card, and the segmented
// controls' track. The dark theme's tokens are its own and are not changed
// here. Chromium and WebKit; no backend, the API is faked.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");

function server(theme) {
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"en",theme:"${theme}",web:false};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang: "en", theme } });
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true } });
    if (url.pathname === "/api/groups") return json({ groups: [] });
    if (url.pathname === "/api/plugins") return json({ plugins: [] });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    await route.fulfill({ body: await fs.readFile(file), contentType });
  };
}

const rgb = (s) => {
  const m = s.match(/rgba?\(([\d.]+)[, ]+([\d.]+)[, ]+([\d.]+)/);
  assert.ok(m, `not a colour: ${s}`);
  return m.slice(1, 4).map(Number);
};
const lum = ([r, g, b]) => [r, g, b].map((v) => {
  v /= 255;
  return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
}).reduce((y, v, i) => y + v * [0.2126, 0.7152, 0.0722][i], 0);
const ratio = (a, b) => {
  const x = lum(a), y = lum(b);
  return (Math.max(x, y) + 0.05) / (Math.min(x, y) + 0.05);
};

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(`${engine}: the light theme's quiet words reach 4.5:1`, async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    const page = await (await browser.newContext({ viewport: { width: 1000, height: 800 }, reducedMotion: "reduce" })).newPage();
    page.setDefaultTimeout(5000);
    page.setDefaultNavigationTimeout(60000);
    await page.route("**/*", server("light"));
    await page.goto("http://myna.test/?view=settings");
    await page.locator("#themeSegs").waitFor();
    const pairs = await page.evaluate(() => {
      const box = document.createElement("span");
      box.style.position = "fixed";
      box.style.left = "-100px";
      document.body.append(box);
      const color = (token, ground, label) => {
        box.style.color = `var(${token})`;
        box.style.background = `var(${ground})`;
        const cs = getComputedStyle(box);
        return [label || `${token} on ${ground}`, cs.color, cs.backgroundColor];
      };
      const out = [];
      for (const token of ["--muted", "--faint"]) for (const ground of ["--bg", "--card", "--seg-track"]) out.push(color(token, ground));
      // the tray panel's light theme has surfaces of its own
      document.body.classList.add("panel");
      for (const token of ["--muted", "--faint"]) for (const ground of ["--bg", "--card"]) out.push(color(token, ground, `panel ${token} on ${ground}`));
      document.body.classList.remove("panel");
      box.remove();
      return out;
    });
    for (const [name, fg, bg] of pairs) {
      const c = ratio(rgb(fg), rgb(bg));
      assert.ok(c >= 4.5, `${name}: ${c.toFixed(2)}:1 (${fg} on ${bg})`);
    }
    await page.context().close();
  });
}