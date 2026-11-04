# Myna 品牌化改造任务

目标：把 magpie 变成自己的发行版 Myna，更新与所有官方网络依赖走自己的 GitHub 仓库，独立配置并迁移旧配置。

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
