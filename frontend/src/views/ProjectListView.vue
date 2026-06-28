<template>
  <div class="project-list">
    <div class="header">
      <h2>Projects</h2>
      <el-button type="primary" @click="showCreate = true">New Project</el-button>
    </div>

    <el-table :data="store.projects" v-loading="store.loading" stripe>
      <el-table-column prop="name" label="Name" min-width="150">
        <template #default="{ row }">
          <router-link :to="`/projects/${row.id}`">{{ row.name }}</router-link>
        </template>
      </el-table-column>
      <el-table-column prop="slug" label="Slug" width="140" />
      <el-table-column prop="description" label="Description" min-width="200" />
      <el-table-column label="Actions" width="140">
        <template #default="{ row }">
          <el-button size="small" @click="handleDelete(row)">Delete</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="showCreate" title="Create Project" width="420px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="Name">
          <el-input v-model="form.name" placeholder="My Project" />
        </el-form-item>
        <el-form-item label="Description">
          <el-input v-model="form.description" type="textarea" placeholder="Optional" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreate = false">Cancel</el-button>
        <el-button type="primary" :loading="creating" @click="handleCreate">Create</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useProjectStore } from '@/stores/project'

const store = useProjectStore()
const router = useRouter()
const showCreate = ref(false)
const creating = ref(false)
const form = ref({ name: '', description: '' })

onMounted(() => store.fetchList())

async function handleCreate() {
  creating.value = true
  try {
    const p = await store.create(form.value)
    showCreate.value = false
    form.value = { name: '', description: '' }
    router.push(`/projects/${p.id}`)
  } finally {
    creating.value = false
  }
}

async function handleDelete(row: { id: string; name: string }) {
  try {
    await store.remove(row.id)
  } catch { /* handled by interceptor */ }
}
</script>

<style scoped>
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
h2 { margin: 0; }
</style>
