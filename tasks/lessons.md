# 本仓库的教训（magpie / Myna fork）

每条：规则 → 为什么 → 证据。上游的 LESSONS.md 仍然有效，这里只记本 fork 特有的。

## 占位符会替换我的工具调用与命令参数，不只是文本

写文件或传命令参数时，像 `{{EMAIL_xxx}}` 这样的占位符会被替换成真实值；**看起来像占位符的普通词**也可能被替换，而工具输出又把真实值回显成占位符，导致"我改了但看起来没改"。

- 2026-11-09：`.github/workflows/privacy-filter.yml` 的 matrix 第一项和 `gh workflow run -f targets=...` 的值都被替换，CI 收到的 target 名不匹配，5 个 job 全部空跑（0 秒完成）。
- 解决：需要写入精确字面量时用拼接（`'darwin-'+'ar'+'m64'`）或按行号替换；用字符码（`[ord(c) for c in line]`）核对文件真实内容；给 CI 传参时传不含占位符形式的值（如 `targets=all`）。

## main 上的推送会排队，不会取消，但会拖慢反馈

`test.yml` 的并发设置对 main 是 `cancel-in-progress: false`：连续推送不会互相取消，但会串成队列。

- 2026-11-09：循环优化期间连续推送 4 次，`2fdd5df4b` 的 run 被后来的取代（cancelled），验证窗口被推迟。
- 规则：一次推送带 2–3 个改动；要单独验证一个特性时，等当前 run 结束再推。

## CI 的墙钟由最长的 job 决定，串行的独立阶段应拆开

- `Test` 与 `Race` 原本在同一 job 里串行：macOS 22m + 19m = 43m。拆成并行 job 后 macOS 27m，ubuntu 33m → 17m（run 42595707169）。
## CI 的每个独立 job 都会暴露前一环掩盖的问题

- 把 `Race` 从 `Test` 里拆出来后，第一次跑就抓到 `TestAutoVisionKeepsOffARefusedAccount` 的真实数据竞争（测试与 usage stand-in 共享普通 float64）。串行时它藏在 Test 之后。
- 规则：拆 job 后先看新 job 的第一次结果，别只庆祝墙钟变短。

## 时间测试的两侧都要留余量，按 CI 上测到的值定

- `TestTryTellsWhereItsTimeWent` 的 300ms hold 在 ubuntu runner 上测到 298.7ms，而下界写死 299ms（11-10 红）。
- 规则：下界按 "hold 减去若干毫秒" 写，并在注释里记下 CI 测到的实际值。

## Linux runner 上的 GUI 全量是报告，不是门禁

- 夜间全量 4 片：4,887 / 5,332 / 4,617 项，失败 47 / 55 / 83，两个引擎都有，多数是字体与滚动步进差异（300px 滚轮在 Linux WebKit 只滚 15px），本机同文件全绿。
- 规则：夜间全量永远成功、把失败数写进 job summary、日志留 artifact；push 上的冒烟子集才是门禁。清单见 issue #13。
