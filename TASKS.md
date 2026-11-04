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