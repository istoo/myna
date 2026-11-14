# 本仓库的教训（magpie / Myna fork）

每条：规则 → 为什么 → 证据。上游的 LESSONS.md 仍然有效，这里只记本 fork 特有的。

## 占位符会替换我的工具调用与命令参数，不只是文本

写文件或传命令参数时，像 `{{EMAIL_xxx}}` 这样的占位符会被替换成真实值；**看起来像占位符的普通词**也可能被替换，而工具输出又把真实值回显成占位符，导致"我改了但看起来没改"。

- 2026-11-09：`.github/workflows/privacy-filter.yml` 的 matrix 第一项和 `gh workflow run -f targets=...` 的值都被替换，CI 收到的 target 名不匹配，5 个 job 全部空跑（0 秒完成）。
- 解决：需要写入精确字面量时用拼接（`'darwin-'+'ar'+'m64'`）或按行号替换；用字符码（`[ord(c) for c in line]`）核对文件真实内容；给 CI 传参时传不含占位符形式的值（如 `targets=all`）。

## 工具输出会把 npm 的依赖目录名换成一个随机串

CI 日志、`ls`、`npm config` 里出现的那个 8 字符小写串，是 npm 的依赖目录名（`node_` 开头、`modules` 结尾，14 个字符）被显示层替换后的样子；**文件本身是对的**。见到"路径不存在"的结论，先用十六进制核对该行再下判断。

- 2026-11-10：`gui.yml` 的 `NODE_PATH=$RUNNER_TEMP/pw/<那个串>` 看起来指向不存在的目录，实际 CI 上解析正常、测试全绿。差点被当成 bug "修正"，那样会打断 GUI 工作流。
- 核对：`python3 -I -c "b=open(p,'rb').read();i=b.find(b'NODE_PATH=');print(b[i:i+40].hex())"`。

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

## GUI 套件按 Makefile 的并发跑，别自己加大

- 2026-11-10：本机用 `--test-concurrency=8` 跑全量，2875 项里 2303 项以 "Promise resolution is still pending but the event loop has already resolved" 整文件失败；
  挑出三个失败文件单独跑，30/22/20 项全绿。是浏览器被并发压垮，不是代码问题。
- 规则：用 `make test-ui`（`UI_TEST_CONCURRENCY` 默认 2）；本机最多 4。整文件报上面这句时，先单独跑该文件再下结论。

## CI 的 GUI 推送只跑冒烟集，全量是“只报告”

- 2026-11-11：上一轮汇报写“CI 的 GUI 全套在 cac2297c7 上全绿”，实际 gui.yml 在 push 时只跑 7 个冒烟文件；
  全量只在每晚定时任务或手动触发 `mode=full` 时跑，而且只跑 Chromium，结果为报告模式（exit 0），全绿的颜色不代表没有红。
  手机标签栏移植后，一个晚上的全量都没跑过，十几个文件就一直红着。
- 规则：说“GUI 全绿”之前，要么本机逐文件跑完，要么 `gh workflow run gui.yml -f mode=full`，然后读日志里的 `shard N/4, chromium: X failed`
  和 `- \`file\`` 清单，不看颜色。汇报里写清楚跑的是冒烟集还是全量。

## zsh 里 `for x in $LIST` 不拆分

- 2026-11-11 同一天第三次踩：`for f in $L` 在 zsh 里把整串当一个值，清单文件只写出一行，容器里的循环全部跑空；
  `docker exec` 不带 `-i` 时 stdin 也传不进去。两次都“成功退出、零输出”。
- 规则：拼文件清单一律 `bash -c '...'` 或直接用数组字面量；循环跑完先 `wc -l` 清单、再看汇总行数是否等于文件数，零输出就是没跑。

## 懒加载脚本要登记进 old-webkit 的闸门

- 2026-11-12：`panelupdate.js` 由 app.js 动态 import，old-webkit 的清单没登记，main 上该测试一直红
  （"panelupdate.js is not loaded by index.html"）。push 只跑冒烟集，没跑到它，所以红了没人看见。
- 规则：新增 `assets/*.js` 并被动态 import 时，同一个提交把它写进 `old-webkit.test.cjs` 的清单和注释，并本地跑一次 old-webkit。

## 发给窗口的单向请求要自己吞掉失败

- 2026-11-12：面板每次重绘都发 `window/fit` 且不等结果；reload 或关窗时请求被取消，未捕获的 rejection 在 WebKit 变成页面错误
  "TypeError: Load failed"，update-hide 在 Linux CI 红（#24）。
- 规则：`api("window/...")` 这类不看返回值的调用一律 `.catch(() => {})`；这样改过的面板测试要断言 `errors` 为空。

## “shard 1/4” 不代表全量

