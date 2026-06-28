export interface API {
  id: string
  project_id: string
  name: string
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE' | 'HEAD' | 'OPTIONS'
  path: string
  operation_id: string
  tags_json?: string
  source: 'openapi' | 'manual'
  deprecated: boolean
  created_at: string
  updated_at: string
}
