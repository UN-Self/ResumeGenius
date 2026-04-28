import { describe, it, expect, vi, beforeEach } from 'vitest'
import { intakeApi, ApiError } from '@/lib/api-client'

describe('apiClient', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
  })

  it('sends X-User-ID header on every request', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: [] }),
    })
    vi.stubGlobal('fetch', mock)

    await intakeApi.listProjects()

    const callArgs = mock.mock.calls[0]
    expect(callArgs[1].headers['X-User-ID']).toBeDefined()
  })

  it('persists the same user ID across requests', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: [] }),
    })
    vi.stubGlobal('fetch', mock)

    await intakeApi.listProjects()
    const firstId = mock.mock.calls[0][1].headers['X-User-ID']

    await intakeApi.listProjects()
    const secondId = mock.mock.calls[1][1].headers['X-User-ID']

    expect(firstId).toBe(secondId)
  })

  it('listProjects calls GET /projects', async () => {
    const projects = [{ id: 1, title: 'test' }]
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: projects }),
    })
    vi.stubGlobal('fetch', mock)

    const result = await intakeApi.listProjects()
    expect(mock).toHaveBeenCalledWith('/api/v1/projects', expect.objectContaining({
      headers: expect.objectContaining({ 'X-User-ID': expect.any(String) }),
    }))
    expect(result).toEqual(projects)
  })

  it('createProject calls POST /projects', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: { id: 1, title: 'test' } }),
    })
    vi.stubGlobal('fetch', mock)

    const result = await intakeApi.createProject('test')
    expect(mock).toHaveBeenCalledWith('/api/v1/projects', expect.objectContaining({
      method: 'POST',
    }))
    expect(result.title).toBe('test')
  })

  it('getProject calls GET /projects/:id', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: { id: 1, title: 'test' } }),
    })
    vi.stubGlobal('fetch', mock)

    const result = await intakeApi.getProject(1)
    expect(mock).toHaveBeenCalledWith('/api/v1/projects/1', expect.objectContaining({
      headers: expect.objectContaining({ 'X-User-ID': expect.any(String) }),
    }))
    expect(result.id).toBe(1)
  })

  it('deleteProject calls DELETE /projects/:id', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: null }),
    })
    vi.stubGlobal('fetch', mock)

    await intakeApi.deleteProject(1)
    expect(mock).toHaveBeenCalledWith('/api/v1/projects/1', expect.objectContaining({ method: 'DELETE' }))
  })

  it('uploadFile sends multipart/form-data', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: { id: 1, type: 'resume_pdf' } }),
    })
    vi.stubGlobal('fetch', mock)

    const file = new File(['pdf-content'], 'resume.pdf', { type: 'application/pdf' })
    await intakeApi.uploadFile(1, file)

    expect(mock).toHaveBeenCalledWith('/api/v1/assets/upload', expect.objectContaining({
      method: 'POST',
    }))
    const body = mock.mock.calls[0][1].body
    expect(body).toBeInstanceOf(FormData)
    expect(body.get('project_id')).toBe('1')
    expect(body.get('file')).toBe(file)
  })

  it('createGitRepo calls POST /assets/git', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: { id: 1, type: 'git_repo' } }),
    })
    vi.stubGlobal('fetch', mock)

    const result = await intakeApi.createGitRepo(1, 'https://github.com/user/repo')
    expect(mock).toHaveBeenCalledWith('/api/v1/assets/git', expect.objectContaining({
      method: 'POST',
    }))
    expect(result.type).toBe('git_repo')
  })

  it('listAssets calls GET /assets?project_id=X', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: [] }),
    })
    vi.stubGlobal('fetch', mock)

    await intakeApi.listAssets(1)
    expect(mock).toHaveBeenCalledWith('/api/v1/assets?project_id=1', expect.objectContaining({
      headers: expect.objectContaining({ 'X-User-ID': expect.any(String) }),
    }))
  })

  it('deleteAsset calls DELETE /assets/:id', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: null }),
    })
    vi.stubGlobal('fetch', mock)

    await intakeApi.deleteAsset(1)
    expect(mock).toHaveBeenCalledWith('/api/v1/assets/1', expect.objectContaining({ method: 'DELETE' }))
  })

  it('createNote calls POST /assets/notes', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: { id: 1, type: 'note' } }),
    })
    vi.stubGlobal('fetch', mock)

    const result = await intakeApi.createNote(1, 'content', 'label')
    expect(mock).toHaveBeenCalledWith('/api/v1/assets/notes', expect.objectContaining({ method: 'POST' }))
    expect(result.type).toBe('note')
  })

  it('updateNote calls PUT /assets/notes/:id', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 0, data: { id: 1, type: 'note', content: 'updated' } }),
    })
    vi.stubGlobal('fetch', mock)

    const result = await intakeApi.updateNote(1, 'updated', 'label')
    expect(mock).toHaveBeenCalledWith('/api/v1/assets/notes/1', expect.objectContaining({ method: 'PUT' }))
    expect(result.content).toBe('updated')
  })

  it('throws ApiError on non-zero code', async () => {
    const mock = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ code: 1004, message: 'project not found' }),
    })
    vi.stubGlobal('fetch', mock)

    await expect(intakeApi.getProject(999)).rejects.toThrow(ApiError)
    await expect(intakeApi.getProject(999)).rejects.toMatchObject({
      message: 'project not found',
    })
  })

  it('ApiError has code property', () => {
    const err = new ApiError(1004, 'project not found')
    expect(err.code).toBe(1004)
    expect(err.message).toBe('project not found')
  })
})
