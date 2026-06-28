import { api } from './client'
import type { API } from '@/types/api'

export function listAPIs(pid: string): Promise<API[]> {
  return api.get<any, API[]>(`/projects/${pid}/apis`)
}

export function createAPI(pid: string, data: { name: string; method: string; path: string; operation_id?: string }): Promise<API> {
  return api.post<any, API>(`/projects/${pid}/apis`, data)
}

export function getAPI(pid: string, apiId: string): Promise<API> {
  return api.get<any, API>(`/projects/${pid}/apis/${apiId}`)
}

export function updateAPI(pid: string, apiId: string, data: Record<string, unknown>): Promise<API> {
  return api.patch<any, API>(`/projects/${pid}/apis/${apiId}`, data)
}

export function deleteAPI(pid: string, apiId: string): Promise<void> {
  return api.delete(`/projects/${pid}/apis/${apiId}`)
}

export function importOpenAPI(pid: string, spec: string): Promise<{ imported: number; total: number }> {
  return api.post<any, any>(`/projects/${pid}/apis/import-openapi`, { spec })
}
