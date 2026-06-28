<template>
  <div class="dashboard">
    <el-container>
      <el-header class="hdr">
        <h2 style="margin: 0">Sentinel · Dashboard</h2>
        <div>
          <span class="user-tag">{{ auth.user?.email ?? 'signed in' }}</span>
          <el-button @click="onLogout" size="small">Sign out</el-button>
        </div>
      </el-header>

      <el-main>
        <!-- Stat cards -->
        <el-row :gutter="16" class="stat-row">
          <el-col :span="6">
            <el-card shadow="hover" class="stat-card">
              <div class="stat-num">{{ stats.projects }}</div>
              <div class="stat-label">Projects</div>
            </el-card>
          </el-col>
          <el-col :span="6">
            <el-card shadow="hover" class="stat-card">
              <div class="stat-num">{{ stats.apis }}</div>
              <div class="stat-label">APIs</div>
            </el-card>
          </el-col>
          <el-col :span="6">
            <el-card shadow="hover" class="stat-card">
              <div class="stat-num">{{ stats.mock_rules }}</div>
              <div class="stat-label">Mock Rules</div>
            </el-card>
          </el-col>
          <el-col :span="6">
            <el-card shadow="hover" class="stat-card">
              <div class="stat-num">{{ stats.test_cases }}</div>
              <div class="stat-label">Test Cases</div>
            </el-card>
          </el-col>
        </el-row>

        <!-- Charts row -->
        <el-row :gutter="16" class="chart-row">
          <el-col :span="12">
            <el-card shadow="hover">
              <template #header><strong>Result Breakdown</strong></template>
              <v-chart :option="pieOption" autoresize style="height: 280px" />
            </el-card>
          </el-col>
          <el-col :span="12">
            <el-card shadow="hover">
              <template #header><strong>Recent Runs</strong></template>
              <v-chart :option="barOption" autoresize style="height: 280px" />
            </el-card>
          </el-col>
        </el-row>

        <!-- Recent runs table -->
        <el-card shadow="hover" class="table-card">
          <template #header><strong>Latest Runs</strong></template>
          <el-table :data="stats.recent_runs" style="width: 100%" size="small" empty-text="No runs yet.">
            <el-table-column prop="name" label="Run" min-width="160" />
            <el-table-column prop="status" label="Status" width="110">
              <template #default="{ row }">
                <el-tag :type="statusTag(row.status)" size="small">{{ row.status }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="total" label="Total" width="70" align="center" />
            <el-table-column prop="passed" label="Pass" width="70" align="center">
              <template #default="{ row }">
                <span class="pass-text">{{ row.passed }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="failed" label="Fail" width="70" align="center">
              <template #default="{ row }">
                <span :class="row.failed ? 'fail-text' : ''">{{ row.failed }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="errored" label="Err" width="70" align="center">
              <template #default="{ row }">
                <span :class="row.errored ? 'err-text' : ''">{{ row.errored }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="Created" width="170">
              <template #default="{ row }">
                {{ fmt(row.created_at) }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-main>
    </el-container>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { fetchDashboardStats, type DashboardStats } from '@/api/dashboard'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { PieChart, BarChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, LegendComponent, GridComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

use([PieChart, BarChart, TitleComponent, TooltipComponent, LegendComponent, GridComponent, CanvasRenderer])

const auth = useAuthStore()
const router = useRouter()

const stats = ref<DashboardStats>({
  projects: 0, apis: 0, mock_rules: 0, test_cases: 0,
  recent_runs: [], status_breakdown: {},
})

const pieOption = computed(() => {
  const b = stats.value.status_breakdown
  return {
    tooltip: { trigger: 'item' as const },
    legend: { bottom: 0 },
    series: [{
      type: 'pie' as const,
      radius: ['50%', '75%'],
      center: ['50%', '45%'],
      data: [
        { value: b.pass ?? 0, name: 'Pass', itemStyle: { color: '#67c23a' } },
        { value: b.fail ?? 0, name: 'Fail', itemStyle: { color: '#f56c6c' } },
        { value: b.error ?? 0, name: 'Error', itemStyle: { color: '#e6a23c' } },
        { value: b.skip ?? 0, name: 'Skip', itemStyle: { color: '#909399' } },
      ].filter(d => d.value > 0),
      label: { show: true, formatter: '{b}: {c}' },
    }],
  }
})

const barOption = computed(() => {
  const runs = [...stats.value.recent_runs].reverse()
  return {
    tooltip: { trigger: 'axis' as const },
    legend: { data: ['Pass', 'Fail', 'Error', 'Skip'], bottom: 0 },
    grid: { left: 8, right: 8, top: 8, bottom: 32 },
    xAxis: { type: 'category' as const, data: runs.map(r => r.name.slice(0, 12) || r.id.slice(-6)), axisLabel: { rotate: 30 } },
    yAxis: { type: 'value' as const },
    series: [
      { name: 'Pass', type: 'bar' as const, stack: 'x', data: runs.map(r => r.passed), itemStyle: { color: '#67c23a' } },
      { name: 'Fail', type: 'bar' as const, stack: 'x', data: runs.map(r => r.failed), itemStyle: { color: '#f56c6c' } },
      { name: 'Error', type: 'bar' as const, stack: 'x', data: runs.map(r => r.errored), itemStyle: { color: '#e6a23c' } },
      { name: 'Skip', type: 'bar' as const, stack: 'x', data: runs.map(r => r.skipped), itemStyle: { color: '#909399' } },
    ],
  }
})

function statusTag(s: string): 'success' | 'danger' | 'warning' | 'info' {
  const m: Record<string, 'success'|'danger'|'warning'|'info'> = {
    success: 'success', failed: 'danger', partial: 'warning',
    running: 'info', queued: 'info', cancelled: 'info',
  }
  return m[s] ?? 'info'
}

function fmt(ts: string): string {
  if (!ts) return ''
  return new Date(ts).toLocaleString()
}

function onLogout() {
  auth.logout()
  router.push('/login')
}

onMounted(async () => {
  try {
    stats.value = await fetchDashboardStats()
  } catch { /* dashboard stats unavailable — stay at zeros */ }
})
</script>

<style scoped>
.dashboard { min-height: 100vh; background: #f5f7fa; }
.hdr { display: flex; align-items: center; justify-content: space-between; background: #fff; border-bottom: 1px solid #e4e7ed; }
.user-tag { margin-right: 12px; color: #606266; }

.stat-row { margin-bottom: 16px; }
.stat-card { text-align: center; }
.stat-num { font-size: 28px; font-weight: 700; color: #303133; }
.stat-label { font-size: 13px; color: #909399; margin-top: 4px; }

.chart-row { margin-bottom: 16px; }

.pass-text { color: #67c23a; font-weight: 600; }
.fail-text { color: #f56c6c; font-weight: 600; }
.err-text { color: #e6a23c; font-weight: 600; }
</style>
