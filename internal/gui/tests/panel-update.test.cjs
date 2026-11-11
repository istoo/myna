// Run with Node's test runner and Playwright on the module path; see README.md.
// The tray panel says an update in a line of its own over its tabs, as
// Settings' AutoRestart left nothing to see: owl restarting into a version
// by itself, the header's Update showed for a moment at most. A newer owl
// downloaded: what it is, a restart, and a × that hides it as the pill's
// does. owl just restarted into one: that it did, and the release notes a
// click away in the window; closing it says the notes were seen. The
// window has no such line, and opens on the notes when asked (notes=1).
// Five languages, Chromium and WebKit, 420px and 300px; the API is faked.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");

const words = {
  en: { ready: "owl 0.1.401 is downloaded", restart: "Restart to update", waiting: "owl restarts into 0.1.401 once the gateway is idle", now: "Restart now", out: "owl 0.1.401 is out", updated: "Updated to owl 0.1.400", news: "What's new", close: "Close", hide: "Hide until the next version" },
  zh: { ready: "owl 0.1.401 已下载好", restart: "重启以更新", waiting: "网关空闲后 owl 会重启到 0.1.401", now: "立即重启", out: "owl 0.1.401 已发布", updated: "已更新到 owl 0.1.400", news: "更新内容", close: "关闭", hide: "隐藏，直到下一个版本" },
  "zh-TW": { ready: "owl 0.1.401 已下載好", restart: "重啟以更新", waiting: "閘道空閒後 owl 會重啟到 0.1.401", now: "立即重啟", out: "owl 0.1.401 已釋出", updated: "已更新到 owl 0.1.400", news: null, close: null, hide: "隱藏，直到下一個版本" },
  ja: { ready: "owl 0.1.401 をダウンロードしました", restart: null, waiting: "ゲートウェイが空いたら owl は 0.1.401 で再起動します", now: null, out: "owl 0.1.401 が公開されました", updated: "owl 0.1.400 に更新しました", news: null, close: null, hide: null },
  de: { ready: "owl 0.1.401 ist heruntergeladen", restart: null, waiting: "owl startet mit 0.1.401 neu, sobald das Gateway frei ist", now: null, out: "owl 0.1.401 ist erschienen", updated: "Auf owl 0.1.400 aktualisiert", news: null, close: null, hide: null },
};

function settingsPayload(lang, ctl) {
  return {
    theme: "light", lang, tray: "panel", quotaLeft: false, currency: "usd", dock: false, dockWindow: false, proxy: "",
    noUpdatePill: false, updateSkip: ctl.skip, version: "0.1.400", dir: "~/.config/owl", gateway: "http://127.0.0.1:3425",
    proxyNow: "none", proxySource: "none", login: false, lanURLs: [], fx: { rate: 7.2, at: new Date().toISOString(), stale: false },
  };
}

// ctl is the backend: the update's state (null: none), whether this run
// is the first of a version owl updated to, and what was asked of it
function server(lang, ctl) {
  return async (route) => {
    const req = route.request(), url = new URL(req.url());
    const json = (data) => route.fulfill({ json: data });
    const asked = url.pathname + url.search.replace(/[?&]lang=[^&]*/, "").replace(/^&/, "?");
    if (req.method() === "POST" && url.pathname.startsWith("/api/")) ctl.posts.push(asked);
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:false};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: settingsPayload(lang, ctl), fx: settingsPayload(lang, ctl).fx });
    if (url.pathname === "/api/settings/update-skip") { ctl.skip = req.postDataJSON().version; return json(settingsPayload(lang, ctl)); }
    if (url.pathname === "/api/settings") return json(settingsPayload(lang, ctl));
    if (url.pathname === "/api/update") return json(ctl.update && ctl.skip !== ctl.update.latest ? { current: "0.1.400", ...ctl.update } : { state: "idle", current: "0.1.400" });
    if (url.pathname === "/api/update/install") { ctl.update = { ...ctl.update, waiting: true }; return json({ current: "0.1.400", ...ctl.update }); }
    if (url.pathname === "/api/whatsnew/seen") { ctl.updated = false; return route.fulfill({ status: 204 }); }
    if (url.pathname === "/api/whatsnew") {
      const all = url.searchParams.get("all");
      return json({ show: false, current: "0.1.400", updated: ctl.updated, releases: all ? [{ version: "0.1.400", notes: "- the notes of 0.1.400" }] : [] });
    }
    if (url.pathname === "/api/plugins") return json({ plugins: [] });
    if (url.pathname === "/api/usage/quotas") return json([]);
    if (url.pathname === "/api/groups") return json({ groups: [], models: [] });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    await route.fulfill({ body: await fs.readFile(file), contentType });
  };
}

