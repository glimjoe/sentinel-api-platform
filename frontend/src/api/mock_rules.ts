import { api } from './client'
import type { MockRule } from '@/types/mock_rule'

export function listRules(apiId: string): Promise<MockRule[]> {
  return api.get<any, MockRule[]>(`/apis/${apiId}/rules`)
}

export function createRule(data: { name: string; match_json: unknown; response_status?: number; response_body_json?: unknown; priority?: number; api_id: string }): Promise<MockRule> {
  const params = new URLSearchParams({ api_id: data.api_id })
  return api.post<any, MockRule>(`/rules?${params}`, data)
}

export function getRule(rid: string): Promise<MockRule> {
  return api.get<any, MockRule>(`/rules/${rid}`)
}

export function updateRule(rid: string, data: Record<string, unknown>): Promise<MockRule> {
  return api.patch<any, MockRule>(`/rules/${rid}`, data)
}

export function deleteRule(rid: string): Promise<void> {
  return api.delete(`/rules/${rid}`)
}
