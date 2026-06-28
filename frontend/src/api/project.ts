import { api } from './client'
import type { Project, ProjectMember } from '@/types/project'

export function listProjects(): Promise<Project[]> {
  return api.get<any, Project[]>('/projects')
}

export function createProject(data: { name: string; description?: string; slug?: string }): Promise<Project> {
  return api.post<any, Project>('/projects', data)
}

export function getProject(pid: string): Promise<Project> {
  return api.get<any, Project>(`/projects/${pid}`)
}

export function updateProject(pid: string, data: Record<string, unknown>): Promise<Project> {
  return api.patch<any, Project>(`/projects/${pid}`, data)
}

export function deleteProject(pid: string): Promise<void> {
  return api.delete(`/projects/${pid}`)
}

export function listMembers(pid: string): Promise<ProjectMember[]> {
  return api.get<any, ProjectMember[]>(`/projects/${pid}/members`)
}
