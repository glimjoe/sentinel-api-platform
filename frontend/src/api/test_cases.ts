import { api } from './client'
import type { TestCase } from '@/types/test_case'

export function listCases(pid: string): Promise<TestCase[]> {
  return api.get<any, TestCase[]>(`/projects/${pid}/cases`)
}

export function createCase(pid: string, data: Partial<TestCase>): Promise<TestCase> {
  return api.post<any, TestCase>(`/projects/${pid}/cases`, data)
}

export function deleteCase(pid: string, caseId: string): Promise<void> {
  return api.delete(`/projects/${pid}/cases/${caseId}`)
}

export function getCase(pid: string, caseId: string): Promise<TestCase> {
  return api.get<any, TestCase>(`/projects/${pid}/cases/${caseId}`)
}

export function updateCase(pid: string, caseId: string, data: Record<string, unknown>): Promise<TestCase> {
  return api.patch<any, TestCase>(`/projects/${pid}/cases/${caseId}`, data)
}
