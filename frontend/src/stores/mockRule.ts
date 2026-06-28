import { defineStore } from 'pinia'
import { ref } from 'vue'
import { listRules, createRule, getRule, updateRule, deleteRule } from '@/api/mock_rules'
import type { MockRule } from '@/types/mock_rule'

export const useMockRuleStore = defineStore('mockRule', () => {
  const rules = ref<MockRule[]>([])
  const current = ref<MockRule | null>(null)
  const loading = ref(false)

  async function fetchList(apiId: string) {
    loading.value = true
    try {
      rules.value = await listRules(apiId)
    } finally {
      loading.value = false
    }
  }

  async function fetchOne(rid: string) {
    current.value = await getRule(rid)
    return current.value
  }

  async function create(data: {
    name: string
    match_json: unknown
    api_id: string
    response_status?: number
    response_body_json?: unknown
    priority?: number
  }): Promise<MockRule> {
    const r = await createRule(data)
    rules.value.push(r)
    return r
  }

  async function update(rid: string, data: Record<string, unknown>): Promise<MockRule> {
    const r = await updateRule(rid, data)
    current.value = r
    const idx = rules.value.findIndex((x) => x.id === rid)
    if (idx >= 0) rules.value[idx] = r
    return r
  }

  async function remove(rid: string) {
    await deleteRule(rid)
    rules.value = rules.value.filter((x) => x.id !== rid)
    if (current.value?.id === rid) current.value = null
  }

  return { rules, current, loading, fetchList, fetchOne, create, update, remove }
})
