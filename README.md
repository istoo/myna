<div align="center">

<a href="https://github.com/istoo/owl"><img src="site/public/img/icon-256.png" width="112" alt="owl"></a>

# owl

### Your agents. Your models. One control point.

**Claude Code on Kimi. Codex on DeepSeek. Gemini CLI on GLM. OpenCode on your ChatGPT plan.**<br>
Each agent takes its model from the same list. A single local gateway serves them all,<br>
moving to your next account before a quota runs dry.

[![Release](https://img.shields.io/github/v/release/istoo/owl?label=release&color=111111)](https://github.com/istoo/owl/releases/latest) [![Stars](https://img.shields.io/github/stars/istoo/owl?style=flat&color=111111)](https://github.com/istoo/owl/stargazers) [![License](https://img.shields.io/badge/license-MIT-111111)](LICENSE)<br>
![macOS](https://img.shields.io/badge/macOS-000000?logo=apple&logoColor=white) ![Windows](https://img.shields.io/badge/Windows-0078D4?logo=windows&logoColor=white) ![Linux](https://img.shields.io/badge/Linux-FCC624?logo=linux&logoColor=black) ![Docker](https://img.shields.io/badge/Docker-2496ED?logo=docker&logoColor=white) ![Termux](https://img.shields.io/badge/Termux-000000?logo=android&logoColor=white)

**[Download](https://github.com/istoo/owl/releases/latest)** &nbsp;·&nbsp; **[Docs](https://github.com/istoo/owl/blob/main/site/public/docs/start.html)** &nbsp;·&nbsp; **[Reference](docs/reference.md)** &nbsp;·&nbsp; **[Plugins](https://github.com/magpie-community/plugins)** &nbsp;·&nbsp; [简体中文](README.zh-CN.md)

<br>

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/agents-dark.png">
  <img src="site/public/img/agents-light.png" width="900" alt="owl's Agents page: Claude Code on Kimi K3, Codex on DeepSeek V4 Pro, Gemini CLI on GLM-5.3, each picked from one list">
</picture>

</div>

<br>

## What owl is

Each agent wants its model in a place of its own: another file, another key,
another base URL. A subscription you pay for is usually wired into one agent
and stays there; pointing a second one at it means editing config by hand.

owl gathers that work onto one screen. Choose a model for an agent, and owl
rewrites only the setting that selects it, leaving comments and formatting
alone. Every request then leaves through a local gateway, so failover,
routing, masking and accounting happen once, in one place, for all of them.

owl earns its place when you run several agents, several providers, or several
accounts on one provider. With a single agent on its vendor's own plan, the
agent's settings are enough.

It runs on macOS, Windows and Linux as a menu bar app, a window, a TUI, a web
UI and a CLI, and also in Docker and on Termux.

## Install

Get the app from **[Releases](https://github.com/istoo/owl/releases/latest)**, or run:

```sh
curl -fsSL https://github.com/istoo/owl/releases/latest/download/install.sh | sh
```

<sub>With a desktop, the script installs the app. On a Linux box without WebKitGTK 4.1 it installs that first through your package manager (sudo will ask for your password); with no desktop, or no terminal to answer the prompt, it installs the terminal build and names the package you still need. Pass `--gui` or `--tui` to choose. Behind a firewall, add `--proxy` or `--mirror`. Files called `owl-cli-*` are terminal builds, not documents: `chmod +x` one and run it, or let the script put `owl` on your PATH. The desktop app is `owl-darwin-*.zip`. You can also `go install github.com/istoo/owl@latest` or use the [Docker image](https://github.com/istoo/owl/blob/main/site/public/docs/docker.html). Every build updates itself, with release notes from this repository. To remove it, run the same command with `sh -s -- --uninstall`; settings in `~/.config/owl` survive unless you add `--purge`, which also disconnects and deletes your agents.</sub>

**First open on macOS.** The app is signed with owl's own certificate, not an Apple Developer ID, and not notarised. A copy
downloaded by a browser carries the quarantine flag, and macOS will refuse it
until you choose **Open Anyway** in **System Settings → Privacy & Security**,
or run `/usr/bin/xattr -dr com.apple.quarantine /Applications/owl.app`. A copy
installed by the script above comes through `curl`, carries no such flag, and
opens straight away.

## Quick start

1. **Add a provider.** In owl, open **Providers → Add provider**. Choose a preset and paste a key, or sign in to a subscription.
2. **Give each agent a model** on the **Agents** page. The next session you start uses it.
3. **Or stay in the terminal:**

```sh
owl provider add deepseek sk-…              # a preset needs only the key
owl claude deepseek/deepseek-v4-pro          # Claude Code on DeepSeek
owl codex moonshot/kimi-k2.5                 # Codex on Kimi
owl group add "Opus anywhere" models=claude/claude-opus-5-5,copilot/claude-opus-5.5 routing=smart
owl claude group/opus-anywhere               # fails over between subscriptions
owl save work && owl use work                # profiles
owl quota                                    # what's left on every plan
owl settings set theme dark                  # every Settings switch, from the terminal
owl tui                                      # the whole thing, in a terminal
```

owl discovers the subscriptions your agents already signed in to by reading
their sign-ins (Claude Code's and Codex's files, the Keychain). To stop it,
turn off **Settings → Privacy → Subscription accounts**, or run
`owl accounts discovery off` (`on` turns it back on). `OWL_NO_ACCOUNTS=1`
disables it for one run without touching the saved setting. API-key providers
keep working either way.

## How it fits together

One gateway listens on `127.0.0.1:3425`. It accepts OpenAI Chat, OpenAI
Responses, Anthropic Messages and Gemini requests, and translates between
them — including streaming, tool calls and reasoning. Your agents point at it;
routing groups, plugins and the mask sit along the way.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/architecture-dark.svg">
  <img src="site/public/img/architecture.svg" alt="How owl fits together: your agents send requests to the owl gateway at 127.0.0.1:3425, which routes them through routing groups and plugins to your API keys, subscriptions and local models.">
</picture>

Anything that accepts a base URL can use the gateway as well:

```sh
export OPENAI_BASE_URL=http://127.0.0.1:3425/v1     OPENAI_API_KEY=owl
export ANTHROPIC_BASE_URL=http://127.0.0.1:3425     ANTHROPIC_API_KEY=owl
export GOOGLE_GEMINI_BASE_URL=http://127.0.0.1:3425 GEMINI_API_KEY=owl
```

## What it does

### Privacy masking

When an agent reads an `.env`, a config file or a log with a token in it, that
content ends up in the prompt. With masking on, owl replaces those values with
placeholders on your computer before the request leaves, and puts the real
values back into the reply. Your agent and your files still see the originals;
the vendor never does.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/masking-dark.png">
  <img src="site/public/img/masking-light.png" width="900" alt="Insights › Masking: 5,812 values kept from vendors in 7 days, flowing from Claude Code and Codex through owl to OpenAI, Kimi and DeepSeek, and each request's masked values">
</picture>

| | |
| :-- | :-- |
| **Two ways to find a value** | Certain shapes are caught by rule: API keys, private keys, tokens, emails, and card or ID numbers whose checksum adds up. For names, addresses and passwords no rule can recognise, an optional model runs on this computer (a 1.6 GB download, CPU) |
| **Restored everywhere** | A value becomes a placeholder like `{{API_KEY_k3v9x2mq}}`. Every placeholder in the reply turns back into the real value, streamed text and tool-call arguments included, so a command the agent runs uses the real key |
| **Stable across turns** | The same value always gets the same placeholder, which keeps the conversation, the prompt cache and thinking signatures valid. Once masked, a value stays masked wherever it appears later |
| **Memory only** | Values are held in memory. The record keeps each value's kind, length and first characters. If masking is enabled but cannot run, the request is held back with a 503 instead of being sent as written |
| **Visible and checkable** | **Insights › Masking** lists every request it changed, side by side with what the vendor received; **Try it** in Settings shows what a piece of text would become |
| **Adjustable** | Secrets and personal data can be switched separately, down to individual kinds (passport, IBAN, MAC, public IP…), with your own words and patterns. Each value can be masked, only warned about, or blocked (**Mask**, **Warn**, **Block**) |

There is a trade-off: the model sees `{{API_KEY_…}}` instead of the key, so a
task that needs the value itself (why is this token refused?) cannot be done
on it. Put such values under **Never mask**. Rules can occasionally be wrong,
and the Masking page is where you would notice. Masking stays off until you
enable it in **Settings → Privacy**. Emails at domains reserved for examples
(`example.com`, `.test`, `.invalid`) are passed through unchanged.

### Providers and subscriptions

Choose a preset and paste a key. The vendor supplies the model list; names and
reasoning levels come from [models.dev](https://models.dev), so a model
released this morning shows up in your picker after the next refresh.

> **Presets include** Anthropic · OpenAI · Google Gemini · Google Vertex AI · Amazon Bedrock (runtime and Mantle) · DeepSeek · Kimi · Zhipu GLM · MiniMax · StepFun · LongCat · Qwen · Baidu Qianfan · Tencent Cloud · Huawei Cloud MaaS · Volcengine Ark · Mistral · Groq · Nous Portal · xAI · OpenRouter · Together · Fireworks · SiliconFlow · NVIDIA NIM · ModelScope · Ollama · LM Studio … and any OpenAI- or Anthropic-compatible URL.

**Import** pulls providers across from CC Switch, Claude Code, Codex and Alma.
An **[Add to owl](https://github.com/istoo/owl/blob/main/site/public/docs/import.html)**
link lets a provider's website hand its config to owl in one click. Any key or
account can be routed through a proxy of its own.

An agent you are already signed in to appears as a **subscription** with its
models, and owl offers it to every other agent as a provider. Add more than
one account to a subscription and owl fails over between them.

- **Claude**: drives the real local `claude` binary and bridges your agent's tools over MCP
- **Codex / ChatGPT**: your ChatGPT plan's models, in every other agent
- **GitHub Copilot**, **Gemini (Code Assist)**, **Antigravity**, **Grok (SuperGrok)** and more
- **Any OpenCode auth plugin or pi package** from npm

**Daily warm-up** opens each Claude and ChatGPT account's 5-hour window at the
times you pick, or as soon as the previous one resets.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/add-dark.png">
  <img src="site/public/img/add-light.png" width="900" alt="Add provider: subscriptions to sign in to, vendors, relays and local servers">
</picture>

### Routing groups

A **routing group** bundles several models under one name an agent can select,
such as `group/daily-coding`. The gateway distributes requests across the
keys and accounts of every member:

| Mode | What it does |
| :-- | :-- |
| **`smart`** | Among subscriptions that still have quota, pick the one resetting soonest, so the least is wasted at reset |
| **`order`** | Stay on the first model until it cannot answer, then try the next |
| **`rotate`** | Advance to the next member each turn |
| **`usage`** | Start with the member used least |
| **`pace`** | Prefer the account with the most of its weekly quota left per hour until reset |

While the vendor's prompt cache is still worth keeping, a conversation **stays
on the account that answered it**. Groups can contain other groups.
**[Intent routing](https://github.com/istoo/owl/blob/main/site/public/docs/intent.html)**
has a small model of your choice read each new turn, so tests reach the strong
model and quick questions the cheap one. The Routing view shows every decision
as it happens.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/routing-dark.png">
  <img width="760" src="site/public/img/routing-light.png" alt="The Routing view: four agents through owl to seven providers, live">
</picture>

### Plugins

```sh
owl plugin add opencode-gemini-auth   # an OpenCode plugin from npm
owl plugin add pi-antigravity         # a pi package, the same way
owl plugin login google-plugin        # its own sign-in flow, in owl
```

Plugins run on [Bun](https://bun.sh), downloaded the first time one is needed.
A plugin signs in, lists its models and makes each request; agents use those
models as they would any provider's. Browse
**[magpie-community/plugins](https://github.com/magpie-community/plugins)** or
[write your own](https://github.com/istoo/owl/blob/main/site/public/docs/plugins.html).

A plugin can also work as **gateway middleware**: JavaScript that sees what
every agent sends and receives. `onRequest` may rewrite a request or refuse
it, `onEvent` observes each streamed event, and `onResponse` sees the finished
reply. If a hook throws or takes too long, the request goes through untouched.

```js
// alias.middleware.js — owl plugin add ./alias.middleware.js
export function onRequest(body, ctx) {
  if (body.model === "fast") return { ...body, model: "deepseek/deepseek-chat" };
}
```

<details>
<summary><b>Ready-made middleware</b>, most of it what <a href="https://github.com/QuantumNous/new-api">New API</a> does for its channels, with the same JSON</summary>

<br>

| Package | What it does |
| :-- | :-- |
| `param-override` | New API's `param_override`: set, delete, move or rewrite request fields under conditions, or refuse a request |
| `model-map` | New API's `model_mapping`: send a model under a different name; the reply keeps the name that was asked for |
| `system-prompt` | Add your system prompt to every request, or only to some agents or models |
| `word-guard` | New API's sensitive-word filter: refuse or mask words in what users send, and in replies |
| `think-tags` | Remove ` thinking…` from replies, or add `reasoning_content` to them |

```sh
owl plugin add @magpie-community/middleware-model-map
owl plugin options model-map '{"mapping": {"fast": "deepseek/deepseek-chat"}}'
```

Find them under Plugins › Discover › Gateway middleware. See [Gateway middleware](https://github.com/istoo/owl/blob/main/site/public/docs/plugins.html#middleware).

</details>

### Usage, sharing and everything else

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/usage-dark.png">
  <img src="site/public/img/usage-light.png" width="900" alt="The Usage page: balances, tokens, cache hit rate and cost">
</picture>

| | |
| :-- | :-- |
| **Tokens and cost** | Tokens, cache reads and writes, reasoning and calls, priced at list price in dollars or yuan; any model can carry a price of your own |
| **Quota** | Balances and quota windows for every key, plan and account (`owl quota`), with a reminder when a window is about to reset with much of it unused |
| **Sessions** | Every agent conversation with its cost and title, reopened in your terminal with one click |
| **Context** | How full each request's context window is, how much came from cache, and what fills it |
| **Bills** | Broken down by account and by upstream key, with OTLP export to your own observability stack |
| **Local network** | Switch on *Share on local network* and give each client a named gateway key with its own daily, weekly or monthly token and cost limit |
| **[Remote owl](https://github.com/istoo/owl/blob/main/site/public/docs/remote.html)** | A laptop can use the providers, accounts and groups of the owl on your desktop, while still wiring its own agents |
| **[Docker](https://github.com/istoo/owl/blob/main/site/public/docs/docker.html)** | Run `ghcr.io/istoo/owl` on a server or NAS and manage it from the web UI |
| **Sync** | Back up to a file, or sync machines over WebDAV (Nutstore, Nextcloud…), S3, or an encrypted GitHub repository backup |

- **Library**: instructions, MCP servers, skills and subagents are written once; owl places them into each agent's files in that agent's format, and removes only what it placed.
- **Web search for any model**: a model without search gets owl's — either a provider that can search, or search APIs (Tavily, Brave, Exa, Firecrawl, You.com, DeepSeek, Doubao Search, Yuanbao Search, SearXNG) in the order you arrange them. The same search can be mounted as an MCP server for any agent, with `web_search` and `web_fetch`.
- **Images and video**: an image tool over MCP for any agent, using the model you choose, and video through a Grok subscription or a MiniMax API key.
- **Agents kept current**: the Agents page shows each CLI's version, updates it the way it was installed, and prints the commands for the ones you don't have.
- **WSL**, **keep awake**, light and dark themes, and your theme's look on [Omarchy](https://omarchy.org).

## Supported agents

<table>
<tr><td>

Claude Code · Claude Desktop · Claude Science · Codex · Gemini CLI · Antigravity CLI · OpenCode · OpenChamber · MiMo Code · DevEco Code · Pi · oh-my-pi · StepCode · MothX · PenguinHarness · Aside · OmO · Prime Agent · Goose · Cursor CLI · Cursor Private Inference · Zed · VS Code Chat · VS Code Insiders · VSCodium Chat · Zoo Code · JetBrains Air · Air (JetBrains IDEs) · Copilot (JetBrains) · Copilot CLI · Crush · DeepSeek Harness · Reasonix Studio · Command Code · fx · Devin · Hermes Agent · Mister Morph · Kimi Code · jcode · Qwen Code · Muse Code · Empryo · MiniMax Code · Droid · Cline · Qoder · Qoder CN · Grok Build · Grok App · ZCode · WorkBuddy · WorkBuddy AI · CodeBuddy · Pencil · OpenPencil · OpenBot · NewMax · Raycast · Kiro Crew · T3 Code · OpenHanako · AtomCode · Snow CLI · Alma · Cindy · Lody · AionUi · Ante · AstrBot · QwenPaw · OpenClaw

</td></tr>
</table>

owl lists only the agents installed on your machine; setup notes for some are in the [reference](docs/reference.md#notes-on-some-agents).

## Documentation

| | |
| :-- | :-- |
| **[Get started](https://github.com/istoo/owl/blob/main/site/public/docs/start.html)** | The guided tour |
| **[Reference](docs/reference.md)** | Every agent, provider option, gateway endpoint, CLI command and file |
| **[Plugins](https://github.com/istoo/owl/blob/main/site/public/docs/plugins.html)** | Use plugins and write your own |
| **[Intent routing](https://github.com/istoo/owl/blob/main/site/public/docs/intent.html)** | Route each turn by what it asks for |
| **[Import links](https://github.com/istoo/owl/blob/main/site/public/docs/import.html)** | "Add to owl" buttons for provider websites |
| **[Remote owl](https://github.com/istoo/owl/blob/main/site/public/docs/remote.html)** | Use one owl's providers and groups from your other computers |
| **[Docker](https://github.com/istoo/owl/blob/main/site/public/docs/docker.html)** | Run owl on a server or a NAS |

## Privacy

Prompts, replies, keys and accounts travel only to the providers you chose.
With [privacy masking](#privacy-masking) enabled, the secrets among them do not
travel even there. This build of owl sends nothing upstream: no usage events,
no partner list, no icon proxy. Its updates, release notes and images come
from, or go to, no service other than this GitHub repository.

## Community

Questions, ideas, or a model that refuses to appear? **[Open an issue](https://github.com/istoo/owl/issues)** — that is where feedback goes.

**This repository does not accept pull requests.** Only maintainers can open them. If you want a fix or a feature, describe it in an issue and we will build it.

If owl saved you from editing one more config file, **a ⭐ helps others find it.**

<a href="https://star-history.com/#istoo/owl&Date">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=istoo/owl&type=Date&theme=dark">
    <img src="https://api.star-history.com/svg?repos=istoo/owl&type=Date" width="600" alt="Star history">
  </picture>
</a>

## License

[MIT](LICENSE)

owl began from [yetone/magpie](https://github.com/yetone/magpie), keeps its
copyright notice and MIT license, and stays compatible with the magpie plugin
contracts (`magpie` plugin key, `@magpie-community/*`, `X-Magpie-*` headers).
The product, its updates and its network dependencies are owl's own.

owl is not affiliated with [Apache Owl](https://owl.apache.org) or the [Apache Software Foundation](https://apache.org).
