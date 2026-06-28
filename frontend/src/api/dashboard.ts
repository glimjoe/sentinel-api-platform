import { api } from './client'

export interface DashboardStats {
  projects: number
  apis: number
  mock_rules: number
  test_cases: number
  recent_runs: RunSummary[]
  status_breakdown: Record<string, number>
}

export interface RunSummary {
  id: string
  name: string
  status: string
  passed: number
  failed: number
  errored: number
  skipped: number
  total: number
  created_at: string
}

export function fetchDashboardStats(): Promise<DashboardStats> {
  return api.get('/dashboard') as Promise<DashboardStats>
}
