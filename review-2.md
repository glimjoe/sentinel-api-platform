# Phase 1 & Phase 2 完成度审查报告

**审查日期**: 2026-06-28
**目标**: 逐项审查 Phase 1（骨架）和 Phase 2（核心域）的实现完成度，与 `~/.claude/plans/expressive-forging-hare.md` 计划要求对照。

---

## 总体统计

| Phase | 区域 | 完成项 | 部分完成 | 缺失子项 |
|-------|------|--------|----------|----------|
| Phase 1 | 后端 | 4 | 1 | 2 (pagination, validator) |
| Phase 1 | 前端 | 2 | 4 | 0 |
| Phase 2 | 后端 | 6 | 3 | 3 (condition.go, validator.go, differ.go) |
| Phase 2 | 前端 | 3 | 3 | 2 (MockConsole, 6 components) |

**总项数: 29 | 完全完成: 15 | 部分完成: 11 | 缺失子项: 7**

---

## Phase 1 — 骨架

### 后端

| # | 要求 | 状态 | 证据 / 差距 |
|---|------|--------|-------------|
| 1 | `internal/pkg/*` 全部子包 | **7/9** | `config/`、`logger/`、`jwt/`、`password/`、`errs/`、`httpx/`、`id/` 已完成；`pagination/` 和 `validator/` 缺失 |
| 2 | `internal/middleware/*` 全部 7 个中间件 | **7/7** | request_id、access log、recovery、CORS、error_mapper、JWT auth、RBAC 全部存在并已注册 |
| 3 | 认证端点 5 个 | **3/5** | register ✅、login ✅、me ✅；refresh ❌、logout ❌ 均未实现（虽然迁移中有 refresh_tokens 表） |
| 4 | 健康检查 /healthz + /readyz | **部分** | 处理函数存在但注册在 `/api/v1/healthz` 和 `/api/v1/readyz`（多了 `/api/v1` 前缀） |
| 5 | 迁移 0001_init (users 表) | **完成** | `migrations/0001_init.up.sql` 含 users + refresh_tokens 表 |
| 6 | main.go 全量连接 + 优雅关闭 | **完成** | 完整的 config→logger→MySQL→Redis→Gin→middleware→handler→graceful shutdown 链路 |

### 前端

| # | 要求 | 状态 | 证据 / 差距 |
|---|------|--------|-------------|
| 1 | Vite 脚手架 + 全部依赖 | **部分** | Vue3/TS/Pinia/router/axios/ElementPlus/vitest 已安装；`@vueuse/core`、`echarts`、`monaco-editor` 未安装；ESLint/Prettier/Playwright 无配置文件；无任何测试文件 |
| 2 | Pinia auth store | **部分** | accessToken/refreshToken/user/isAuthenticated/login/logout/register 均有；**`refresh()` 操作缺失** |
| 3 | Axios 客户端自动刷新 | **部分** | baseURL + token 拦截器正常；401 仅清除 token 并跳转 `/login`，**未调用 refresh 端点重试原请求** |
| 4 | 路由守卫 | **完成** | beforeEach 检查 isAuthenticated，公开路由 meta.public=true，未认证跳转 /login |
| 5 | Login/Register/Dashboard 页面 | **部分** | 三个 .vue 文件均存在；**Dashboard 路由未注册**（`/dashboard` 不在 router/index.ts），登录/注册成功跳转 `/dashboard` 会 404 |
| 6 | `npm run dev` 端口 5180 | **完成** | vite.config.ts 设置 port:5180 + host:0.0.0.0 + /api 代理到 :8081 |

---

## Phase 2 — 核心域

### 后端

| # | 要求 | 状态 | 证据 / 差距 |
|---|------|--------|-------------|
| 1 | Projects CRUD + 成员管理 | **完成** | Create/List/Get/Update/Delete/AddMember/ListMembers — 7/7 端点，三层实现完整 |
| 2 | APIs CRUD + OpenAPI 导入 | **完成** | Create/List/Get/Update/Delete/AddTag/RemoveTag/ImportOpenAPI — 8/8 端点 |
| 3 | Mock Rules CRUD + hits 日志 | **完成** | Create/Get/Update/Delete/ListRules/RecordHit/ListHits — 7/7 端点 |
| 4 | RBAC 中间件 | **完成** | `RequireProjectRole` 闭包中间件 + `ProjectService.RoleFor`（owner 隐式 admin + project_members 查表） |
| 5a | mock/engine.go — Serve 入口 | **完成** | 完整 match→delay→extract→substitute→respond 流程，约 146 行 |
| 5b | mock/matcher.go — 评分匹配 | **完成** | literal(+50)/regex(+30)/type(+10) 评分；priority→rule_id 决胜；X-Mock-Match-Reason 响应头 |
| 5c | mock/extractor.go — 变量提取 | **完成** | gjson 路径提取 + regex 捕获组；写入 VarBag |
| 5d | mock/varbag.go — Redis 变量袋 | **完成** | key `mock:vars:{slug}:{sessionId}`；30min 滑动 TTL；`{{var}}` 替换；X-Mock-Session 头 |
| 5e | mock/delay.go | **部分** | engine.go 内联了 `time.After(delayMs)`；无独立 delay.go 文件或脚本化延迟 |
| 5f | mock/condition.go | **缺失** | 条件响应（if-else 链）未实现 |
| 5g | mock/recorder.go | **部分** | HitRecorder 接口定义在 engine.go；MockHitRepo 存在；**main.go:238 传入 nil**，实际未连接 |
| 6a | contract/loader.go | **部分** | 仅接受 `[]byte` 入参（HTTP 请求体）；不从文件或 URL 加载 |
| 6b | contract/parser.go | **完成** | `ExtractAPIs` 用 kin-openapi 解析为 `[]*model.API`；含 schema/tags/spec 提取 |
| 6c | contract/validator.go | **缺失** | JSON Schema 运行时校验未实现 |
| 6d | contract/differ.go | **缺失** | 版本 diff / 破坏性变更检测未实现 |
| 7 | Mock 端点 `/mock/:slug/*path` | **完成** | `r.Any("/mock/:projectSlug/*path", mockEngine.Serve)` — 公开，无需认证 |
| 8 | 审计服务 | **部分** | 模型 + 仓库 + 中间件 + 迁移均完成；`service/audit_service.go` 文件不存在（审计查询层缺失） |
| 9 | Phase 2 迁移 | **完成** | 0002（projects/members/apis/rules/hits 5 表）+ 0003（audit_logs）完整 DDL |

