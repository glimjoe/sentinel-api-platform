import { api } from './client'
import type { TestRun, TestResult, RunEvent } from '@/types/test_run'

export function listRuns(pid: string): Promise<TestRun[]> {
  return api.get<any, TestRun[]>(`/projects/${pid}/runs`)
}

export function createRun(pid: string, data: { name: string; target_base_url: string }): Promise<TestRun> {
  return api.post<any, TestRun>(`/projects/${pid}/runs`, data)
}

export function startRun(pid: string, runId: string): Promise<TestRun> {
  return api.post<any, TestRun>(`/projects/${pid}/runs/${runId}/start`)
}

export function cancelRun(pid: string, runId: string): Promise<void> {
  return api.post(`/projects/${pid}/runs/${runId}/cancel`)
}

export function streamRun(pid: string, runId: string, onEvent: (e: RunEvent) => void): () => void {
  const url = `/api/v1/projects/${pid}/runs/${runId}/stream`
  const es = new EventSource(url)
  es.onmessage = (msg) => {
    try {
      onEvent(JSON.parse(msg.data))
    } catch { /* ignore parse errors */ }
  }
  es.onerror = () => es.close()
  return () => es.close()
}

export function getResults(pid: string, runId: string): Promise<TestResult[]> {
  return api.get<any, TestResult[]>(`/projects/${pid}/runs/${runId}/results`)
}
