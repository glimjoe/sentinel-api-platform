export interface MockRule {
  id: string
  api_id: string
  name: string
  match_json: unknown
  response_status: number
  response_headers_json?: unknown
  response_body_json?: unknown
  priority: number
  delay_ms: number
  enabled: boolean
  hit_count: number
  created_at: string
  updated_at: string
}