### 前端

| # | 要求 | 状态 | 证据 / 差距 |
|---|------|--------|-------------|
| 1 | Project 列表 + 编辑器 | **完成** | `/projects` 列表视图 + `/projects/:pid` 详情视图（含标签页） |
| 2 | API 列表 + 编辑器 | **部分** | ProjectDetailView 的 APIs Tab 有列表 + 创建对话框 + 导入；**无专用编辑/详情页，无搜索** |
| 3 | MockRule 列表 + 编辑器 | **部分** | ProjectDetailView 的 Mock Rules Tab 有列表 + 删除；**无创建/编辑对话框** |
| 4 | MockConsole 页面 | **缺失** | 完全未实现——无路由、无视图、无 API 调用 |
| 5 | Pinia stores (project/api/mockRule) | **部分** | `project.ts` store 存在；**`api` store 和 `mockRule` store 缺失** |
| 6 | 路由 `/projects` + `/projects/:pid` | **完成** | 带懒加载和 auth guard 的完整路由 |
| 7 | 可复用组件 6 个 | **缺失** | `src/components/` 目录为空——AppShell/DataTable/MethodBadge/JsonEditor/JsonViewer/ConfirmDialog 均不存在 |
| 8 | API 客户端函数 | **完成** | project.ts / apis.ts / mock_rules.ts 均导出完整 CRUD 函数 |

---

## 按严重程度排序的差距

### 🔴 P0 — 阻断用户核心流程

| # | 问题 | 影响 |
|---|------|------|
| 1 | **前端 Dashboard 路由缺失** | 登录/注册成功后跳转 `/dashboard` 返回 404，用户无法完成注册→登录→首页闭环 |
| 2 | **前端 MockRule 无法创建** | 仅能列表和删除，无法添加或编辑规则——Mock 功能不可用 |
| 3 | **前端 MockConsole 完全缺失** | Phase 2 核心差异化功能不可用 |

### 🟠 P1 — 功能不完整

| # | 问题 | 影响 |
|---|------|------|
| 4 | 后端 `POST /auth/refresh` 缺失 | 令牌过期后无法续期，用户必须重新登录 |
| 5 | 后端 `POST /auth/logout` 缺失 | 无服务端登出端点 |
| 6 | 后端 mock hit 记录器未连接 (main.go 传 nil) | mock_hits 表永远为空，mock 请求无轨迹可查 |
| 7 | 前端 auth store 缺 `refresh()` | 即使后端实现了 refresh 端点，前端也无法调用 |
| 8 | 前端 401 拦截器无自动刷新 | 令牌过期直接踢到登录页，用户体验差 |
| 9 | 前端缺 `api` + `mockRule` store | API/规则状态散落在组件中，无集中状态管理 |
| 10 | 前端 6 个可复用组件全缺失 | 视图直接使用 el-table/el-tag/el-dialog，无封装复用 |

### 🟡 P2 — 基础设施 / 远期

| # | 问题 | 影响 |
|---|------|------|
| 11 | `pagination/` + `validator/` 包缺失 | 未来需要分页或入参校验时需新写 |
| 12 | ESLint/Prettier/Playwright 无配置 | 代码风格不统一，E2E 测试无法运行 |
| 13 | `/healthz` 多了 `/api/v1` 前缀 | 若外部监控探针期望 `/healthz` 则探测失败 |
| 14 | `contract/validator.go` + `differ.go` 缺失 | API schema 校验和变更检测不可用 |

---

## 结论

Phase 1 和 Phase 2 的核心 CRUD 后端端点全部就位（项目/API/规则三层齐全），RBAC 和 mock 引擎核心也已完成。但前端存在 **3 个 P0 阻断级缺口**（Dashboard 路由、MockRule 创建 UI、MockConsole），使 Phase 2 的用户闭环无法走通。

**建议优先修复 P0 三项**（预计 2–3 小时），再按 P1 → P2 顺序逐项补齐。
