// Run with Node's test runner and Playwright on the module path; see README.md.
// Phone navigation and content stay usable in both languages (#391). The
// desktop screenshots match BASE_REF within a small rendering tolerance
// (origin/main by default), but for the Agents page while BASE_REF hasn't its
// 「接入」 rows yet. Until BASE_REF has the visibility manager, its new footer
// alone is excluded; agent-bulk-visibility and agent-grip cover that footer's
// layout and controls. Its height is rounded only for the pixel comparison,
// so unchanged text below keeps the baseline's subpixel antialiasing. API
// boundaries are faked; the page's scrolling helpers are left intact.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { execFileSync } = require("node:child_process");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");
const { reach, openPage } = require("./reader.cjs");
const { PNG } = require("playwright-core/lib/utilsBundle");

const assets = path.resolve(__dirname, "../assets");
const base = process.env.BASE_REF || "origin/main";
const baseline = new Map();
const at = "2026-09-30T12:00:00.000Z";
const groups = [{ name: "Claude Code", icon: "claudecode-color", calls: 123, errors: 2,
  input: 123456, output: 67890, cache_read: 543210, cache_write: 1234, reasoning: 12345, cost: 12.34, unpriced: 0 }];
const usage = { ...groups[0], path: "/test/usage.jsonl", bucket: "day", agents: groups,
  models: [{ ...groups[0], name: "a-model-with-a-long-name" }],
  series: Array.from({ length: 30 }, (_, i) => ({ ...groups[0], label: "09-" + String(i + 1).padStart(2, "0") })) };
// Populated rows are essential: an empty Agents list cannot expose clipped
// model/effort controls or make the desktop comparison cover those rows.
// Connected, so each opens on its pickers.
const modelOptions = [{ value: "claude-sonnet-4.5", label: "Claude Sonnet 4.5", icon: "claude-color", ref: "claude/claude-sonnet-4.5" },
  { value: "gpt-6.1-sol", label: "GPT-6.1-Sol", icon: "openai", ref: "openai/gpt-6.1-sol" }];
const modelField = (key, label, value = "") => ({ key, label, value, options: modelOptions });
const agentRows = [
  { id: "claude", name: "Claude Code", path: "/test/claude.json", icon: "claudecode-color", wired: true, fields: [
    modelField("model", "model", "claude-sonnet-4.5"),
    { key: "effort", label: "thinking", value: "high", options: [{ value: "high", label: "High" }] },
  ] },
  { id: "codex", name: "Codex", path: "/test/codex.toml", icon: "codex-color", wired: true, fields: [
    modelField("model", "model", "gpt-6.1-sol"),
    { key: "effort", label: "effort", value: "high", options: [{ value: "high", label: "High" }] },
    modelField("subagent", "subagents"),
    { key: "signin", label: "sign-in", value: "owl", options: [{ value: "owl", label: "owl" }, { value: "chatgpt", label: "ChatGPT" }] },
  ] },
  { id: "omp", name: "omp", path: "/test/omp.json", icon: "omp", wired: true, fields: [
    modelField("model", "model", "gpt-6.1-sol"), modelField("subagent", "subagents"),
    modelField("small", "smol"), modelField("slow", "slow"),
  ] },
];
// the labels app.js groups into omp's roles square (OMP_ROLES)
const OMP_ROLES = ["subagents", "smol", "slow", "plan", "vision", "advisor"];
const views = ["agents", "providers", "gateway", "routing", "usage", "library", "plugins", "settings"];

// BASE_REF's assets, all read in one git call the first time one is asked
// for. A git show per file blocked the fake server for each of the ~100
// the original page loads: under load its goto passed the 5s timeout, or
// its icons came after the screenshot (WebKit, the two OpenAI marks on
// Providers, 338 pixels).
const baselineHas = (file) => { try { baselineFile(file); return true; } catch { return false; } };
function baselineFile(file) {
  if (!baseline.size) {
    const root = path.resolve(assets, "../../.."), dir = "internal/gui/assets/";
    const names = execFileSync("git", ["ls-tree", "-r", "--name-only", base, dir], { cwd: root }).toString().split("\n").filter(Boolean);
    const out = execFileSync("git", ["cat-file", "--batch"], { cwd: root, input: names.map((n) => `${base}:${n}\n`).join(""), maxBuffer: 256 * 1024 * 1024 });
    let at = 0;
    for (const name of names) {
      const nl = out.indexOf(10, at);
      const size = Number(out.subarray(at, nl).toString().split(" ")[2]);
      baseline.set(name.slice(dir.length), out.subarray(nl + 1, nl + 1 + size));
      at = nl + 1 + size + 1;
    }
  }
  if (!baseline.has(file)) throw new Error(`${file} isn't in ${base}`);
  return baseline.get(file);
}

