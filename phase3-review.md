# Phase 3 进度审查报告

**审查日期**: 2026-06-28
**审查范围**: Phase 3 — 测试执行（test_cases + test_runs + runner 引擎 + SSE + 前端）
**基准计划**: `~/.claude/plans/expressive-forging-hare.md` § Phase 3 (line 462–469)

---

## 总体评估：约 65–70% 完成

Phase 3 的核心数据层和基础流程已就位，但存在多个功能缺口和 bug，其中部分 bug 会导致运行时行为异常。

---

## 一、已完成项 ✅

| 组件 | 状态 | 证据 |
|---|---|---|
| 数据模型 (test_case/run/result) | ✅ | `internal/model/test_case.go:5`, `test_run.go:5`, `test_result.go:5` |
| 数据库迁移 | ✅ | `migrations/0004_test_runner.up.sql` 含完整 DDL |
| Repository 层 (3 个) | ✅ | `test_case_repo.go`, `test_run_repo.go`, `test_result_repo.go` |
| TestCase 服务 CRUD + RBAC | ✅ | `service/test_case_service.go` |
| TestRun 服务 (create/start/cancel) | ✅ | `service/test_run_service.go` |
| Runner 引擎 (顺序模式) | ✅ | `runner/runner.go` + `executor.go` |
| 断言引擎 (status/exact/contains/regex) | ✅ | `runner/assertion.go` — 3/6 种 body match 已实现 |
| SSE 实时流 (Redis pub/sub) | ✅ | `runner/sse.go` + ADR-0006 |
| 15s 心跳 | ✅ | `runner/sse.go:64` |
| 路由注册 (全部 Phase 3 端点) | ✅ | `cmd/server/main.go:184-226` |
| 前端 TestCase/Run 类型定义 | ✅ | `types/test_case.ts`, `test_run.ts` |
| 前端 API 客户端 | ✅ | `api/test_cases.ts`, `test_runs.ts` 含 EventSource |
| 前端列表页 (Tab 形式) | ✅ | `ProjectDetailView.vue` 含 Cases/Runs Tab |
| 前端实时进度弹窗 | ✅ | SSE `streamRun` + progress dialog |
| ADR-0006/0007/0008 | ✅ | `docs/adr/` 目录齐全 |

---

## 二、Bug（会引发运行时错误或静默数据错误）

### 🔴 BUG-1: `Start()` 同步阻塞会导致 HTTP 超时

- **文件**: `backend/internal/service/test_run_service.go:74-83`
- **严重程度**: P0 — 阻断 Run 端到端流程
- **问题**: `Start()` 方法在同一个 goroutine 中同步调用 `runner.Run()`，该调用会阻塞直到所有测试用例执行完毕。对于任何非平凡测试集，HTTP 请求会在 `runner.Run` 返回前超时。
- **影响**: 启动测试运行后前端永远收不到响应；实际执行可能已完成但客户端视为失败。
- **修复方向**: 将 `runner.Run()` 放入 goroutine 中执行，`Start()` 立即返回 `{"status": "started"}`。

### 🔴 BUG-2: `schema` 和 `jsonpath` 断言静默通过

- **文件**: `backend/internal/runner/assertion.go:46-72`
- **严重程度**: P0 — 静默数据错误，测试结果不可信
- **问题**: 模型声明了 6 种 body match 类型 (`exact`, `contains`, `jsonpath`, `schema`, `regex`, `none`)，但 `assertBody()` switch 中：
  - `jsonpath` 落到 case 体返回 nil（注释说 "deferred"）
  - `schema` 不在 switch 分支中，落到 default 返回 nil
- **影响**: 用户设置了 `expected_body_match: "schema"` 或 `"jsonpath"` 的用例**永远报告 pass**，无论实际响应是否符合预期。这是静默数据错误——用户会误以为测试通过。
- **修复方向**: 要么实现 schema/jsonpath 断言，要么对未实现的 match 类型返回 `AssertionFailure{Type: "body.unsupported", Message: "match type not implemented"}`。

### 🔴 BUG-3: Runner 循环不检查取消信号

- **文件**: `backend/internal/runner/runner.go:51-72`
- **严重程度**: P1 — 取消操作不会立即生效
- **问题**: 顺序执行的 for 循环不检查 `ctx.Done()`。虽然 `Execute()` 会将 context 传给 `http.NewRequestWithContext`（可取消飞行中的请求），但循环在用例之间不会停止——cancel 后仍会继续执行下一个用例。
- **影响**: 取消操作可能在等待当前用例超时（最多 30s）后才生效，且可能多执行一个用例。用户感知的取消延迟可达 30 秒。
- **修复方向**: 在 for 循环顶部添加 `select { case <-ctx.Done(): return ctx.Err(); default: }`。

