import { defineStore } from 'pinia'
import { ref } from 'vue'
import { listProjects, createProject, getProject, deleteProject, updateProject } from '@/api/project'
import type { Project } from '@/types/project'

export const useProjectStore = defineStore('project', () => {
  const projects = ref<Project[]>([])
  const current = ref<Project | null>(null)
  const loading = ref(false)

  async function fetchList() {
    loading.value = true
    try {
      projects.value = await listProjects()
    } finally {
      loading.value = false
    }
  }

  async function fetchOne(pid: string) {
    current.value = await getProject(pid)
  }

  async function create(data: { name: string; description?: string }) {
    const p = await createProject(data)
    projects.value.unshift(p)
    return p
  }

  async function update(pid: string, data: Record<string, unknown>) {
    const p = await updateProject(pid, data)
    current.value = p
    const idx = projects.value.findIndex(x => x.id === pid)
    if (idx >= 0) projects.value[idx] = p
    return p
  }

  async function remove(pid: string) {
    await deleteProject(pid)
    projects.value = projects.value.filter(x => x.id !== pid)
    if (current.value?.id === pid) current.value = null
  }

  return { projects, current, loading, fetchList, fetchOne, create, update, remove }
})
