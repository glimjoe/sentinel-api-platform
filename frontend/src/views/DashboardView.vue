<template>
  <div class="dashboard">
    <el-container>
      <el-header class="hdr">
        <h2 style="margin: 0;">Sentinel · Dashboard</h2>
        <div>
          <span style="margin-right: 12px; color: #606266;">
            {{ auth.user?.email ?? 'signed in' }}
          </span>
          <el-button @click="onLogout" size="small">Sign out</el-button>
        </div>
      </el-header>
      <el-main>
        <el-card>
          <template #header>
            <strong>Phase 1 MVP placeholder</strong>
          </template>
          <p>Welcome, <strong>{{ auth.user?.email ?? 'user' }}</strong>.</p>
          <p>Phase 1 (skeleton) is wired up: backend serves <code>/healthz</code> and <code>/readyz</code>;
             the frontend talks to the API through Vite's proxy at <code>/api</code> → <code>:8081</code>.</p>
          <p>Full feature set lands in Phases 2–5 per the project plan.</p>
        </el-card>
        <el-card style="margin-top: 16px;">
          <template #header><strong>Quick links</strong></template>
          <ul>
            <li><a href="/api/v1/healthz" target="_blank">/api/v1/healthz</a> — backend liveness</li>
            <li><a href="/api/v1/readyz" target="_blank">/api/v1/readyz</a> — backend readiness (DB + Redis)</li>
          </ul>
        </el-card>
      </el-main>
    </el-container>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

function onLogout() {
  auth.logout()
  router.push('/login')
}
</script>

<style scoped>
.dashboard { min-height: 100vh; background: #f5f7fa; }
.hdr { display: flex; align-items: center; justify-content: space-between; background: #fff; border-bottom: 1px solid #e4e7ed; }
</style>
