# Phase 4 代码审查报告 — 2026-06-28

**审查范围**: Phase 4 AI 模块全量代码（commit `e46d66e`），29 个文件，1524 行新增  
**审查方法**: 10 角度并行 agent review（correctness / removed-behavior / cross-file / go-pitfalls / wrapper-proxy / reuse / simplification / efficiency / altitude / conventions）+ sweep 查漏  
**测试状态**: `go test ./internal/ai/...` 全部通过（17/17），`go vet` 无告警

---

## Phase 4 目标对照

| 目标 | 状态 | 证据 |
|---|---|---|
| Provider 抽象层（mock/anthropic/openai/noop） | ⚠️ 部分 | mock + noop 完成；anthropic/openai 为 stub（返回 NoopProvider） |
| Guard 预算执行（日/月限额） | ⚠️ 部分 | 逻辑正确，但月度超标返回错误哨兵（BUG-2），AI 错误映射为 500（BUG-3） |
| Cache（Redis 提示词-响应缓存） | ⚠️ 部分 | 基本功能正确，但 hash 遗漏 MaxTokens/Temperature（BUG-6） |
| 3 个 AI 函数（Attribution/Completion/Prioritization） | ❌ 阻断 | **RoleFor 参数顺序错误导致所有端点返回 403**（BUG-1） |
| API Handler（4 个端点） | ⚠️ 部分 | 路由注册正确，但错误映射缺失（BUG-3），Budget 返回硬编码零值（BUG-5） |
| 前端 AiPanel 组件 | ⚠️ 部分 | UI 完整，但错误提示不区分类型，Mock 补全提示词匹配失败（BUG-14） |
| Feature Flag AI_ENABLED | ❌ 未实现 | 配置已加载但从未检查，路由无条件注册（BUG-4） |

---

## 严重问题

### 🔴 CRITICAL-1: RoleFor 参数顺序错误 —— 所有 AI 端点返回 403

- **文件**: `backend/internal/service/ai_service.go:42, 50, 75`
- **问题**: `projectRoleChecker.RoleFor` 接口签名为 `RoleFor(ctx, projectID, userID)`，但 ai_service 中三处调用均传入 `RoleFor(ctx, callerID, projectID)`（参数顺序颠倒）。项目内其他所有服务均正确使用 `RoleFor(ctx, projectID, callerID)`。
- **影响**: 所有 AI 端点（Attribution/Complete/Prioritize）对任何用户均返回 403 Forbidden。`ProjectService.RoleFor` 将用户的 UUID 当作项目 ID 查找（`FindByID(callerID)`），必然返回 `ErrNotFound`，被映射为 `ErrForbidden`。
- **修复**: 将三处 `RoleFor(ctx, callerID, projectID)` 改为 `RoleFor(ctx, projectID, callerID)`

---

## 高危问题

### 🟠 HIGH-2: 月度预算超标返回 ErrDailyBudgetExceeded（错误哨兵）

- **文件**: `backend/internal/ai/guard.go:42-43`
- **问题**: `if monthly >= g.monthlyLimit { return ErrDailyBudgetExceeded }` — 月度限额触发时返回"每日预算超标"的错误。`ErrMonthlyBudgetExceeded` 不存在。测试 `TestGuard_Allow_MonthlyExceeded` 仅检查 `err == nil` 而未断言具体错误值。
- **影响**: 用户月度预算用完后看到"daily budget exceeded"提示，误以为次日即可恢复。
- **修复**: 新增 `ErrMonthlyBudgetExceeded` 哨兵错误并在 guard.go:43 返回，同时更新测试断言具体错误值。

### 🟠 HIGH-3: AI 哨兵错误落入 HTTP 500 兜底

- **文件**: `backend/internal/middleware/error_mapper.go:31-52`
- **问题**: `WriteError` 的 switch 不处理 `ai.ErrAIDisabled` 和 `ai.ErrDailyBudgetExceeded`，均落入 `default` → HTTP 500 "internal server error"。CLAUDE.md 明确要求 AI 禁用时返回 503。
- **影响**: 用户无法区分"AI 已禁用"(503)、"预算超标"(429) 和"服务器崩溃"(500)。
- **修复**: 在 `WriteError` 中添加对这两个 AI 哨兵错误的判断，映射为合适的 HTTP 状态码（ErrAIDisabled → 503，预算超标 → 429）。

