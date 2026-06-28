import { defineStore } from 'pinia'
import { ref } from 'vue'
import { listAPIs, createAPI, getAPI, updateAPI, deleteAPI, importOpenAPI } from '@/api/apis'
import type { API } from '@/types/api'

export const useAPIStore = defineStore('api', () => {
  const apis = ref<API[]>([])
  const current = ref<API | null>(null)
  const loading = ref(false)

  async function fetchList(pid: string) {
    loading.value = true
    try {
      apis.value = await listAPIs(pid)
    } finally {
      loading.value = false
    }
  }

  async function fetchOne(pid: string, apiId: string) {
    current.value = await getAPI(pid, apiId)
    return current.value
  }

  async function create(pid: string, data: { name: string; method: string; path: string }): Promise<API> {
    const a = await createAPI(pid, data)
    apis.value.push(a)
    return a
  }

  async function update(pid: string, apiId: string, data: Record<string, unknown>): Promise<API> {
    const a = await updateAPI(pid, apiId, data)
    current.value = a
    const idx = apis.value.findIndex((x) => x.id === apiId)
    if (idx >= 0) apis.value[idx] = a
    return a
  }

  async function remove(pid: string, apiId: string) {
    await deleteAPI(pid, apiId)
    apis.value = apis.value.filter((x) => x.id !== apiId)
    if (current.value?.id === apiId) current.value = null
  }

  async function importSpec(pid: string, spec: string) {
    return await importOpenAPI(pid, spec)
  }

  return { apis, current, loading, fetchList, fetchOne, create, update, remove, importSpec }
})
