# Phase 3 代码审查报告 — 2026-06-28

**审查范围**: 全量 Phase 3 实现 + 未提交工作区变更  
**审查方法**: 5 角度并行 agent review（correctness / silent-failure / completeness / frontend / go-pitfalls）+ 逐文件验证  
**测试状态**: `go test ./...` 全部通过，`go vet` 无告警，`vue-tsc --noEmit` 通过

---

## Phase 3 DoD 对照

| DoD 条件 | 状态 | 证据 |
|---|---|---|
| 5 个手动用例 → 运行 → 结果实时流 | ⚠️ 部分通过 | 后端管道完整，但 **cancelled 状态被覆写** (BUG-1)，SSE 连接**无法携带 auth token** (BUG-9) |
| JSON Schema 断言能捕获故意错配 | ✅ 通过 | `assertion.go:191-206` — 使用 `gojsonschema` 完整实现，6 种断言全部可用 |
| 中途取消生效 | ❌ 未通过 | cancel context 传播正确，但 **run.Status 被无条件覆写** (BUG-1)，且 parallel 模式 **不写 DB** (BUG-2) |
| 报告可导出 JSON | ✅ 通过 | `GET /projects/:pid/runs/:runId/results` → `ExportResults()` → `ListByRun()` |

**DoD 达成率: 2/4 完全满足，2/4 部分满足（因 BUG-1/BUG-2 阻断）**

---

## 严重问题

### 🔴 CRITICAL-1: 取消的运行状态被覆写为 "success"/"failed"

- **文件**: `backend/internal/runner/runner.go:57-63`
- **问题**: `Run()` 在 `runSequential`/`runParallel` 返回后无条件执行：
  ```go
  if run.Failed > 0 || run.Errored > 0 {
      run.Status = "failed"
  } else {
      run.Status = "success"
  }
  ```
  当 run 被取消时，`runSequential:89` 或 `runParallel:111` 已将 `run.Status` 设为 `"cancelled"`，但此处被无条件覆写。没有检查 `run.Status == "cancelled"` 的守卫。
- **影响**: 取消的运行在数据库中永久记录为 "success" 或 "failed"，无法区分完成和中断。数据损坏。
- **修复**: 在第 59 行前添加：
  ```go
  if run.Status == "cancelled" {
      finished := time.Now()
      run.FinishedAt = &finished
      updater.Update(ctx, run.ID, map[string]any{"status": "cancelled", "finished_at": run.FinishedAt})
      return nil
  }
  ```

### 🔴 CRITICAL-2: Parallel 模式取消不写 DB

- **文件**: `backend/internal/runner/runner.go:108-128`
- **问题**: `runParallel` 函数签名**不接受** `updater RunUpdater` 参数（对比 `runSequential` 接受了 `updater`）。取消时 (line 111) 仅设置 `run.Status = "cancelled"` 并发布 SSE 事件，**从不调用** `updater.Update()`。
- **影响**: 取消的 parallel run 在 DB 中永远保持 "running" 状态。前端永远显示 progress spinner。
- **修复**: 在 `runParallel` 签名中添加 `updater RunUpdater`（与 `runSequential` 对齐），并在取消路径调用 `updater.Update()`。

### 🔴 CRITICAL-3: Refresh token Consume 存在竞态条件

- **文件**: `backend/internal/repository/refresh_token_repo.go:61-76`
- **问题**: `Consume` 先 `SELECT ... WHERE revoked_at IS NULL` 再 `Save` 设置 `revoked_at`。两个并发请求可同时看到 token 未撤销，双双返回有效 `userID`。无 `FOR UPDATE` 锁，无原子 `UPDATE ... WHERE revoked_at IS NULL`。
- **影响**: 被盗 refresh token 可被攻击者和合法用户同时刷新，各自获得新 token pair。token 轮转安全机制被完全绕过。
- **修复**: 替换为原子 UPDATE + 后查询：
  ```go
  result := r.db.WithContext(ctx).Model(&model.RefreshToken{}).
      Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", hashStr, time.Now()).
      Update("revoked_at", time.Now())
  if result.RowsAffected == 0 {
      return "", fmt.Errorf("refresh_token_repo: invalid or expired token")
  }
  ```

---

## 高危问题

### 🟠 HIGH-4: Refresh 在确认新 token 前撤销旧 token

- **文件**: `backend/internal/service/auth_service.go:151-171`
- **问题**: `Refresh()` 先调用 `Consume()`（验证 + 撤销合一），再执行 `FindByID → jwt.Mint → GenerateToken`。若 `GenerateToken` 失败（DB 瞬时故障），旧 token 已撤销但新 token 未创建 → 用户会话永久丢失。
- **修复**: 分离验证与撤销，先创建新 token 再撤销旧 token。或使用事务。

