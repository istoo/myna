// A gateway key's value is always one owl makes, random: adding a key asks
// for its name alone, rotating asks for nothing, and neither sends a value
// of the user's own.
const assert = require("node:assert/strict");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");
const { fixture, keyAction } = require("./fixtures/caller-keys.cjs");

const words = {
  en: { create: "Create", rotate: "Rotate key", made: "Gateway key created. Use Copy on its row to connect a client.", rotated: "Gateway key rotated. Update clients on other computers." },
  zh: { create: "创建", rotate: "轮换密钥", made: "网关密钥已创建，用该行的复制按钮配置客户端。", rotated: "网关密钥已轮换，请更新其他电脑上的客户端。" },
  "zh-TW": { create: "建立", rotate: "輪換金鑰", made: "閘道金鑰已建立，用該行的複製按鈕設定用戶端。", rotated: "閘道金鑰已輪換，請更新其他電腦上的用戶端。" },
  ja: { create: "作成", rotate: "キーを再発行", made: "ゲートウェイキーを作成しました。行のコピーを使ってクライアントを接続してください。", rotated: "ゲートウェイキーを更新しました。他のコンピュータのクライアントを更新してください。" },
  de: { create: "Erstellen", rotate: "Schlüssel erneuern", made: "Zugangsschlüssel erstellt. Mit „Kopieren“ in seiner Zeile einen Client verbinden.", rotated: "Zugangsschlüssel erneuert. Aktualisieren Sie die Clients auf anderen Computern." },
};

const said = (page, m) => page.waitForFunction((m) => document.body.textContent.includes(m), m);

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const lang of Object.keys(words)) {
    for (const width of [1000, 400]) {
      test(`${engine} ${lang} ${width}px: a gateway key's value is always owl's`, async (t) => {
        const w = words[lang];
        const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
        t.after(() => browser.close());
        const page = await (await browser.newContext({ viewport: { width, height: 760 }, reducedMotion: "reduce" })).newPage();
        page.setDefaultTimeout(6000);
        page.setDefaultNavigationTimeout(60000);
        const events = [], errors = [];
        page.on("pageerror", (e) => errors.push(e.message));
        await page.route("**/*", fixture(lang, "light", events, { lan: true }));
        await page.goto("http://owl.test/?view=gateway");
        const row = page.locator('#gatewayKeys .acc[data-key="laptop"]');
        await row.waitFor();
        // adding: the name, and no field for a value
        await page.locator("#addGatewayKey").click();
        const fields = page.locator("#gatewayKeys .adding .kf input");
        await fields.first().waitFor();
        assert.equal(await fields.count(), 1);
        assert.equal(await page.locator('#gatewayKeys .adding input[type="password"]').count(), 0);
        const name = fields.first();
        assert((await name.boundingBox()).width >= 140, "the name field is wide enough to type in");
        await name.fill("Family");
        await page.getByRole("button", { name: w.create, exact: true }).click();
        await page.locator("#gatewayKeys .acc[data-key]", { hasText: "Family" }).waitFor();
        assert.deepEqual(events.filter((e) => e.action === "add-key").at(-1).body, { name: "Family" });
        await said(page, w.made);
        // rotating: a confirmation with no field
        await keyAction(page, row, w.rotate);
        const modal = page.locator("#modal");
        await modal.locator(".editor").waitFor();
        assert.equal(await modal.locator("input").count(), 0);
        await modal.getByRole("button", { name: w.rotate, exact: true }).click();
        await modal.waitFor({ state: "hidden" });
        assert.deepEqual(events.filter((e) => e.action === "rotate-key").at(-1).body, { key: "laptop" });
        await said(page, w.rotated);
        await page.waitForFunction(() => document.querySelector('#gatewayKeys .acc[data-key="laptop"]')?.textContent.includes("sk-owl-key-…"));
        assert.deepEqual(errors, []);
      });
    }
  }
}
