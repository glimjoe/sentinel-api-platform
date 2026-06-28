<template>
  <el-container>
    <el-header v-if="auth.isAuthenticated" class="nav">
      <div class="nav-left">
        <router-link to="/projects" class="logo">Sentinel</router-link>
      </div>
      <div class="nav-right">
        <span class="user">{{ auth.user?.email }}</span>
        <el-button size="small" @click="auth.logout(); $router.push('/login')">Logout</el-button>
      </div>
    </el-header>
    <el-main>
      <router-view />
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'

const auth = useAuthStore()
const $router = useRouter()
</script>

<style>
html, body, #app { height: 100%; margin: 0; padding: 0; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }
.nav { display: flex; justify-content: space-between; align-items: center; background: #409eff; color: white; padding: 0 24px; }
.nav a { color: white; text-decoration: none; }
.logo { font-size: 18px; font-weight: bold; }
.nav-right { display: flex; align-items: center; gap: 12px; }
.user { font-size: 14px; }
</style>
