import { api } from './client'
import type { AttributionResult, CompleteResponse, PriorityItem, BudgetInfo } from '@/types/ai'

export function attributeResult(pid: string, resultJson: string): Promise<AttributionResult> {
  return api.post<any, AttributionResult>(`/projects/${pid}/ai/attribution`, { result_json: resultJson })
}

export function generateCases(pid: string, apiId?: string): Promise<CompleteResponse> {
  return api.post<any, CompleteResponse>(`/projects/${pid}/ai/complete`, { api_id: apiId || undefined })
}

export function prioritizeCases(pid: string, caseIds?: string[]): Promise<{ priorities: PriorityItem[] }> {
  return api.post<any, { priorities: PriorityItem[] }>(`/projects/${pid}/ai/prioritize`, {
    case_ids: caseIds || undefined,
  })
}

export function getBudget(): Promise<BudgetInfo> {
  return api.get<any, BudgetInfo>('/ai/budget')
}