// the files being served now: a picture still on its way (an icon's mask is
// fetched by the page itself, apart from any preload) isn't drawn yet
let serving = 0;

function server(lang, original = false, opened = [], headless = false) {
  return async route => {
    const url = new URL(route.request().url());
    const json = data => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:true};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: agentRows, profiles: [], settings: { lang, theme: "light" } });
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true, calls: [], groups: [] } });
    if (url.pathname === "/api/groups") return json({ groups: [], pools: [], models: [] });
    if (url.pathname === "/api/usage/quotas") {
      // main's usage render can reveal the empty quota heading again if it
      // finishes last. Deliver the empty allowances after the totals paint,
      // so both screenshot pages compare the same completed API state.
      // Only Usage's ask waits: Routing asks too, and its ask held until the
      // test reached Usage outlived its page on a busy machine, failing
      // whichever test ran then (#10).
      const page = route.request().frame().page();
      if (await page.locator("#view-usage:not([hidden])").count().catch(() => 0))
        await page.locator("#usageAgents .row.stat").first().waitFor({ state: "attached", timeout: 30000 }).catch(() => {});
      return json([]).catch(() => {}); // the page closed meanwhile
    }
    if (url.pathname === "/api/usage") return json(usage);
    if (url.pathname === "/api/gateway/trace") return json({ mine: true, now: at, seq: 0, routes: [], totals: { requests: 0, rerouted: 0, errors: 0 } });
    if (url.pathname === "/api/gateway/history") return json({ cut: false, days: [], routes: [] });
    if (url.pathname === "/api/library") return json({ dir: "/test/library", home: "/test", agents: [], instructions: { agents: [], sets: [] }, servers: [], skills: [], projects: [], foundServers: [], foundSkills: [] });
    if (url.pathname === "/api/plugins") return json({ bun: true, plugins: [], movable: [] });
    if (url.pathname === "/api/plugins/listings") return json({ listings: [] });
    if (url.pathname === "/api/plugins/npm") return json({ npm: {} });
    if (url.pathname === "/api/plugins/market") return json({ listings: [], state: { bun: true, plugins: [] } });
    if (url.pathname === "/api/library/reveal") { opened.push(route.request().postDataJSON()); return json({}); }
    if (url.pathname.startsWith("/api/")) return json({});
    const file = url.pathname === "/" ? "index.html" : url.pathname.slice(1);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    serving++;
    try {
      let body;
      if (original) body = baselineFile(file);
      else body = await fs.readFile(path.join(assets, file));
      // The Agents and Providers pages' head and the Agents page's cards
      // (pagehead.js, page-head.test.cjs): until BASE_REF has them, the
      // desktop picture's current page is drawn without the module and its
      // styles, the block app.css ends with, so the rest is still compared.
      if (headless && file === "pagehead.js") body = "export function start() {}";
      if (headless && file === "app.css") body = body.toString().split("\n/* ---- a page's head (pagehead.js)")[0];
      await route.fulfill({ body, contentType });
    } catch { await route.fulfill({ status: 404 }); } finally { serving--; }
  };
}

async function open(browser, lang, width, { original = false, touch = true, opened = [], headless = false } = {}) {
  const page = await browser.newPage({ viewport: { width, height: 900 }, hasTouch: touch, reducedMotion: "reduce", colorScheme: "light" });
  page.setDefaultTimeout(5000);
  page.setDefaultNavigationTimeout(60000);
  await page.addInitScript(at => {
    const RealDate = Date;
    window.Date = class extends RealDate {
      constructor(...args) { super(...(args.length ? args : [at])); }
      static now() { return new RealDate(at).getTime(); }
    };
  }, at);
  await page.route("http://owl.test/**", server(lang, original, opened, headless));
  await page.goto("http://owl.test/");
  await page.locator("#nav button").first().waitFor();
  return page;
}

