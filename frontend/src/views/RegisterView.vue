<template>
  <div class="auth-page">
    <el-card class="auth-card">
      <template #header>
        <h2 style="margin: 0; text-align: center;">Sentinel · Create account</h2>
      </template>
      <el-form :model="form" label-position="top" @submit.prevent="onSubmit">
        <el-form-item label="Email">
          <el-input v-model="form.email" type="email" placeholder="you@example.com" />
        </el-form-item>
        <el-form-item label="Display name (optional)">
          <el-input v-model="form.displayName" />
        </el-form-item>
        <el-form-item label="Password">
          <el-input v-model="form.password" type="password" show-password />
        </el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading" style="width: 100%;">
          Create account
        </el-button>
        <p style="text-align: center; margin-top: 12px;">
          <router-link to="/login">Already have an account? Sign in</router-link>
        </p>
      </el-form>
      <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon style="margin-top: 12px;" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()
const form = reactive({ email: '', password: '', displayName: '' })
const loading = ref(false)
const error = ref<string | null>(null)

async function onSubmit() {
  loading.value = true
  error.value = null
  try {
    await auth.register(form.email, form.password, form.displayName || undefined)
    router.push('/dashboard')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Registration failed'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-page { min-height: 100vh; display: flex; align-items: center; justify-content: center; background: #f5f7fa; }
.auth-card { width: 380px; box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08); }
</style>
