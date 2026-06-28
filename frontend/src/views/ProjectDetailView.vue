<template>
  <div class="project-detail" v-loading="loading">
    <div class="header">
      <h2>{{ project?.name }}</h2>
      <el-tag>{{ project?.slug }}</el-tag>
      <el-button size="small" @click="$router.push(`/projects/${pid}/console`)">Mock Console</el-button>
    </div>

    <el-tabs v-model="activeTab">
      <!-- APIs -->
      <el-tab-pane label="APIs" name="apis">
        <div class="tab-header">
          <span>{{ apis.length }} API(s)</span>
          <div>
            <el-button type="primary" size="small" @click="showCreateAPI = true">Add API</el-button>
            <el-upload :auto-upload="false" :show-file-list="false" accept=".json,.yaml,.yml" @change="handleImport" style="display:inline;margin-left:8px">
              <el-button size="small" :loading="importing">Import OpenAPI</el-button>
            </el-upload>
          </div>
        </div>
        <el-table :data="apis" stripe size="small">
          <el-table-column label="Method" width="80">
            <template #default="{ row }"><el-tag :type="methodColor(row.method)" size="small">{{ row.method }}</el-tag></template>
          </el-table-column>
          <el-table-column prop="path" label="Path" min-width="180" />
          <el-table-column prop="name" label="Name" min-width="130" />
          <el-table-column label="Actions" width="80">
            <template #default="{ row }"><el-button size="small" type="danger" @click="handleDeleteAPI(row)">Del</el-button></template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- Mock Rules -->
      <el-tab-pane label="Mock Rules" name="rules">
        <div class="tab-header">
          <el-select v-model="selectedApiId" placeholder="Choose API" @change="fetchRules" style="width:320px">
            <el-option v-for="a in apis" :key="a.id" :label="`${a.method} ${a.path}`" :value="a.id" />
          </el-select>
          <el-button type="primary" size="small" :disabled="!selectedApiId" @click="openCreateRule">New Rule</el-button>
        </div>
        <el-table v-if="rules.length" :data="rules" stripe size="small">
          <el-table-column prop="name" label="Name" min-width="140" />
          <el-table-column prop="response_status" label="Status" width="70" />
          <el-table-column prop="priority" label="Prio" width="60" />
          <el-table-column label="Actions" width="140">
            <template #default="{ row }">
              <el-button size="small" @click="openEditRule(row)">Edit</el-button>
              <el-button size="small" type="danger" @click="handleDeleteRule(row)">Del</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-else-if="selectedApiId" description="No rules" />
      </el-tab-pane>

      <!-- Test Cases -->
      <el-tab-pane label="Test Cases" name="cases">
        <div class="tab-header">
          <span>{{ cases.length }} case(s)</span>
          <el-button type="primary" size="small" @click="showCreateCase = true">New Case</el-button>
        </div>
        <el-table :data="cases" stripe size="small">
          <el-table-column label="Method" width="80">
            <template #default="{ row }"><el-tag :type="methodColor(row.method)" size="small">{{ row.method }}</el-tag></template>
          </el-table-column>
          <el-table-column prop="path" label="Path" min-width="160" />
          <el-table-column prop="name" label="Name" min-width="120" />
          <el-table-column prop="expected_status" label="Expect" width="70" />
          <el-table-column label="Actions" width="140">
            <template #default="{ row }">
              <el-button size="small" @click="openEditCase(row)">Edit</el-button>
              <el-button size="small" type="danger" @click="handleDeleteCase(row)">Del</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- Test Runs -->
      <el-tab-pane label="Test Runs" name="runs">
        <div class="tab-header">
          <span>{{ runs.length }} run(s)</span>
          <el-button type="primary" size="small" @click="showCreateRun = true">New Run</el-button>
        </div>
        <el-table :data="runs" stripe size="small">
          <el-table-column prop="name" label="Name" min-width="140" />
          <el-table-column prop="status" label="Status" width="90">
            <template #default="{ row }"><el-tag :type="runStatusColor(row.status)" size="small">{{ row.status }}</el-tag></template>
          </el-table-column>
          <el-table-column label="Progress" min-width="160">
            <template #default="{ row }">
              <el-progress :percentage="runPercent(row)" :color="row.status==='failed'?'#f56c6c':'#67c23a'" :stroke-width="16">
                <span style="font-size:11px">{{ row.passed }}/{{ row.total }}</span>
              </el-progress>
            </template>
          </el-table-column>
          <el-table-column label="Actions" width="160">
            <template #default="{ row }">
              <el-button v-if="row.status==='queued'" size="small" type="success" @click="handleStartRun(row)">Start</el-button>
              <el-button v-if="row.status==='running'" size="small" type="warning" @click="handleCancelRun(row)">Cancel</el-button>
              <el-button v-if="row.status==='running'" size="small" @click="watchRun(row)">Watch</el-button>
              <el-button v-if="row.status==='success'||row.status==='failed'" size="small" @click="viewResults(row)">Results</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- AI -->
      <el-tab-pane label="AI" name="ai">
        <AiPanel :apis="apis" @case-created="onAiCaseCreated" />
      </el-tab-pane>
    </el-tabs>

    <!-- Create API Dialog -->
    <el-dialog v-model="showCreateAPI" title="Add API" width="400px">
      <el-form :model="apiForm" label-width="80px">
        <el-form-item label="Name"><el-input v-model="apiForm.name" /></el-form-item>
        <el-form-item label="Method"><el-select v-model="apiForm.method"><el-option v-for="m in methods" :key="m" :label="m" :value="m" /></el-select></el-form-item>
        <el-form-item label="Path"><el-input v-model="apiForm.path" placeholder="/pets" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="showCreateAPI = false">Cancel</el-button><el-button type="primary" :loading="creatingAPI" @click="handleCreateAPI">Create</el-button></template>
    </el-dialog>

    <!-- Create Case Dialog -->
    <el-dialog v-model="showCreateCase" title="New Test Case" width="450px">
      <el-form :model="caseForm" label-width="90px">
        <el-form-item label="Name"><el-input v-model="caseForm.name" placeholder="Get pets" /></el-form-item>
        <el-form-item label="Method"><el-select v-model="caseForm.method"><el-option v-for="m in methods" :key="m" :label="m" :value="m" /></el-select></el-form-item>
        <el-form-item label="Path"><el-input v-model="caseForm.path" placeholder="/pets" /></el-form-item>
        <el-form-item label="Expected Status"><el-input-number v-model="caseForm.expected_status" :min="100" :max="599" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="showCreateCase = false">Cancel</el-button><el-button type="primary" :loading="creatingCase" @click="handleCreateCase">Create</el-button></template>
    </el-dialog>

    <!-- Create Run Dialog -->
    <el-dialog v-model="showCreateRun" title="New Test Run" width="420px">
      <el-form :model="runForm" label-width="110px">
        <el-form-item label="Name"><el-input v-model="runForm.name" placeholder="Smoke test" /></el-form-item>
        <el-form-item label="Target Base URL"><el-input v-model="runForm.target_base_url" placeholder="http://localhost:8081/mock/petstore" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="showCreateRun = false">Cancel</el-button><el-button type="primary" :loading="creatingRun" @click="handleCreateRun">Create</el-button></template>
    </el-dialog>

    <!-- Run Progress Dialog (SSE) -->
    <el-dialog v-model="showProgress" title="Run Progress" width="480px" :close-on-click-modal="false">
      <div v-if="progress" class="progress-box">
        <el-progress :percentage="runPercent(progress)" :color="progress.status==='failed'?'#f56c6c':'#67c23a'" :stroke-width="20">
          <span>{{ progress.passed }}/{{ progress.total }}</span>
        </el-progress>
        <div class="stats">
          <el-tag type="success">Pass: {{ progress.passed }}</el-tag>
          <el-tag type="danger">Fail: {{ progress.failed }}</el-tag>
          <el-tag type="warning">Error: {{ progress.errored }}</el-tag>
        </div>
        <div v-if="progress.status==='success'" class="result success">All passed!</div>
        <div v-if="progress.status==='failed'" class="result failed">Run failed</div>
      </div>
      <template #footer><el-button @click="closeProgress">Close</el-button></template>
    </el-dialog>

    <!-- Edit Case Dialog -->
    <el-dialog v-model="showEditCase" title="Edit Test Case" width="420px">
      <el-form :model="editCaseForm" label-width="110px">
        <el-form-item label="Name"><el-input v-model="editCaseForm.name" /></el-form-item>
        <el-form-item label="Expected Status"><el-input-number v-model="editCaseForm.expected_status" :min="100" :max="599" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="showEditCase = false">Cancel</el-button><el-button type="primary" :loading="savingCase" @click="handleSaveCase">Save</el-button></template>
    </el-dialog>

    <!-- Results Dialog -->
    <el-dialog v-model="showResults" :title="'Results — ' + selectedRunName" width="600px">
      <el-table :data="results" stripe size="small" max-height="400">
        <el-table-column label="Status" width="70">
          <template #default="{ row }"><el-tag :type="row.status==='pass'?'success':'danger'" size="small">{{ row.status }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="actual_status" label="HTTP" width="60" />
        <el-table-column prop="duration_ms" label="Time" width="70">
          <template #default="{ row }">{{ row.duration_ms }}ms</template>
        </el-table-column>
        <el-table-column prop="error_msg" label="Error" min-width="160" />
      </el-table>
      <template #footer><el-button @click="showResults = false">Close</el-button></template>
    </el-dialog>

    <!-- Create / Edit Rule Dialog -->
    <el-dialog v-model="showRuleDialog" :title="editingRuleId ? 'Edit Rule' : 'New Rule'" width="500px">
      <el-form :model="ruleForm" label-width="110px">
        <el-form-item label="Name"><el-input v-model="ruleForm.name" placeholder="e.g. Success response" /></el-form-item>
        <el-form-item label="Match JSON"><el-input v-model="ruleForm.match_json" type="textarea" :rows="3" placeholder='{"path":"/pets","method":"GET"}' /></el-form-item>
        <el-form-item label="Response Status"><el-input-number v-model="ruleForm.response_status" :min="100" :max="599" /></el-form-item>
        <el-form-item label="Response Body"><el-input v-model="ruleForm.response_body_json" type="textarea" :rows="3" placeholder='{"id":1,"name":"dog"}' /></el-form-item>
        <el-form-item label="Priority"><el-input-number v-model="ruleForm.priority" :min="0" :max="100" /></el-form-item>
        <el-form-item label="Delay (ms)"><el-input-number v-model="ruleForm.delay_ms" :min="0" :max="30000" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showRuleDialog = false">Cancel</el-button>
        <el-button type="primary" :loading="savingRule" @click="handleSaveRule">{{ editingRuleId ? 'Save' : 'Create' }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useProjectStore } from '@/stores/project'
