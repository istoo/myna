import { GATEWAY } from "./config.js";

const zh = /^zh/i.test(navigator.language);
const say = zh
  ? { on: "已连接 Owl", off: "未连接 Owl", onHint: "通过 Owl 运行的智能体可以使用这个浏览器里的标签页。它们打开的标签页放在各自的 “Owl” 分组里。", offHint: "请确认 Owl 正在运行。扩展会自动重连。" }
  : { on: "Connected to Owl", off: "Not connected to Owl", onHint: "Agents you run through Owl can use this browser's tabs. Each puts the tabs it opens in a \"Owl\" group of its own.", offHint: "Check that Owl is running. The extension reconnects by itself." };

async function show() {
  const { status } = await chrome.storage.session.get("status");
  const on = !!(status && status.connected);
  document.getElementById("dot").classList.toggle("on", on);
  document.getElementById("state").textContent = on ? say.on : say.off;
  document.getElementById("hint").textContent = on ? say.onHint : say.offHint;
}

const gw = document.getElementById("gw");
chrome.storage.local.get("gateway").then(({ gateway }) => { gw.value = gateway || GATEWAY; });
document.getElementById("save").addEventListener("click", async () => {
  const v = gw.value.trim();
  if (!v || v === GATEWAY) await chrome.storage.local.remove("gateway");
  else await chrome.storage.local.set({ gateway: v.replace(/^http/, "ws") });
  chrome.runtime.sendMessage({ type: "reconnect" }).catch(() => {});
});
chrome.storage.onChanged.addListener((c, area) => { if (area === "session" && c.status) show(); });
show();
