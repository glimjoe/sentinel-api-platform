export interface TestRun {
  id: string
  project_id: string
  name: string
  status: 'queued' | 'running' | 'success' | 'failed' | 'cancelled' | 'partial'
  mode: 'sequential' | 'parallel'
  target_base_url: string
  total: number
  passed: number
  failed: number
  errored: number
  started_at?: string
  finished_at?: string
  created_at: string
}

export interface TestResult {
  id: string
  run_id: string
  case_id: string
  status: 'pass' | 'fail' | 'error' | 'skip'
  actual_status?: number
  duration_ms: number
  error_msg?: string
}

export interface RunEvent {
  type: 'progress' | 'complete' | 'heartbeat'
  run_id: string
  total: number
  passed: number
  failed: number
  errored: number
  status?: string
  ts: number
}
