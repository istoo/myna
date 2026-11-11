// The tray panel's line about an update, over its tabs: a newer owl the
// header's Update offers, said in words, with its click and its ×; else,
// on the first run of a version owl updated to (Settings' AutoRestart
// included, which left nothing to see), that it did, with the release
// notes a click away in the window, until closed. app.js imports this
// only when the panel looks (panelUpdate), so it weighs nothing on the
// core, which the window pays for.
export async function panelUpdate(u) {
  const box = $("#panelUpdate"), pill = $("#update");
  const draw = (text, act, go, x, xLabel) => {
    const line = el("span", "pu-text", text);
    const parts = [line];
    if (act) {
      const b = el("button", "text pu-go", act);
      b.type = "button";
      b.onclick = go;
      parts.push(b);
    }
    if (x) {
      const c = el("button", "icon pu-x");
      c.type = "button";
      c.setAttribute("aria-label", xLabel);
      c.title = xLabel;
      c.append(svg("M4 4l8 8M12 4l-8 8", 10, 1.6));
      c.onclick = x;
      parts.push(c);
    }
    box.replaceChildren(...parts);
    if (box.hidden) box.hidden = false;
    fit();
  };
  if (u && !pill.hidden) {
    if (pill.classList.contains("busy")) return draw(pill.querySelector("span").textContent);
    const v = u.latest;
    if (u.state === "ready" && u.waiting) return draw(t("owl restarts into {v} once the gateway is idle", { v }), t("Restart now"), () => pill.click());
    if (u.state === "ready" && !u.error) return draw(t("owl {v} is downloaded", { v }), t("Restart to update"), () => pill.click(), () => skipUpdate(v), t("Hide until the next version"));
    return draw(t("{v} is out", { v: "owl " + v }), t(u.state === "ready" ? "Update failed" : "Update"), () => pill.click(), () => skipUpdate(v), t("Hide until the next version"));
  }
  const w = await api(updatePath("whatsnew")).catch(() => null);
  if (w?.updated) {
    const forget = () => api("whatsnew/seen", {}).catch(() => {}).then(() => panelUpdate());
    return draw(t("Updated to owl {v}", { v: w.current }), t("What's new"), () => { api("window/main?view=settings&notes=1", {}); forget(); }, forget, t("Close"));
  }
  if (!box.hidden) {
    box.hidden = true;
    box.replaceChildren();
    fit();
  }
}

