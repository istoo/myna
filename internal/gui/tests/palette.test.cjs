// Run with Node's test runner and Playwright on the module path; see README.md.
// ⌘K / Ctrl+K opens the command palette (palette.js, imported when first
// asked for): pages, Settings' rows, providers, agents and plugin pages,
// picked with the arrows and Enter, Escape away; "/" puts the cursor in the
// page's own search box.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");
const { navTo } = require("./reader.cjs");

const assets = path.resolve(__dirname, "../assets");

function serve() {
  const state = {
    agents: [{ id: "codex", name: "Codex", icon: "codex-color", path: "/home/me/.codex", fields: [], wired: false }],
    profiles: [], settings: { lang: "en", theme: "light", guideHidden: true },
  };
  const providers = { providers: [{ id: "relay", name: "Relay", icon: "generic", chat: "https://relay.example.com/v1", responses: "", anthropic: "", catalog: "", models: [], agents: [], fallback: [], keyList: [], key: { set: true, masked: "sk-…one" }, balanceToken: { takes: false, set: false } }], presets: [], excluded: [], gateway: { running: true, window: true } };
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"en",theme:"light",web:false};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json(state);
    if (url.pathname === "/api/providers") return json(providers);
    if (url.pathname === "/api/groups") return json({ groups: [] });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    await route.fulfill({ body: await fs.readFile(file), contentType });
  };
}

async function open(engine, t) {
  const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
  t.after(() => browser.close());
  const page = await (await browser.newContext({ viewport: { width: 1100, height: 760 }, reducedMotion: "reduce" })).newPage();
  page.setDefaultTimeout(5000);
  page.setDefaultNavigationTimeout(60000);
  const errors = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.route("**/*", serve());
  await page.goto("http://owl.test/");
  await page.waitForFunction(() => !document.querySelector("#view-agents").classList.contains("loading"));
  return { page, errors };
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(`${engine}: Ctrl+K searches pages, providers, agents and Settings`, async (t) => {
    const { page, errors } = await open(engine, t);
    await page.keyboard.press("Control+k");
    const box = page.locator(".palette-dialog");
    await box.waitFor();
    const q = box.locator(".palette-q");
    assert.equal(await q.evaluate((e) => e === document.activeElement || document.activeElement?.closest(".palette-dialog")), true, "the palette takes the focus");

    // pages and providers by their names
    await q.fill("relay");
    await box.locator(".palette-item", { hasText: "Relay" }).waitFor();
    assert.equal(await box.locator(".palette-kind").first().textContent(), "Provider");
    await page.keyboard.press("Enter");
    await page.locator(".editor").waitFor();
    assert.equal(await page.locator("#view-providers").isVisible(), true);
    // Escape closes the editor the provider opened (an empty draft asks nothing)
    await page.keyboard.press("Escape");
    await page.locator(".editor").waitFor({ state: "detached" });

    // the arrows move and Enter opens the page picked
    await page.keyboard.press("Control+k");
    await box.waitFor();
    await page.locator(".palette-q").fill("agents");
    const pageItem = box.locator(".palette-item", { hasText: "Agents" }).first();
    await pageItem.waitFor();
    await page.keyboard.press("Enter");
    await page.waitForFunction(() => document.querySelector("#nav button.on")?.dataset.view === "agents");

    // a Settings row: the same words find it there, newly opened
    await page.keyboard.press("Control+k");
    await page.locator(".palette-q").fill("Reduce motion");
    const setting = page.locator(".palette-item", { hasText: "Reduce motion" }).first();
    await setting.waitFor();
    await page.keyboard.press("Enter");
    await page.locator("#view-settings").waitFor({ state: "visible" });
    assert.equal(await page.locator("#setQ").inputValue(), "Reduce motion");
    await page.locator("#setPage-general").getByText("Reduce motion", { exact: true }).waitFor();

    // "/" puts the cursor in this page's own search box
    await navTo(page, "providers");
    await page.locator("#view-providers").waitFor({ state: "visible" });
    await page.locator("#pvQ").waitFor({ state: "visible" });
    await page.keyboard.press("/");
    assert.equal(await page.evaluate(() => document.activeElement?.id), "pvQ");
    assert.deepEqual(errors, []);
  });

  test(`${engine}: Escape closes the palette`, async (t) => {
    const { page, errors } = await open(engine, t);
    await page.keyboard.press("Control+k");
    await page.locator(".palette-dialog").waitFor();
    await page.keyboard.press("Escape");
    await page.locator(".palette-dialog").waitFor({ state: "detached" });
    assert.deepEqual(errors, []);
  });
}