- 2026-11-12：push 触发的 GUI 也是 4 个分片，容易当成全量；实际只跑 7 个冒烟文件。全量要读日志里的 `mode=full`
  和 `shard N/4, browser: X failed of Y`——它是报告模式，有红也 exit 0。
- 规则：说“全量全绿”必须贴 `failed of` 的数字；本机跑全量并发最多 4（并发 6 在负载 27 时出现 180 秒卡顿）。

## 活过请求的结构不能存请求的 context

- 2026-11-13：`toolScan` 只为了 `ctx.Err()` 存了整个 `r.Context()`，而 Route 留 60 条、`recent` 留 40 条；
  context 链上挂着这个请求的 hold（含请求本身与 body），于是每条保留的路由都钉住一整个请求体。
  60 个 1 MB 请求 GC 后残留 147 MB，改成只存 `Done()` channel 后 31 MB（再给 `modelOf` 加 `strings.Clone` 后 18 MB）。
- 规则：要判断"请求还在不在"就存 `context.Done()`；凡是会被长期结构保留的、从 body 读出的字符串（gjson/decodeJSON 可能共享底层字节），
  一律 `strings.Clone`。新增长期结构时用"60 个 1 MB 请求后活堆不涨"的测试兜底（`route_bodies_test.go`）。

## 打标签前必须先读等待器的结论

- 2026-11-13：我用 `cat 等待器输出; git tag …` 串成一条命令，等待器输出的是 `failure attempt 1`（gofmt 因新测试文件少了行尾换行而红），标签却照样打上并推了，Release 已开始构建才被我发现；只能取消工作流、删本地与远端标签、修格式、重推、等新 run 再打。
- 规则：标签只在读到 `success` 的分支里打——`grep -q '^success' || exit 1`，或先单独确认结论再执行下一条命令；本地提交前跑 `gofmt -l $(git ls-files '*.go')`，新文件尤其容易漏行尾换行。

## 规范化函数可能原样返回入参，回归测试要用真实请求的形状

- 2026-11-13：`effortOf` 写的是 `return strings.ToLower(s)`；s 已是小写时 Go 原样返回 s，而 s 是 decodeJSON 共享的 body 字节。
  Route.Tries[].Effort 于是钉住整份改写后的 body（占位说明副本、翻译 buffer），0.2.17 真机 15 分钟活堆 779 MB。
  上一轮的回归测试只发了不带 effort 的纯文本请求，所以绿着放过了它；fork 用简单请求"排除"了它，也是同一个盲区。
- 规则：`strings.ToLower/TrimSpace/TrimPrefix`、切片、`NearestEffort` 这类函数都可能把入参原样交出——从 body 读出的值要长期保存，
  就映射成常量或 `strings.Clone`。内存回归测试的请求体要照真实 agent 来写：effort、service_tier、脱敏占位、跨协议翻译、流式都带上。

## 反向代理在读完请求体前回复，net/http 会和代理抢读请求体（2026-11-14，#18）

- 规则：handler 把请求体交给 ReverseProxy 往上游转发、而上游可能在读完请求体之前就开始回复（流式接口）时，先对 rw 调 `http.NewResponseController(rw).EnableFullDuplex()`。
- 原因：响应头写出时如果请求体还没读完，net/http 会自己把剩下的请求体读掉丢弃（chunkWriter.writeHeader），和代理的 Transport 抢同一个 body。代理读到的 body 变短就报写错误，上游连接在流中途被关掉。body 还在路上时，响应头要等它整个到达才发得出去。
- 复现：单独压测 900 次都是绿的。用 io.Pipe 压住请求体的后半段，等第一条事件到了再发，这样能稳定在旧代码上失败。测试里的 Go 假服务自己也要开 full duplex，否则卡住的是假服务而不是被测代码。

## 按整个堆量的内存上限，要在只跑它自己的进程里量（2026-11-14，880db0b90 的 CI 红）

- 规则：断言「某操作后存活堆增长不超过 N」的测试，用子进程只跑它自己来量（照 `TestClaudeIdleEvictionEnds` 的写法）。上限的依据是两种状态之间的差值（填满小请求体，再全部换成大请求体），不是一个绝对总量。
- 原因：`runtime.ReadMemStats` 量的是整个进程的堆。同一个包里前面测试留下、还在跑的东西会算进来：全包顺序下同一个量起伏 70、58、-84 MB，CI 上 49 MB 越过了 45 MB 的上限，其实什么都没被扣住。
- 怎么验证：分别把这个上限要防的几种回退手动放回去（这次是去掉 `strings.Clone`、撤回 effortOf 修复），看测试都会失败。正常情况要离上限远，回退的情况也要离上限远。
