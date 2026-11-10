<div align="center">

<a href="https://github.com/istoo/owl"><img src="site/public/img/icon-256.png" width="112" alt="owl"></a>

# owl

### 每个 agent 的模型，都在一个地方。

**Claude Code 跑 Kimi。Codex 跑 DeepSeek。Gemini CLI 跑 GLM。OpenCode 跑你的 ChatGPT 订阅。**<br>
一个列表里任选，给任意 agent。一个本地网关服务所有 agent，<br>
某个额度用完时，它默默换到你的下一个账号。

> **owl 是 [yetone/magpie](https://github.com/yetone/magpie) 的二次开发。**
> 它是一个衍生发行版：插件生态与契约（`magpie` 插件键、`@magpie-community/*`、
> `X-Magpie-*` 请求头）保留，品牌、更新源与网络依赖独立维护。
> 原项目的版权与 MIT 许可证一并保留。

[![Release](https://img.shields.io/github/v/release/istoo/owl?label=release&color=111111)](https://github.com/istoo/owl/releases/latest) [![Stars](https://img.shields.io/github/stars/istoo/owl?style=flat&color=111111)](https://github.com/istoo/owl/stargazers) [![License](https://img.shields.io/badge/license-MIT-111111)](LICENSE)<br>
![macOS](https://img.shields.io/badge/macOS-000000?logo=apple&logoColor=white) ![Windows](https://img.shields.io/badge/Windows-0078D4?logo=windows&logoColor=white) ![Linux](https://img.shields.io/badge/Linux-FCC624?logo=linux&logoColor=black) ![Docker](https://img.shields.io/badge/Docker-2496ED?logo=docker&logoColor=white) ![Termux](https://img.shields.io/badge/Termux-000000?logo=android&logoColor=white)

**[下载](https://github.com/istoo/owl/releases/latest)** &nbsp;·&nbsp; **[文档](https://github.com/istoo/owl/blob/main/site/public/docs/zh/start.html)** &nbsp;·&nbsp; **[完整参考（英文）](docs/reference.md)** &nbsp;·&nbsp; **[插件](https://github.com/magpie-community/plugins)** &nbsp;·&nbsp; [English](README.md)

<br>

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/agents-dark.png">
  <img src="site/public/img/agents-light.png" width="900" alt="owl 的 Agents 页：Claude Code 跑 Kimi K3，Codex 跑 DeepSeek V4 Pro，Gemini CLI 跑 GLM-5.3，都从一个列表里选">
</picture>

</div>

<br>

## 为什么用 owl

每个 agent 都把自己的模型写在各自的文件里，格式不同，密钥和 base URL 也不同。你付费的订阅只能在一个 agent 里用。owl 把这些都放到一个应用和一个本地网关后面：

- **任意模型，任意 agent。** 从一个列表里选；owl 只改 agent 配置里真正起作用的那一个键，原来的注释和排版不动。
- **一个供应商，处处可用。** 一个 API key、一个本地模型，或者你已登录的 Claude、ChatGPT、Copilot、Gemini、Grok 订阅。
- **额度用完自动续上。** 同一供应商的下一个账号接手；路由分组会换到另一个模型或供应商，agent 看不到错误。只有一个账号又没有分组时，无处可换，错误才会到 agent 手上。
- **密钥不离开你的电脑。** 打开脱敏后，提示词里的密钥、密码和邮箱会在本机换成占位符，回复时再换回来。

同时用多个 agent、多个供应商或多个账号，owl 就是为你做的。如果只有一个 agent、用它自己厂商的订阅就够，你不需要它。

支持 macOS、Windows 和 Linux（菜单栏应用、窗口、TUI、网页界面和 CLI），也能跑在 Docker 与 Termux 上。

## 安装

从 **[Releases](https://github.com/istoo/owl/releases/latest)** 下载应用，或者运行：

```sh
curl -fsSL https://github.com/istoo/owl/releases/latest/download/install.sh | sh
```

<sub>在桌面上，脚本安装桌面应用。Linux 上没有 WebKitGTK 4.1 时，脚本先用包管理器装上（sudo 会问密码）；没有桌面、或者没有终端可以询问时，安装终端版，并告诉你要补哪个包。`--gui` 或 `--tui` 可以指定。防火墙后面用 `--proxy` 或 `--mirror`。名字带 `owl-cli-*` 的是终端版——它们是可执行文件，不是文本文件：`chmod +x` 之后在终端运行，或者用上面的脚本把 `owl` 放进 PATH。图形界面是 `owl-darwin-*.zip`。也可以用 `go install github.com/istoo/owl@latest`，或者 [Docker 镜像](https://github.com/istoo/owl/blob/main/site/public/docs/zh/docker.html)。每个构建都会自己更新，更新说明只来自这个仓库。卸载时用同一条命令加 `sh -s -- --uninstall`；`~/.config/owl` 里的设置会保留，除非加上 `--purge`——那会断开你的 agent 并一起删掉。</sub>

**macOS 首次打开。** 应用是 ad-hoc 签名，没有公证。浏览器下载的副本带着隔离标记，
在你允许之前 macOS 会拒绝打开：到 **系统设置 → 隐私与安全性** 点 **仍要打开**，
或者运行 `/usr/bin/xattr -dr com.apple.quarantine /Applications/owl.app`。用上面的脚本安装的副本
是用 `curl` 下载的，没有隔离标记，可以直接打开。

## 快速上手

1. **添加供应商。** 打开 owl，进入 **供应商 → 添加供应商**。选一个预设、粘贴 key，或者用订阅登录。
2. **给每个 agent 选模型。** 在 **Agents** 页选好，开一个新会话就生效。
3. **或者全在终端里做：**

```sh
owl provider add deepseek sk-…              # 预设只需要 key
owl claude deepseek/deepseek-v4-pro          # Claude Code 跑 DeepSeek
owl codex moonshot/kimi-k2.5                 # Codex 跑 Kimi
owl group add "Opus anywhere" models=claude/claude-opus-5-5,copilot/claude-opus-5.5 routing=smart
owl claude group/opus-anywhere               # 在多个订阅之间故障转移
owl save work && owl use work                # 配置档
owl quota                                    # 每个订阅还剩多少
owl settings set theme dark                  # 设置里的每一项，都能在终端改
owl tui                                      # 整个应用，在终端里跑
```

owl 通过读取 agent 的登录信息（Claude Code 和 Codex 的文件、钥匙串）来发现你已经登录的订阅。不想让它读，关掉 **设置 → 隐私 → 订阅账号**，或者运行 `owl accounts discovery off`（`on` 打开）。`OWL_NO_ACCOUNTS=1` 只在这一次运行里关掉，不改已保存的设置。用 API key 的供应商不受影响。

## 它是怎么连起来的

`127.0.0.1:3425` 上的网关会说 OpenAI Chat、OpenAI Responses、Anthropic Messages 和 Gemini，并在它们之间翻译——流式输出、工具调用和推理都包括。agent 指向它；路由分组、插件和脱敏都在路径上。

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/architecture-dark.svg">
  <img src="site/public/img/architecture.svg" alt="owl 的连接方式：agent 把请求发给 127.0.0.1:3425 上的网关，网关经过路由分组和插件，送到你的 API key、订阅和本地模型">
</picture>

任何接受 base URL 的工具也能用这个网关：

```sh
export OPENAI_BASE_URL=http://127.0.0.1:3425/v1     OPENAI_API_KEY=owl
export ANTHROPIC_BASE_URL=http://127.0.0.1:3425     ANTHROPIC_API_KEY=owl
export GOOGLE_GEMINI_BASE_URL=http://127.0.0.1:3425 GEMINI_API_KEY=owl
```

## 它能做什么

### 隐私脱敏

agent 会把读到的东西放进提示词：一个 `.env`、一份配置、一段带 token 的日志。打开脱敏后，owl 在请求离开电脑之前把这些值换成占位符，回复时再换回来。agent 和你的文件保留真实值，供应商看不到。

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/masking-dark.png">
  <img src="site/public/img/masking-light.png" width="900" alt="洞察 › 脱敏：7 天里从供应商手里留下的 5,812 个值，从 Claude Code 和 Codex 经过 owl 流到 OpenAI、Kimi 和 DeepSeek，以及每个请求被脱敏的值">
</picture>

| | |
| :-- | :-- |
| **两种发现方式** | 规则找出确定的（API key、私钥、token、邮箱，以及校验和能对上的卡号和证件号）；如果你愿意，还可以在本机跑一个模型，找出规则找不到的姓名、地址和密码（下载 1.6 GB，用 CPU） |
| **到处都能换回来** | 一个值变成 `{{API_KEY_k3v9x2mq}}` 这样的占位符，回复里每个占位符都换回原值——流式文本和工具调用参数也一样，所以 agent 执行的命令拿到的是真 key |
| **不会弄坏对话** | 同一个值每次都得到同一个占位符，对话、提示词缓存和 thinking 签名都保持有效。一个值被脱敏过一次，之后出现在哪里都会脱敏 |
| **不落盘** | 值只保存在内存里。记录里只有它的类型、长度和开头几个字符。脱敏开着但跑不起来时，请求会被拦下（503），不会原样发出去 |
| **可以检查** | **洞察 › 脱敏** 列出每一个被改过的请求，和供应商实际收到的内容并排显示；设置里的 **试试看** 会告诉你一段文字会变成什么 |
| **你自己的规则** | 密钥和个人信息可以分开开关，还有更细的类型（护照、IBAN、MAC、公网 IP……）、你自己的词和模式，以及命中后是脱敏、只警告，还是拦下（**脱敏**、**警告**、**拦截**） |

代价是：模型看到的是 `{{API_KEY_…}}`，不是 key 本身，所以"这个 token 为什么被拒"这类针对值本身的任务做不了。把这类值放进 **永不脱敏**。规则偶尔也会判断错，脱敏页就是发现它的地方。它默认关闭，在 **设置 → 隐私** 里打开。示例域名（`example.com`、`.test`、`.invalid`）下的邮箱会原样保留。

### 供应商与订阅

选预设、粘贴 key。模型列表来自供应商，名称和推理档位由 [models.dev](https://models.dev) 补全，所以今早刚发布的模型，下次刷新就在你的选择列表里。

> **预设包括** Anthropic · OpenAI · Google Gemini · Google Vertex AI · Amazon Bedrock（runtime 与 Mantle）· DeepSeek · Kimi · 智谱 GLM · MiniMax · StepFun · LongCat · Qwen · 百度千帆 · 腾讯云 · 华为云 MaaS · 火山引擎方舟 · Mistral · Groq · Nous Portal · xAI · OpenRouter · Together · Fireworks · SiliconFlow · NVIDIA NIM · ModelScope · Ollama · LM Studio……以及任何 OpenAI 或 Anthropic 兼容的 URL。

**导入** 可以从 CC Switch、Claude Code、Codex 和 Alma 带入供应商。**[Add to owl](https://github.com/istoo/owl/blob/main/site/public/docs/zh/import.html)** 链接让供应商官网一键把配置交给 owl。每个 key、每个账号都可以走自己的代理。

你已经登录的 agent 是一个**订阅**，后面有一串模型，于是 owl 可以把它当成供应商给所有其他 agent 用。同一个订阅可以加多个账号，owl 会在它们之间故障转移。

- **Claude**：驱动本机真实的 `claude` 二进制，并通过 MCP 桥接你的 agent 的工具
- **Codex / ChatGPT**：你的 ChatGPT 订阅里的模型，给其他所有 agent 用
- **GitHub Copilot**、**Gemini（Code Assist）**、**Antigravity**、**Grok（SuperGrok）** 等
- npm 上**任何 OpenCode 登录插件或 pi 包**

**每日预热** 会在你指定的时间，或者上一个窗口重置的那一刻，开启每个 Claude 和 ChatGPT 账号的 5 小时窗口。

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/add-dark.png">
  <img src="site/public/img/add-light.png" width="900" alt="添加供应商：可登录的订阅、厂商、中转和本地服务器">
</picture>

### 路由分组

**路由分组**是几个模型，agent 把它们当成一个来选，比如 `group/daily-coding`。网关把请求分到每个成员的 key 和账号上：

| 模式 | 它会做什么 |
| :-- | :-- |
| **`smart`** | 在还有额度的订阅里，用最早重置的那个，重置时浪费得最少 |
| **`order`** | 一直用第一个模型，直到它答不上来，再换下一个 |
| **`rotate`** | 每一轮换到下一个成员 |
| **`usage`** | 先用用得最少的成员 |
| **`pace`** | 用每小时剩余额度最多的账号，直到它重置 |

只要供应商的提示词缓存还值得保留，一段对话就**留在回答它的那个账号上**。分组里可以再放分组。**[意图路由](https://github.com/istoo/owl/blob/main/site/public/docs/zh/intent.html)** 让你选一个小模型读每一轮的新请求，测试交给强模型，简单问题交给便宜的模型。Routing 页会实时显示每一次决定。

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/routing-dark.png">
  <img width="760" src="site/public/img/routing-light.png" alt="Routing 页：四个 agent 经过 owl 到七个供应商，实时">
</picture>

### 插件

```sh
owl plugin add opencode-gemini-auth   # npm 上的 OpenCode 插件
owl plugin add pi-antigravity         # pi 包，同样的装法
owl plugin login google-plugin        # 它自己的登录流程，在 owl 里
```

插件跑在 [Bun](https://bun.sh) 上，第一次需要时自动下载。插件负责登录、列出模型、发出请求；agent 像用其他供应商一样用这些模型。去 **[magpie-community/plugins](https://github.com/magpie-community/plugins)** 找，或者[自己写一个](https://github.com/istoo/owl/blob/main/site/public/docs/zh/plugins.html)。

插件也可以是**网关中间件**：看得见每个 agent 发出去和收回来的一切的 JavaScript。`onRequest` 可以改写请求或拦下它，`onEvent` 看得见每个流式事件，`onResponse` 看到完整回复。抛错或者跑太久的钩子不会影响请求，请求按原样发出。

```js
// alias.middleware.js — owl plugin add ./alias.middleware.js
export function onRequest(body, ctx) {
  if (body.model === "fast") return { ...body, model: "deepseek/deepseek-chat" };
}
```

<details>
<summary><b>现成的中间件</b>，大部分是 <a href="https://github.com/QuantumNous/new-api">New API</a> 给渠道做的功能，JSON 也一样</summary>

<br>

| 包 | 它做什么 |
| :-- | :-- |
| `param-override` | New API 的 `param_override`：按条件设置、删除、移动或改写请求字段，或者拦下请求 |
| `model-map` | New API 的 `model_mapping`：用另一个名字发出模型；回复仍然用被请求的名字 |
| `system-prompt` | 给每个请求加上你的 system prompt，或者只给某些 agent、某些模型 |
| `word-guard` | New API 的敏感词过滤：拦下或替换用户发来的词，以及回复里的词 |
| `think-tags` | 把 ` thinking…` 从回复里拿掉，或者把 `reasoning_content` 放进去 |

```sh
owl plugin add @magpie-community/middleware-model-map
owl plugin options model-map '{"mapping": {"fast": "deepseek/deepseek-chat"}}'
```

在 插件 › 发现 › 网关中间件 里能找到它们。见[网关中间件](https://github.com/istoo/owl/blob/main/site/public/docs/zh/plugins.html#middleware)。

</details>

### 用量、共享和其他

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/usage-dark.png">
  <img src="site/public/img/usage-light.png" width="900" alt="用量页：余额、token、缓存命中率和花费">
</picture>

| | |
| :-- | :-- |
| **Token 与花费** | token、缓存读写、推理和调用次数，按标价用美元或人民币计价；任何模型都可以设自己的价格 |
| **额度** | 每个 key、订阅和账号的余额与额度窗口（`owl quota`），窗口重置前还有大量没用会提醒 |
| **会话** | 每一段 agent 对话和它的花费、标题，一键在终端里重新打开 |
| **上下文** | 每个请求的上下文窗口有多满、多少来自缓存、是什么占满的 |
| **账单** | 按账号和按上游 key 统计，并可导出 OTLP 到你自己的可观测性平台 |
| **局域网** | 打开 **在局域网共享**，给每个客户端一个命名的网关 key，各自有每天、每周或每月的 token 和花费上限 |
| **[远程 owl](https://github.com/istoo/owl/blob/main/site/public/docs/zh/remote.html)** | 笔记本用你台式机上 owl 的供应商、账号和分组，同时仍然接自己的 agent |
| **[Docker](https://github.com/istoo/owl/blob/main/site/public/docs/zh/docker.html)** | 在服务器或 NAS 上跑 `ghcr.io/istoo/owl`，从网页界面管理 |
| **同步** | 备份到文件，或者通过 WebDAV（坚果云、Nextcloud……）、S3，或 GitHub 上的加密仓库在机器之间同步 |

- **资料库**：指令、MCP 服务器、技能和子 agent 只写一次；owl 按各 agent 的格式写进它的文件，并且只删除自己写进去的东西。
- **给任何模型加联网搜索**：不会搜索的模型可以用 owl 的搜索——一个能搜的供应商，或者搜索 API（Tavily、Brave、Exa、Firecrawl、You.com、DeepSeek、豆包搜索、腾讯元宝搜索、SearXNG），按你拖动的顺序尝试。任何 agent 都能把同一个搜索挂成 MCP 服务器，带 `web_search` 和 `web_fetch`。
- **图片与视频**：通过 MCP 给任何 agent 图片工具，模型由你选；视频用 Grok 订阅或 MiniMax 的 API key。
- **让 agent 保持最新**：Agents 页显示每个 CLI 的版本，按它安装的方式更新，没有安装的给出安装命令。
- **WSL**、**保持唤醒**、明暗主题，以及在 [Omarchy](https://omarchy.org) 上跟随主题。

## 支持的 agent

<table>
<tr><td>

Claude Code · Claude Desktop · Claude Science · Codex · Gemini CLI · Antigravity CLI · OpenCode · OpenChamber · MiMo Code · DevEco Code · Pi · oh-my-pi · StepCode · MothX · PenguinHarness · Aside · OmO · Prime Agent · Goose · Cursor CLI · Cursor Private Inference · Zed · VS Code Chat · VS Code Insiders · VSCodium Chat · Zoo Code · JetBrains Air · Air（JetBrains IDE）· Copilot（JetBrains）· Copilot CLI · Crush · DeepSeek Harness · Reasonix Studio · Command Code · fx · Devin · Hermes Agent · Mister Morph · Kimi Code · jcode · Qwen Code · Muse Code · Empryo · MiniMax Code · Droid · Cline · Qoder · Qoder CN · Grok Build · Grok App · ZCode · WorkBuddy · WorkBuddy AI · CodeBuddy · Pencil · OpenPencil · OpenBot · NewMax · Raycast · Kiro Crew · T3 Code · OpenHanako · AtomCode · Snow CLI · Alma · Cindy · Lody · AionUi · Ante · AstrBot · QwenPaw · OpenClaw

</td></tr>
</table>

owl 只显示你机器上装了的 agent；其中一些的安装说明在[完整参考](docs/reference.md#notes-on-some-agents)里。

## 文档

| | |
| :-- | :-- |
| **[快速上手](https://github.com/istoo/owl/blob/main/site/public/docs/zh/start.html)** | 一步步的导览 |
| **[完整参考（英文）](docs/reference.md)** | 每个 agent、供应商选项、网关端点、CLI 命令和文件 |
| **[插件](https://github.com/istoo/owl/blob/main/site/public/docs/zh/plugins.html)** | 使用插件，以及自己写 |
| **[意图路由](https://github.com/istoo/owl/blob/main/site/public/docs/zh/intent.html)** | 按每一轮要做什么来路由 |
| **[导入链接](https://github.com/istoo/owl/blob/main/site/public/docs/zh/import.html)** | 给供应商官网用的 "Add to owl" 按钮 |
| **[远程 owl](https://github.com/istoo/owl/blob/main/site/public/docs/zh/remote.html)** | 在别的电脑上用一台 owl 的供应商和分组 |
| **[Docker](https://github.com/istoo/owl/blob/main/site/public/docs/zh/docker.html)** | 在服务器或 NAS 上跑 owl |

## 隐私

你的提示词、回复、密钥和账号只发给你用的供应商；打开[隐私脱敏](#隐私脱敏)后，其中的密钥连供应商也收不到。这个 owl 构建不向官方上报任何东西：没有用量事件、没有伙伴列表、没有图标代理。它的更新、发行说明和显示的图片都不经过别的服务，只来自这个 GitHub 仓库。

## 社区

有问题、有想法，或者某个模型就是不出现？**[开一个 issue](https://github.com/istoo/owl/issues)**，反馈都走这里。

**本仓库不接受 Pull Request**，只有维护者能提交。想要修复或新功能，在 issue 里描述清楚，我们来实现。

如果 owl 帮你少改了一次配置文件，**点个 ⭐ 能让更多人发现它。**

<a href="https://star-history.com/#istoo/owl&Date">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=istoo/owl&type=Date&theme=dark">
    <img src="https://api.star-history.com/svg?repos=istoo/owl&type=Date" width="600" alt="Star 历史">
  </picture>
</a>

## 许可证

[MIT](LICENSE)

owl 与 [Apache Owl](https://owl.apache.org) 及 [Apache Software Foundation](https://apache.org) 无关。
