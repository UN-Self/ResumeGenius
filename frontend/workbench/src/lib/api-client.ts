const BASE = '/api/v1'

export async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const mergedHeaders: HeadersInit = {
    'Content-Type': 'application/json',
    ...((options?.headers as Record<string, string>) ?? {}),
  }

  const res = await fetch(`${BASE}${path}`, {
    credentials: 'include',
    ...options,
    headers: mergedHeaders,
  })
  const json = await res.json()
  if (json.code !== 0) {
    throw new ApiError(json.code, json.message)
  }
  return json.data as T
}

async function upload<T>(path: string, formData: FormData): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method: 'POST',
    credentials: 'include',
    body: formData,
  })
  const json = await res.json()
  if (json.code !== 0) {
    throw new ApiError(json.code, json.message)
  }
  return json.data as T
}

export class ApiError extends Error {
  code: number
  constructor(code: number, message: string) {
    super(message)
    this.code = code
  }
}

// --- Types ---

export interface Project {
  id: number
  title: string
  status: string
  current_draft_id: number | null
  created_at: string
}

export interface Asset {
  id: number
  project_id: number
  type: string
  uri?: string
  content?: string
  label?: string
  created_at: string
}

export interface User {
  id: string
  username: string
}

export const authApi = {
  login: (username: string, password: string) =>
    request<User>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  me: () => request<User>('/auth/me'),
  logout: () => request<null>('/auth/logout', { method: 'POST' }),
}

// --- Intake API ---

export const intakeApi = {
  // Projects
  listProjects: () => request<Project[]>('/projects'),
  createProject: (title: string) =>
    request<Project>('/projects', { method: 'POST', body: JSON.stringify({ title }) }),
  getProject: (id: number) => request<Project>(`/projects/${id}`),
  deleteProject: (id: number) =>
    request<null>(`/projects/${id}`, { method: 'DELETE' }),

  // Assets
  listAssets: (projectId: number) =>
    request<Asset[]>(`/assets?project_id=${projectId}`),
  uploadFile: (projectId: number, file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    fd.append('project_id', String(projectId))
    return upload<Asset>('/assets/upload', fd)
  },
  createGitRepo: (projectId: number, repoUrl: string) =>
    request<Asset>('/assets/git', {
      method: 'POST',
      body: JSON.stringify({ project_id: projectId, repo_url: repoUrl }),
    }),
  deleteAsset: (id: number) =>
    request<null>(`/assets/${id}`, { method: 'DELETE' }),

  // Notes
  createNote: (projectId: number, content: string, label: string) =>
    request<Asset>('/assets/notes', {
      method: 'POST',
      body: JSON.stringify({ project_id: projectId, content, label }),
    }),
  updateNote: (id: number, content: string, label: string) =>
    request<Asset>(`/assets/notes/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ content, label }),
    }),
}

// --- Workbench API ---

import type { Draft } from '@/types/editor'

export type { Draft }

export const workbenchApi = {
  getDraft: (id: number) => request<Draft>(`/drafts/${id}`),
  createDraft: (projectId: number) =>
    request<Draft>('/drafts', {
      method: 'POST',
      body: JSON.stringify({ project_id: projectId }),
    }),
}

// --- Parsing API ---

export interface ParsedImage {
  description: string
  data_base64: string
}

export interface ParsedContent {
  asset_id: number
  type: string
  label: string
  text: string
  images?: ParsedImage[]
}

// --- Agent API ---

export interface AISession {
  id: number
  draft_id: number
  created_at: string
}

export interface AIMessageItem {
  id: number
  role: 'user' | 'assistant'
  content: string
  created_at: string
}

export const agentApi = {
  listSessions: (draftId: number) =>
    request<AISession[]>(`/ai/sessions?draft_id=${draftId}`),
  createSession: (draftId: number) =>
    request<AISession>('/ai/sessions', {
      method: 'POST',
      body: JSON.stringify({ draft_id: draftId }),
    }),
  getHistory: (sessionId: number) =>
    request<{ items: AIMessageItem[] }>(`/ai/sessions/${sessionId}/history`),
}

export const parsingApi = {
  parseProject: (projectId: number) =>
    request<{ parsed_contents: ParsedContent[] }>('/parsing/parse', {
      method: 'POST',
      body: JSON.stringify({ project_id: projectId }),
    }),
}