### 🟠 HIGH-5: 所有 updater.Update 调用静默丢弃 error

- **文件**: `backend/internal/runner/runner.go:35,74,90`
- **问题**: 三处 `updater.Update()` 调用均不检查 error。包括初始 "running" 更新、取消状态更新（仅 sequential）、最终完成更新。
- **影响**: DB 写入失败时 run 永久停留在 "queued"，用户看到卡住的 run 无法恢复。
- **修复**: 至少 `log.Printf("updater.Update: %v", err)`。

### 🟠 HIGH-6: io.ReadAll error 静默忽略

- **文件**: `backend/internal/runner/executor.go:62`
- **问题**: `body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))` — 连接中断时 body 为空，断言给出 "body does not match" 而非 "failed to read body"。
- **修复**: 检查 error，失败时设置 `result.Status = "error"` + `result.ErrorMsg`。

### 🟠 HIGH-7: Headers JSON 解析失败时静默丢弃所有 headers

- **文件**: `backend/internal/runner/executor.go:43-47`
- **问题**: `if err := json.Unmarshal(tc.HeadersJSON, &headers); err == nil { ... }` — JSON 拼写错误时请求不带任何自定义 headers 发送。
- **修复**: 解析失败时设置 `result.Status = "error"` + `result.ErrorMsg = "invalid headers_json"`。

### 🟠 HIGH-8: ErrForbidden/ErrNotFound 被映射为 HTTP 500

- **文件**: `backend/internal/api/test_case.go:29,38,63,73` 和 `backend/internal/api/test_run.go:44,54,63,81,91`
- **问题**: handler 将 `errs.ErrForbidden` (403) / `errs.ErrNotFound` (404) / `errs.ErrBadRequest` (400) 统一返回 `http.StatusInternalServerError, 50000`。仅 `Get` 方法使用了 404。
- **影响**: 用户无权限时看到 "500 Internal Server Error" 而非 "403 Forbidden"。
- **修复**: 统一使用 `middleware.WriteError(c, err)`（`api/auth.go` 已采用此模式）。

### 🟠 HIGH-9: SSE EventSource 无法携带 auth token

- **文件**: `frontend/src/api/test_runs.ts:20-30`
- **问题**: `streamRun` 使用 `new EventSource(url)`，无法设置 `Authorization` header。endpoint 在 `protected` 路由组下需要 Bearer token。
- **影响**: SSE 连接收到 401 → `onerror` 关闭连接 → 实时进度流静默失败，用户看到空白弹窗。
- **修复**: 将 SSE endpoint 移出 auth middleware，使用 URL 短 token 鉴权；或改用 `fetch` + ReadableStream。

---

## 中等问题

### 🟡 MEDIUM-10: Logout handler 泄露原始 error 信息

- **文件**: `backend/internal/api/auth.go:137-139`
- **修复**: 使用 `middleware.WriteError(c, err)` 替代 `httpx.Fail(c, ..., err.Error())`。

### 🟡 MEDIUM-11: pub.Publish error 被忽略（4 处）

- **文件**: `backend/internal/runner/runner.go:66,92,113,142`
- **影响**: Redis 宕机时 SSE 事件静默丢失，取消路径 "complete" 事件丢失 → SSE 客户端永久挂起。

### 🟡 MEDIUM-12: json.Marshal 错误在 assertion failure 序列化时被忽略

- **文件**: `backend/internal/runner/executor.go:84`

### 🟡 MEDIUM-13: JSONPath 数组索引解析失败静默用 index 0

- **文件**: `backend/internal/runner/assertion.go:165`
- **问题**: `fmt.Sscanf(..., "%d", &n)` 对 `$.items[abc]` 失败但不检查返回值，`n` 保持为 0。

### 🟡 MEDIUM-14: DiffViewer 组件创建但从未使用

- **文件**: `frontend/src/components/DiffViewer.vue` — Phase 3 计划要求 "DiffViewer" 但无任何 view 引用此组件。结果详情弹窗仅显示汇总表，无 expected-vs-actual 对比界面。

### 🟡 MEDIUM-15: Phase 3 核心包测试覆盖率严重不足

- **覆盖率**: runner: 29.9% | service: 41.6% | api: 53.6% | repository: 41.3% | 前端: 0%
- **计划要求**: ≥80%（Phase 5a 目标）

---

## 低危 / 观测项

