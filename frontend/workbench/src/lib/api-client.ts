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
  // Canonical asset body for AI / sidebar consumption.
  // Notes write it directly; file and git assets are expected to be backfilled by parsing after cleanup.
  content?: string
  label?: string
  metadata?: Record<string, unknown>
  created_at: string
}

export interface User {
  id: string
  username: string
}

export interface AuthUser {
  id: string
  username: string
  email?: string
  email_verified?: boolean
  avatar_url?: string
  points: number
  plan: string
  plan_started_at?: string
  plan_expires_at?: string
}

export interface PointsRecord {
  id: number
  user_id: string
  amount: number
  balance: number
  type: string
  note: string
  created_at: string
}

export interface PointsStats {
  balance: number
  month_used: number
  total_earned: number
}

export interface DailyUsage {
  date: string
  used: number
  earned: number
}

export interface CategoryUsage {
  type: string
  total: number
}

export interface PointsDashboard {
  balance: number
  month_used: number
  total_earned: number
  daily_usage: DailyUsage[]
  categories: CategoryUsage[]
}

export interface CheckAvailability {
  available: boolean
}

export const authApi = {
  login: (username: string, password: string) =>
    request<User>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  me: () => request<User>('/auth/me'),
  logout: () => request<null>('/auth/logout', { method: 'POST' }),
  register: (username: string, password: string, email: string) =>
    request<AuthUser>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username, password, email }),
    }),
  sendCode: (email: string) =>
    request<{ dev_code?: string } | null>('/auth/send-code', {
      method: 'POST',
      body: JSON.stringify({ email }),
    }),
  verifyEmail: (email: string, code: string) =>
    request<AuthUser>('/auth/verify-email', {
      method: 'POST',
      body: JSON.stringify({ email, code }),
    }),
  checkUsername: (q: string) =>
    request<CheckAvailability>(`/auth/check-username?q=${encodeURIComponent(q)}`),
  checkEmail: (q: string) =>
    request<CheckAvailability>(`/auth/check-email?q=${encodeURIComponent(q)}`),
  updateProfile: (nickname: string) =>
    request<AuthUser>('/auth/profile', {
      method: 'PUT',
      body: JSON.stringify({ nickname }),
    }),
  changePassword: (old_password: string, new_password: string) =>
    request<null>('/auth/password', {
      method: 'PUT',
      body: JSON.stringify({ old_password, new_password }),
    }),
  uploadAvatar: (file: File) => {
    const fd = new FormData()
    fd.append('avatar', file)
    return upload<AuthUser>('/auth/avatar', fd)
  },
  getPointsRecords: () =>
    request<{ items: PointsRecord[] }>('/auth/points/records'),
  getPointsStats: () =>
    request<PointsStats>('/auth/points/stats'),
  getPointsDashboard: () =>
    request<PointsDashboard>('/auth/points/dashboard'),
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
  uploadFile: (projectId: number, file: File, replaceAssetId?: number, folderId?: number | null) => {
    const fd = new FormData()
    fd.append('file', file)
    fd.append('project_id', String(projectId))
    if (replaceAssetId !== undefined) {
      fd.append('replace_asset_id', String(replaceAssetId))
    }
    if (folderId !== undefined && folderId !== null) {
      fd.append('folder_id', String(folderId))
    }
    return upload<Asset>('/assets/upload', fd)
  },
  createFolder: (projectId: number, name: string, parentFolderId?: number | null) =>
    request<Asset>('/assets/folders', {
      method: 'POST',
      body: JSON.stringify({ project_id: projectId, name, parent_folder_id: parentFolderId ?? null }),
    }),
  createGitRepo: (projectId: number, repoUrls: string[]) =>
    request<Asset[]>('/assets/git', {
      method: 'POST',
      body: JSON.stringify({ project_id: projectId, repo_urls: repoUrls }),
    }),
  updateAsset: (id: number, payload: { content?: string; label?: string }) =>
    request<Asset>(`/assets/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(payload),
    }),
  deleteAsset: (id: number) =>
    request<null>(`/assets/${id}`, { method: 'DELETE' }),
  moveAsset: (id: number, targetFolderId: number | null) =>
    request<Asset>(`/assets/${id}/move`, {
      method: 'PATCH',
      body: JSON.stringify({ target_folder_id: targetFolderId }),
    }),

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
  // Temporary parse preview payload. The canonical persisted body should live in assets.content.
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
  thinking?: string
  created_at: string
}

export interface ToolCallEntry {
  name: string
  status: 'running' | 'completed' | 'failed'
  params?: Record<string, unknown>
  result?: string
  created_at?: string
}

export async function undoDraft(draftId: number) {
  return request<{ html_content: string }>(`/ai/drafts/${draftId}/undo`, { method: 'POST' })
}

export async function redoDraft(draftId: number) {
  return request<{ html_content: string }>(`/ai/drafts/${draftId}/redo`, { method: 'POST' })
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
    request<{ items: AIMessageItem[]; tool_calls?: ToolCallEntry[] }>(`/ai/sessions/${sessionId}/history`),
}

export const parsingApi = {
  parseProject: (projectId: number) =>
    request<{ parsed_contents: ParsedContent[] }>('/parsing/parse', {
      method: 'POST',
      body: JSON.stringify({ project_id: projectId }),
    }),
  parseAsset: (assetId: number) =>
    request<ParsedContent>(`/parsing/assets/${assetId}/parse`, {
      method: 'POST',
    }),
}

// --- Render API ---

export interface Version {
  id: number
  label: string
  created_at: string
}

export interface VersionDetail extends Version {
  html_snapshot: string
}

export const renderApi = {
  listVersions: (draftId: number) =>
    request<{ items: Version[]; total: number }>(`/drafts/${draftId}/versions`),
  getVersion: (draftId: number, versionId: number) =>
    request<VersionDetail>(`/drafts/${draftId}/versions/${versionId}`),
  createVersion: (draftId: number, label: string) =>
    request<Version>(`/drafts/${draftId}/versions`, {
      method: 'POST',
      body: JSON.stringify({ label }),
    }),
  rollback: (draftId: number, versionId: number) =>
    request<{
      draft_id: number
      updated_at: string
      new_version_id: number
      new_version_label: string
      new_version_created_at: string
    }>(`/drafts/${draftId}/rollback`, {
      method: 'POST',
      body: JSON.stringify({ version_id: versionId }),
    }),
}