### 🟠 HIGH-4: AI_ENABLED 功能开关已加载但从未检查

- **文件**: `backend/cmd/server/main.go:239-257`
- **问题**: `cfg.AI.Enabled` 从环境变量加载（config.go:155）但从未在 main.go 或任何处理器中读取。AI 路由无条件注册。`Budget()` 硬编码 `"enabled": true`。
- **影响**: 设置 `AI_ENABLED=false` 无效——路由仍注册，按钮仍显示。与 CLAUDE.md 声明"AI_ENABLED=false 完全隐藏按钮并 503 端点"矛盾。
- **修复**: 在 main.go 中用 `if cfg.AI.Enabled` 包裹 AI 路由注册；在 Budget() 中读取真实 enabled 状态。

### 🟠 HIGH-5: Budget() 返回硬编码零值

- **文件**: `backend/internal/service/ai_service.go:99-105`
- **问题**: `Budget()` 返回 `"used": 0` 和硬编码限额 `1.0/20.0`，从不查询 `AiRepo.GetDailyUsage()/GetMonthlyUsage()`。`AIService` 未持有 `UsageStore` 或 `Guard` 引用。
- **影响**: `/ai/budget` 端点永远显示零用量，仪表盘/监控完全失效。
- **修复**: 将 `UsageStore` 注入 `AIService`，在 `Budget()` 中调用 `GetDailyUsage()/GetMonthlyUsage()` 获取真实数据。

### 🟠 HIGH-6: 缓存键 Hash 遗漏 MaxTokens 和 Temperature

- **文件**: `backend/internal/ai/cache.go:45-53`
- **问题**: `hashPrompt` 仅对 `SystemPrompt` + `Messages` 做哈希，忽略 `MaxTokens` 和 `Temperature`。当 Temperature 变为可配置后，不同参数的相同提示词会命中同一缓存。
- **影响**: 低温度(0.1)确定性请求可能被缓存，后续高温度(0.9)探索性请求命中同一缓存，返回非预期输出。
- **修复**: 将 `req.MaxTokens` 和 `req.Temperature` 纳入哈希计算。

### 🟠 HIGH-7: 提示词模板硬编码在 Go 源码中（违反 CLAUDE.md + ADR-0005）

- **文件**: `backend/internal/ai/prompts.go:3-23`
- **问题**: 三个长提示词作为 Go `const` 硬编码。CLAUDE.md 规定"All prompt templates live in docs/ai-workflow/prompts/*.md — never hard-code long prompts in Go source." ADR-0005 规定提示词版本独立于代码版本。相同提示词已存在于 `docs/ai-workflow/prompts/*.md`。
- **影响**: 每次提示词调整需修改 Go 源码并重新编译，违背设计意图。
- **修复**: 使用 `embed.FS` 或 `os.ReadFile` 从 `docs/ai-workflow/prompts/*.md` 加载提示词，删除 prompts.go 中的硬编码常量。

### 🟠 HIGH-8: 缺少 0005_ai_module 的 down 迁移

- **文件**: `backend/migrations/`（仅有 `0005_ai_module.up.sql`）
- **问题**: 前四个迁移（0001-0004）均有 up/down 配套文件。0005 缺少 down 迁移，回滚脚本将中断。
- **影响**: 执行迁移回滚时在 0005 处中断，操作员需在事故压力下手写 down 迁移。
- **修复**: 添加 `backend/migrations/0005_ai_module.down.sql`：`DROP TABLE IF EXISTS ai_usage;`

---

## 中等问题

### 🟡 MEDIUM-9: Complete 路径静默丢弃 ListByProject 错误

- **文件**: `backend/internal/service/ai_service.go:68`
- **问题**: `cases, _ := s.caseStore.ListByProject(ctx, projectID)` — 错误被丢弃。对比 Prioritize（第 78 行）正确检查了同一调用的错误。DB 故障时 cases 为 nil，`json.Marshal(nil)` 产生 `"null"`，LLM 收到畸形输入。
- **影响**: DB 故障被完全隐藏，用户看到"Generation succeeded"但结果异常，无法定位问题。
- **修复**: 参照 Prioritize 的写法添加错误检查：`if err != nil { return nil, fmt.Errorf("list existing cases: %w", err) }`

### 🟡 MEDIUM-10: guard.Allow 热路径上两次串行 DB 查询