import { listAPIs, createAPI, deleteAPI, importOpenAPI } from '@/api/apis'
import { listRules, createRule, updateRule, deleteRule } from '@/api/mock_rules'
import { listCases, createCase, deleteCase, getCase, updateCase } from '@/api/test_cases'
import { listRuns, createRun, startRun, cancelRun, streamRun, getResults } from '@/api/test_runs'
import type { TestResult } from '@/types/test_run'
import type { API } from '@/types/api'
import type { MockRule } from '@/types/mock_rule'
import type { TestCase } from '@/types/test_case'
import type { TestRun, RunEvent } from '@/types/test_run'
import AiPanel from '@/components/AiPanel.vue'
import type { GeneratedCase } from '@/types/ai'
import { ElMessage } from 'element-plus'

const route = useRoute()
const store = useProjectStore()
const pid = route.params.pid as string

const project = ref(store.current)
const loading = ref(false)
const activeTab = ref('apis')
const apis = ref<API[]>([])
const rules = ref<MockRule[]>([])
const cases = ref<TestCase[]>([])
const runs = ref<TestRun[]>([])
const selectedApiId = ref('')
const methods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE']

const showCreateAPI = ref(false); const creatingAPI = ref(false); const importing = ref(false)
const apiForm = ref({ name: '', method: 'GET', path: '' })
const showCreateCase = ref(false); const creatingCase = ref(false)
const caseForm = ref({ name: '', method: 'GET', path: '', expected_status: 200 })
const showCreateRun = ref(false); const creatingRun = ref(false)
const runForm = ref({ name: '', target_base_url: '' })
const showProgress = ref(false); const progress = ref<RunEvent | null>(null)
let closeStream: (() => void) | null = null
const showEditCase = ref(false); const savingCase = ref(false)
const editCaseForm = ref({ id: '', name: '', expected_status: 200 })
const showResults = ref(false); const results = ref<TestResult[]>([])
const selectedRunName = ref('')
const showRuleDialog = ref(false); const savingRule = ref(false); const editingRuleId = ref('')
const ruleForm = ref({ name: '', match_json: '', response_status: 200, response_body_json: '', priority: 0, delay_ms: 0 })

