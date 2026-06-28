<template>
  <div class="project-detail" v-loading="loading">
    <div class="header">
      <h2>{{ project?.name }}</h2>
      <el-tag v-if="project">{{ project.slug }}</el-tag>
    </div>
    <p class="desc">{{ project?.description }}</p>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="APIs" name="apis">
        <div class="tab-header">
          <span>{{ apis.length }} API(s)</span>
          <el-button type="primary" size="small" @click="showCreateAPI = true">Add API</el-button>
        </div>
        <el-table :data="apis" stripe size="small">
          <el-table-column label="Method" width="80">
            <template #default="{ row }">
              <el-tag :type="methodColor(row.method)" size="small">{{ row.method }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="path" label="Path" min-width="180" />
          <el-table-column prop="name" label="Name" min-width="130" />
          <el-table-column prop="source" label="Source" width="90" />
          <el-table-column label="Actions" width="80">
            <template #default="{ row }">
              <el-button size="small" type="danger" @click="handleDeleteAPI(row)">Del</el-button>
            </template>
          </el-table-column>
        </el-table>

        <!-- Import OpenAPI -->
        <el-divider />
        <el-upload :auto-upload="false" :show-file-list="false" accept=".json,.yaml,.yml"
          @change="handleImport">
          <el-button size="small" :loading="importing">Import OpenAPI Spec</el-button>
        </el-upload>
      </el-tab-pane>

      <el-tab-pane label="Mock Rules" name="rules">
        <div class="tab-header">
          <span>Select an API to manage its rules</span>
        </div>
        <el-select v-model="selectedApiId" placeholder="Choose API" @change="fetchRules" style="width:300px;margin-bottom:12px">
          <el-option v-for="a in apis" :key="a.id" :label="`${a.method} ${a.path}`" :value="a.id" />
        </el-select>

        <el-table v-if="rules.length" :data="rules" stripe size="small">
          <el-table-column prop="name" label="Name" min-width="140" />
          <el-table-column prop="response_status" label="Status" width="70" />
          <el-table-column prop="priority" label="Priority" width="70" />
          <el-table-column label="Enabled" width="80">
            <template #default="{ row }">
              <el-switch :model-value="row.enabled" disabled size="small" />
            </template>
          </el-table-column>
          <el-table-column label="Actions" width="80">
            <template #default="{ row }">
              <el-button size="small" type="danger" @click="handleDeleteRule(row)">Del</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-else-if="selectedApiId" description="No rules for this API" />
      </el-tab-pane>
    </el-tabs>

    <!-- Create API Dialog -->
    <el-dialog v-model="showCreateAPI" title="Add API" width="400px">
      <el-form :model="apiForm" label-width="80px">
        <el-form-item label="Name"><el-input v-model="apiForm.name" /></el-form-item>
        <el-form-item label="Method">
          <el-select v-model="apiForm.method">
            <el-option v-for="m in methods" :key="m" :label="m" :value="m" />
          </el-select>
        </el-form-item>
        <el-form-item label="Path"><el-input v-model="apiForm.path" placeholder="/pets" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateAPI = false">Cancel</el-button>
        <el-button type="primary" :loading="creatingAPI" @click="handleCreateAPI">Create</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useProjectStore } from '@/stores/project'
import { listAPIs, createAPI, deleteAPI, importOpenAPI } from '@/api/apis'
import { listRules, deleteRule } from '@/api/mock_rules'
import type { API } from '@/types/api'
import type { MockRule } from '@/types/mock_rule'
import { ElMessage } from 'element-plus'

const route = useRoute()
const store = useProjectStore()
const pid = route.params.pid as string

const project = ref(store.current)
const loading = ref(false)
const activeTab = ref('apis')
const apis = ref<API[]>([])
const rules = ref<MockRule[]>([])
const selectedApiId = ref('')

const showCreateAPI = ref(false)
const creatingAPI = ref(false)
const importing = ref(false)
const methods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS']
const apiForm = ref({ name: '', method: 'GET' as string, path: '' })

onMounted(async () => {
  loading.value = true
  try {
    await store.fetchOne(pid)
    project.value = store.current
    apis.value = await listAPIs(pid)
  } finally {
    loading.value = false
  }
})

async function handleCreateAPI() {
  creatingAPI.value = true
  try {
    const a = await createAPI(pid, apiForm.value)
    apis.value.push(a)
    showCreateAPI.value = false
    apiForm.value = { name: '', method: 'GET', path: '' }
  } finally { creatingAPI.value = false }
}

async function handleDeleteAPI(row: API) {
  try {
    await deleteAPI(pid, row.id)
    apis.value = apis.value.filter(a => a.id !== row.id)
  } catch { /* handled */ }
}

async function handleImport(file: any) {
  importing.value = true
  try {
    const reader = new FileReader()
    reader.onload = async (e) => {
      const text = e.target?.result as string
      const result = await importOpenAPI(pid, text)
      ElMessage.success(`Imported ${result.imported} of ${result.total} APIs`)
      apis.value = await listAPIs(pid)
      importing.value = false
    }
    reader.readAsText(file.raw)
  } catch { importing.value = false }
}

async function fetchRules() {
  if (!selectedApiId.value) return
  rules.value = await listRules(selectedApiId.value)
}

async function handleDeleteRule(row: MockRule) {
  await deleteRule(row.id)
  rules.value = rules.value.filter(r => r.id !== row.id)
}

function methodColor(m: string) {
  const map: Record<string, string> = { GET: 'success', POST: 'primary', PUT: 'warning', PATCH: 'warning', DELETE: 'danger' }
  return map[m] || 'info'
}
</script>

<style scoped>
.header { display: flex; align-items: center; gap: 12px; margin-bottom: 8px; }
.desc { color: #666; margin-bottom: 16px; }
h2 { margin: 0; }
.tab-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
</style>