async function go(page, view) {
  // the page's tab, or on a phone's tab bar More and its row for a page
  // past the fifth (phonetabs.js)
  if (view === "settings") await page.locator("#prefs").click();
  await (view === "settings" ? page.locator(`#view-${view}`).waitFor() : openPage(page, view));
  if (view === "library") await page.locator("#view-library .lib-more:not(:disabled)").waitFor();
  if (view === "plugins") await page.locator("#view-plugins .pm-find").waitFor();
  if (view === "usage") {
    await page.locator("#usageAgents .row.stat").waitFor();
    await page.locator("#quotaHead").waitFor({ state: "hidden" });
  }
  await page.evaluate(() => document.fonts.ready);
  // every icon drawn: a mask's picture paints once it has come, which the
  // 800ms below doesn't promise on a loaded machine (nothing served for a
  // while, then a frame)
  for (let quiet = 0; quiet < 3; quiet = serving ? 0 : quiet + 1) await page.waitForTimeout(50);
  await page.evaluate(() => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r))));
  await page.waitForTimeout(800); // let the tab's spring and content entrance settle
}

// a screenshot once two in a row are the same: WebKit fetches a mask's
// picture first when it paints it, which a screenshot can be the first to do
// (an icon still on its way when the page was drawn), so the screenshot
// waits for what it asked for and is taken again
async function still(page) {
  let last;
  for (let i = 0; i < 10; i++) {
    const png = await page.screenshot({ animations: "disabled" });
    if (last && png.equals(last)) break;
    last = png;
    for (let quiet = 0; quiet < 3; quiet = serving ? 0 : quiet + 1) await page.waitForTimeout(50);
  }
  return last;
}

async function save(page, name) {
  if (!process.env.ARTIFACT_DIR) return;
  await fs.mkdir(process.env.ARTIFACT_DIR, { recursive: true });
  await page.screenshot({ path: path.join(process.env.ARTIFACT_DIR, name + ".png"), animations: "disabled" });
}

async function fits(page, view) {
  const dimensions = await page.locator(`#view-${view}`).evaluate(v => ({
    view: [v.scrollWidth, v.clientWidth], body: [document.body.scrollWidth, innerWidth],
    document: [document.documentElement.scrollWidth, innerWidth],
  }));
  for (const [name, [actual, available]] of Object.entries(dimensions)) assert(actual <= available + 1, `${view}: ${name} scrolls sideways (${actual} > ${available})`);
}

async function agentsFit(page) {
  for (const agent of agentRows) {
    const row = page.locator(`.row.agent[data-id="${agent.id}"]`);
    await row.waitFor();
    // opened, its pickers in the row; the last row opened can push this one
    // out of sight, so it is wheeled to once that one has slid open
    await page.waitForFunction(() => !document.getAnimations().length);
    await reach(page, row.locator(".ag-link"));
    await row.locator(".ag-link").click();
    await row.locator(".ag-exp").waitFor();
    const bad = await row.evaluate(row => {
      const box = row.getBoundingClientRect(), bad = [];
      for (const field of row.querySelectorAll(".field")) {
        const b = field.getBoundingClientRect();
        if (b.left < box.left - 1 || b.right > box.right + 1 || b.top < box.top - 1 || b.bottom > box.bottom + 1) bad.push("field cropped: " + field.dataset.key);
      }
      const model = row.querySelector('.field[data-key="model"] .v');
      if (!model?.textContent || model.scrollWidth > model.clientWidth + 1) bad.push("model name cropped");
      return bad;
    });
    assert.deepEqual(bad, [], agent.name + ": every control and model name must be visible");
    // omp's subagents, smol and slow roles share one square, as Claude
    // Code's tiers do (#1397, fc5dfa3c): it opens their menu
    const keys = [...new Set(agent.fields.map((f) => agent.id === "omp" && OMP_ROLES.includes(f.label) ? "tiers" : f.key))];
    for (const key of keys) {
      try {
        const field = row.locator(`.field[data-key="${key}"]`);
        await page.waitForFunction(() => !document.getAnimations().length);
        await reach(page, field);
        await field.click();
        await page.locator("#pop").waitFor();
      } catch (e) { throw new Error(`${agent.id} ${key} open: ${e.message.split("\n")[0]}`); }
      assert.equal(await page.locator("#pop").isVisible(), true, agent.name + ": " + key + " opens");
      // shut by a tap on the header's name: a picker opened low in a tall
      // row (Codex's holds its own settings) can cover the phone's tab bar,
      // and the page's lead line can be scrolled under the header's tabs
      const brand = await page.locator("header.top .brand").boundingBox();
      await page.mouse.click(brand.x + brand.width / 2, brand.y + brand.height / 2);
      try { await page.locator("#pop").waitFor({ state: "hidden" }); }
      catch (e) { throw new Error(`${agent.id} ${key} shut: ${e.message.split("\n")[0]}`); }
    }
  }
}