onMounted(async () => {
  loading.value = true
  try {
    await store.fetchOne(pid)
    project.value = store.current
    apis.value = await listAPIs(pid)
    cases.value = await listCases(pid)
    runs.value = await listRuns(pid)
  } finally { loading.value = false }
})

// APIs
async function handleCreateAPI() { creatingAPI.value = true; try { apis.value.push(await createAPI(pid, apiForm.value)); showCreateAPI.value = false; apiForm.value = { name: '', method: 'GET', path: '' } } finally { creatingAPI.value = false } }
async function handleDeleteAPI(row: API) { try { await deleteAPI(pid, row.id); apis.value = apis.value.filter(a => a.id !== row.id) } catch { /* */ } }
async function handleImport(file: any) { importing.value = true; try { const r = new FileReader(); r.onload = async (e) => { await importOpenAPI(pid, e.target?.result as string); apis.value = await listAPIs(pid); importing.value = false }; r.readAsText(file.raw) } catch { importing.value = false } }
async function fetchRules() { if (selectedApiId.value) rules.value = await listRules(selectedApiId.value) }
async function handleDeleteRule(row: MockRule) { await deleteRule(row.id); rules.value = rules.value.filter(r => r.id !== row.id) }
function openCreateRule() { editingRuleId.value = ''; ruleForm.value = { name: '', match_json: '', response_status: 200, response_body_json: '', priority: 0, delay_ms: 0 }; showRuleDialog.value = true }
function openEditRule(row: MockRule) { editingRuleId.value = row.id; ruleForm.value = { name: row.name, match_json: JSON.stringify(row.match_json), response_status: row.response_status, response_body_json: typeof row.response_body_json === 'string' ? row.response_body_json as string : JSON.stringify(row.response_body_json), priority: row.priority, delay_ms: row.delay_ms }; showRuleDialog.value = true }
async function handleSaveRule() {
  savingRule.value = true
  try {
    let matchJSON: unknown; try { matchJSON = JSON.parse(ruleForm.value.match_json) } catch { matchJSON = {} }
    const body: Record<string, unknown> = { name: ruleForm.value.name, match_json: matchJSON, response_status: ruleForm.value.response_status, priority: ruleForm.value.priority, delay_ms: ruleForm.value.delay_ms }
    if (ruleForm.value.response_body_json) { try { body.response_body_json = JSON.parse(ruleForm.value.response_body_json) } catch { body.response_body_json = ruleForm.value.response_body_json } }
    if (editingRuleId.value) { await updateRule(editingRuleId.value, body) } else { body.api_id = selectedApiId.value; await createRule(body as any) }
    showRuleDialog.value = false
    await fetchRules()
  } finally { savingRule.value = false }
}

