export interface TestCase {
  id: string
  project_id: string
  api_id?: string
  name: string
  description?: string
  method: string
  path: string
  headers_json?: unknown
  query_json?: unknown
  body_json?: unknown
  expected_status: number
  expected_body_json?: unknown
  expected_body_match: 'exact' | 'contains' | 'jsonpath' | 'schema' | 'regex' | 'none'
  expected_body_pattern?: string
  priority: 'p0' | 'p1' | 'p2' | 'p3'
  created_at: string
}
