import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  intakeApi, parsingApi,
  ApiError, type Asset,
} from '@/lib/api-client'
import AssetList from '@/components/intake/AssetList'
import DeleteConfirm from '@/components/intake/DeleteConfirm'
import UploadDialog from '@/components/intake/UploadDialog'
import GitRepoDialog from '@/components/intake/GitRepoDialog'
import NoteDialog from '@/components/intake/NoteDialog'
import { useProjectData } from '@/hooks/useProjectData'
import { Button } from '@/components/ui/button'
import { Alert } from '@/components/ui/alert'
import { FullPageState } from '@/components/ui/full-page-state'

export default function ProjectDetail() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const pid = Number(projectId)

  const { project, assets, loading, error, reload } = useProjectData(pid)

  // UI state
  const [assetActionError, setAssetActionError] = useState('')
  const [uploadOpen, setUploadOpen] = useState(false)
  const [gitOpen, setGitOpen] = useState(false)
  const [noteOpen, setNoteOpen] = useState(false)
  const [editingNote, setEditingNote] = useState<Asset | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<{ type: 'project' | 'asset'; id: number } | null>(null)
  const [deleteError, setDeleteError] = useState('')
  const [deleting, setDeleting] = useState(false)

  // --- Intake handlers ---
  const handleUpload = async (file: File, replaceAssetId?: number) => {
    setAssetActionError('')
    try {
      await intakeApi.uploadFile(pid, file, replaceAssetId)
      await parsingApi.parseProject(pid)
    } catch (uploadError) {
      setAssetActionError(uploadError instanceof Error ? uploadError.message : '上传或解析失败')
    } finally {
      reload()
    }
  }

  const handleCreateGit = async (repoUrl: string) => {
    setAssetActionError('')
    try {
      await intakeApi.createGitRepo(pid, repoUrl)
      await parsingApi.parseProject(pid)
    } catch (createGitError) {
      setAssetActionError(createGitError instanceof Error ? createGitError.message : '创建 Git 素材失败')
    } finally {
      reload()
    }
  }

  const handleCreateNote = async (content: string, label: string) => {
    setAssetActionError('')
    try {
      await intakeApi.createNote(pid, content, label)
    } catch (createNoteError) {
      setAssetActionError(createNoteError instanceof Error ? createNoteError.message : '创建备注失败')
    } finally {
      reload()
    }
  }

  const handleUpdateNote = async (content: string, label: string) => {
    if (!editingNote) return
    setAssetActionError('')
    try {
      await intakeApi.updateNote(editingNote.id, content, label)
      setEditingNote(null)
    } catch (updateNoteError) {
      setAssetActionError(updateNoteError instanceof Error ? updateNoteError.message : '更新备注失败')
    } finally {
      reload()
    }
  }

  const handleDeleteAsset = async () => {
    if (!deleteTarget || deleteTarget.type !== 'asset') return
    try {
      setDeleting(true)
      setDeleteError('')
      await intakeApi.deleteAsset(deleteTarget.id)
      reload()
    } catch (deleteError) {
      setDeleteError(deleteError instanceof ApiError ? deleteError.message : '删除失败')
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
    } catch (err) {
      setDeleteError(err instanceof ApiError ? err.message : '删除失败')
    } finally {
      setDeleting(false)
      setDeleteTarget(null)
    }
  }

  const handleEditNote = (asset: Asset) => {
    setEditingNote(asset)
    setNoteOpen(true)
  }

  // --- Loading / error states ---
  if (loading) {
    return <FullPageState variant="loading" />
  }

  if (!project) {
    return <FullPageState variant="error" message={error || '项目不存在'} />
  }

  // --- Intake content (rendered inside left panel) ---
  const intakeContent = (
    <div className="max-w-2xl mx-auto px-6 py-10">
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
        <Button
          variant="danger"
          size="sm"
          className="shrink-0 ml-4"
          onClick={() => setDeleteTarget({ type: 'project', id: pid })}
        >
          删除项目
        </Button>
      </div>

      {error && (
        <Alert className="mb-4">{error}</Alert>
      )}
      {assetActionError && (
        <Alert className="mb-4">{assetActionError}</Alert>
      )}
      {deleteError && (
        <Alert className="mb-4">删除失败：{deleteError}</Alert>
      )}

      <div className="flex gap-2 mb-5">
        <Button
          size="md"
          className="h-9"
          onClick={() => setUploadOpen(true)}
        >
          上传文件
        </Button>
        <Button
          variant="secondary"
          size="md"
          className="h-9"
          onClick={() => setGitOpen(true)}
        >
          接入 Git
        </Button>
        <Button
          variant="secondary"
          size="md"
          className="h-9"
          onClick={() => { setEditingNote(null); setNoteOpen(true) }}
        >
          添加备注
        </Button>
      </div>

      <AssetList
        assets={assets}
        onDelete={(id) => setDeleteTarget({ type: 'asset', id })}
        onEditAsset={handleEditNote}
        canEditAsset={(asset) => asset.type === 'note'}
      />

      {assets.length > 0 && (
        <Button
          size="lg"
          className="mt-6 w-full h-11"
          onClick={() => navigate(`/projects/${pid}/edit`)}
        >
          {project.current_draft_id ? '继续编辑' : '开始编辑'}
        </Button>
      )}

      <UploadDialog
        open={uploadOpen}
        onClose={() => setUploadOpen(false)}
        onUpload={handleUpload}
        existingAssets={assets}
      />
      <GitRepoDialog open={gitOpen} onClose={() => setGitOpen(false)} onSubmit={handleCreateGit} />
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
  )

  return (
    <div className="h-screen bg-background overflow-y-auto">
      {intakeContent}
    </div>
  )
}
