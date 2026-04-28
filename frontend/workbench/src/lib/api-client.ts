const BASE = '/api/v1'

const USER_ID_KEY = 'rg_user_id'

function getUserId(): string {
  let id = localStorage.getItem(USER_ID_KEY)
  if (!id) {
    id = crypto.randomUUID()
    localStorage.setItem(USER_ID_KEY, id)
  }
  return id
}

function userHeaders(): Record<string, string> {
  return { 'X-User-ID': getUserId() }
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...userHeaders(),
    },
    ...options,
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
    headers: userHeaders(),
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