async function usageFits(page) {
  const bad = await page.evaluate(() => {
    const bad = [];
    for (const r of document.querySelectorAll(".row.stat")) {
      const box = r.getBoundingClientRect();
      for (const e of r.querySelectorAll(".num, .cost")) {
        const b = e.getBoundingClientRect();
        if (b.left < box.left || b.right > box.right + 1 || e.scrollWidth > e.clientWidth + 1) bad.push("tokens or cost cropped: " + e.textContent);
      }
    }
    for (const e of document.querySelectorAll("#stats small")) if (e.scrollWidth > e.clientWidth + 1) bad.push("stat explanation cropped: " + e.textContent);
    let last;
    const dates = [];
    for (const e of document.querySelectorAll("#chart .labels span")) {
      if (!e.textContent || getComputedStyle(e).visibility === "hidden") continue;
      dates.push(e.textContent);
      const text = document.createRange(); text.selectNodeContents(e);
      const b = text.getBoundingClientRect();
      if (last && b.left < last.right + 2) bad.push("date labels overlap");
      const box = e.parentElement.getBoundingClientRect();
      if (b.left < box.left - 1 || b.right > box.right + 1) bad.push("date label outside chart");
      last = b;
    }
    if (dates.length < 3 || dates[0] !== "09-01" || dates[dates.length - 1] !== "09-30") bad.push("chart must retain its endpoints and intermediate dates");
    return bad;
  });
  assert.deepEqual(bad, []);
  for (const selector of ["#usageAgents .cost", "#usageModels .cost"]) assert.match(await page.locator(selector).first().textContent(), /≈\$12\.34/);
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(engine + ": owl web on phones", async t => {
    assert(["chromium", "webkit"].includes(engine), "BROWSER must be chromium or webkit");
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    for (const lang of ["en", "zh"]) {
      for (const width of [360, 390, 430, 820]) {
        await t.test(`${lang} ${width}: every tab and page fits`, async () => {
          const opened = [], errors = [], failures = [], page = await open(browser, lang, width, { opened });
          page.on("pageerror", e => errors.push(e.message));
          try {
            for (const view of views) {
              try {
              await go(page, view);
              await save(page, `${engine}-${lang}-${width}-${view}`);
              await fits(page, view);
              assert.equal(await page.locator(`#view-${view}`).isVisible(), true);
              if (view === "agents") await agentsFit(page);
              if (view === "usage") await usageFits(page);
              if (view === "library") {
                await Promise.all([
                  page.waitForResponse(r => new URL(r.url()).pathname === "/api/library/reveal"),
                  page.getByRole("button", { name: lang === "en" ? "Library folder" : "资源库文件夹", exact: true }).click(),
                ]);
                assert.deepEqual(opened, [{ path: "/test/library" }], "Library folder must be reachable");
              }
              } catch (e) { failures.push(view + ": " + e.message); }
            }
            assert.deepEqual(errors, []);
            assert.deepEqual(failures, []);
          } finally { await page.close(); }
        });
      }
      await t.test(lang + ": idle scripts cannot scroll, touch momentum can, then protection returns", async () => {
        const page = await open(browser, lang, 390);
        try {
          // Settings now has separate parts. Its Usage part in a short phone
          // viewport has real content to scroll; General fits without scrolling.
          await page.setViewportSize({ width: 390, height: 600 });
          await page.goto("http://owl.test/?view=settings&tab=usage");
          await go(page, "settings");
          await page.waitForTimeout(500);
          const view = page.locator("#view-settings");
          assert(await view.evaluate(v => v.scrollHeight > v.clientHeight + 400));
          await view.evaluate(v => { v.scrollTop = 300; });
          await page.waitForTimeout(100);
          assert.equal(await view.evaluate(v => v.scrollTop), 0, "idle code cannot move a phone page");
          await view.evaluate(async v => {
            for (const name of ["touchstart", "touchmove", "touchend"]) v.dispatchEvent(new Event(name, { bubbles: true }));
            // OS momentum outlasts the original 250ms input window; the scroll
            // events arrive 60ms apart, as they do while a finger's fling slows.
            for (let i = 1; i <= 10; i++) {
              await new Promise(r => setTimeout(r, 60));
              v.scrollTop = i * 30;
            }
          });
          await page.waitForTimeout(100);
          assert.equal(await view.evaluate(v => v.scrollTop), 300, "continuous momentum must not be pulled back");
          await page.waitForTimeout(350);
          await view.evaluate(v => { v.scrollTop = 450; });
          await page.waitForTimeout(100);
          assert.equal(await view.evaluate(v => v.scrollTop), 300, "protection returns after momentum stops");
        } finally { await page.close(); }
      });
      for (const width of [900, 1280]) {
        await t.test(`${lang} ${width}: desktop matches ${base}`, async () => {
          const current = await open(browser, lang, width, { touch: false, headless: !baselineHas("pagehead.js") });
          const original = await open(browser, lang, width, { touch: false, original: true });
          try {
            for (const view of views) {
              if (view === "agents" && !baselineFile("app.js").includes("function connectPanel")) continue;
              await go(current, view); await go(original, view);
              await current.mouse.move(0, 0); await original.mouse.move(0, 0);
              // Removing hover can repaint the tab's shadow in a later frame.
              await current.waitForTimeout(350); await original.waitForTimeout(350);
              // The new management footer intentionally differs from main.
              // Compare the rest at its new position: the footer also moves
              // Profiles down. Compare the footer too once main contains it.
              // Do not skip the Agents page or increase the pixel tolerance.
              let addedFooter = view === "agents" && !baselineFile("app.js").includes("function agentsManageButton")
                ? await current.locator("#agentsManage").boundingBox() : null;
              if (addedFooter) {
                // A fractional new height changes text antialiasing below it,
                // even after translating the screenshot back. Normalize only
                // that new height; the footer's real layout is tested separately.
                await current.locator("#agentsManage").evaluate((bar, height) => {
                  bar.style.boxSizing = "border-box";
                  bar.style.height = Math.ceil(height) + "px";
                }, addedFooter.height);
                addedFooter = await current.locator("#agentsManage").boundingBox();
              }
              // Plugins' Built-in tab is new: until BASE_REF has it, the
              // page is compared with that tab taken out of its strip (the
              // sidebar's list follows the strip); builtins.test.cjs covers
              // the tab and its list.
              if (view === "plugins" && !baselineFile("plugins.js").includes("function drawBuiltin")) {
                await current.locator("#view-plugins .lib-tabs .opt").first().evaluate((b) => b.remove());
                await current.waitForTimeout(350);
              }
              // Usage's lists fold under their headings and the tokens tile's
              // label is a pop-up (#2378): until BASE_REF has them, each is
              // put back to the plain label it was, and the rest of the page
              // is compared as ever; usage-overview-fold.test.cjs covers them.
              if (view === "usage" && !baselineFile("plugins/usage/page.js").toString().includes("function mountUsageFolds")) {
                await current.evaluate(() => {
                  for (const b of document.querySelectorAll("#usagePane .row-head .fold, #stats .kpi-pick")) b.replaceWith(b.querySelector(".label, span"));
                });
                await current.waitForTimeout(350);
              }
              // Replies' Short Claude system prompt row is new: until BASE_REF
              // has it, the page is compared with the row taken out;
              // claude-lean-prompt-setting.test.cjs covers the row.
              if (view === "settings" && !baselineFile("app.js").includes("claudeLeanRow")) {
                await current.locator("#claudeLeanRow").evaluate((r) => r.remove());
                await current.waitForTimeout(350);
              }
              // Replies' Claude Code's interactive mode row is new: until
              // BASE_REF has it, the page is compared with the row taken
              // out; claude-tui-setting.test.cjs covers the row.
              if (view === "settings" && !baselineFile("app.js").includes("claudeTuiRow")) {
                await current.locator("#claudeTuiRow").evaluate((r) => r.remove());
                await current.waitForTimeout(350);
              }
              // A setting row's line wraps whole (arculongus on Discord):
              // until BASE_REF has that, the page is compared with the lines
              // held to one line as they were; settings-lines-wrap.test.cjs
              // covers the wrapping.
              if (view === "settings" && !baselineFile("app.css").includes("arculongus")) {
                await current.addStyleTag({ content: ".row.pref .sub { white-space: nowrap; overflow-wrap: normal; }" });
                await current.waitForTimeout(350);
              }
              // The sidebar's button sits ahead of the owl in both layouts
              // (ztf4080 on X), so the header beside the sidebar no longer
              // starts with it: until BASE_REF has it, the button, the owl
              // and the header beside the sidebar are left out of both pictures and the rest
              // compared as ever; side-toggle.test.cjs covers where they are.
              if (!baselineFile("app.js").includes("append(btn, brand)")) {
                for (const p of [current, original]) await p.addStyleTag({ content: "#sideToggle, .brand, body.side-nav .top { visibility: hidden !important; }" });
                await current.waitForTimeout(350);
              }
              // The light theme's quiet words are darker so they reach 4.5:1
              // (light-contrast.test.cjs): until BASE_REF has them, both
              // pictures use the new values and the rest is compared as ever.
              if (!baselineFile("app.css").includes("--muted: #5f5f67")) {
                for (const p of [current, original]) await p.addStyleTag({ content: "html, body.panel { --muted: #5f5f67 !important; --faint: #66666e !important; }" });
                await current.waitForTimeout(350);
              }
              // A row with two states is a switch now (settings-switch.test.cjs):
              // until BASE_REF has it, the controls in Settings' rows are
              // left out of both pictures, so the rows and what they say are
              // compared as ever; the control itself is covered by its test.
              if (!baselineFile("app.js").includes("function settingSwitch")) {
                for (const p of [current, original]) await p.addStyleTag({ content: "#view-settings .row.pref :is(.segs, .setting-switch) { display: none !important; }" });
                await current.waitForTimeout(350);
              }
              // A segmented control's picked option is heavier: until
              // BASE_REF has it, both pictures keep the old weight.
              if (!baselineFile("app.css").includes(".opt.on { color: var(--fg); font-weight: 560")) {
                for (const p of [current, original]) await p.addStyleTag({ content: ".opt.on { font-weight: 400 !important; }" });
                await current.waitForTimeout(350);
              }
              // Gateway's line now says "one address for every API format" (the review
              // asked for it in place of "five APIs, one URL"): until BASE_REF
              // has it, both pictures show the same line; gateway-systemone
              // checks the words.
              if (view === "gateway" && !baselineFile("plugins/gateway/client.js").toString().includes("one address for every API format")) {
                for (const p of [current, original]) await p.evaluate(() => {
                  const s = document.querySelector("#gateway .who > .sub");
                  if (s) s.textContent = "N models · no agent routed through it yet · one address for every API format";
                });
                await current.waitForTimeout(350);
              }
              // Usage's cost note now says "at your model prices" (the review asked
              // for it in place of "effective prices"): until BASE_REF has
              // it, both pictures show the same words.
              if (view === "usage" && !baselineFile("plugins/usage/page.js").toString().includes("at your model prices")) {
                for (const p of [current, original]) await p.evaluate(() => {
                  const s = document.querySelector("#usageCost > span:not([class])");
                  if (s) s.textContent = "at your model prices";
                });
                await current.waitForTimeout(350);
              }
              // The sidebar's Search takes a row of its own above the pages:
              // until BASE_REF has it, both pictures show the sidebar without
              // it, the pages back at the height they had.
              if (!baselineFile("index.html").toString().includes('id="sideFind"')) {
                for (const p of [current, original]) await p.evaluate(() => document.querySelector(".side-find-row")?.remove());
                await current.waitForTimeout(350);
              }
              // Usage with no calls now says why and gives the two ways to
              // fill it: until BASE_REF has that, both pictures show the one
              // line it had.
              if (view === "usage" && !baselineFile("plugins/usage/page.js").toString().includes("none-acts")) {
                const html = await original.evaluate(() => document.querySelector("#stats")?.innerHTML || "");
                await current.evaluate((h) => {
                  const s = document.querySelector("#stats");
                  if (!s) return;
                  s.innerHTML = h;
                  const n = s.querySelector(".none");
                  if (n) n.setAttribute("style", "display:block;padding:18px 16px;color:var(--muted);text-align:center;line-height:1.5");
                }, html);
                await current.waitForTimeout(350);
              }
              // The window's polish (nav-polish.test.cjs): one set of sidebar
              // icons, a provider's small switch, sections' names in their own
              // case, 28px finds, a usage row's breakdown that gives way. Until
              // BASE_REF has it, the current page takes the original's icons
              // and sizes back, so the rest of the page is still compared.
              if (!baselineFile("app.css").toString().includes(".provider .pswitch.on i { transform: translateX(12px); }")) {
                const icons = await original.evaluate(() => [...document.querySelectorAll("#nav button[data-view] > svg.nav-ic, #prefs > svg")].map((s) => [s.closest("button").dataset.view || "", s.outerHTML]));
                await current.evaluate((icons) => {
                  for (const [v, html] of icons) {
                    const s = document.querySelector(v ? `#nav button[data-view="${v}"] > svg.nav-ic` : "#prefs > svg");
                    if (s) s.outerHTML = html;
                  }
                }, icons);
                await current.addStyleTag({ content: `
                  .provider .pswitch { width: 32px; height: 19px; } .provider .pswitch i { width: 15px; height: 15px; } .provider .pswitch.on i { transform: translateX(13px); }
                  .label, .kpi span, .sess-chart-head .label, .sess-card-head .label, .warm-tabs button, .set-searching .set-page::before { font-size: 11px; letter-spacing: .05em; text-transform: uppercase; }
                  .sess-filter { height: 26px; padding: 0 9px; border-radius: 7px; }
                  body.side-nav #prefs svg { width: 15px; height: 15px; margin: 0 .5px; }
                  .row.stat .num { flex: none; } .row.stat .num small { max-width: none; overflow: visible; }` });
                await current.waitForTimeout(350);
              }
              const before = PNG.sync.read(await still(original));
              const after = PNG.sync.read(await still(current));
              const sameSize = before.width === after.width && before.height === after.height;
              let changedPixels = 0;
              // Chromium can repaint antialiased edges differently even when
              // comparing HEAD with itself. Count a pixel once only when one
              // of its channels changes by more than 48; size changes always fail.
              const scale = before.width / width;
              if (sameSize) for (let i = 0; i < before.data.length; i += 4) {
                const row = Math.floor(i / 4 / before.width), col = (i / 4) % before.width;
                // Only the footer's own column moves: the sidebar beside the
                // list (at 1280) stays where it was.
                const shifted = addedFooter && row >= Math.floor(addedFooter.y * scale)
                  && col >= Math.floor(addedFooter.x * scale) && col < Math.ceil((addedFooter.x + addedFooter.width) * scale)
                  ? Math.round(addedFooter.height * scale) : 0;
                // Both images retain their real viewport size. The added
                // footer pushes this last slice below the current viewport.
                if (row + shifted >= after.height) continue;
                const j = i + shifted * after.width * 4;
                for (let channel = 0; channel < 4; channel++) {
                  if (Math.abs(before.data[i + channel] - after.data[j + channel]) > 48) {
                    changedPixels++;
                    break;
                  }
                }
              }
              const same = sameSize && changedPixels <= 300;
              if (!same) {
                await save(original, `${engine}-${lang}-${width}-${view}-desktop-before`);
                await save(current, `${engine}-${lang}-${width}-${view}-desktop-after`);
              }
              assert(same, `${view}: desktop screenshot changed (${sameSize ? changedPixels + " pixels differ by more than 48; limit 300" : "dimensions differ"})`);
            }
          } finally { await current.close(); await original.close(); }
        });
      }
    }
  });
}
