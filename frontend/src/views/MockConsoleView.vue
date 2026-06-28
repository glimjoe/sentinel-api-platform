<template>
  <div class="mock-console" v-loading="loading">
    <div class="header">
      <h2>Mock Console</h2>
      <span v-if="project" class="subtitle">{{ project.name }} ({{ project.slug }})</span>
    </div>

    <el-row :gutter="16">
      <!-- Request panel -->
      <el-col :span="12">
        <el-card header="Request">
          <el-form label-width="80px" size="small">
            <el-form-item label="API">
              <el-select v-model="selectedApiId" placeholder="Choose endpoint" style="width:100%" @change="onApiSelect">
                <el-option v-for="a in apis" :key="a.id" :label="`${a.method} ${a.path}`" :value="a.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="Method">
              <el-select v-model="reqMethod">
                <el-option v-for="m in methods" :key="m" :label="m" :value="m" />
              </el-select>
            </el-form-item>
            <el-form-item label="Path">
              <el-input v-model="reqPath" placeholder="/pets" />
            </el-form-item>
            <el-form-item label="Headers">
              <el-input v-model="reqHeadersStr" type="textarea" :rows="3" placeholder='{"Authorization":"Bearer x"}' />
            </el-form-item>
            <el-form-item label="Body">
              <el-input v-model="reqBody" type="textarea" :rows="4" placeholder='{"name":"test"}' />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="sending" @click="sendRequest">Send Request</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <!-- Response panel -->
      <el-col :span="12">
        <el-card header="Response">
          <div v-if="!response && !sendError" class="placeholder">Send a request to see the response.</div>
          <div v-else>
            <div class="resp-status">
              <el-tag v-if="respStatus" :type="respStatus < 400 ? 'success' : 'danger'">{{ respStatus }} {{ respStatusText }}</el-tag>
              <span v-if="respDuration" style="margin-left:8px;color:#909399">{{ respDuration }}ms</span>
            </div>
            <el-divider v-if="respHeadersStr" />
            <pre v-if="respHeadersStr" class="headers">{{ respHeadersStr }}</pre>
            <el-divider v-if="respBody" />
            <pre v-if="respBody" class="body">{{ respBody }}</pre>
            <el-alert v-if="sendError" type="error" :title="sendError" :closable="false" style="margin-top:12px" />
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useProjectStore } from '@/stores/project'
import { listAPIs } from '@/api/apis'
import type { API } from '@/types/api'
import axios from 'axios'

const route = useRoute()
const store = useProjectStore()
const pid = route.params.pid as string

const project = ref(store.current)
const loading = ref(false)
const apis = ref<API[]>([])
const selectedApiId = ref('')
const methods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE']

const reqMethod = ref('GET')
const reqPath = ref('')
const reqHeadersStr = ref('')
const reqBody = ref('')
const sending = ref(false)

const response = ref<any>(null)
const respStatus = ref(0)
const respStatusText = ref('')
const respHeadersStr = ref('')
const respBody = ref('')
const respDuration = ref(0)
const sendError = ref('')

onMounted(async () => {
  loading.value = true
  try {
    await store.fetchOne(pid)
    project.value = store.current
    apis.value = await listAPIs(pid)
  } finally { loading.value = false }
})

function onApiSelect(apiId: string) {
  const a = apis.value.find(x => x.id === apiId)
  if (a) { reqMethod.value = a.method; reqPath.value = a.path }
}

async function sendRequest() {
  sending.value = true; sendError.value = ''; response.value = null
  respStatus.value = 0; respHeadersStr.value = ''; respBody.value = ''; respDuration.value = 0
  const t0 = performance.now()
  try {
    const url = `/mock/${project.value!.slug}${reqPath.value}`
    let headers: Record<string, string> = {}
    try { headers = JSON.parse(reqHeadersStr.value) } catch { /* ignore */ }
    const resp = await axios.request({ method: reqMethod.value, url, headers, data: reqBody.value || undefined, validateStatus: () => true })
    respDuration.value = Math.round(performance.now() - t0)
    respStatus.value = resp.status; respStatusText.value = resp.statusText
    respHeadersStr.value = JSON.stringify(resp.headers, null, 2)
    respBody.value = typeof resp.data === 'string' ? resp.data : JSON.stringify(resp.data, null, 2)
  } catch (e: any) {
    respDuration.value = Math.round(performance.now() - t0)
    sendError.value = e.message ?? 'Request failed'
  } finally { sending.value = false }
}
</script>

<style scoped>
.header { display: flex; align-items: baseline; gap: 12px; margin-bottom: 16px; }
.header h2 { margin: 0; }
.subtitle { color: #909399; }
.placeholder { color: #c0c4cc; font-style: italic; padding: 24px 0; text-align: center; }
.resp-status { margin-bottom: 8px; }
pre.headers, pre.body { max-height: 300px; overflow: auto; background: #f5f7fa; padding: 8px; border-radius: 4px; font-size: 12px; white-space: pre-wrap; }
</style>