### 🟡 BUG-4: Handler 注释错位

- **文件**: `backend/internal/api/test_run.go:78-79`
- **严重程度**: P3 — 不影响运行
- **问题**: 第 78 行注释写的是 `// Stream handles GET ...` 但下一行实际定义的是 `func (h *TestRunHandler) Cancel(...)`。正确的 `Stream` 方法在第 88 行。
- **影响**: 代码可读性问题，可能误导后续维护者。

---

## 三、功能缺口 ❌

| 缺口 | 计划要求 | 当前状态 |
|---|---|---|
| **并发执行模式** | "顺序/并发两种模式" | `runner.go:50` 注释明确写 "parallel deferred to later Phase 3 iteration"，未实现 |
| **JSON Schema 断言** | "JSON Schema 断言能捕获故意错配" | `schema` body match 类型静默通过 (BUG-2) |
| **JSONPath 断言** | 6 种断言之一是 jsonpath | 同样静默通过 (BUG-2) |
| **报告导出 JSON** | "报告可导出 JSON" | 无任何导出端点 |
| **TestCase 编辑器** | "TestCase 列表/编辑器" | 仅有简单创建弹窗，无独立编辑/详情视图 |
| **TestRun 详情视图** | "TestRun 列表/详情" | 仅有列表+进度弹窗，无详情页含 test_results 列表 |
| **DiffViewer** | "DiffViewer" | 未实现 |
| **结果抽屉** | "结果抽屉" | 仅有简易进度弹窗，无结果详情 |
| **前端 getCase/updateCase** | CRUD 完整性 | `api/test_cases.ts` 缺少 `getCase` 和 `updateCase` 函数 |

---

## 四、测试覆盖严重不足

| 包 | 覆盖率 | 缺失测试 |
|---|---|---|
| `runner` | **23.4%** | 无 `runner_test.go`、`executor_test.go`、`sse_test.go` |
| `service` | **43.4%** | 无 `test_case_service_test.go`、`test_run_service_test.go` |
| `api` | **56.4%** | 无 test_case handler 测试、test_run handler 测试 |
| `repository` | **47.6%** | 无 `test_case_repo_test.go`、`test_run_repo_test.go`、`test_result_repo_test.go` |
| 前端 | **0%** | 无任何 vitest 测试文件 |

计划要求 Phase 5a 达到 ≥80% 覆盖率。当前 Phase 3 核心包的覆盖率仅 23–56%，后续补测试的工作量较大。

---

## 五、Phase 3 DoD 对照

| DoD 条件 | 状态 | 说明 |
|---|---|---|
| 5 个手动用例 → 运行 → 结果实时流 | ⚠️ | 后端已布线但 BUG-1 阻断端到端流程；需 goroutine 化 Start() 后重新验证 |
| JSON Schema 断言能捕获故意错配 | ❌ | schema 断言静默通过 (BUG-2)，完全无法捕获错配 |
| 中途取消生效 | ⚠️ | Cancel 端点存在但 BUG-3 使取消非即时，可能多执行一个用例 |
| 报告可导出 JSON | ❌ | 完全未实现，无任何导出端点或逻辑 |

**DoD 达成率: 0/4**（4 项均未完全满足）

---

## 六、优先修复顺序

| 优先级 | 问题 | 理由 |
|---|---|---|
| **P0** | BUG-1 — Start() 同步阻塞 | 阻断 Run 端到端流程，不修复则 Phase 3 不可用 |
| **P0** | BUG-2 — schema/jsonpath 静默通过 | 静默数据错误，测试结果不可信 |
| **P1** | BUG-3 — 取消循环不检查 ctx.Done() | 取消功能不可靠 |
| **P1** | 并发模式 (parallel) | 计划明确要求的核心功能 |
| **P1** | 报告导出 JSON | DoD 条件之一 |
| **P2** | BUG-4 — 注释错位 | 代码卫生 |
| **P2** | 前端 TestCase 编辑/详情视图 | 功能完整性 |
| **P2** | 前端 TestRun 详情视图 + DiffViewer + 结果抽屉 | 功能完整性 |
| **P2** | 前端 getCase/updateCase API | CRUD 完整性 |
| **P2** | 测试覆盖补齐 | 不应全部拖延到 Phase 5a |

---

## 七、结论

Phase 3 的数据层、核心引擎和 SSE 管道是扎实的，但存在 **3 个 P0/P1 级 bug** 和 **约 9 个功能缺口**，导致 4 项 DoD 条件无一完全满足。建议优先修复 3 个 bug（预计 1–2 小时），再补齐并发、导出和前端视图（预计 3–5 小时），最后补测试。

**当前进度可信度: 65–70%**（基础架构 90% + 功能完整性 50% + 测试覆盖 30%）。
