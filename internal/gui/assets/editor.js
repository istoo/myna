// The provider editor: the Providers page's largest piece, in a classic
// script app.js loads the first time an editor is drawn (editor.js is not in
// index.html, so the core's budget doesn't count it). It shares app.js's
// globals: a classic script's top level is the same global lexical
// environment, so el, t, api, draft, editing and the rest are in reach.
function drawEditor(p, presetID) {
  const pr = presetID ? providers.presets.find((x) => x.id === presetID) : p?.preset ? providers.presets.find((x) => x.id === p.preset) : null;
  const isNew = !p, custom = !pr && !p?.account, decides = !!(p?.decide || pr?.decide);
  // a preset already added is added again only through "Add another": one
  // more provider of it, under a name and id of its own
  const another = isNew && !!pr?.added;
  // an editor opened afresh has Names, prices & levels folded: it stayed open in
  // every editor after, until owl was restarted (Hu9956, #868)
  if (!draft) naming = unpickedShown = picksLeft = null;
  draft = draft || (p
    ? draftOf(p)
    : pr
      ? { id: pr.id, name: pr.name, preset: pr.id, key: "", chosen: [], extra: [], headers: [] }
      : { id: "", name: "", preset: "", chat: "", responses: "", anthropic: "", decide: "", catalog: "", key: "", api: "openai", chosen: [], extra: [], headers: [], icon: "" });
  const ed = el("div", "editor roomy" + (isNew ? " new" : ""));
  ed.onclick = (e) => e.stopPropagation();

  {
    const h = el("div", "ehead");
    const copyOf = draft.copyOf && providers.providers.find((x) => x.id === draft.copyOf);
    h.append(icon(p?.icon || (copyOf && draft.icon) || pr?.icon || "generic"), el("b", "", p ? p.name : copyOf ? t("Copy of {name}", { name: copyOf.name }) : pr ? pr.name : t("Custom provider")));
    if (pr?.kind === "partner") { if (partnerNote(pr)) h.append(el("span", "note", partnerNote(pr))); } else if (pr?.note) h.append(el("span", "note", t(pr.note)));
    h.append(el("span", "grow"));
    // a region's own page (Tencent Cloud's), following the region picked
    // below; a partner's /go link is named by where it goes (siteHost)
    const at = regionAt(pr, draft)?.website ? regionAt(pr, draft) : pr;
    const site = at?.website || p?.website || hostSite(p);
    if (site) {
      const b = el("button", "link site", hostOf(at?.siteHost || site) + " ↗");
      b.dataset.url = site;
      b.onclick = () => api("open", { url: b.dataset.url });
      h.append(b);
    }
    if (p) h.append(providerSwitch(p));
    ed.append(h);
    if (p?.off) ed.append(el("div", "hint off-note", t("Switched off: agents aren't given its models and no request goes to it. Its keys and settings are kept; switch it on to use it again.")));
  }

  // a vendor's global and China endpoints are presets of their own, one row
  // in the add sheet: here the new provider picks between them, the key kept
  const pair = isNew && pr ? (globalOf(pr) ? [globalOf(pr), pr] : chinaOf(pr) ? [pr, chinaOf(pr)] : null) : null;
  if (pair) {
    const seg = el("div", "segs area");
    pair.forEach((x, i) => {
      const b = el("button", "opt" + (x.id === pr.id ? " on" : ""), t(i ? "China" : "Global"));
      b.title = hostOf(x.chat || x.responses || x.anthropic) + (x.added ? " · " + t("Added") : "");
      b.onclick = () => {
        if (x.id === pr.id) return;
        editing = { preset: x.id };
        draft = { id: x.id, name: x.name, preset: x.id, key: draft.key, chosen: [], extra: [], headers: [] };
        renderProviders();
      };
      seg.append(b);
    });
    queueMicrotask(() => slide(seg, "area"));
    ed.append(...field(t("Region"), seg, ""));
  }

  // who uses it: just the agents already pointed here, so a click changes
  // one's model. Pointing a new agent at the provider happens in the Agent
  // tab's picker, not here.
  if (p) {
    const on = p.agents.filter((a) => a.current);
    if (on.length) {
      const chips = el("div", "achips");
      for (const a of on) {
        const c = el("button", "achip on");
        c.append(icon(a.icon), el("span", "n", a.name), el("span", "m", a.group || a.model));
        c.title = a.group ? t("{agent} is on the routing group {group}, {model} here among its members — click to change", { agent: a.name, group: a.group, model: a.model })
          : t("{agent} is on {model} — click to change", { agent: a.name, model: a.model });
        c.onclick = (ev) => pickForAgent(a, p, c, ev);
        chips.append(c);
      }
      ed.append(...field(t("Agents"), chips, ""));
    }
  }

  let name, url;
  if (another) {
    name = input(draft.name === pr.name ? "" : draft.name, t("e.g. {name} · Work", { name: pr.name }));
    const hint = el("div", "hint");
    const idHint = () => { hint.textContent = t("id {id} — a number is added if it is taken", { id: draft.id }); };
    name.oninput = () => { draft.name = name.value.trim() || pr.name; draft.id = slug(name.value) || pr.id; idHint(); };
    idHint();
    const wrap = el("div");
    wrap.append(name, hint);
    ed.append(el("label", "", t("Name")), wrap);
  }
  // an added preset's name is the user's, as a custom provider's is: two
  // of a vendor's sites (Kimi and Kimi (China)) named as they tell them
  // apart, in the list and every picker (yapechan on Discord). Left empty,
  // it is the preset's again
  if (p && pr && !p.account) {
    name = input(draft.name === pr.name ? "" : draft.name, pr.name);
    name.className = "provider-name";
    name.oninput = () => { draft.name = name.value.trim() || pr.name; };
    ed.append(...field(t("Name"), name));
  }
  // an added provider's id can change: its models are picked by it, and
  // the agents and routing groups on them move to the new one
  const idField = () => {
    const idIn = input(draft.id, p.id);
    const hint = el("div", "hint");
    const idOf = () => slug(draft.id) || p.id;
    const show = () => {
      hint.textContent = t(decideOnly(p) ? "Routing groups name its models as {id} for their classifier" : "Agents pick its models as {id}", { id: idOf() + "/…" }) +
        (idOf() !== p.id ? " · " + t("agents and routing groups on {id} move to it", { id: p.id + "/…" }) : "");
    };
    idIn.oninput = () => { draft.id = idIn.value; show(); };
    idIn.onblur = () => { draft.id = idIn.value = idOf(); show(); };
    show();
    const w = el("div");
    w.append(idIn, hint);
    ed.append(el("label", "", t("ID")), w);
  };
  if (p && !p.account && !custom) idField();
  let fillEndpoints = () => {};
  // what a pasted curl command fills beyond the key (curlOf): a custom
  // provider's Base URL and its API, and a new one's models
  let curlBase = null, curlModel = null;
  // a URL typed and not saved that a protocol switch carried to another
  // protocol: the protocol it was typed for, and the URL as typed
  let carried = null;
  // the Web search row, shown while there is an API it can search on
  const searchable = () => !!((draft.anthropic || "").trim() || (draft.responses || "").trim());
  let showSearch = () => {};
  // the Redaction row, shown while every address is on this computer or
  // the local network
  let showLocal = () => {};
  // the Certificate row, shown while a provider of the user's own has an
  // https:// address (#2383)
  let showTLS = () => {};
  // the Key header row, shown while the provider has a third-party
  // Anthropic API, which is sent its key in both headers by default
  let showAuth = () => {};
  // the Compaction row, shown while the provider has a Responses API
  // (Provider.CarriesCompaction): Codex's own compaction can go only there
  let showCompaction = () => {};
  const carriesCompaction = () => !!((draft.responses ?? (isNew && pr ? pr.responses : "")) || "").trim();
  if (custom) {
    name = input(draft.name, t("e.g. My Relay"));
    name.oninput = () => { draft.name = name.value; if (isNew) draft.id = slug(name.value); };
    ed.append(...field(t("Name"), name));
    if (p) idField();

    // the base URL is the one the chosen protocol is asked at; a vendor
    // that serves only the Responses API is added (and tested) with that
    // alone, since /chat/completions would only fail (#73)
    const seg = el("div", "segs base-apis");
    for (const [v, hint] of [["openai", "…/v1 — chat completions, and responses when the vendor has it"], ["responses", "…/v1 — for a vendor that serves only the Responses API, not chat completions"], ["anthropic", "the root URL, what ANTHROPIC_BASE_URL would take"], ["gemini", "…/v1beta — Google's Gemini API (generateContent), or one that answers as it does; the key goes in x-goog-api-key"], ["decide", "a decision API's root, POST …/systemone under it: its models are a routing group's classifier, never an agent's"]]) {
      const b = el("button", "opt" + (draft.api === v ? " on" : ""), t(API_NAMES[v]));
      b.title = t(hint);
      b.onclick = () => {
        if (draft.api === v) return;
        // each protocol keeps its own URL (#105); an unsaved one moves to a
        // protocol without one, in its spelling (#73), and goes back once
        // that protocol is given its own (#1231)
        const from = apiField[draft.api], to = apiField[v];
        carried = null;
        if (!draft[to] && draft[from] && draft[from] !== (p?.[from] || "")) {
          carried = { from, value: draft[from] };
          draft[to] = respellURL(draft[from], v);
          draft[from] = p?.[from] || "";
        }
        showApi(v);
      };
      b.dataset.api = v;
      seg.append(b);
    }
    // the protocol the base URL is for, shown as picked: by its button,
    // or by a detection taken (useDetected)
    const showApi = (v) => {
      draft.api = v;
      url.value = draft[apiField[v]] || "";
      for (const x of seg.querySelectorAll(".opt")) x.classList.toggle("on", x.dataset.api === v);
      slide(seg, "api");
      url.placeholder = urlHint(v);
      fillEndpoints();
      showSearch();
      showLocal();
      showTLS();
      showAuth();
      showCompaction();
    };
    queueMicrotask(() => slide(seg, "api"));
    url = input(draft[apiField[draft.api]], urlHint(draft.api), "url");
    url.classList.add("base-url");
    url.oninput = () => {
      draft[apiField[draft.api]] = url.value;
      if (carried) { draft[carried.from] = carried.value; carried = null; fillEndpoints(); }
      showSearch(); showLocal(); showTLS(); showAuth(); showCompaction();
    };
    const urlWrap = el("div", "stack");
    urlWrap.append(seg, url);
    // the APIs that answered a detection, taken for the provider: their
    // URLs set, one that wasn't found there (404, 405) cleared, and the
    // base URL's protocol one of those that answered
    const useDetected = (rs) => {
      carried = null;
      for (const x of rs) {
        if (x.ok) draft[x.protocol] = x.base;
        else if ((x.status === 404 || x.status === 405) && (draft[x.protocol] || "").trim().replace(/\/+$/, "") === x.base) draft[x.protocol] = "";
      }
      const v = draft[apiField[draft.api]] ? draft.api : ["openai", "responses", "anthropic", "gemini", "decide"].find((a) => draft[apiField[a]]);
      showApi(v || draft.api);
      draft.onModelPrefs?.(); // the APIs a model can be given follow the URLs
    };
    curlBase = (api, base) => {
      carried = null;
      draft[apiField[api]] = base;
      showApi(api);
      draft.onModelPrefs?.();
    };
    // Detect asks OpenAI's, Anthropic's and Gemini's APIs (#1346)
    urlWrap.append(detectAPIs(p, () => url.value, useDetected));
    ed.append(...field("Base URL", urlWrap));
  }

  // the Cline API serves a model from whichever host its AI gateway picks;
  // pinned, its DeepSeek models are served by DeepSeek's own API alone and
  // its GLM models by Z.ai's (sdytman on Discord), which keeps their prompt
  // cache (White Immortal on Discord). One tick for both makers. ClinePass,
  // or the Cline plugin's provider (ARNO on Discord)
  const upstreamField = () => {
    const [ptk, pcb] = tick(t("DeepSeek and GLM models only from their makers' own APIs"), !!draft.pinUpstream);
    ptk.classList.add("pin-upstream");
    pcb.onchange = () => { draft.pinUpstream = pcb.checked; };
    return field(t("Upstream"), ptk, t("Cline serves a model from any host its gateway picks; pinned, a request for a DeepSeek model goes only to DeepSeek and one for a GLM model only to Z.ai, which keeps its prompt cache, and fails when that API can't take it"));
  };

  // the settings few need — headers, proxy, limits, windows, fallback —
  // folded under Advanced, out of the way of the key and the models: open
  // when one of them is set, or as it was left in this editor
  const adv = edAligned(el("details", "more advanced"));
  adv.open = advanced && advanced.of === editing ? advanced.open : advancedSet(draft, p);
  const of = editing;
  adv.onfold = (open) => { advanced = { of, open }; };
  adv.append(el("summary", "", t("Advanced")));
  const advIn = el("div", "inner");
  adv.append(advIn);

  const unmaskRow = (hint, urls) => {
    const [ltk, lcb] = tick(t("Send requests unmasked"), !!draft.unredacted);
    const row = field(t("Redaction"), ltk, hint), note = el("div", "hint unmasked-away");
    row[1].append(note);
    showLocal = () => {
      const host = draft.unredacted && (urls ? awayHost(urls()) : p.name);
      note.hidden = !(note.textContent = host ? t("{host} isn't on this computer or your local network: turned on, requests go to it unmasked.", { host }) : "");
      if (draft.unredacted && urls) askLANNames(urls(), showLocal);
    };
    lcb.onchange = () => { draft.unredacted = lcb.checked; showLocal(); };
    showLocal();
    return row;
  };

  let routePick = null;
  if (p?.account) {
    // the sign-in belongs to the agent; owl only borrows it
    const a = p.account;
    if (subOf(a.agent)) {
      // no Claude Code on Windows: the gateway runs the one in WSL
      if (a.agent === "claude" && !p.switchOnly) ed.append(claudeModeRow());
      const via = a.wsl ? t("Runs through WSL · {distro}: Claude Code there signs in and answers; Windows has none of its own.", { distro: a.wsl }) + " " : "";
      ed.append(...field(t("Accounts"), renderAccounts(a, p), via + (p.routing ? t("Tick every account to use; Routing says how requests spread over them.") : subOf(a.agent).single ? t("{agent} keeps one account; the gateway runs it for every request. Signing in to another replaces it.", { agent: a.agentName }) : subOf(a.agent).own && (a.logins || []).some((l) => l.own) ? t("The gateway uses the first. Tick more and it moves on to the next when the one before it is out of quota. {agent} itself stays signed in as it is.", { agent: a.agentName && a.agentName !== a.agent ? a.agentName : p.name }) : subOf(a.agent).own || subOf(a.agent).plugin ? t("The gateway uses the first. Tick more and it moves on to the next when the one before it is out of quota.") : t("{agent} signs in to the first. Tick more and owl moves on to the next when the one before it is out of quota. Sessions already running keep theirs until restarted.", { agent: a.agentName }))));
      if ((a.logins || []).filter((l) => l.active || l.on).length > 1) { routePick = renderRouting(p); ed.append(...routePick.nodes); }
      if (p.move) ed.append(...renderMove(p));
    } else {
      const acct = el("div", "acct");
      acct.append(icon(a.agentIcon), el("span", "n", a.user), hideAccounts.eye(a.user, p.id), el("span", "plan", accountPlan(a)));
      ed.append(...field(t("Account"), acct, t("{agent}'s sign-in, read from its own files. Sign out there and this provider goes away.", { agent: a.agentName })));
    }
    const modelsField = field(t("Models"), renderModels(p), pluginFilterNote(subOf(a.agent)));
    if (routePick) routePick.slot.append(...modelsField);
    else ed.append(...modelsField);
    if (p.drawIds?.length) ed.append(...renderDrawers(p));
    if (p.filmIds?.length) ed.append(...renderVideomakers(p));
    // a subscription's window too: Codex's backend says 272K for models
    // that take 872K (#120)
    const cx = input(draft.contexts || "", t("e.g. 128k · or gpt-6=1m, comma separated"));
    advIn.append(...field(t("Context window"), contextPicks(p, cx), t("How long a request the models take, told to the agents; empty leaves it to the vendor and models.dev")));
    advIn.append(...outputField());
    advIn.append(...compactField());
    if (p.carriesCompaction) advIn.append(...nativeCompactionField(() => true)[0]);
    advIn.append(...field(t("Fallback"), renderFallback(p), fallbackHint(p)));
    const proxies = el("div", "stack");
    proxies.append(proxyPicker());
    const perAccount = subOf(a.agent) ? accountProxyPicker(a, p) : null;
    if (perAccount) proxies.append(perAccount);
    advIn.append(...field(t("Proxy"), proxies));
    advIn.append(...concurrencyField(p));
    advIn.append(...priceRateField(), ...countFreeField(), ...rateWindowsField());
    if (p.cline) advIn.append(...upstreamField());
    advIn.append(...unmaskRow(""));
    ed.append(...pluginOptionsField(subOf(a.agent)));
    // a plugin's provider is reached inside owl: its plugin:// URLs go
    // nowhere to show or test
    const urls = [p.chat, p.responses, p.anthropic].filter(Boolean);
    if (urls.length && !urls.some((u) => u.startsWith("plugin://"))) ed.append(...field(t("Endpoints"), renderEndpoints(p, p)));
    ed.append(adv);
  mergeRouteAdv(adv, advIn, routePick);
    const bar = el("div", "bar");
    // removing only hides it from owl; the agent stays signed in
    const del = el("button", "text danger", t("Remove"));
    del.title = t("{agent} stays signed in; owl just stops offering it", { agent: a.agentName });
    del.onclick = () => providerAction("delete", { id: p.id }, t("{name} removed", { name: p.name }));
    bar.append(del, el("span", "grow"));
    const cancel = el("button", "text", t("Cancel"));
    cancel.onclick = cancelEdit;
    const saveBtn = el("button", "text primary", t("Save"));
    saveBtn.onclick = () => {
      if (pickingNone()) return editorError(t("Pick a model first."), "warn");
      const cx = parseContexts(draft.contexts || "");
      if (cx.error) return contextError(ed, cx.error);
      const ox = parseContexts(draft.outputs || "");
      if (ox.error) return outputError(ed, ox.error);
      const cpx = parseContexts(draft.compacts || "");
      if (cpx.error) return compactError(ed, cpx.error);
      const proxy = proxyOfDraft();
      if (proxy === null) return proxyError(ed);
      const own = accountProxiesOfDraft();
      if (own.missing) {
        const line = [...ed.querySelectorAll(".acct-proxy")].find((x) => x.dataset.user.toLowerCase() === own.missing);
        shown(line?.querySelector(".proxy-url"))?.focus(NO_SCROLL);
        return editorError(t("Proxy of {user}: type its address, like http://127.0.0.1:7890", { user: line?.dataset.user || own.missing }), "warn");
      }
      const maxConcurrency = concurrencyOfDraft();
      if (maxConcurrency === undefined) return concurrencyError(ed);
      const maxAccounts = concurrencyOfDraft("maxAccounts");
      if (maxAccounts === undefined) return accountsError(ed);
      const queue = queueOfDraft();
      if (queue.bad) return queueError(ed, queue.bad);
      const priceRate = priceRateOfDraft();
      if (priceRate === undefined) return priceRateError(ed);
      const rateWindows = self.rateWindowsOfDraft?.();
      if (rateWindows === false) return rateWindowsError(ed);
      if (priceTypedError(ed)) return;
      const picked = accountModelsDraft(p);
      if (picked === false) return pickError();
      const body = { id: p.id, models: chosenIds(), noModels: noneChosen(), bestInFamily: bestChosen(), unlisted: draft.unlisted, hiddenDrawers: draft.hiddenDrawers, hiddenVideomakers: draft.hiddenVideomakers, fallback: draft.fallback, contexts: cx.map, outputs: ox.map, compacts: cpx.map, proxy, accountProxies: own.map, maxConcurrency, maxAccounts, ...queue, priceRate, rateWindows, modelPrefs: modelPrefsOfDraft(), pinUpstream: !!draft.pinUpstream, unredacted: !!draft.unredacted, countFree: !!draft.countFree, ...(p.carriesCompaction ? { nativeCompaction: !!draft.nativeCompaction } : {}), ...routingOfDraft(p) };
      saving(saveBtn, t("Saving…"));
      saveAccountModelsDraft(picked).then(() => savePluginOptions(subOf(a.agent))).then(() => providerAction("save", body, t("{name} saved", { name: p.name })), (e) => { doneSaving(saveBtn); editorError(e.message || String(e)); }); };
    bar.append(cancel, saveBtn);
    ed.append(bar);
    return ed;
  }

  // a vendor reached at the user's own resource (Azure OpenAI): no URL of
  // the preset's, the one the resource is at is typed or pasted here, and
  // owl puts it on the API the preset speaks when it is saved
  let endpoint = null;
  if (pr?.endpoint) {
    if (draft.chat === undefined) { draft.chat = p?.chat || ""; draft.responses = p?.responses || ""; }
    endpoint = input(draft.chat || draft.responses || "", pr.endpoint, "url");
    endpoint.classList.add("endpoint");
    endpoint.oninput = () => { draft.chat = draft.responses = endpoint.value.trim(); refreshEndpoints(); showCompaction(); };
    ed.append(...field(t("Endpoint"), endpoint, pr.endpointHint ? t(pr.endpointHint) : ""));
  }

  // a local server's address moves each URL, keeping its path
  // (provider.AtAddress); shown first, never folded (mengxiangziyou on X)
  let address = null, addressField = null;
  if (pr?.kind === "local") {
    // a saved provider's URLs (or a duplicate's) start it; a new one's are
    // the preset's, sent as none unless moved, so the gateway takes them
    const own = draft.chat !== undefined;
    // oMLX's decision API (#1641) is the same server's, and moves with it
    draft.addressFrom ??= { chat: draft.chat ?? pr.chat ?? "", responses: draft.responses ?? pr.responses ?? "", anthropic: draft.anthropic ?? pr.anthropic ?? "", decide: draft.decide ?? (p?.decide || pr.decide || "") };
    draft.addressWas ??= own ? originOf(draft.addressFrom) : "";
    draft.address ??= draft.addressWas;
    address = input(draft.address, originOf(pr));
    address.inputMode = "url";
    address.classList.add("address");
    // the Jev endpoint field shows where the decision API moved to, until
    // one is typed there
    address.oninput = () => {
      draft.address = address.value;
      refreshEndpoints();
      const du = ed.querySelector("input.decide-url");
      if (du && draft.decide === draft.addressFrom.decide) du.value = movedURLs(draft)?.decide || draft.decide;
    };
    const keys = address.onkeydown;
    address.onkeydown = (e) => { keys(e); if (e.key === "Enter" && isNew) save(); };
    addressField = field(t("Server address"), address, t("Where the server listens; change the port, or give another computer's address"));
  }

  // null, not false, when there is none: false?.key is undefined, and a
  // provider with no key read .set of it (willz: a local Ollama without a
  // key opened no editor, its row just toggling)
  const copied = (!p && draft.copyOf && providers.providers.find((x) => x.id === draft.copyOf)) || null;
  const key = input(draft.key || "", p?.key.set ? t("{masked} · paste a new key to replace it", { masked: p.key.masked }) : copied?.key.set ? t("{masked} · {name}'s key, or paste another", { masked: copied.key.masked, name: copied.name }) : t(pr?.keyHint || (pr?.noKey || p?.key.optional ? "optional for local servers" : "paste an API key")), "password");
  // many keys pasted at once (361 on Discord): one a line, or separated by
  // commas or spaces, are kept as one list; saved, the first is the key
  // and the others are added to its accounts, as Paste several does
  const keyCount = el("span", "hint key-count");
  const countKeys = () => {
    const n = splitKeys(key.value).length;
    keyCount.textContent = n > 1 ? t("{n} keys", { n }) : "";
    keyCount.title = n > 1 ? t("The first is tried first; the other {n} are added to its accounts", { n: n - 1 }) : "";
  };
  key.oninput = () => { draft.key = key.value; countKeys(); };
  // a curl command from a vendor's docs pasted into the key or the Base URL
  // fills the form (@mark_li21 on X), saying what it took; a key it only
  // stands for is left out, and the key's box is where to paste one
  const fromCurl = (e) => {
    const c = curlOf(e.clipboardData?.getData("text") || "");
    if (!c) return false;
    e.preventDefault();
    const took = [], notes = [];
    if (curlBase && c.api) { curlBase(c.api, c.base); took.push(t("Base URL") + " (" + t(API_NAMES[c.api]) + ")"); }
    // no key box when keys are Accounts: none taken unseen
    if (c.key && key.isConnected) { key.value = draft.key = c.key; countKeys(); took.push(t("API key")); }
    if (curlModel && c.model) { curlModel(c.model); took.push(t("model {model}", { model: c.model })); }
    if (took.length) notes.push(t("Filled in from the curl command: {what}.", { what: took.join(" · ") }));
    if (curlBase && !c.api) notes.push(t("Its URL isn't an address ending in /chat/completions, /messages or /responses: type the base URL."));
    if (curlBase && c.urlHole) notes.push(t("Replace {part} in the base URL.", { part: c.urlHole }));
    if (c.placeholder) notes.push(t("Its key, {key}, only stands for one: paste your own API key.", { key: c.placeholder }));
    if (!c.key) shown(key)?.focus(NO_SCROLL);
    editorError(notes.join(" "), "note");
    return true;
  };
  // not fromCurl itself: an on-handler giving back false cancels the paste
  if (url) url.onpaste = (e) => { fromCurl(e); };
  key.onpaste = (e) => {
    if (fromCurl(e)) return;
    const keys = splitKeys(e.clipboardData?.getData("text") || "");
    if (keys.length < 2) return;
    e.preventDefault();
    // an input keeps no line breaks: the keys are kept comma separated
    key.value = draft.key = keys.join(", ");
    countKeys();
  };
  countKeys();
  key.onkeydown = (e) => { e.stopPropagation(); if (e.key === "Enter" && isNew) save(); else if (e.key === "Escape") cancelEdit(); };
  const side = el("div", "side");
  side.append(keyCount);
  const eye = el("button", "text", t("Show"));
  let revealed = false; // the saved key is in the box, not a draft
  eye.onclick = async () => {
    if (key.type === "password") {
      if (!key.value && p?.key.set) {
        try { key.value = (await api("provider/key", { id: p.id })).key; revealed = true; } catch (e) { statusErr(e); return; }
      }
      key.type = "text"; eye.textContent = t("Hide");
    } else {
      if (revealed && !draft.key) key.value = "";
      revealed = false;
      key.type = "password"; eye.textContent = t("Show");
    }
  };
  side.append(eye);
  const keysUrl = p?.keysUrl || pr?.keysUrl;
  // the link follows the plan picked: a region's keysUrl goes with its
  // endpoints, and one without falls back to the preset's own page
  if (keysUrl) {
    const b = el("button", "link", t("Get a key ↗"));
    b.onclick = () => {
      if (pr?.kind === "partner") countPartner("keys", [pr.id]);
      api("open", { url: draft?.keysUrl || pr?.keysUrl || p?.keysUrl });
    };
    side.append(b);
  }
  const keyWrap = el("div", "pair");
  keyWrap.append(key, side);
  // Google Vertex AI takes no key: it is asked at the user's own Google
  // Cloud project with a token their Google credentials mint, so its
  // project, location, credentials file and a service account to go as
  // are asked in its place, the project first
  const vertex = p ? !!p.vertex : !!pr?.vertex;
  let vertexProject = null;
  if (vertex) draft.vertex = draft.vertex || { project: p?.vertex?.project || "", location: p?.vertex?.location || "", credentials: p?.vertex?.credentials || "", impersonate: p?.vertex?.impersonate || "" };
  const vertexFields = () => {
    const box = (k, placeholder, cls) => {
      const i = input(draft.vertex[k], placeholder);
      i.classList.add(cls);
      i.oninput = () => { draft.vertex[k] = i.value; refreshEndpoints(); };
      i.onkeydown = key.onkeydown; // Enter adds it, as from the key's box
      return i;
    };
    vertexProject = box("project", "my-project-123", "vertex-project");
    return [
      ...field(t("Project ID"), vertexProject, t("Your Google Cloud project, with the Agent Platform API (aiplatform.googleapis.com) enabled: requests are made, and billed, there")),
      ...field(t("Location"), box("location", "global", "vertex-location"), t("global, us, eu or a region such as us-central1. Each location serves its own models; global serves them all.")),
      ...field(t("Credentials file"), box("credentials", "~/.config/gcloud/application_default_credentials.json", "vertex-credentials"), t("Optional. Left empty, gcloud's Application Default Credentials are used (gcloud auth application-default login), or the file GOOGLE_APPLICATION_CREDENTIALS names. A service account's key file works too.")),
      ...field(t("Service account"), box("impersonate", t("optional · a service account's email"), "vertex-impersonate"), t("Requests are then made as this service account, impersonated with the credentials above: their account needs roles/iam.serviceAccountTokenCreator on it.")),
    ];
  };
  if (addressField) ed.append(...addressField);
  if (p?.keyList?.length) ed.append(...field(t("Accounts"), renderKeyAccounts(p), p.routing ? t("Tick every key to use; Routing says how requests spread over them.") : t("Tick every key to use. Requests go to the first; when it runs out of quota or hits a rate limit, the next ticked key takes over.")));
  if (p?.keyList?.filter((k) => k.on).length > 1) { routePick = renderRouting(p); ed.append(...routePick.nodes); }
  else if (vertex) ed.append(...vertexFields());
  // keys are replaced in Accounts; the box is for no key yet (Kylin_Cloud on X)
  else if (!p?.keyList?.length) ed.append(...field(t("API key"), keyWrap, isNew ? t("Kept in ~/.config/owl/providers.json, readable by you alone. Nothing is read from your shell.") : ""));

  // A user-defined provider can have its own picture; presets keep theirs.
  if (custom) ed.append(...field(t("Icon"), iconPicker(ed), ""));

  // Request headers of the user's own, for a preset's provider as much as a
  // custom one — which workspace a key is for, who the app is; signed-in
  // accounts returned above use the agent's own authentication headers.
  if (custom) {
    if (!draft.headers.length) draft.headers.push(["", ""]);
    advIn.append(...field(t("Headers"), headerEditor(), t("Extra HTTP headers sent to the vendor, applied after auth. For gateways that need a private scheme. ${sessionId} in a value is the conversation's id; ${threadId} is the thread's, where the agent names one (Codex), else the same.")));
  } else {
    advIn.append(...field(t("Headers"), headerEditor(pr?.headerHints || []), t("Optional headers sent with every request to {p}, applied after auth. ${sessionId} in a value is the conversation's id; ${threadId} is the thread's, where the agent names one (Codex), else the same.", { p: pr?.name || p?.name })));
  }
  // and JSON fields of the user's own for each request's body (#1604)
  advIn.append(...field(t("Extra body"), extraBodyEditor(), t("JSON fields added to the body of every request to the vendor. A field the request has already is kept; write \"!name\" to replace it. ${sessionId} is the conversation's id; ${threadId} is the thread's, where the agent names one (Codex), else the same.")));
  // and headers of the vendor's reply passed on to the agent (#4145)
  const replyIn = input(draft.replyHeaders || "", "X-Trace-Id, X-Foo");
  replyIn.classList.add("reply-headers");
  replyIn.oninput = () => { draft.replyHeaders = replyIn.value; };
  advIn.append(...field(t("Reply headers"), replyIn, t("Headers of the vendor's reply passed on to the agent, separated by commas, in any case. Request IDs (X-Request-Id, Request-Id, X-OneAPI-Request-ID) always are; cookies, CORS and the body's own headers never.")));
  const passIn = input(draft.passHeaders || "", "X-Tenant-Id");
  passIn.classList.add("pass-headers");
  passIn.oninput = () => { draft.passHeaders = passIn.value; };
  advIn.append(...field(t("Request headers to pass"), passIn, t("Headers of the agent's request passed on as sent, separated by commas. The agent's value stands over a header above. Keys, cookies and the connection's own never are.")));
  const proxies = el("div", "stack");
  proxies.append(proxyPicker());
  const perKey = keyProxyPicker(p);
  if (perKey) proxies.append(perKey);
  advIn.append(...field(t("Proxy"), proxies));
  advIn.append(...concurrencyField(p));
  advIn.append(...priceRateField(), ...countFreeField(), ...rateWindowsField());

  if (p?.cline || (isNew && pr?.id === "clinepass")) advIn.append(...upstreamField());

  // a relay that searches the web as its vendor does gets a client's search as sent (#359)
  if (custom) {
    const [stk, scb] = tick(t("Searches the web by itself"), !!draft.searches);
    scb.onchange = () => { draft.searches = scb.checked; };
    const row = field(t("Web search"), stk, t("For a relay in front of Anthropic's or OpenAI's own API: Claude Code's WebSearch and Codex's web_search go to it as they were sent, not through owl's search. owl doesn't automatically search with it for other models, but you can name it as the searcher."));
    showSearch = () => { for (const e of row) e.style.display = searchable() ? "" : "none"; };
    showSearch();
    advIn.append(...row);
  }

  // a third-party Anthropic API's key header: both, or one for a service refusing two (Aloha)
  const twoKeyHeaders = () => {
    if (p?.account || vertex) return false;
    if (p && !custom) return !!p.bothAuth;
    const a = ((draft.anthropic ?? (isNew && pr ? pr.anthropic : "")) || "").trim();
    if (!a) return false;
    let host = "";
    try { host = new URL(/^[a-z]+:\/\//i.test(a) ? a : "https://" + a).hostname.toLowerCase(); } catch { return true; }
    return !host.endsWith("anthropic.com") && !/^bedrock-runtime\..*\.amazonaws\.com$/.test(host) && !/^bedrock-mantle\..*\.api\.aws$/.test(host) && !host.endsWith(".azure.com");
  };
  if (!p?.account && !vertex) {
    const seg = segs([["", t("Both")], ["x-api-key", "x-api-key"], ["bearer", "Authorization: Bearer"]], draft.authHeader || "", (v) => { draft.authHeader = v; });
    seg.classList.add("auth-header");
    const row = field(t("Key header"), seg, t("Which header the API key goes in on the Anthropic API. Most services take both; pick one for a service that refuses the two together. owl also switches to one by itself when a service says so."));
    showAuth = () => { for (const e of row) e.style.display = twoKeyHeaders() ? "" : "none"; };
    showAuth();
    advIn.append(...row);
  }

  // the agent's User-Agent (#1851); an account or ownClient keeps its own
  const uaPassable = () => !p?.account && !(p ? p.ownClient : pr?.ownClient);
  if (uaPassable()) {
    const [utk, ucb] = tick(t("Send the agent's own User-Agent"), !!draft.passUserAgent);
    ucb.onchange = () => { draft.passUserAgent = ucb.checked; };
    advIn.append(...field(t("User-Agent"), utk, t("For a plan that serves only the coding agents it names: each request goes with the User-Agent the agent sent (pi, OpenCode, Claude Code…), not owl's. Requests owl makes itself, and ones the agent sent without one, still go as owl. A User-Agent set in Headers above always wins.")));
  }

  const addressOf = (k) => (draft[k] ?? (isNew && pr ? pr[k] : "")) || "";
  const unmaskable = () => (p?.preset || pr?.id) !== "remote-owl";
  if (unmaskable()) {
    advIn.append(...unmaskRow(t("For a model running on this computer or your local network: secrets, personal data and your masked words go to it as written. Leave it off for a local relay or proxy that passes requests on to a vendor."), () => ["chat", "responses", "anthropic", "gemini", "decide"].map(addressOf)));
  }

  // a user's provider whose certificate nothing trusts (#2383, provider.TakesInsecureTLS)
  const tlsTakes = () => custom && (!p || !!p.takesInsecureTLS);
  const onHTTPS = () => ["chat", "responses", "anthropic", "gemini", "decide"].map(addressOf).some((u) => /^\s*https:\/\//i.test(u));
  if (tlsTakes()) {
    const [ctk, ccb] = tick(t("Don't check its certificate"), !!draft.insecureTLS);
    ctk.classList.add("insecure-tls");
    ccb.onchange = () => { draft.insecureTLS = ccb.checked; };
    const row = field(t("Certificate"), ctk, t("For your own server with a self-signed certificate or one from your company's CA. owl then can't tell it from another machine answering at its address: turn it on only on a network you trust."));
    showTLS = () => { for (const e of row) e.style.display = onHTTPS() ? "" : "none"; };
    showTLS();
    ed.append(...row);
  }
  // off, a 429 or quota refusal rests none of its keys (LimitRestOff, zoecats)
  if (custom) {
    const [rtk, rcb] = tick(t("Set it aside when out of quota or rate limited"), !draft.limitRestOff);
    rcb.onchange = () => { draft.limitRestOff = !rcb.checked; };
    advIn.append(...field(t("Rests"), rtk, t("Off: a 429 or an out-of-quota refusal doesn't rest its keys, and the next request is sent to it as usual. For a relay whose limits pass in a moment.")));
  }

  // a balance token of the vendor's own (AiHubMix), or a custom one's (balanceFix)
  let balFix = null;
  if (p?.balanceToken?.takes || custom) {
    const saved = !!p?.balanceToken?.set;
    const tok = input(draft.balanceToken || "", saved && !draft.clearBalanceToken ? t("saved · paste a new one to replace it") : t("optional · the account's system access token"), "password");
    tok.oninput = () => { draft.balanceToken = tok.value.trim(); };
    tok.onkeydown = (e) => { e.stopPropagation(); if (e.key === "Escape") cancelEdit(); };
    const pair = el("div", "pair");
    pair.append(tok);
    if (saved && !draft.clearBalanceToken) {
      const side = el("div", "side");
      const drop = el("button", "text", t("Remove"));
      drop.onclick = () => { draft.clearBalanceToken = true; draft.balanceToken = ""; tok.value = ""; tok.placeholder = t("optional · the account's system access token"); drop.remove(); balFix?.refresh(); };
      side.append(drop);
      pair.append(side);
    }
    const tokHelp = custom || p.balanceURL
      ? t("What the Balance URL is asked with in place of the key, when it wants the account's own token: a new-api relay's System Access Token, or a sub2api panel's login token (a JWT, sent as a Bearer); it is used for nothing else.")
      : t("A key tells only what is left on itself. For the whole account's balance on the Usage page, generate a System Access Token in {p}'s settings and paste it here; it is used for nothing else.", { p: pr?.name || p.name });
    const [label, wrap] = field(t("Account balance"), pair, tokHelp);
    if (custom) {
      balFix = balanceFix(p);
      wrap.append(balFix);
      // whatever is typed may make it or unmake it, but the User ID keeps its place
      ed.addEventListener("input", (e) => { if (!e.target.closest?.(".bal-uid")) balFix.refresh(); });
    }
    advIn.append(label, wrap);
  }

  // StepFun tells a Step Plan's 5-hour, weekly and credit windows only to
  // its platform's sign-in, never to a key: signed in once in a window of
  // owl's, the Usage page shows them
  if (p?.stepPlan) ed.append(...field(t("Step Plan allowances"), renderStepPlan(p.stepPlan, p.id), t("A key tells only the balance. The Step Plan's 5-hour, weekly and credit windows are told only to a StepFun sign-in: bring yours here once and owl keeps it (for 30 days), used for nothing else.")));

  // a key on a team's GLM Coding Plan is told the team's windows only with
  // the team's organization and project, which the console shows (#236)
  const team = p ? !!p.zhipuTeam : !!pr?.zhipuTeam;
  if (team) {
    draft.zhipuTeam = draft.zhipuTeam || { org: p?.zhipuTeam?.org || "", project: p?.zhipuTeam?.project || "" };
    const org = input(draft.zhipuTeam.org, t("optional · only for a team plan"));
    org.classList.add("team-org");
    org.oninput = () => { draft.zhipuTeam.org = org.value; };
    const proj = input(draft.zhipuTeam.project, t("optional · only for a team plan"));
    proj.classList.add("team-project");
    proj.oninput = () => { draft.zhipuTeam.project = proj.value; };
    ed.append(...field(t("Team org ID"), org, ""));
    ed.append(...field(t("Team project ID"), proj, t("Only for a key on a team's GLM Coding Plan: both are in {p}'s console, under the team's organization and project. With them the Usage page shows the team's 5-hour and weekly windows.", { p: pr?.name || p.name })));
  }

  // plan windows told only to an access key (Volcengine #1427, Tencent #1574);
  // the saved Secret never comes back here
  const access = p ? p.accessKey : pr?.accessKey ? { vendor: pr.accessKeyVendor || "volcengine", id: "", secretSet: false } : null;
  let accessAt = () => {}, accessShown = () => !!access;
  if (access) {
    const tc = access.vendor === "tencent";
    const idName = tc ? t("SecretId") : t("AccessKey ID"), secretName = tc ? t("SecretKey") : t("Secret Access Key");
    if (draft.accessKeyID === undefined) draft.accessKeyID = access.id || "";
    const ak = input(draft.accessKeyID, idName);
    ak.classList.add("volc-ak");
    ak.oninput = () => { draft.accessKeyID = ak.value.trim(); };
    const saved = access.secretSet && !draft.clearAccessKey;
    const sk = input(draft.secretAccessKey || "", saved ? t("saved · paste a new one to replace it") : secretName, "password");
    sk.classList.add("volc-sk");
    sk.oninput = () => { draft.secretAccessKey = sk.value.trim(); };
    sk.onkeydown = (e) => { e.stopPropagation(); if (e.key === "Escape") cancelEdit(); };
    const pair = el("div", "pair");
    pair.append(sk);
    if (saved) {
      const side = el("div", "side");
      const drop = el("button", "text", t("Remove"));
      drop.onclick = () => { draft.clearAccessKey = true; draft.accessKeyID = ""; draft.secretAccessKey = ""; ak.value = ""; sk.value = ""; sk.placeholder = secretName; drop.remove(); };
      side.append(drop);
      pair.append(side);
    }
    const rows = [
      ...field(idName, ak, ""),
      ...field(secretName, pair, tc
        ? t("Token Plan Enterprise's usage is told only to the Tencent Cloud account's API key (console → Access management → API keys), not to the plan's API key. Give its SecretId and SecretKey here for the Usage page to show it; a sub-user's key that may read TokenHub is enough. It is used for nothing else.")
        : t("The plan's 5-hour, weekly and monthly windows are told only to the account's access key (Volcengine console → API Access Keys), not to the plan's API key. Give it here for the Usage page to show them; it is used for nothing else.")),
    ];
    ed.append(...rows);
    // a preset with the plan at some of its regions only (Tencent Cloud's
    // 企业版 beside its personal plan) asks for it at those, and at an
    // endpoint none of its regions has, which may be one of them
    const at = pr?.accessKeyRegions;
    if (at?.length) {
      accessAt = (r) => {
        const on = !r || at.includes(r.id) || !regionAt(pr, draft);
        for (const x of rows) x.hidden = !on;
      };
      accessShown = () => !rows[0].hidden;
    }
  }

  // a relay that offers several regional endpoints, or a vendor whose plans
  // are served at their own: one selector, and the provider's base URLs follow it
  let refreshEndpoints = () => {}, onRegion = () => {};
  if (pr?.regions?.length) {
    // Bedrock's ten regions don't fit the width: they scroll
    const seg = el("div", "segs regions");
    const cur = regionAt(pr, draft) || pr.regions.find((r) => r.decide && (r.id === draft.region || (!draft.region && workspaceOf(r.decide, draft.decide ?? p?.decide ?? pr.decide) !== null))) || pr.regions[0];
    for (const r of pr.regions) {
      const b = el("button", "opt" + (r.id === cur.id ? " on" : ""), t(r.name));
      b.onclick = () => {
        draft.chat = r.chat || ""; draft.responses = r.responses || ""; draft.anthropic = r.anthropic || "";
        draft.keysUrl = r.keysUrl || "";
        const site = ed.querySelector(".ehead .link.site");
        if (site && pr.regions.some((x) => x.website)) {
          site.dataset.url = r.website || pr.website;
          site.textContent = hostOf((r.website ? r : pr).siteHost || site.dataset.url) + " ↗";
        }
        onRegion(r);
        accessAt(r);
        for (const x of seg.querySelectorAll(".opt")) x.classList.toggle("on", x === b);
        slide(seg, "regions");
        refreshEndpoints();
      };
      seg.append(b);
    }
    accessAt(cur);
    queueMicrotask(() => {
      slide(seg, "regions");
      // the region in use in view, where they scroll
      const on = seg.querySelector(":scope > .on");
      if (on && seg.scrollWidth > seg.clientWidth) seg.scrollLeft = on.offsetLeft - (seg.clientWidth - on.offsetWidth) / 2;
    });
    ed.append(...field(t(pr.regionLabel || "Region"), seg, t(pr.splitAccounts ? "pick the region of your {p} account" : "which endpoint {p} is reached through", { p: pr.name })));
  }

  if (p) {
    const modelsField = field(t("Models"), renderModels(p), "");
    if (routePick) routePick.slot.append(...modelsField);
    else ed.append(...modelsField);
  }
  // Azure OpenAI's models are its deployments, which owl lists with the
  // key once the resource is added (#1995): said here, so no one takes the
  // add page for one where they must type them
  else if (pr?.id === "azure") ed.append(...field(t("Models"), el("div", "hint azure-models", t("No need to type them: once added, owl lists the deployments this key can call. Type a deployment's name as a model id only if that list can't be fetched.")), ""));
  if (p?.drawIds?.length) ed.append(...renderDrawers(p));
  if (p?.filmIds?.length) ed.append(...renderVideomakers(p));
  {
    // the window agents are told a model has, over what the vendor or
    // models.dev says: one for all of them, and model=size for one
    const cx = input(draft.contexts || "", t("e.g. 128k · or gpt-6=1m, comma separated"));
    if (!decideOnly(p || pr)) advIn.append(...field(t("Context window"), contextPicks(p, cx), t("How long a request the models take, told to the agents; empty leaves it to the vendor and models.dev")));
    if (!decideOnly(p || pr)) advIn.append(...outputField());
    if (!decideOnly(p || pr)) {
      advIn.append(...compactField());
      const [row, refresh] = nativeCompactionField(carriesCompaction);
      advIn.append(...row);
      showCompaction = refresh;
    }
  }
  if (p && !decideOnly(p)) advIn.append(...field(t("Fallback"), renderFallback(p), fallbackHint(p)));
  else if (custom) {
    const ex = input(draft.extra.join(", "), t("model ids, comma separated · e.g. gpt-5.5, claude-sonnet-5"));
    const pick = addFormModels(ex);
    ex.oninput = () => { draft.extra = typedIds(ex.value); pick.draw(); };
    curlModel = (id) => {
      if (!draft.extra.includes(id)) draft.extra = [...draft.extra, id];
      ex.value = draft.extra.join(", ");
      pick.draw();
    };
    ed.append(...field(t("Models"), pick.box, t("Optional: Fetch models to pick from the vendor's list, or type ids; none picked, owl asks for the list after saving.")));
  }

  // Jev's endpoint can be the one its gateway's docs give — Cloudflare's
  // names the account, …/accounts/<id>/ai/run — in place of the preset's
  if (decides && !custom && (p?.preset || pr?.id) !== "remote-owl") {
    if (draft.decide === undefined) draft.decide = p?.decide || pr?.decide || "";
    const du = input((address && movedURLs(draft)?.decide) || draft.decide, pr?.decide || "https://…", "url");
    du.classList.add("decide-url");
    // Bailian's is at the host of the key's workspace (#647): its id is
    // typed on its own, the plan's address filled in with it
    if (!draft.region && pr?.regions?.some((r) => r.decide)) draft.region = (pr.regions.find((r) => r.decide && workspaceOf(r.decide, draft.decide) !== null) || pr.regions[0]).id;
    const tmpl = () => (pr?.regions || []).find((r) => r.id === draft.region)?.decide || (draft.region ? "" : pr?.decide) || "";
    const wsIn = input((draft.workspace ?? workspaceOf(tmpl(), draft.decide)) || "", t("e.g. ws-…, from the Bailian console"));
    wsIn.classList.add("workspace-id");
    const wsRow = field(t("Workspace ID"), wsIn, t("The workspace your API key belongs to, in the Bailian console; its decision model is asked at that workspace's host"));
    const fill = () => {
      const tp = tmpl();
      const ws = tp.includes(WORKSPACE);
      for (const x of wsRow) x.hidden = !ws;
      if (!ws) return;
      draft.decide = du.value = tp.split(WORKSPACE).join((draft.workspace ?? wsIn.value).trim() || WORKSPACE);
    };
    wsIn.oninput = () => { draft.workspace = wsIn.value; fill(); };
    du.oninput = () => { draft.decide = du.value; };
    onRegion = (r) => { draft.region = r.id; if (r.decide) { draft.decide = du.value = r.decide; fill(); } };
    ed.append(...wsRow);
    fill();
    // Jev's own presets (TypeSafe, Vercel, Cloudflare) name it and the
    // gateway docs' address; a provider that also serves conversations
    // (OpenRouter, OpenCode Zen, Ollama …) asks its own decision models
    // on System One, and Jev's title and Cloudflare's example aren't its
    const jev = decideOnly(p || pr) && /jev/i.test(pr?.name || p?.name || "");
    ed.append(...field(t(jev ? "Jev endpoint" : "System One endpoint"), du, tmpl().includes(WORKSPACE) ? t("The address the decision model is asked at, POST …/systemone under it; the workspace ID above fills it in") : jev ? t("The address Jev is asked at; paste the one from your gateway's docs, e.g. Cloudflare's …/accounts/<account id>/ai/run") : t("The address its decision models are asked at, POST …/systemone under it")));
  }

  if (!custom && !(decides && !p)) {
    const ebox = el("div");
    const row = field(t("Endpoints"), ebox, "");
    refreshEndpoints = () => {
      const base = p || pr || {};
      const src = { chat: draft.chat || base.chat || "", responses: draft.responses || base.responses || "", anthropic: draft.anthropic || base.anthropic || "", decide: base.decide || "" };
      // Vertex AI's is the address its project and location make, shown
      // with its Test once a project is typed, saved one or not: with none
      // there is nowhere to ask
      if (vertex) {
        src.gemini = vertexBase(draft.vertex);
        for (const x of row) x.hidden = !src.gemini;
      }
      Object.assign(src, address && movedURLs(draft));
      ebox.replaceChildren(renderEndpoints(p, src));
    };
    refreshEndpoints();
    ed.append(...row);
  }
  ed.append(adv);
  mergeRouteAdv(adv, advIn, routePick);

  if (custom) {
    const more = edAligned(el("details", "more"));
    more.append(el("summary", "", t("More endpoints")));
    const inner = el("div", "inner");
    // the other protocols' URLs, drawn again when the base URL's changes
    const eps = el("div");
    eps.style.display = "contents";
    fillEndpoints = () => {
      eps.replaceChildren();
      const add = (label, key, ph, hint) => {
        if (apiField[draft.api] === key) return;
        const i = input(draft[key], ph, "url");
        i.oninput = () => { draft[key] = i.value; if (carried?.from === key) carried = null; showSearch(); showLocal(); showTLS(); showAuth(); showCompaction(); };
        eps.append(...field(t(label), i, t(hint)));
      };
      add("OpenAI URL", "chat", "https://…/v1", "if the vendor also serves chat completions");
      add("Anthropic URL", "anthropic", "https://…", "if the vendor also serves Anthropic messages");
      add("Responses URL", "responses", "https://…/v1", "if the vendor serves the OpenAI Responses API (Codex uses it natively)");
      add("Gemini URL", "gemini", "https://…/v1beta", "if the vendor serves Google's Gemini API (generateContent)");
    };
    fillEndpoints();
    inner.append(eps);
    const mu = input(draft.modelsURL, "https://…/v1/models", "url");
    mu.oninput = () => { draft.modelsURL = mu.value; };
    inner.append(...field(t("Models URL"), mu, t("Where the vendor lists its models, when that isn't under the base URL; asked with the key")));
    const cat = input(draft.catalog, t("models.dev ids, e.g. openai, deepseek"));
    cat.oninput = () => { draft.catalog = cat.value; };
    inner.append(...field(t("Catalog"), cat, t("Display names, reasoning levels and list prices for the models; for a gateway that serves several vendors, list them all, first match wins")));
    const bal = input(draft.balanceURL, "https://…/api/usage/token", "url");
    bal.classList.add("bal-url");
    bal.oninput = () => { draft.balanceURL = bal.value; balPath.placeholder = balanceFieldOf(bal.value); };
    // how the URL is asked: GET, or POST with no body for a vendor that
    // answers only a POST (#2368)
    const method = segs([["", "GET"], ["POST", "POST"]], draft.balanceMethod === "POST" ? "POST" : "", (v) => { draft.balanceMethod = v; });
    method.classList.add("bal-method");
    method.setAttribute("role", "group");
    method.setAttribute("aria-label", t("Balance request method"));
    method.title = t("How the Balance URL is asked: GET, or POST with no body for a vendor that answers only a POST");
    const balAsk = el("div", "pair bal-ask");
    balAsk.append(method, bal);
    inner.append(...field(t("Balance URL"), balAsk, t("Where the vendor tells what is left on the key, asked with it like a chat request; {key} in it or in a header is each key's own, for a vendor that takes the key in the URL (…?key={key}); shown on the Usage page")));
    const balPath = input(draft.balancePath, balanceFieldOf(draft.balanceURL));
    balPath.classList.add("bal-path");
    balPath.oninput = () => { draft.balancePath = balPath.value; };
    // asked as the form has it, before a Save: what the Usage page would show
    const balPair = el("div", "pair");
    balPair.append(balPath);
    if (p) {
      const side = el("div", "side");
      const check = el("button", "text action", t("Check balance"));
      check.title = t("Ask the Balance URL now, as the form has it");
      side.append(check);
      balPair.append(side);
      const res = el("div", "bal-res");
      check.onclick = async () => {
        check.classList.add("busy");
        res.className = "bal-res wait"; res.textContent = "…"; res.title = "";
        const body = { id: p.id, balanceURL: (draft.balanceURL || "").trim(), balancePath: (draft.balancePath || "").trim(), balanceMethod: draft.balanceMethod || "", headers: headersOf(draft.headers) };
        if (draft.balanceToken) body.balanceToken = draft.balanceToken;
        else if (draft.clearBalanceToken) body.clearBalanceToken = true;
        try {
          const r = await api("provider/balance", body);
          res.className = "bal-res " + (r.error ? "bad" : r.ok ? "ok" : "");
          res.textContent = r.error ? balanceError(r.error) || r.error : r.ok ? t("Balance") + " " + r.amount : t("No Balance URL to ask");
          res.title = r.error || "";
        } catch (e) { res.className = "bal-res bad"; res.textContent = e.message; }
        check.classList.remove("busy");
      };
      balPair.append(res);
      balPair.classList.add("wrap");
    }
    inner.append(...field(t("Balance field"), balPair, t("Where the amount is in the reply, e.g. data.balance; it can be a sum with + - * / and brackets, e.g. data.total / 500000 or (1 - credits.used / 70) %; \"$\" in front adds the sign, \"%\" after it shows a percent, with a bar; several, each with a label, go apart by \";\", e.g. 5h: a.used / a.cap %; $credits.left; a list's items are a.*.used, or a[status=active].used for those that match, put in sum(), max(), min() or count(); an amount the reply hasn't shows \"—\" and the others still show")));
    more.append(inner);
    ed.append(more);
  }

  const bar = el("div", "bar");
  if (p) {
    const del = el("button", "text danger", t("Remove"));
    del.onclick = () => providerAction("delete", { id: p.id }, t("{name} removed", { name: p.name }));
    bar.append(del);
  }
  if (p && pr) {
    // another key of the vendor, or the same key for another workspace
    const more = el("button", "text", t("Add another {name}", { name: pr.name }));
    more.title = vertex ? t("One more {name} provider, with its own project, credentials, headers and models", { name: pr.name }) : t("One more {name} provider, with its own key, headers and models", { name: pr.name });
    more.onclick = () => { editing = { preset: pr.id }; draft = null; renderProviders(); };
    bar.append(more);
  }
  if (p) {
    const dup = el("button", "text", t("Duplicate"));
    dup.title = vertex ? t("A new provider with {name}'s project, credentials, headers and models, to change before adding", { name: p.name }) : t("A new provider with {name}'s URLs, key, headers, models and balance settings, to change before adding", { name: p.name });
    dup.onclick = () => duplicateProvider(p);
    bar.append(dup);
  }
  bar.append(el("span", "grow"));
  const cancel = el("button", "text", t("Cancel"));
  cancel.onclick = cancelEdit;
  const saveBtn = el("button", "text primary", t(isNew ? "Add" : "Save"));
  const save = () => {
    if (p && pickingNone()) return editorError(t("Pick a model first."), "warn");
    // a typed icon address first; one that fails stays (hoobnn)
    const waiting = custom && ed.iconFlush?.();
    if (waiting) return waiting.then((ok) => ok && save());
    // new: an Add never replaces a provider that has the id already
    const body = { id: p ? slug(draft.id) || p.id : draft.id, from: p?.id, name: draft.name, preset: draft.preset, key: draft.key || "", chat: draft.chat, responses: draft.responses, anthropic: draft.anthropic, gemini: (draft.gemini || "").trim(), catalog: draft.catalog, models: p ? chosenIds() : draft.extra, headers: headersOf(draft.headers), new: isNew };
    if (isNew && draft.copyOf) body.copyOf = draft.copyOf;
    const xb = extraBodyOf(draft.extraBody);
    if (xb.error) { shown(ed.querySelector(".extra-body textarea"))?.focus(NO_SCROLL); return editorError(t("Extra body: {why}", { why: xb.error }), "warn"); }
    body.extraBody = xb.value;
    const rh = replyHeadersOf(draft.replyHeaders);
    if (rh.error) { shown(ed.querySelector(".reply-headers"))?.focus(NO_SCROLL); return editorError(t("Reply headers: {why}", { why: rh.error }), "warn"); }
    body.replyHeaders = rh.value;
    const ph = replyHeadersOf(draft.passHeaders, true);
    if (ph.error) { shown(ed.querySelector(".pass-headers"))?.focus(NO_SCROLL); return editorError(t("Request headers to pass: {why}", { why: ph.error }), "warn"); }
    body.passHeaders = ph.value;
    if (decides || custom) body.decide = (custom && draft.api !== "decide" && !p?.decide ? "" : draft.decide || "").trim();
    if ((body.decide || "").includes(WORKSPACE)) { ed.querySelector(".workspace-id")?.focus(NO_SCROLL); return editorError(t("Give the workspace ID your API key belongs to, or pick the Token Plan"), "warn"); }
    if (custom) {
      // the API picked for the Base URL is saved with it, and one left with
      // no URL is said rather than lost: the editor would open on another
      if (!(draft[apiField[draft.api]] || "").trim() && ["chat", "responses", "anthropic", "gemini", "decide"].some((k) => (draft[k] || "").trim())) {
        ed.querySelector(".base-url")?.focus(NO_SCROLL);
        return editorError(t("Base URL: type the {api} URL, or pick the API the URL you have is for", { api: t(API_NAMES[draft.api]) }), "warn");
      }
      body.baseAPI = draft.api === "openai" ? "chat" : draft.api;
    }
    if (custom) { body.icon = draft.icon || "generic"; body.iconFrom = draft.iconFrom || ""; body.balanceURL = (draft.balanceURL || "").trim(); body.balancePath = (draft.balancePath || "").trim(); body.balanceMethod = draft.balanceMethod || ""; body.modelsURL = (draft.modelsURL || "").trim(); }
    if (p) { body.fallback = draft.fallback; body.unlisted = draft.unlisted; body.hiddenDrawers = draft.hiddenDrawers; body.hiddenVideomakers = draft.hiddenVideomakers; body.noModels = noneChosen(); body.bestInFamily = bestChosen(); body.modelPrefs = modelPrefsOfDraft(); Object.assign(body, routingOfDraft(p)); }
    body.searches = !!draft.searches && searchable();
    body.pinUpstream = !!draft.pinUpstream;
    if (twoKeyHeaders()) body.authHeader = draft.authHeader || "";
    body.unredacted = !!draft.unredacted && unmaskable();
    body.countFree = !!draft.countFree;
    if (tlsTakes()) body.insecureTLS = !!draft.insecureTLS && onHTTPS();
    if (custom) body.limitRestOff = !!draft.limitRestOff;
    if (!decideOnly(p || pr)) body.nativeCompaction = !!draft.nativeCompaction && carriesCompaction();
    body.passUserAgent = !!draft.passUserAgent && uaPassable();
    const cx = parseContexts(draft.contexts || "");
    if (cx.error) return contextError(ed, cx.error);
    body.contexts = cx.map;
    if (!decideOnly(p || pr)) {
      const ox = parseContexts(draft.outputs || "");
      if (ox.error) return outputError(ed, ox.error);
      body.outputs = ox.map;
      const cpx = parseContexts(draft.compacts || "");
      if (cpx.error) return compactError(ed, cpx.error);
      body.compacts = cpx.map;
    }
    body.proxy = proxyOfDraft();
    if (body.proxy === null) return proxyError(ed);
    if (perKey) {
      const own = accountProxiesOfDraft();
      if (own.missing) {
        const line = ed.querySelector(`.acct-proxy[data-key="${own.missing}"]`);
        shown(line?.querySelector(".proxy-url"))?.focus(NO_SCROLL);
        return editorError(t("Proxy of {user}: type its address, like http://127.0.0.1:7890", { user: line?.querySelector(".who")?.textContent || own.missing }), "warn");
      }
      body.accountProxies = own.map;
    }
    body.maxConcurrency = concurrencyOfDraft();
    if (body.maxConcurrency === undefined) return concurrencyError(ed);
    body.maxAccounts = concurrencyOfDraft("maxAccounts");
    if (body.maxAccounts === undefined) return accountsError(ed);
    const queue = queueOfDraft();
    if (queue.bad) return queueError(ed, queue.bad);
    Object.assign(body, queue);
    body.priceRate = priceRateOfDraft();
    if (body.priceRate === undefined) return priceRateError(ed);
    body.rateWindows = self.rateWindowsOfDraft?.();
    if (body.rateWindows === false) return rateWindowsError(ed);
    if (p && priceTypedError(ed)) return;
    if (draft.balanceToken) body.balanceToken = draft.balanceToken;
    else if (draft.clearBalanceToken) body.clearBalanceToken = true;
    if (team) {
      // both or neither: {} clears them
      const org = (draft.zhipuTeam.org || "").trim(), project = (draft.zhipuTeam.project || "").trim();
      if (!org !== !project) { ed.querySelector(org ? ".team-project" : ".team-org")?.focus(NO_SCROLL); return editorError(t("Team plan: give both the organization ID and the project ID"), "warn"); }
      body.zhipuTeam = org ? { org, project } : {};
    }
    if (access) {
      // the Secret is sent only when a new one is typed: blank keeps the
      // saved one, unless it was removed
      body.accessKeyID = draft.accessKeyID || "";
      if (draft.secretAccessKey) body.secretAccessKey = draft.secretAccessKey;
      else if (draft.clearAccessKey) body.clearAccessKey = true;
      if (accessShown() && body.accessKeyID && !body.secretAccessKey && !(access.secretSet && !draft.clearAccessKey)) { ed.querySelector(".volc-sk")?.focus(NO_SCROLL); return editorError(t("Give the access key's Secret too"), "warn"); }
    }
    if (vertex) body.vertex = vertexOfDraft();
    if (isNew && custom && !body.name) { name.focus(); return editorError(t("Give it a name"), "warn"); }
    if (isNew && custom && !body.chat && !body.anthropic && !body.responses && !body.gemini && !body.decide) { url.focus(); return editorError(t("A base URL is needed"), "warn"); }
    if (endpoint && !body.chat && !body.responses) { endpoint.focus(); return editorError(t(pr.endpointNeeded || "Your resource's endpoint is needed"), "warn"); }
    if (vertex && !body.vertex.project) { vertexProject?.focus(NO_SCROLL); return editorError(t("Google Vertex AI needs the id of your Google Cloud project"), "warn"); }
    if (address) {
      const a = (draft.address || "").trim();
      if (localAddress(a) === null) { address.focus(NO_SCROLL); return editorError(t("Address: {v} isn't an address like http://localhost:11434", { v: a }), "warn"); }
      Object.assign(body, movedURLs(draft));
    }
    editorError("");
    const picked = accountModelsDraft(p);
    if (picked === false) return pickError();
    saving(saveBtn, t(isNew ? "Adding…" : "Saving…"));
    const okMsg = t(isNew ? "{name} added" : "{name} saved", { name: draft.name || draft.id });
    saveAccountModelsDraft(picked).then(() => providerAction("save", body, okMsg), (e) => { doneSaving(saveBtn); editorError(e.message || String(e)); });
  };
  saveBtn.onclick = save;
  bar.append(cancel, saveBtn);
  // a plugin's panels for a provider (owl.panels' provider), told which
  if (p) { const slot = owl.slot?.("provider", { provider: { id: p.id, name: p.name || p.id } }); if (slot) ed.append(slot); }
  ed.append(bar);
  setTimeout(() => (isNew ? (custom || another ? name : endpoint || vertexProject || key) : null)?.focus(), 0);
  return ed;
}