// Test Cases
async function handleCreateCase() { creatingCase.value = true; try { cases.value.push(await createCase(pid, caseForm.value)); showCreateCase.value = false; caseForm.value = { name: '', method: 'GET', path: '', expected_status: 200 } } finally { creatingCase.value = false } }
async function handleDeleteCase(row: TestCase) { try { await deleteCase(pid, row.id); cases.value = cases.value.filter(c => c.id !== row.id) } catch { /* */ } }

// Test Runs
async function handleCreateRun() { creatingRun.value = true; try { runs.value.unshift(await createRun(pid, runForm.value)); showCreateRun.value = false; runForm.value = { name: '', target_base_url: '' } } finally { creatingRun.value = false } }
async function handleStartRun(row: TestRun) { try { await startRun(pid, row.id); await refreshRuns() } catch { /* */ } }
async function handleCancelRun(row: TestRun) { try { await cancelRun(pid, row.id); await refreshRuns() } catch { /* */ } }
function watchRun(row: TestRun) { showProgress.value = true; closeStream = streamRun(pid, row.id, (e) => { progress.value = e; if (e.type === 'complete') { refreshRuns() } }) }
function closeProgress() { if (closeStream) closeStream(); showProgress.value = false; progress.value = null }
async function refreshRuns() { runs.value = await listRuns(pid) }

async function openEditCase(row: TestCase) {
  const tc = await getCase(pid, row.id)
  editCaseForm.value = { id: tc.id, name: tc.name, expected_status: tc.expected_status }
  showEditCase.value = true
}
async function handleSaveCase() {
  savingCase.value = true
  try {
    await updateCase(pid, editCaseForm.value.id, { name: editCaseForm.value.name, expected_status: editCaseForm.value.expected_status })
    showEditCase.value = false
    cases.value = await listCases(pid)
  } finally { savingCase.value = false }
}
async function viewResults(row: TestRun) {
  selectedRunName.value = row.name
  results.value = await getResults(pid, row.id)
  showResults.value = true
}

function onAiCaseCreated(c: GeneratedCase) {
  caseForm.value = { name: c.name, method: c.method, path: c.path, expected_status: c.expected_status }
  showCreateCase.value = true
}
function methodColor(m: string) { const map: Record<string, string> = { GET: 'success', POST: 'primary', PUT: 'warning', PATCH: 'warning', DELETE: 'danger' }; return map[m] || 'info' }
function runStatusColor(s: string) { const map: Record<string, string> = { queued: 'info', running: 'warning', success: 'success', failed: 'danger', cancelled: 'info' }; return map[s] || 'info' }
function runPercent(r: { total: number; passed: number }) { return r.total ? Math.round((r.passed / r.total) * 100) : 0 }
</script>

<style scoped>
.header { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
h2 { margin: 0; }
.tab-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.progress-box { text-align: center; }
.stats { display: flex; gap: 8px; justify-content: center; margin-top: 16px; }
.result { margin-top: 16px; font-size: 16px; font-weight: bold; }
.success { color: #67c23a; }
.failed { color: #f56c6c; }
</style>
