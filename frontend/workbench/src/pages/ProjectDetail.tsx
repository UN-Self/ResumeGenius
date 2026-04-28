import { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { intakeApi, ApiError, type Project, type Asset } from '@/lib/api-client'
import AssetList from '@/components/intake/AssetList'
import DeleteConfirm from '@/components/intake/DeleteConfirm'
import UploadDialog from '@/components/intake/UploadDialog'
import GitRepoDialog from '@/components/intake/GitRepoDialog'
import NoteDialog from '@/components/intake/NoteDialog'

export default function ProjectDetail() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const pid = Number(projectId)

  const [project, setProject] = useState<Project | null>(null)
  const [assets, setAssets] = useState<Asset[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  // Dialogs
  const [uploadOpen, setUploadOpen] = useState(false)
  const [gitOpen, setGitOpen] = useState(false)
  const [noteOpen, setNoteOpen] = useState(false)
  const [editingNote, setEditingNote] = useState<Asset | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<{ type: 'project' | 'asset'; id: number } | null>(null)
  const [deleting, setDeleting] = useState(false)

  const load = useCallback(async () => {
    try {
      setLoading(true)
      const [proj, asts] = await Promise.all([
        intakeApi.getProject(pid),
        intakeApi.listAssets(pid),
      ])
      setProject(proj)
      setAssets(asts)
      setError('')
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }, [pid])

  useEffect(() => { load() }, [load])

  const handleUpload = async (file: File) => {
    await intakeApi.uploadFile(pid, file)
    await load()
  }

  const handleCreateGit = async (repoUrl: string) => {
    await intakeApi.createGitRepo(pid, repoUrl)
    await load()
  }

  const handleCreateNote = async (content: string, label: string) => {
    await intakeApi.createNote(pid, content, label)
    await load()
  }

  const handleUpdateNote = async (content: string, label: string) => {
    if (!editingNote) return
    await intakeApi.updateNote(editingNote.id, content, label)
    setEditingNote(null)
    await load()
  }

  const handleDeleteAsset = async () => {
    if (!deleteTarget || deleteTarget.type !== 'asset') return
    try {
      setDeleting(true)
      await intakeApi.deleteAsset(deleteTarget.id)
      await load()
    } finally {
      setDeleting(false)
      setDeleteTarget(null)
    }
  }

  const handleDeleteProject = async () => {
    if (!deleteTarget || deleteTarget.type !== 'project') return
    try {
      setDeleting(true)
      await intakeApi.deleteProject(pid)
      navigate('/')
    } catch {
      setError('删除失败')
    } finally {
      setDeleting(false)
      setDeleteTarget(null)
    }
  }

  const handleEditNote = (asset: Asset) => {
    setEditingNote(asset)
    setNoteOpen(true)
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <p className="text-muted-foreground text-sm">加载中...</p>
      </div>
    )
  }

  if (!project) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <p className="text-destructive text-sm">{error || '项目不存在'}</p>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-2xl mx-auto px-6 py-10">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div className="min-w-0 flex-1">
            <button
              onClick={() => navigate('/')}
              className="text-xs text-muted-foreground hover:text-foreground transition-colors mb-1"
            >
              &larr; 返回项目列表
            </button>
            <h1 className="font-serif text-xl font-semibold text-foreground truncate">
              {project.title}
            </h1>
          </div>
          <button
            onClick={() => setDeleteTarget({ type: 'project', id: pid })}
            className="shrink-0 ml-4 text-xs text-muted-foreground hover:text-destructive px-3 py-1.5 rounded-lg border border-border hover:border-destructive/30 transition-colors"
          >
            删除项目
          </button>
        </div>

        {error && (
          <div className="mb-4 px-4 py-2.5 text-sm rounded-lg bg-destructive/10 text-destructive border border-destructive/20">
            {error}
          </div>
        )}

        {/* Action buttons */}
        <div className="flex gap-2 mb-5">
          <button
            onClick={() => setUploadOpen(true)}
            className="h-9 px-4 text-sm font-medium rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
          >
            上传文件
          </button>
          <button
            onClick={() => setGitOpen(true)}
            className="h-9 px-4 text-sm font-medium rounded-lg border border-border bg-card text-foreground hover:bg-accent transition-colors"
          >
            接入 Git
          </button>
          <button
            onClick={() => { setEditingNote(null); setNoteOpen(true) }}
            className="h-9 px-4 text-sm font-medium rounded-lg border border-border bg-card text-foreground hover:bg-accent transition-colors"
          >
            添加备注
          </button>
        </div>

        {/* Asset list */}
        <AssetList
          assets={assets}
          onDelete={(id) => setDeleteTarget({ type: 'asset', id })}
          onEditNote={handleEditNote}
        />

        {/* Dialogs */}
        <UploadDialog
          open={uploadOpen}
          onClose={() => setUploadOpen(false)}
          onUpload={handleUpload}
        />
        <GitRepoDialog
          open={gitOpen}
          onClose={() => setGitOpen(false)}
          onSubmit={handleCreateGit}
        />
        <NoteDialog
          open={noteOpen}
          onClose={() => { setNoteOpen(false); setEditingNote(null) }}
          onSubmit={editingNote ? handleUpdateNote : handleCreateNote}
          initialNote={editingNote ?? undefined}
        />
        <DeleteConfirm
          open={deleteTarget !== null}
          title={deleteTarget?.type === 'project' ? '删除项目' : '删除资料'}
          message={deleteTarget?.type === 'project'
            ? '删除后该项目下的所有资料和文件将被永久删除，此操作不可撤销。'
            : '删除后该资料将被永久删除，此操作不可撤销。'}
          onConfirm={deleteTarget?.type === 'project' ? handleDeleteProject : handleDeleteAsset}
          onCancel={() => setDeleteTarget(null)}
          loading={deleting}
        />
      </div>
    </div>
  )
}
