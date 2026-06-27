<template>
  <div class="auth-page">
    <el-card class="auth-card">
      <template #header>
        <h2 style="margin: 0; text-align: center;">Sentinel · Sign in</h2>
      </template>
      <el-form :model="form" label-position="top" @submit.prevent="onSubmit">
        <el-form-item label="Email">
          <el-input v-model="form.email" type="email" placeholder="you@example.com" />
        </el-form-item>
        <el-form-item label="Password">
          <el-input v-model="form.password" type="password" show-password />
        </el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading" style="width: 100%;">
          Sign in
        </el-button>
        <p style="text-align: center; margin-top: 12px;">
          <router-link to="/register">Create an account</router-link>
        </p>
      </el-form>
      <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon style="margin-top: 12px;" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const form = reactive({ email: '', password: '' })
const loading = ref(false)
const error = ref<string | null>(null)

async function onSubmit() {
  loading.value = true
  error.value = null
  try {
    await auth.login(form.email, form.password)
    const redirect = (route.query.redirect as string) || '/dashboard'
    router.push(redirect)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Login failed'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f7fa;
}
.auth-card {
  width: 380px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}
</style>
