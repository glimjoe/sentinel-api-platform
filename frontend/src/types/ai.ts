export interface AttributionResult {
  analysis: string
  root_cause: string
  confidence: number
  suggested_fix?: string
}

export interface GeneratedCase {
  name: string
  method: string
  path: string
  expected_status: number
  expected_body_match?: string
  expected_body_pattern?: string
}

export interface CompleteResponse {
  test_cases: GeneratedCase[]
}

export interface PriorityItem {
  case_id: string
  priority: 'p0' | 'p1' | 'p2' | 'p3'
  reasoning: string
}

export interface BudgetInfo {
  enabled: boolean
  daily: { used: number; limit: number }
  monthly: { used: number; limit: number }
}
