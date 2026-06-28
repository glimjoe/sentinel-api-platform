<template>
  <div class="ai-panel">
    <el-card header="Failure Attribution" class="section">
      <p class="desc">Analyze a failed test result to find the root cause.</p>
      <el-input v-model="attrJson" type="textarea" :rows="4" placeholder="Paste test result JSON here..." />
      <el-button type="primary" :loading="attrLoading" @click="runAttribution" style="margin-top:8px">
        Analyze
      </el-button>
      <div v-if="attrResult" class="result-box">
        <div><strong>Root Cause:</strong> {{ attrResult.root_cause }}</div>
        <div><strong>Confidence:</strong> {{ (attrResult.confidence * 100).toFixed(0) }}%</div>
        <div><strong>Analysis:</strong> {{ attrResult.analysis }}</div>
        <div v-if="attrResult.suggested_fix"><strong>Suggested Fix:</strong> {{ attrResult.suggested_fix }}</div>
      </div>
    </el-card>

    <el-card header="Test Case Generation" class="section">
      <p class="desc">Generate test cases from API specifications.</p>
      <el-select v-model="genApiId" placeholder="All APIs" clearable style="width:240px">
        <el-option v-for="a in apis" :key="a.id" :label="`${a.method} ${a.path}`" :value="a.id" />
      </el-select>
      <el-button type="success" :loading="genLoading" @click="runGeneration" style="margin-left:8px">
        Generate
      </el-button>
      <el-table v-if="genCases.length" :data="genCases" stripe size="small" style="margin-top:12px">
        <el-table-column prop="name" label="Name" min-width="140" />
        <el-table-column prop="method" label="Method" width="70" />
        <el-table-column prop="path" label="Path" min-width="140" />
        <el-table-column prop="expected_status" label="Status" width="70" />
        <el-table-column label="Action" width="80">
          <template #default="{ row }">
            <el-button size="small" @click="$emit('caseCreated', row)">Apply</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card header="Priority Suggestion" class="section">
      <p class="desc">Get AI-suggested priorities for test cases.</p>
      <el-button type="warning" :loading="prioLoading" @click="runPrioritization">
        Suggest Priorities
      </el-button>
      <el-table v-if="priorities.length" :data="priorities" stripe size="small" style="margin-top:12px">
        <el-table-column prop="case_id" label="Case ID" width="180" />
        <el-table-column label="Priority" width="80">
          <template #default="{ row }"><el-tag :type="prioColor(row.priority)" size="small">{{ row.priority }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="reasoning" label="Reasoning" min-width="180" />
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { attributeResult, generateCases, prioritizeCases } from '@/api/ai'
import type { GeneratedCase, PriorityItem, AttributionResult } from '@/types/ai'
import type { API } from '@/types/api'
import { ElMessage } from 'element-plus'

defineProps<{ apis: API[] }>()
defineEmits<{ caseCreated: [c: GeneratedCase] }>()

const attrJson = ref('')
const attrLoading = ref(false)
const attrResult = ref<AttributionResult | null>(null)

async function runAttribution() {
  attrLoading.value = true
  try {
    attrResult.value = await attributeResult(pid(), attrJson.value)
  } catch {
    ElMessage.error('Attribution failed')
  } finally { attrLoading.value = false }
}

const genApiId = ref('')
const genLoading = ref(false)
const genCases = ref<GeneratedCase[]>([])

async function runGeneration() {
  genLoading.value = true
  try {
    const resp = await generateCases(pid(), genApiId.value || undefined)
    genCases.value = resp.test_cases
  } catch {
    ElMessage.error('Generation failed')
  } finally { genLoading.value = false }
}

const prioLoading = ref(false)
const priorities = ref<PriorityItem[]>([])

async function runPrioritization() {
  prioLoading.value = true
  try {
    const resp = await prioritizeCases(pid(), undefined)
    priorities.value = resp.priorities
  } catch {
    ElMessage.error('Prioritization failed')
  } finally { prioLoading.value = false }
}

function pid(): string {
  return window.location.pathname.split('/')[2]
}

function prioColor(p: string) {
  const m: Record<string, string> = { p0: 'danger', p1: 'warning', p2: 'info', p3: '' }
  return m[p] || 'info'
}
</script>

<style scoped>
.section { margin-bottom: 16px; }
.desc { color: #909399; font-size: 13px; margin-bottom: 8px; }
.result-box { margin-top: 12px; background: #f5f7fa; padding: 12px; border-radius: 4px; }
.result-box div { margin-bottom: 6px; }
</style>
