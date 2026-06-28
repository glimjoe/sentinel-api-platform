export interface Project {
  id: string
  name: string
  slug: string
  owner_id: string
  description: string
  default_base_url: string
  created_at: string
  updated_at: string
}

export interface ProjectMember {
  project_id: string
  user_id: string
  role: 'admin' | 'engineer' | 'viewer'
  email?: string
  display_name?: string
}