- **文件**: `backend/internal/ai/guard.go:30-44`
- **问题**: `GetDailyUsage` 和 `GetMonthlyUsage` 是两个独立的 DB 查询（结构相同的 SUM 聚合，仅 WHERE 日期范围不同），在每次 AI 请求的热路径上串行执行——且在缓存检查*之前*。缓存命中的请求白白消耗 2 次 DB 查询。
- **影响**: 50ms DB 延迟下每次 AI 请求额外消耗 100ms。高并发时累积明显。
- **修复**: 合并为单个 SQL（CASE WHEN 分区聚合）或将缓存检查移到 guard 之前。

### 🟡 MEDIUM-11: Prioritize 过滤使用 O(n×m) 嵌套循环

- **文件**: `backend/internal/service/ai_service.go:82-92`
- **问题**: 当 `caseIDs` 非空时使用嵌套循环过滤：对每个 case 遍历所有 caseIDs。500 个用例 × 100 个 ID = 50,000 次比较。
- **影响**: 大型项目下同步阻塞请求路径，增加不必要的延迟。
- **修复**: 先构建 `map[string]struct{}` 集合（O(m)），再单次遍历 cases（O(n)），总复杂度 O(n+m)。

### 🟡 MEDIUM-12: time.Now() 与 time.Now().UTC() 时区不一致

- **文件**: `backend/internal/repository/ai_repo.go:25 vs :34,49`
- **问题**: `RecordUsage` 使用 `time.Now()`（本地时间）存储 `CreatedAt`，但 `GetDailyUsage` 和 `GetMonthlyUsage` 使用 `time.Now().UTC()` 构建查询范围边界。服务器时区非 UTC 时，午夜附近的记录可能被计入错误的日期桶。
- **影响**: 服务器时区 UTC+8 时，凌晨 0:00-7:59 的使用记录可能被排除在当日预算统计之外，预算保护出现窗口期漏洞。
- **修复**: 统一使用 `time.Now().UTC()` 或改用 `CURDATE()` 在数据库侧计算日期边界。

### 🟡 MEDIUM-13: json.Marshal 错误被静默丢弃（3 处）

- **文件**: `backend/internal/service/ai_service.go:67, 68, 94`
- **问题**: `apiJSON, _ := json.Marshal(apis)` 等三处丢弃序列化错误。序列化失败时变量为空字符串，LLM 收到空提示词。
- **影响**: 未来数据模型变更引入不可序列化字段时，静默产生空提示词 → 幻觉输出，无错误信号。
- **修复**: 检查并返回序列化错误。

### 🟡 MEDIUM-14: Mock Provider 补全提示词匹配失败

- **文件**: `backend/internal/ai/mock_provider.go:23`
- **问题**: mock 的 switch 检查 `"completion"` 或 `"generate test case"` 关键字，但实际 `promptCompletion` 常量（"You are a test case generation engine..."）均不包含这两个字符串。mock 命中 default 分支，返回 `{"message":"Mock response..."}`，`json.Unmarshal` 静默失败。
- **影响**: 开发环境（mock provider）下补全功能永远返回零条生成用例，开发者无法验证功能。
- **修复**: 匹配逻辑应使用 `function` 参数（engine.call 传入的 `"completion"` 字符串）而非 SystemPrompt 内容；或将 `function` 参数传递到 mock provider。

### 🟡 MEDIUM-15: guard.Record 错误被静默丢弃

- **文件**: `backend/internal/ai/engine.go:60-68`
- **问题**: `_ = e.guard.Record(ctx, ...)` — 数据库插入失败时静默丢弃。用量记录缺失导致 Guard 低估真实花费。
- **影响**: 连续 20 次成功 LLM 调用全部记录失败 → Guard 仍认为用量为 $0 → 后续调用不受预算限制，可能产生严重超支。
- **修复**: 至少 `log.Printf("guard.Record: %v", err)`；严重时可考虑在 pipeline 中返回部分错误。

---

## 低危 / 观测项

