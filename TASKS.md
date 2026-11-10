# Owl 品牌化改造任务（2026-11-10 由 Myna 全面改名）

目标：把 magpie 变成自己的发行版 Myna，更新与所有官方网络依赖走自己的 GitHub 仓库，独立配置并迁移旧配置。

## 改名 Myna → Owl（2026-11-10）

- [x] 机械改名：module `github.com/istoo/owl` 与全部 import（3146 文件）、二进制/应用名、`~/.config/owl`/`~/.cache/owl`、
      Bundle ID 与 LaunchAgent `com.istoo.owl`、Windows Run 值 `owl`、Omarchy widget `useowl.owl`、网关 `/v1/owl/*`、`OWL_*`、
      文档与站点。文件名含 myna 的 45 个路径一并改名（`git mv`）。
- [x] 旧数据迁移：`appdir` 依次从 `~/.config/myna`、`~/.config/magpie` 复制（新测试覆盖"myna 优先、不改 owl 已有"）；
      启动时删除旧 LaunchAgent `com.istoo.myna.plist`、Windows Run 值 `myna`、旧 Omarchy widget `usemyna.myna`（各有测试）。
- [x] 图标：`build/icon/owl.svg`、`owl-small.svg` 换成正面猫头鹰（圆身、两只大眼、三角喙、双耳羽），
      header logo（`index.html` 的 `#bird` 与 body/tail/wing clip）、favicon、add-to 徽章、架构图、PWA 图标全部重画；
      `make icons` 重建 icns/ico/png；Dock 与托盘的 flap 保留三分片与关键帧（折叠的双翅只转 1°）。
- [x] 仓库改名：`gh repo rename owl`（istoo/myna → istoo/owl，旧地址重定向），本地 origin 同步。
- [x] 验证：`go build ./...`、`go vet ./...`、`go test ./internal/{appdir,autostart,omarchy,update}`（203）、
      `go test ./internal/gui -run 'Icon|Flap|Logo|Manifest|Tray'`（62）、`go test ./internal/provider -run 'Logo|Signin|Brand'`（5）、
      GUI 冒烟 7 文件 56/56（chromium）、`make icons`。PR #15 已合并（squash `4c331d207`），main 的 Test/GUI/Docker 全绿。
- [x] 真实迁移验证（沙箱 HOME）：放 `~/.config/myna/{providers.json,settings.json}` 后运行编译出的 owl，
      日志 `appdir: brought owl's files over from …/.config/myna`，两个文件都出现在 `~/.config/owl`。
- [x] 兼容与发布 `v0.1.2910`：release 同时上传 `myna-*` 资产（Mac 的 zip 内部是 `myna.app`、二进制与签名同一份），
      `latest.json` 列两套名字；网关把 `/v1/myna/*` 改写到 `/v1/owl/*`，插件子进程把 `OWL_*` 以 `MYNA_*` 再给一份
      （下一版撤）。`magpie-community/plugins` 全仓库零处 myna 引用，无需改。
      验证：19 个资产（8 myna + 8 owl + install/latest/notes）；`latest.json` 16 条；下载 `myna-darwin-arm64.zip` 检查为
      `myna.app/` 且 `Info.plist` 是 owl；Release 与 Docker 两个 workflow 全绿；Latest 指向 2910。