| # | 文件 | 行 | 描述 |
|---|---|---|---|
| L1 | `frontend/src/api/client.ts` | 47-53 | `localStorage.setItem('access_token', undefined)` 存储 `"undefined"` 字符串 |
| L2 | `frontend/src/views/ProjectDetailView.vue` | 253,273 | 删除操作 `catch {}` 空体，UI 乐观删除但 DB 可能未删 |
| L3 | `frontend/src/views/ProjectDetailView.vue` | 254 | FileReader `onerror`/`onabort` 未处理，loading spinner 可能永久旋转 |
| L4 | `frontend/src/views/MockConsoleView.vue` | 111 | `project.value!.slug` 非空断言，API 失败时运行时 `Cannot read properties of null` |
| L5 | `backend/internal/runner/executor.go` | 49 | `Content-Type: application/json` 无条件覆盖测试用例的自定义 Content-Type |
| L6 | `backend/internal/contract/validator.go` | 41-45 | `IsValidSchema` 返回 bare bool，调用方无法获知失败原因是 JSON 解析失败还是 schema 校验失败 |
| L7 | `backend/internal/api/auth_test.go` | 62-63 | `fakeRefreshTokenStore.Consume` 总是返回 `("", nil)` — 未来加 Refresh handler 测试会踩坑 |
| L8 | `backend/internal/api/test_run.go` | 87 | 注释 "Stream handles GET..." 错放在 `Export` 函数上（上次审查 BUG-4，仍未修复） |
| L9 | `frontend/src/stores/api.ts` + `mockRule.ts` | — | 两个 Pinia store 定义了 CRUD 但零引用 — dead code |
| L10 | `frontend/.eslintrc.cjs` | — | ESLint 9.x 不兼容旧格式 `.eslintrc.cjs`，缺少 `@typescript-eslint/eslint-plugin` 依赖，`npm run lint` 无法运行 |

---

## Phase 3 完成度总结

| 维度 | 完成度 | 说明 |
|---|---|---|
| 数据模型 + 迁移 | 100% | test_case/run/result 三个 model + migration 完整 |
| Runner 引擎 | 95% | 顺序/并发模式均已实现，但取消状态写 DB 有 CRITICAL bug |
| 6 种断言 | 100% | status/exact/contains/regex/jsonpath/schema 全部实现，包括 gojsonschema 校验 |
| SSE 实时流 | 90% | Redis pub/sub + heartbeat 正确，但 EventSource 无法携带 auth token |
| 取消功能 | 70% | context 传播正确，但状态覆写 bug 导致 DB 状态错误 (CRITICAL) |
| JSON 导出 | 100% | Export endpoint + service 完成 |
| 前端 TestCase CRUD | 85% | 基础 CRUD 完整，但编辑弹窗缺 body_match/pattern/headers 等高级字段 |
| 前端 TestRun 视图 | 80% | 列表+进度弹窗可用，但 DiffViewer 未集成，结果弹窗缺 assertion detail |
| **测试覆盖** | **30%** | runner 29.9%、service 41.6%、api 53.6%、repository 41.3%、前端 0% — 远低于 ≥80% |

**总体评估**: Phase 3 核心功能架构完成度约 **85%**。关键风险集中在 3 个 CRITICAL bug（取消状态覆写 + parallel 不写 DB + refresh token 竞态），修复这些 bug 后 Phase 3 可视为功能可用。测试覆盖率是后续 Phase 5a 的主要工作量。

---

## 优先修复顺序

| 优先级 | 编号 | 问题 | 预估 |
|---|---|---|---|
| **P0** | CRIT-1 | Cancelled status 被覆写 | 30 min |
| **P0** | CRIT-2 | Parallel 取消不写 DB | 20 min |
| **P0** | CRIT-3 | Refresh token Consume 竞态 | 30 min |
| **P1** | HIGH-4 | Refresh 撤销顺序错误 | 30 min |
| **P1** | HIGH-5 | updater.Update error 静默丢弃 | 15 min |
| **P1** | HIGH-9 | SSE 无 auth token | 1 hr |
| **P1** | HIGH-8 | HTTP 500 错误映射 | 30 min |
| **P2** | HIGH-6/7, MED-10~15 | 其他 HIGH/MEDIUM 问题 | 2-3 hr |
| **P3** | 测试覆盖 | runner/service/api/repo + 前端测试 | 1-2 天 |
| **P3** | DiffViewer 集成 | 结果详情弹窗连接 DiffViewer | 2 hr |

---

*审查由 4 个并行 agent（correctness / silent-failure / completeness / frontend）执行，逐文件验证*  
*AI-Assisted: claude-code*