| # | 文件 | 行 | 描述 |
|---|------|-----|------|
| L1 | `ai_service.go` / `ai_repo.go` | — | service 层和 repository 层零单元测试（覆盖率要求 ≥80%） |
| L2 | `AiPanel.vue` | 68-102 | 所有 API 调用 catch 块使用相同通用错误提示，无法区分禁用/预算/网络错误 |
| L3 | `main.go` | 244 | Temperature 硬编码 0.3，AIConfig 无 Temperature 字段 |
| L4 | `provider.go` | 52-55 | anthropic/openai 分支返回 NoopProvider（stub），无启动日志警告 |
| L5 | `engine.go:62` + `ai_repo.go:24` | — | engine 和 repo 双重调用 `id.New()`，engine 生成的 ID 被 repo 覆盖 |
| L6 | `engine.go` | 62-68 | ProjectID 在所有 AiUsage 记录中永为 NULL，无法按项目归因成本 |
| L7 | `provider_test.go` | 740-753 | 工厂测试未覆盖 anthropic/openai/空字符串 分支 |
| L8 | `config.go` | 73-75,157-159 | ModelAttributor/ModelCompleter/ModelPrioritizer 已解析但从未使用（死配置） |
| L9 | `config.go` | 165 | AI_TIMEOUT_SECONDS 已解析但从未应用（无 context.WithTimeout） |
| L10 | `prompts.go` / `completer.go` | — | 部分文件未通过 gofmt 格式检查 |

---

## 完成度总结

| 维度 | 完成度 | 说明 |
|---|---|---|
| Provider 层 | 40% | mock + noop 完成；anthropic/openai 为 stub（返回 ErrAIDisabled） |
| Guard 预算 | 70% | 逻辑正确但月度返回错误哨兵，Record 错误丢弃 |
| Cache 缓存 | 85% | 基本正确但 hash 不完整，时区不一致 |
| 3 个 AI 函数 | 0% | **RoleFor 参数 bug 导致全部返回 403 Forbidden** |
| API Handler | 60% | 路由正确但错误映射缺失，Budget 硬编码 |
| 前端 AiPanel | 65% | UI 完整但错误处理弱，mock 补全不工作 |
| Feature Flag | 0% | 配置已加载但从未实施，与 CLAUDE.md 约定矛盾 |
| 数据库迁移 | 50% | up 迁移正确，down 迁移缺失 |
| 测试覆盖 | 25% | ai 包 17 测试通过，service/repo 层零测试 |
| 提示词管理 | 0% | 硬编码在 Go 源码中，违背 ADR-0005 设计决策 |

**总体评估**: Phase 4 架构设计合理、分层清晰（Handler → Service → Engine → Provider/Cache/Guard），但由于 **RoleFor 参数错误（CRITICAL-1）导致所有 AI 端点不可用**。修复此 bug 后，需解决 2-8 号 HIGH 问题才能使 AI 模块在生产和开发环境正常工作。预估修复全部 CRITICAL + HIGH 问题需 **3-4 小时**，MEDIUM 问题需额外 **2-3 小时**。

---

## 优先修复顺序

| 优先级 | 编号 | 问题 | 预估 |
|---|---|---|---|
| **P0** | CRIT-1 | RoleFor 参数顺序错误 | 5 min |
| **P0** | HIGH-8 | 缺少 down 迁移 | 2 min |
| **P1** | HIGH-2 | 月度预算返回错误哨兵 | 10 min |
| **P1** | HIGH-3 | AI 错误映射为 500 | 20 min |
| **P1** | HIGH-4 | AI_ENABLED 未检查 | 30 min |
| **P1** | HIGH-6 | 缓存 hash 不完整 | 10 min |
| **P1** | HIGH-7 | 提示词硬编码 | 30 min |
| **P1** | HIGH-5 | Budget 硬编码零值 | 30 min |
| **P2** | MED-9 | Complete 静默丢弃错误 | 5 min |
| **P2** | MED-14 | Mock 补全匹配失败 | 15 min |
| **P2** | MED-12 | 时区不一致 | 5 min |
| **P2** | MED-13 | json.Marshal 错误丢弃 | 5 min |
| **P2** | MED-10 | 两次串行 DB 查询 | 30 min |
| **P2** | MED-11 | O(n×m) 嵌套循环 | 10 min |
| **P2** | MED-15 | guard.Record 错误丢弃 | 10 min |
| **P3** | L1-L10 | 测试覆盖 + 配置清理 + 前端改进 | 4-6 hr |

---

*审查由 10 个并行 agent 执行（5 correctness + 3 cleanup + 1 altitude + 1 conventions）+ 1 sweep 查漏，逐文件验证*  
*AI-Assisted: claude-code*