- [x] 彻底撤掉 myna（2026-11-10，用户定：代码与 GitHub 一起删）：
      代码里删掉 `/v1/myna/*` 改写、插件环境 `MYNA_*` 镜像、`~/.config/myna`/`~/.cache/myna` 迁移、
      旧 LaunchAgent/autostart/widget 清理、会话与 provider id 的 `myna` 识别、provider id 保留名、
      release 的 `myna-*` 资产别名；相关 fixture 改名（GitHub/npm 外部快照保留原文）。
      保留上游 magpie 契约（`magpie` 插件键、`@magpie-community`、`X-Magpie-*`、`~/.config/magpie` 迁移）。
      GitHub：仓库描述改 Owl；删掉 2910 的 8 个 `myna-*` 资产并重传去掉它们的 `latest.json`；
      `notes.json` 与发布说明里的旧名行删掉；v0.1.2896–2909 共 14 个 release 删除（tag 保留）。
      验证：`go build ./...`、`go vet`、`gofmt -l` 干净；`go test ./internal/{appdir,sessions,autostart,omarchy,provider}` 3076 绿、
      gateway 路由 331 绿、plugin 除本机既有红 `TestHostToldTheGatewayAddress` 外全绿（CI 上该包绿）；
      线上 2910 只剩 11 个 owl 资产，`latest.json`/`notes.json` 各 0 处 myna。
      CI 红另记 [#17](https://github.com/istoo/owl/issues/17)（`TestKeepOffWaitsForTheSetting`，macOS，本地 5/5 通过）。
- [x] 社区只保留 GitHub（2026-11-10）：站点社区卡片、文档页头与页脚、应用 About 行都去掉 Discord；
      README 重写为 owl 版本（命令、端点与事实沿用，文档链接改指真实存在的 `site/public/docs/*.html`）。
- [ ] **下一版发布前**：撤掉网关的 `/v1/myna/*` 改写与插件环境的 `MYNA_*` 镜像（用户 11-10 定：按计划撤），
      并在发布说明里写明；社区插件仓库经查零处引用，撤掉不影响它们。Mac 上从旧版升级的 `myna.app` 目录不重命名（签名/LaunchServices 风险）。
- [x] routing-sky-under（CI 3 条）：两处——第三只请求的点在飞行途中被读到（900ms 固定等待），改为等所有点位置连续 5 帧不变；
      另一处是几何：点半径 4.5、560px 中日文下点边距卡片 3.98px，边沿相交 0.52px，比 0.5 容差多 0.02，判定改为 1px（两像素贴边即相接）。
      容器 webkit 20/20（原 18/20）、macOS chromium 20/20、macOS webkit 20/20。`c5ddec61b`
- [ ] 未验证：Mac 上 Dock 磁贴的 flap 动画（新图形）在真机上的观感。
- 记录原则：`LESSONS.md`、`tasks/lessons.md` 与本文件的历史叙述保持原样（它们是历史事实），新内容用新名。

- [x] 0. 创建 istoo/myna 仓库，设置 origin/upstream
- [x] 1. module 路径与全局标识改名（magpie → myna，com.yetone → com.istoo，MAGPIE_ → MYNA_）
- [x] 2. 更新链路改为 istoo/myna Releases（feed、资产名、Release 页面、notes）
- [x] 3. 去掉 usemagpie.ai 依赖（图标代理、合作伙伴、文档/官网链接、MCP client_uri、统计上报）
- [x] 4. 配置/缓存/日志目录改为 myna，并从 ~/.config/magpie 自动迁移
- [x] 5. Bundle/LaunchAgent/守护进程身份改 com.istoo.myna
- [x] 6. Makefile 产物改名，app 构建后 ad-hoc 签名（更新校验要求）
- [x] 7. 新图标（app 图标已替换；托盘沿用同一只鸟的模板图）
- [x] 8. CI：tag 触发构建、签名、打包、生成 latest.json/notes.json 并发布
- [x] 9. 构建 + go test 全量验证（gateway 单跑 601s 通过；lane/SOCKS 为环境性失败，已分别证实）
- [x] 10. 安装最终版 Myna，验证网关与自动更新可用（0.1.2892 → 0.1.2896 真机更新通过）
- [x] 11. 清理旧 magpie（应用、LaunchAgent、登录项已移除；旧 MCP 子进程等客户端重启后换新）

兼容性边界（有意保留 magpie 字样）：
- 插件 package.json 的 `magpie` 键与 `@magpie-community` 生态：第三方插件契约。
- 插件请求头 `X-Magpie-*`：第三方插件读取的协议。
- 会话/配置里上游写下的 `magpie_*` JSON 键：旧数据读取。
- dsh 的 `magpie*` 路由名：旧会话归类。

验证方式：`go vet ./...`、`go test ./...`、`make app` 后 `codesign --verify --deep --strict`、本地起 feed 服务走完一次自更新、网关 200。
## 后续架构任务（审计遗留，需真实环境验证，不能盲改）

- [x] 插件间请求劫持（审计发现的那一面）：`globalThis.fetch` 按插件作用域解析
      （`host.js` 的 `pluginALS`/`pluginFetches`，Pi 加载器保留兜底），一个插件换掉的
      fetch 不再作用于别的插件；`Bun.fetch` 实测为只读、不可配置（Bun 1.4.3），
      插件改不了。`node:http` 等内建模块的默认导出仍可被插件改写，只影响直接用它们的
      插件；要堵住它需要每插件独立 Bun 子进程 + `input.client` RPC 代理。
      评估结论：暂不做独立进程——插件在宿主内本就是受信代码（可调 `input.client`、
      读宿主环境），进程隔离不构成新的安全边界，只防误伤；代价是每插件一个运行时
      （内存显著上升），且必须在真实 grok/cursor/workbuddy 插件上逐家重验
      sign-in、models、usage、请求，不能盲改。将来要做时按这条起。
- [x] 回环对端 OS 用户检查（`internal/gateway/peeruid`、`peeruser.go`）：
      macOS `sysctl net.inet.tcp.pcblist` 的 `xsocket.so_uid`，Linux/Android
      `/proc/net/tcp[6]` 的 uid 列，Windows `GetExtendedTcpTable` + token SID；
      每连接在 `ConnContext` 读一次，结果并入 `local()`，另一 OS 用户
      的回环请求按“另一台机器”处理。
      验证：macOS 实机 `TestSameOwnConnection` 通过；Linux/aarch64 实车
      （5.15 内核）上整包测试通过，含 `setuid(65534)` 子进程的
      `TestSameRefusesAnotherUser`；Windows 交叉编译 + vet，CI 新增
      windows-latest 运行 `TestSameOwnConnection` 的步骤。
- [x] 取走的账号不被迟到的轮换写回来（#1）：`host.js` 的 `client.auth.set`
      现在只更新"请求所属账号"仍在的账号；被登出/被搬移后，插件手里那份
      旧 token 不再把它签回来（登录流程与插件写另一 provider 的记录不受限）。
      验证：新测试 `TestTakenAccountNotSignedBackInByALateListing` 在旧
      host.js 上红、修复后绿；`TestMoveCutShort` 负载下 200 次由 10 红变 0 红。
- [x] `TestClientGoneRecordedAsCanceled`（#2）：不再"开始后睡 200ms 再取消"，
      改为等回复内容真正写到 writer 后取消（`saysWhen`），CI 负载下不再
      在回复开始前就结束 run。
- [x] 新连接被对端关闭时，读端报 EOF 也按“未发出”重试
      （`internal/gateway/upstream_transport.go` 的 `countedBody`/
      `closedByPeer`）：写循环没读完请求体就说明没写完，写端、读端哪个先
      看到关闭都重试；读完整的不重试。测试改成逐片写体（`dripped`）。
      验证：`-run '^TestClosed' -count=20` 80 通过；旧代码下 8KB 体复现
      CI 的 `EOF, want errClosedUnsent`，8MB 慢体 3 次中 1 次复现。
- [x] Claude Code 退出时的错误带上它最后画的内容：`claudetui` 的终端读取
      结束（`readDone`）后再取屏幕（`endedScreen`），不再在 `Wait` 回收进程
      的瞬间取到空屏（ubuntu CI 的 `TestShimNeverPrints/failed_start`）。
      验证：`go test ./internal/claudetui -count=1` 48 通过；
      `TestShimNeverPrints -count=5` 通过；全量除 #3 的环境项外全绿。
- [x] 压缩请求不被 keepalive 提前提交 200：`askCompaction` 标记
      `noKeepAliveKey`，`holdWriter.keepAlive` 跳过；字节上限的 400 不再
      变成“200 + 流内错误”，`codexCompact` 能继续剥图并本地摘要。
      验证：四个相关测试 `-race -count=3` 24 通过（改动前 4 红）；
      网关整包 `-race`（1291s）除 #3 环境项外全绿。
- [x] `-race` 下三个大请求/大行测试的限值放宽（`roomForRace`、24MB 行
      30s）：CI 的 Race 分片在 `-race` 下是机器慢而不是回归。
      验证：三个测试 `-race` 7 通过；网关整包 `-race`（1281s）除 #3 外全绿。
- [x] 客户端离开时的回复结尾：`TestClientGoneRecordedAsCanceled` 接受
      `message_stop` 或 `event: error` 两种终态（#4822 的竞态），仍要求内容
      出现且不计为完整回复（499）。验证：`-race -count=10` 通过。
- [x] 用户确认 11-05：保留 v0.1.2902 发布；站点 worker 文件保留；
      插件不做独立进程隔离；GUI 测试的“会话无法判定即跳过”策略保留。
- [x] CI 全绿：`10b82b3aa` 的 run 41946918501，ubuntu/macOS/Windows 与两个 CLI 全部
      通过（含 Race 分片 45m）。#1、#2 已关闭；#3（本机环境）、#4（宿主端 CAS）为
      已记录的遗留。
- [x] `-race` 余量补齐到其余 26–42MB 的字节上限测试（`roomForRace`），
      ubuntu Race 分片的 504 不再出现。验证：四个测试 `-race` 66s 通过。
- [x] 站点 worker 文件：保留（用户 11-05 确认）。`site/worker.js` 及
      `partners.js`/`sponsors.js`/`i18n.js` 是站点源码，隐私页（三种语言）与
      `docs/reference.md` 直接引用它们；本 fork 不部署该 worker，但删除会
      让这些文档失效，属于内容问题而非可安全删除的死代码。
- [x] GUI 套件的既有红（另开任务，用户 11-06 决定）：为验证“删除脱敏
      演示”，1182 个文件分 12 片跑完——19,913 项，19,561 通过，352 失败；
      把 67 个失败文件单独复跑（1,312 项）仍有 284 例真实失败，不是并发
      负载。产品资源与 `v0.1.2903` 逐字节一致（
      `git diff v0.1.2903 HEAD -- internal/gui/assets` 为空），且 GUI 套件
      不在 CI，所以这些红在此前发布即存在，与本次改动无关。主要文件：
      usage-agents-hide(20)、model-chip-steady(20)、segs-press-redraw(16)、
      rest-try-now(15)、provider-fetch-again(14)、page-width-steady(11)、
      routing-rule-remove(10)、recovered-file(10)、plugin-registry(10)、
      zero-priced-free(9)、vertex-editor(8)、models-works-only(8)、
      models-select-all-api(7)、context-map(5)、usage-ledger(5) 等 67 个。
      复跑证据：`/tmp/ui-chunk-*.log`、`/tmp/ui-redo.log`。
      11-07 收尾：44 个复跑仍红的文件已全部重跑，43 个通过；`recovered-file`
      的 5 个失败是本机缺少 Playwright headless shell（环境问题），安装后
      10/10。产品修复：路由故事框在 WebKit 上因滚动条宽度变化丢掉已测高度
      （`d9943a481`）；其余是测试的点击位置与等待条件（不按固定时长）。
      app.js 的三处产品修复让核心脚本超出预算 1,049 字节，以同处注释删减
      支付（`b9ad112e8`，app.js 1,311,658）。
      11-07 全量复核：1,182 个文件分 12 片跑完，19,913 项、19,870 通过、
      43 条失败（17 个文件）。25 条随修复转绿，含产品回归 `b45cd3f3c`
      （选中的请求被每次 trace 清掉）、`eb38e2a36`（规则条件 chip 在手机
      宽度溢出到框外）；5 条当时只在多片并发时出现、单跑全绿；剩余
      2 个文件 7 条（plugin-card-update 的 WebKit 卡片按钮、routing-many-keys
      的页脚键数）当时判为既有差异。这个判断已撤回：两者随后都修了，
      最终全量里均为 0 失败（见下）。
      11-07 第二轮全量：各片跑在不同提交上，09、11 片补跑；
      19,943 项、19,941 通过，剩下 2 条在 routing-many-keys。原因是测试的 wheel 指针
      固定在 (700,500)，滚过头后落在请求列表 `.rt-reqs` 上，该列表
      `overscroll-behavior: contain` 且 scrollTop 为 0，吞掉了向上的
      wheel（与 LESSONS 10-28 同类，产品行为正确）。现在每次滚动前把
      指针放到 view 内没有自滚动祖先的点（`bed8780b9`）。
      11-08 同一提交上的完整全量：`299abdb80` 上 12 片有 4 条红，3 个文件。
      tray-cell-click 是产品 bug：`f4bdc3da2` 依赖的 `quotaFocus` 在高亮
      前已清空，从未生效；改为卡片记住被点名的账户，重绘后把它留在
      视野内（`e05642e52`）。routing-rule-remove 也是产品 bug：菜单里的
      方向键被全局 keydown 记成读者滚动，400ms 内的重绘滚动就会关掉菜单
      （`1a16703cf`，新断言在旧代码上 20/20 红）。routing-rules-line 是
      测试问题：页面未停稳就点击（`114b28493`）。
      再在 `114b28493` 上跑 12 片：19,913 项、19,910 通过，3 条红都在
      library-all，原因是测量时 `.lib-body` 的入场动画还没结束，测试只等了
      行的父元素（`b8b29195e`，注入后旧写法红、新写法 26/26）。
      最终：`b8b29195e` 与 `114b28493` 之间只差 library-all 一个测试
      文件，第 06 片（含该文件）已在 `b8b29195e` 上整片重跑：1,812/1,812。
      合计：12 片 19,913 项、19,913 通过、0 失败（11 片在 `114b28493`，
      第 06 片在 `b8b29195e`，两者的产品代码相同）。

## 精简 / 交互 / 性能（2026-11-08 起）

约束：所有面向用户的功能保留（网关、订阅路由、全部 agent 接入、插件体系、附加能力）。
“精简”只删死代码、冗余实现、无引用资源。不 commit / push，等用户确认。
全量 `go test ./...` 最后由主会话统一跑一次（同机只跑一套）。

- [x] A. 网关 CPU/内存（主会话）结论：**无空转、无泄漏**。
      - 30s CPU profile：13% 单核，全部用在当时 4 个 Claude 会话的请求处理上（脱敏 25%、翻译/流式、PII CoreML 推理、GC）；去掉请求路径后只剩 GC 与上游 TLS 读。
      - 两次 heap 差分（间隔 5 分钟）增长只在处理中的请求体。footprint 734MB（ps 的 1GB 含已归还页），Go 堆约 240MB，GPU/图形 116MB。
      - 根因是规模：1MB 的 Claude Code 请求每次分配 22.8MB（21.7 倍），6.5h 合计 395GB。分配分散在多次解码/脱敏遍历中，不存在单点。
      - 试过并撤回（实测无收益）：hold 脱敏后请求体；mcp 子进程 GOMAXPROCS/FreeOSMemory（footprint 26MB→26MB，主要是框架加载）。
      - 待用户决定：单次解析流水线重构（大）；redact walker 零拷贝（隐私链路，需审计）；mcp 独立小二进制。
- [x] B. 死代码/冗余：已核对并应用到主工作区（54 文件 +96/−359；U1000 39→6，deadcode 32→4，剩余为构建标签保护项；18 包测试 ok，日志 /tmp/myna-deadcode/test-final.log）。
- [x] C1. 已合入：/api/state 冷启动 417→57ms（15 次串行 mdfind 改并发预取+in-flight 去重，主会话补“过期后再预取”及先红后绿测试）、首屏可交互 530→143ms、myna ls 0.46→0.19s、网关回复读缓冲按长度取（同负载分配 −17%）。网关附加首字节中位 0.26ms，切页 0 long task，app.js 0 死函数。
- [ ] C2. GUI 设计/易用性审查（subagent，只读）：出带截图的问题清单与方案，布局改动经用户确认后再做。
- [x] C2. GUI 审查完成（报告见会话；截图在 /tmp/myna-review/shots）。已核实：#2 CLI 写入非法 effort、#4 Esc 不关对话框、#5 浅色 muted/faint 色值。
- [x] E. 已合入主工作区（25 文件 +295/−34，26 字段 Closed，Codex/Grok 回落补全，WSL 克隆补 Closed；agent/cli/profile/gui ok）。CLI 封闭枚举校验：Field 加 Closed，Spell 校验；24 个 effort/subagent_effort 字段逐家核对 vendor 实际接受值
      （Codex 未入目录模型回落 low..xhigh，但 Codex 本身接受 none/minimal，不能误伤）。复现：`myna codex effort highish` 写入 config.toml。
- [x] F. 已合入（245 文件，均在 internal/gui/assets|tests；与 F-CLI/G/H 无冲突）。主会话修 update-download ja@420：hold() 的“停在顶部”阈值由 min(24,参照物高) 改为 24px（14px 的 code 参照物遇两行折行被当成整行放回）；22/22 绿，预算内。原：（C1 合并后，一个 subagent 顺序做，均改 app.js）用户 11-08 授权按业界最优自定：
      F1 非布局：#1 读取失败错误态+重试、#4 Esc 统一关对话框（经 confirmDiscard）、#5 浅色 muted/faint ≥4.5:1、#7 设置搜索孤立标题、
         #15 快速上手计数、#21 tablist aria-orientation、#16 侧栏网关行加 chevron、#17“隐藏侧边栏”改名、#18 长命令可换行、#14 文案。
      F2 布局：#3 窄屏导航 priority+（放不下的收进“更多”菜单，可键盘达）、#13 路由策略改单选下拉+当前项说明，限流/会话保持/模型收进“高级”、
         #20 账号行邮箱可换行+完整值 title、#6 行可聚焦 Enter 打开、#12 两态设置统一 switch。
      F3 #19 Cmd+K 命令面板（页面/设置项复用 searchSettings/供应商/Agent/插件），独立模块动态 import，不抬 app.js 预算。
      F4 #9 会话授权卡：加“暂不允许”，主按钮改“允许所选”，“全部允许”降为次要（避免诱导最宽授权）。
      [x] CLI/TUI 已合入（主会话把缺 key 示例从 provider.Save 挪到 CLI add 路径：ErrNeedsKey，GUI/TUI 不再显示 CLI 命令；suggest/tui/cli/provider ok）。CLI：#10 未知 agent did-you-mean（复用 groups_cli 的实现）、#11 `myna help <cmd>` 等同 `--help`、provider 缺 key 时给示例；TUI #8 `?` 帮助浮层+footer 按宽度截短。
- [x] G. 已合入（主会话审计零拷贝存储点：placeholder/scores/texts/tails/shape 均显式 Clone，bytesOf 仅同步只读；去 Clone 测试变红已复现）。1MB 脱敏翻译请求 33.6→20.7MB/req（−38.5%）、79.8k→37.8k mallocs（−52.6%），上游字节归一化后一致。网关分配削减（C1 合并后）：探针（1MB Claude Code 请求、脱敏开、翻译路由，基线 22.8MB/21.7x）作分配上限测试，逐项削减并保留数字；
      redact walker 零拷贝只在审计占位符存储不会钉住请求体后做，并按 LESSONS 把秘密放在工具参数/消息正文/嵌套对象里测。
- [x] 决定不做：MCP 独立小二进制。footprint 21MB/进程且多为共享框架页，换来打包与各 agent 配置路径变更，收益不抵风险。
- [x] H. 已实现并合入（codex_socket_keep.go 新增；gateway.go 两行接线；文档同步；11 个新测试，-race -count=20 通过，主工作区 37+90 通过）。11-09 真实 chatgpt.com（free 账号，codex/gpt-5.6-terra）验证：`MODEL=codex/gpt-5.6-terra scripts/check-codex-kept-socket.sh` 第一轮 whole 18374 B，第二轮 delta 18455 of 18622 B，回复 one/two 均 completed；省下的只是 input 部分，instructions 和 tools 每轮照发（与 codex-rs 相同），会话越长省得越多。ChatGPT Codex 后端 WS 增量接续（用户 11-08 提出）：codex_socket.go 每轮新开 socket、发全量、完成即关。
      改为按（主机+账号身份+session-id/thread-id/prompt_cache_key）保留上游连接；本轮除 input/client_metadata 外与上轮逐字节相同、
      且 input 以“上轮 input+上轮 k 个输出项（类型/call_id 校验）”开头时，只发 previous_response_id+新增项；否则全量。
      previous_response_not_found/回复前断开 → 新连接全量重发；并发轮次另开连接；空闲回收。验证：真实 ChatGPT 账号两轮，上游第二轮消息体只含新增。
- [x] D. 汇总验证（11-09）：go vet 三平台 ok；go test ./... 87 包 ok，claudebridge TestCallbackGoesPastTheAgentsProxy 与 netproxy TestLANModelListNotThroughProxy
      红，origin/main 6d4fd95f6 同样红（本机 Little Snitch 拦截未签名测试二进制的回环/LAN 连接），待开 issue。
      GUI：合入后 3 红（account-arrange/icon-url/key-weight）+ 补跑出 2 红（update-network/vertex-editor）均已按根因修（lastScrollInput 初值、
      按钮宽度保持、药丸显隐同步 fitTop）；hold() 阈值仅对“点击元素已重绘”的旁位参照用 24px。负载下偶发：update-served-anew ja@420、update-hide webkit zh，待开 issue。
      GUI 全量 3 分片通过后推送 main，CI 绿后已发布 v0.1.2908（Release run 42538105753）。负载下的 4 个 GUI 红与 2 个 Little Snitch 红分别为 #7–#10、#5–#6。
- [x] I. 收尾（11-09）：
      #5/#6：两个 LAN 回调测试改用 `testenv.OwnAddrSkip`（与 #3 同一守卫），netproxy 客户端加 30s 期限；
        注入 192.0.2.1 时两测试 0.00s 跳过并说明原因，正常路径 4/4。`acd3f5eeb`。
      #7：`reloadOnce` 一次重载（servedAnew/backAsNew 共用）；CDP 12 倍降频下旧码日志 `/@0.1.401` 两次，新码 17/17 ×2。`e10acc6e9`。
      #8：测试先等 fonts.ready、navmore.js 与额外一帧再置 tight；排队帧下旧测 2/2 红（'block'），新测 18/18。`22edf98de`。
      #9：`focusSoon`（library 三处 + routing 一处）焦点已移开则不抢；旧码 4/4 红"a late frame took the focus"，新码 16/16；
        5 路 WebKit + 600ms 抖动 30/30（旧 7/30）。`8db0b3c04`。
      #10：配额假接口只在 Usage 页等待并容忍页面已关；等待压到 3s 时旧 10/15 红（9 条为 issue 同款），新 15/15；正常 30/30。`fece23c07`。
      #11（本轮发现的 main 既有红）：update-network chromium en@390 在头栏 fitTop 稳定前读宽度；改为等 class 三帧不变，22/22 ×3。`a1f8494c0`。
      可达 GUI 文件回归：99 个（update/market/library/routing 组编辑器等）98 通过，唯一红即 #11，已修。

## 三项大工程（2026-11-09 用户确认开工）

- [x] J. GUI 套件进 CI（`d2ade2d9c`，issue #12）：新增 `.github/workflows/gui.yml`，先在 ubuntu 上跑一个冒烟子集
      （不依赖字体像素基线的文件），验证 Playwright/Chromium/WebKit 在 CI 可用后，
      再用 `--test-shard` 扩大；全量放 schedule + workflow_dispatch 控成本。
      同步改 `docs/subsystems/gui-shell.md` 与 `internal/gui/tests/README.md`（现在写着 CI 不跑）。
- [x] K. app.js 预算腾余量（`b0d99d478`）：drawEditor 移到 `assets/editor.js`（classic script，首绘 Providers 页时加载），
      app.js 1,311,824 → 1,247,685，核心 1,534,015 → 1,469,600，两处预算随之下调；73 个 provider 相关 GUI 文件 71 通过，
      2 个读编辑器的测试改为等待后 52/52 绿。原计划：
      找出边界清晰、可独立加载的页面代码移出核心（按 plugins/*/page.js 的既有模式），
      移动后把 `appScriptBudget`/`coreScriptBudget` 下调到新体积，并跑全量 GUI 相关文件。
- [~] L. 网关"只解析一次"（调研中）：已经有一次解析 + `walk` 零拷贝（unsafe 包装）+ 输出缓冲裁剪（#3383）+ heldsize/分配上限测试；先 profile 复现 1MB Claude Code 请求的分配（TASKS G 项基线 20.7MB/req），
      审计脱敏链路对请求体的持有，给出分步方案（每步独立可验证、可回滚），再实现。
      密码学/隐私链路：按 LESSONS 把秘密放在工具参数、消息正文、嵌套对象里测。
      11-09 profile（TestMaskedTranslatedRequestAlloc，1MB 请求 20.6MB/37.8k 次；`/tmp/mem.out`）：
      大头是 `unquoteJSON` 4.2MB/req、`bytes.Clone`（fillJSONIn 的 RawMessage 复制，低于 heldShareMost 的小片段）4MB/req、
      `io.ReadAll` 2.8MB、`bytes.growSlice` 1.8MB、`gjson.getBytes` 1.7MB、`marshalSized` 1.6MB、`insertAt`/`snipsOf` 各 1.4MB。
      结论：共享机制（inHeldBody/heldShareMost）已在，剩余是"共享会不会钉住 1MB body"的权衡，
      下一批按单点做并各自保留分配数字，不整体重写。
- [x] M. 本地脱敏模型的准确率（用户 11-09 要求）：CoreML 早已接入（`internal/pii/guard/coreml.go`，MLComputeUnitsAll）。
      新增 `guard/eval_test.go`：46 条中英标注请求。本机（M 系列）：CoreML 39/42，P 0.975，R 0.929，F1 0.951；
      bert.go 38/42，F1 0.927；971 token 窗口 7.2ms vs 441ms。漏检：无服务词的英文 IP、中国大陆身份证号、成对出现的第二个 IP；
      误遮 2 字节（"11 月 3 日"的 11）。数字与明细见提交 `6c3135f81`。
      两模型对比（`TestEvalBothModels`，同一 46 条集，11-09）：AstrLink Guard 39/42，P 0.951，R 0.929，F1 0.940，
      636/702 字节，误遮 6；Privacy Filter 30/42，P 0.938，R 0.714，F1 0.811，553/702，误遮 19，耗时 8.0s vs 0.85s。
      注意：本集以自然语言为主，Privacy Filter 的强项（代码里的哈希/产品名）未覆盖，结论不可外推到代码场景。
      Privacy Filter 在本仓库目前装不上：`pf-v3` 的 pf-serve 资产 404（Release 未构建）。本地已用
      `native/privacy-filter/build.sh darwin-arm64` 构建并手工装好、下载模型（sha256 与 pin 一致），对比才得以进行。
      待办：`docs/reference.md:2767,2800` 的 install.sh 链接是仓库根路径（404），应改为 releases/latest/download。

## GUI 全量在 Linux 上的红（issue #13，2026-11-10 起）

- [x] 本地 Linux oracle：Playwright 官方镜像 + `NODE_PATH`，命令写进 `docs/subsystems/gui-shell.md`。
      1.58.2 复现了 chromium 的 gateway-card 与 webkit 的 quit-ask-setting；1.65 镜像两次拉取失败（网络），
      且 `editor-roomy` 在 1.58 里不复现——本地结论要按"日志说什么"复核。
- [x] gateway-card（chromium 10 条）：中线偏 0.586px，容差 0.25→1px。`3673d3245`
- [x] quit-ask-setting（webkit 5 条）：测量前把开关滚进视野，点击本身才是被测的位移。`3673d3245`
- [x] 分片 summary 列出失败测试的 `文件:行` 与名字；日志无 totals 时写明被截断。`c14a03359`
- [x] 全量分片时限 Run 140→190m、job 150→200m：首轮 webkit 的 shard 3 在 140m 被杀，totals 与失败清单都没写。`0df956cdc`
- [x] click-scroll「late-heard wheel」：`wheel(-40)` 后只读一次 `scrollTop`，滚动晚一两帧落地就误判"没动"；
      改为等 `scrollTop < from`（2s 上限）再断言。容器 1.58.2 连跑 4 次 24/24（修复前 3 次红 2），macOS 24/24。
- [x] 「click moved the page/nothing」一类（currency、plain-names、omarchy、redact-rule-edit，共 11 条）：
      根因是记录位置时滚轮惯性还没停（应用 flings 250ms），固定等 300ms 在重载 runner 上不够。
      新增 `fixtures/wheel-rest.cjs`（8 帧不变且 ≥250ms，2s 上限），`reach`/`inView` 用它替换固定 300ms。
      容器 1.58.2：12/12、4/4、14/14、20/20；macOS 五个文件 132/132，冒烟集 108/108。
- [ ] model-load 390 窗口「busy models go last」点击后视图跳 229px：与本次改动无关（回滚 reader.cjs 同样复现），是独立问题。
- [x] full webkit（42679120343，190 分钟时限）四片全绿，单片 57–88 分钟：140 分钟被杀的问题已解决。
- [ ] 但 full 是报告模式（exit 0），artifact 只在 `failure()` 时上传——这一轮的日志全丢了。
      改为 `failure() || mode == 'full'`：summary 只列最多 40 条，断言原文只在日志里。
- [x] `2c63d12df` 的 GUI 冒烟四片全绿（Linux 冒烟门禁通过，click-scroll 稳定）。
- [x] usage-heatmap 的 tooltip 差一天：指针落在下一行的格子上——记录格子位置时滚轮惯性未停。改用 `wheelRest`。
      修完后该文件剩余的红是另一件事：`#view-usage` 横向溢出 3–4px（scrollWidth 414 vs clientWidth 410），
      越界元素是 ledger 的 `table.led`（宽 475，本应在自己的滚动容器里）。量级很小，先不改布局，待查。
- [x] 共享 `reach` 的"等滚轮停稳"顺带修掉：routing-bodies 20/20、routing-time-went、sessions-pending-fill、
      session-identity、settings-proxy-auth 全绿（镜像 1.58.2，一批 110 项）。
- [x] routing-rule-models：Playwright 的 `hover()` 会先把控件滚进视野、被应用撤销，悬停超时。改成本地 `hover()`
      （真实滚轮 + 停稳后定位），Linux 镜像 30/30，macOS 60/60。
- [x] rest-try-now：本地 `reach` 的 200ms 固定等待换成 `wheelRest`；剩 2 条间歇性"按钮点不到"（webkit en/de@420），单跑不复现。
- [x] title-bar-tools：4 条红全部来自 WebKit 取消导航时报告的错误；`e.message` 是截断的（以 `/myna.test/api/... due to access control checks.` 开头），
      过滤按 `e.stack` 才匹配。加负载跑（4 个 CPU 忙循环）时 111 条错误全部滤掉，5/5 绿。
- [ ] codex-predictions：加了 `wheelRest` 后失败差从 ~300px 变成 ~50px，仍是应用在选完后自己滚动（#14 记录），测试不再放宽。
- [x] 第一次**完整**的 full webkit（42699937825，四片全部跑完，单片 57–88 分钟）：10,152 项、166 红、134 个不同(file,test)。
      清单与分类见 #13；日志已作为 artifact 保留。
- [x] `.rt-day` 裸选择器命中了同 class 的隐藏分组按钮（1.65 上必现）：routing-bodies、call-headers、routing-cline-route、
      routing-time-went、routing-upstream、phone-rows 改为 `.rt-days .rt-day`。
- [x] 本地 Linux 镜像升级到 Playwright 1.65 自己的浏览器（模块 `/opt/pw`，浏览器 `/ms-playwright`：webkit-2388、chromium-1254）。
      routing-bodies + call-headers 40/40。注意：旧选择器在本地 1.65 上不复现那 9 条（CI 上是重载下的重绘竞态），
      所以 `.rt-days .rt-day` 的依据是 CI 日志里 nth(1) 解析到隐藏的分组按钮，而不是本地红绿。
- [x] allow-ips、lan-docker 的"Add/开关移动了页面"：同样是滚轮惯性未停就取基准，加 `wheelRest` 后 1.65 上 30/30。
- [x] usage 视图横向溢出 4px：源头是趋势卡的指标页签（`.led-bar .segs` 383px vs 卡片 358px），
      按编辑器里同款规则改为换行；1.65 上 usage 五个文件 93/100（回退该规则 89/100），usage-heatmap 26/26 且 macOS 26/26。`d5ea0a139`
- [ ] #14 新证据（11-10）：model-load 390 zh 点击后 400→108。应用日志显示点击重绘把 view 拉到 0，而这次滚动落在点击自己开的
      `purposeUntil` 窗口里（249ms、995ms），滚动处理器的 `purposeUntil` 分支先于“点击保持”（`held?.v === v`）执行，
      于是把 0/108 记成了读者位置（`leaves`），随后的回填按 0/108 还原。修法需整体读 `keepHeld`/`setRoom`/`held`，风险高，测试未放宽。
      证据已回帖 #14。
- [ ] 应用侧待修（#14）：codex-predictions 选完模型视图回跳（420 宽确定性 5/5；根因是"点击保持期间的滚轮惯性滚动被保留却未记为读者位置"）；
      model-load 390 宽点击跳 229px。应用滚动记账逻辑暂不改（牵动整套 GUI 滚动，风险高）。
- [ ] view-open-wheel（webkit 约 20 条）：320 个子测试，Linux 上单条 180–234 秒，父测试 15 分钟超时后余下子测试瞬时失败；
      测的是"听不到滚轮"时的记账（waitsBack），属应用侧。同类还有 view-wheel-after-click（3 条）。
- [x] 第二份完整 full webkit（42720704172，`cd12a16ba`）：127 红 / 101 个不同 (file,test)（上一份 166 / 134）。
      69 条消失，36 条是同一断言换语言/宽度，没有回归。清单与分布已更新到 #13。
- [x] provider-levels-save（en、zh）与 model-reasonless（1100 de）：WebKitGTK 把 sheet 的 `.ebody` 网格首帧按 616(含滚动条) 算宽，
      字段比实际多 10px，直到 sheet 里**第一个控件拿到焦点**才重排；德国文案的行正好在换行边界，行高 131→191，
      复选框在按下与松开之间下移 60px，点击落到旁边的输入框（provider-levels-save 报 "Clicking the checkbox did not change its state"）。
      新增 `fixtures/press.cjs`：先 focus，等控件位置连续 3 帧不变，再 click。容器 1.65 webkit 12/12、macOS chromium 12/12、macOS webkit 12/12。`414e1ad3a`
      应用侧未改：给 `.ebody` 加 `scrollbar-gutter: stable` 或 `overflow-y: scroll` 都只能让首帧宽 10px 或 20px（重排仍发生），
      是 WebKitGTK 的网格首帧计算问题，不是本次测试错。桌面版（body.linux，`.ebody` 已是 `overflow-y: scroll`）是否也丢第一下点击未验证。
- [x] user-select（10 条，en/zh/ja/de × 420/1100）：`-webkit-user-select` 在 WebKitGTK 2.52 上算作 `auto`、标准属性才是规则（WebKit 16 相反），
      取"非 auto 的那个"后 8 条转绿；剩下的 420 ja 是拖选起点取了滚轮惯性未停的位置（选到上一行的 "disconnect/reset before headers"），
      改为 `wheelRest` 等停稳。容器 webkit 8/8、macOS chromium 8/8、macOS webkit 8/8。`97878c1d9`
- [ ] view-open-wheel 本地镜像与 CI 结论相反：CI 全量里 chromium 360s、webkit 366s 全绿；本地 webkit 的 `unheard` 子测试单条 57–250s 并有 5 条红
      （CI 上同样 160 条只要 6 分钟）。属镜像环境差异，测试未改；不再用本地红绿判断这个文件。
- [ ] 已知限制：CI 的字体集与本地镜像不同，德语 560 宽的 9px/6px 溢出本地复现不了；已在镜像里跑 `playwright install-deps` 试图对齐。

## 2026-11-10 反馈帖 + 上游同步（第 N 轮）
- [x] 抓取反馈帖 195 条，官方 v0.1.3501–3612 修了大量问题；清单见 ~/.local/share/owl-feedback/bugs.md
- [x] backport 18c932385（Fable 5.2 压缩）→ f767a791c
- [x] backport 1b2ad557f（Alma/OpenHanako 卸载后可关；Trash 不算已安装）→ 636eaab02
- [ ] CI 绿后决定是否发版（owl 自身 tag 已到 v0.1.2910；上游 2911–3612 的 tag 已占用）
- [ ] 下一批候选：543eb39b9（改 effort 不重启本轮）、10bef5fe0（Codex 项目共享）、6bac8f8a5（手机导航）
- [x] 2026-11-10 23:5x：反馈帖二次抓取 387 条；backport b2509b9a8（ptmx EAGAIN）→ 2627b3f49
- [ ] 下一批：0791c91c5（非法 UTF-8）、93a33d817（OpenBot）
- [x] 2026-11-11 00:4x：backport 0791c91c5（非法 UTF-8 → U+FFFD）→ f5f171f03，已推送
- CI：fb4ad6568 的 Test 全绿（含 Alma/OpenHanako 修复）；7bc72e281 的 test(macos) 15 分钟无步骤被取消，属 runner 侧，代码由 f5f171f03 的新 run 覆盖
- [ ] 等 f5f171f03 的 Test 全绿（含 claudetui ptmx）
- [x] 2026-11-11 01:0x：backport 93a33d817（OpenBot 图标/请求归属/Library skills）→ e9b52ce34
- [ ] OpenBot 的 GUI 浏览器测试未跑（本机无 Playwright）；待 CI 或装 Playwright 后补
- [x] 2026-11-11 02:2x：38f43bdf1 的 test(macos) 红：TestTOMLBigFileParsesInLinearTime 比值 24.6>24（负载擦边）；TestSearchQueueKeepsCodexAlive/cancel 挂 19m 致整包 30m 超时 → 测试清理已修（99a5924ba）
- [ ] TOML 比值阈值 24 在慢 runner 上擦边，未改；若再红需换成更稳的度量或放宽
- [x] 2026-11-11 02:4x：56da6020c 的 test(macos) 红：TestSpotlightStaleAsksEveryAppAgain 5s 等状态不够 → 上限改 30s
- [x] 2026-11-11 04:2x：backport 388ca9ad4（Claude Desktop 导入提示）→ 本地提交
- [x] d4ccee0d4 的 Test 全绿（effort 修复）