const fresh = (over) => ({ update: null, updated: false, skip: "", posts: [], ...over });

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(engine + ": the tray panel says an update in a line of its own", async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    const open = async (lang, ctl, query, width = 420) => {
      const errors = [];
      const page = await (await browser.newContext({ viewport: { width, height: 560 }, reducedMotion: "reduce" })).newPage();
      page.setDefaultTimeout(5000);
      page.setDefaultNavigationTimeout(60000);
      page.on("pageerror", (e) => { if (!/^ResizeObserver loop completed with undelivered notifications/.test(e.message)) errors.push(e.message); });
      await page.route("**/*", server(lang, ctl));
      await page.goto("http://owl.test/" + query);
      return { page, errors };
    };
    // the line is whole, inside the panel, its buttons apart
    const fits = async (page, width) => {
      const box = await page.locator("#panelUpdate").boundingBox();
      assert(box && box.x >= 0 && box.x + box.width <= width + 0.5, `line ${JSON.stringify(box)} in a ${width}px panel`);
      for (const b of await page.locator("#panelUpdate button").all()) {
        const r = await b.boundingBox();
        assert(r.x >= box.x && r.x + r.width <= box.x + box.width + 0.5, "button inside the line");
      }
      const text = page.locator("#panelUpdate .pu-text");
      assert.equal(await text.evaluate((e) => e.scrollWidth <= e.clientWidth + 1), true, "text not cut");
    };
    for (const lang of Object.keys(words)) {
      const w = words[lang];
      for (const width of [420, 300]) {
        await t.test(`${lang} ${width}px: a downloaded owl, its restart and its ×`, async () => {
          const ctl = fresh({ update: { state: "ready", latest: "0.1.401" } });
          const { page, errors } = await open(lang, ctl, "?mode=panel", width);
          const line = page.locator("#panelUpdate");
          await line.waitFor({ state: "visible" });
          assert.equal(await line.locator(".pu-text").textContent(), w.ready);
          if (w.restart) assert.equal(await line.locator(".pu-go").textContent(), w.restart);
          if (w.hide) assert.equal(await line.locator(".pu-x").getAttribute("aria-label"), w.hide);
          await fits(page, width);
          // over the tabs: each tab keeps it
          for (const tab of ["usage", "stats", "routing"]) {
            await page.evaluate((tab) => { document.body.dataset.ptab = tab; }, tab);
            assert(await line.isVisible(), "shown on " + tab);
          }
          await page.evaluate(() => { delete document.body.dataset.ptab; });
          // its button is the header's Update: a restart (the panel asks no notes first)
          await line.locator(".pu-go").click();
          await page.waitForFunction(() => document.querySelector("#panelUpdate .pu-text")?.textContent.includes("0.1.401") && !document.querySelector("#panelUpdate .pu-x"));
          assert(ctl.posts.includes("/api/update/install"), JSON.stringify(ctl.posts));
          if (w.waiting) assert.equal(await line.locator(".pu-text").textContent(), w.waiting);
          if (w.now) assert.equal(await line.locator(".pu-go").textContent(), w.now);
          assert.deepEqual(errors, []);
        });
      }
      await t.test(`${lang}: the × hides it for that version, as the pill's does`, async () => {
        const ctl = fresh({ update: { state: "ready", latest: "0.1.401" } });
        const { page, errors } = await open(lang, ctl, "?mode=panel");
        const line = page.locator("#panelUpdate");
        await line.waitFor({ state: "visible" });
        await line.locator(".pu-x").click();
        await line.waitFor({ state: "hidden" });
        assert.equal(ctl.skip, "0.1.401");
        assert(await page.locator("#update").isHidden(), "the header's pill goes too");
        assert.deepEqual(errors, []);
      });
      await t.test(`${lang}: a newer owl out, not downloaded`, async () => {
        const ctl = fresh({ update: { state: "found", latest: "0.1.401", size: 3e7 } });
        const { page, errors } = await open(lang, ctl, "?mode=panel");
        const line = page.locator("#panelUpdate");
        await line.waitFor({ state: "visible" });
        assert.equal(await line.locator(".pu-text").textContent(), w.out);
        assert.equal(await line.locator(".pu-go").textContent(), await page.evaluate(() => t("Update")));
        assert.deepEqual(errors, []);
      });
      for (const width of [420, 300]) {
        await t.test(`${lang} ${width}px: owl restarted into a version by itself`, async () => {
          const ctl = fresh({ updated: true });
          const { page, errors } = await open(lang, ctl, "?mode=panel", width);
          const line = page.locator("#panelUpdate");
          await line.waitFor({ state: "visible" });
          assert.equal(await line.locator(".pu-text").textContent(), w.updated);
          if (w.news) assert.equal(await line.locator(".pu-go").textContent(), w.news);
          if (w.close) assert.equal(await line.locator(".pu-x").getAttribute("aria-label"), w.close);
          await fits(page, width);
          await line.locator(".pu-go").click();
          await line.waitFor({ state: "hidden" });
          assert(ctl.posts.includes("/api/window/main?view=settings&notes=1"), JSON.stringify(ctl.posts));
          assert(ctl.posts.includes("/api/whatsnew/seen"), JSON.stringify(ctl.posts));
          // seen: the next look has none
          await page.reload();
          await page.locator("#agents").waitFor({ state: "attached" });
          await page.waitForTimeout(400);
          assert(await line.isHidden(), "gone once seen");
          assert.deepEqual(errors, []);
        });
      }
      await t.test(`${lang}: its × says the notes were seen`, async () => {
        const ctl = fresh({ updated: true });
        const { page, errors } = await open(lang, ctl, "?mode=panel");
        const line = page.locator("#panelUpdate");
        await line.waitFor({ state: "visible" });
        await line.locator(".pu-x").click();
        await line.waitFor({ state: "hidden" });
        assert.deepEqual(ctl.posts.filter((p) => p.startsWith("/api/whatsnew") || p.startsWith("/api/window/main")), ["/api/whatsnew/seen"]);
        assert.deepEqual(errors, []);
      });
    }
    await t.test("the window has no such line, and opens on the notes asked for", async () => {
      const ctl = fresh({ update: { state: "ready", latest: "0.1.401" }, updated: true });
      const { page, errors } = await open("en", ctl, "?view=settings&notes=1", 900);
      await page.locator(".editor.whatsnew").waitFor();
      assert.match(await page.locator(".editor.whatsnew").textContent(), /the notes of 0\.1\.400/);
      assert(await page.locator("#panelUpdate").isHidden(), "no line in the window");
      assert.equal(new URL(page.url()).searchParams.get("notes"), null, "notes=1 is read once");
      assert.deepEqual(errors, []);
    });
    await t.test("no update and none made: no line", async () => {
      const ctl = fresh();
      const { page, errors } = await open("en", ctl, "?mode=panel");
      await page.locator("#agents").waitFor({ state: "attached" });
      await page.waitForTimeout(600);
      assert(await page.locator("#panelUpdate").isHidden());
      assert.deepEqual(errors, []);
    });
  });
}
