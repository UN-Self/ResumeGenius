import { useMemo, useState } from 'react'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { intakeApi, parsingApi, type Asset } from '@/lib/api-client'
import AssetList from './AssetList'
import AssetEditorDialog from './AssetEditorDialog'
import DeleteConfirm from './DeleteConfirm'
import GitRepoDialog from './GitRepoDialog'
import NoteDialog from './NoteDialog'
import UploadDialog from './UploadDialog'

interface AssetSidebarProps {
  projectId: number
  assets: Asset[]
  onReload: () => Promise<void>
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function isDerivedImageAsset(asset: Asset) {
  if (asset.type !== 'resume_image' || !isRecord(asset.metadata)) {
    return false
  }

  const parsing = asset.metadata.parsing
  if (!isRecord(parsing)) {
    return false
  }

  return parsing.derived === true
}

function canEditAsset(asset: Asset) {
  return asset.type !== 'resume_image'
}

export default function AssetSidebar({ projectId, assets, onReload }: AssetSidebarProps) {
  const [error, setError] = useState('')
  const [uploadOpen, setUploadOpen] = useState(false)
  const [gitOpen, setGitOpen] = useState(false)
  const [noteOpen, setNoteOpen] = useState(false)
  const [editingAsset, setEditingAsset] = useState<Asset | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Asset | null>(null)
  const [deleting, setDeleting] = useState(false)

  const visibleAssets = useMemo(
    () => assets.filter((asset) => !isDerivedImageAsset(asset)),
    [assets]
  )

  const refreshAssets = async () => {
    await onReload()
  }

  const handleUpload = async (file: File) => {
    setError('')
    try {
      await intakeApi.uploadFile(projectId, file)
      await parsingApi.parseProject(projectId)
    } finally {
      await refreshAssets()
    }
  }

  const handleCreateGit = async (repoUrl: string) => {
    setError('')
    try {
      await intakeApi.createGitRepo(projectId, repoUrl)
      await parsingApi.parseProject(projectId)
    } finally {
      await refreshAssets()
    }
  }

  const handleCreateNote = async (content: string, label: string) => {
    setError('')
    await intakeApi.createNote(projectId, content, label)
    await refreshAssets()
  }

  const handleUpdateAsset = async (content: string, label: string) => {
    if (!editingAsset) return
    setError('')
    await intakeApi.updateAsset(editingAsset.id, { content, label })
    setEditingAsset(null)
    await refreshAssets()
  }

  const handleDeleteAsset = async () => {
    if (!deleteTarget) return
    try {
      setDeleting(true)
      setError('')
      await intakeApi.deleteAsset(deleteTarget.id)
      await refreshAssets()
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : '\u5220\u9664\u5931\u8d25')
    } finally {
      setDeleting(false)
      setDeleteTarget(null)
    }
  }

  return (
    <div className="h-full overflow-y-auto p-4">
      <div className="flex flex-wrap gap-2">
        <Button size="sm" onClick={() => setUploadOpen(true)}>
          {'\u4e0a\u4f20\u6587\u4ef6'}
        </Button>
        <Button size="sm" variant="secondary" onClick={() => setGitOpen(true)}>
          {'\u63a5\u5165 Git'}
        </Button>
        <Button size="sm" variant="secondary" onClick={() => setNoteOpen(true)}>
          {'\u6dfb\u52a0\u5907\u6ce8'}
        </Button>
      </div>

      {error && (
        <Alert className="mt-3">{error}</Alert>
      )}

      <div className="mt-4">
        <AssetList
          assets={visibleAssets}
          onDelete={(id) => {
            const target = visibleAssets.find((asset) => asset.id === id) ?? null
            setDeleteTarget(target)
          }}
          onEditAsset={(asset) => setEditingAsset(asset)}
          canEditAsset={canEditAsset}
        />
      </div>

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
        onClose={() => setNoteOpen(false)}
        onSubmit={handleCreateNote}
      />
      <AssetEditorDialog
        open={editingAsset !== null}
        asset={editingAsset}
        onClose={() => setEditingAsset(null)}
        onSubmit={handleUpdateAsset}
      />
      <DeleteConfirm
        open={deleteTarget !== null}
        title="删除素材"
        message="删除后该素材将被永久删除，此操作不可撤销。"
        onConfirm={handleDeleteAsset}
        onCancel={() => setDeleteTarget(null)}
        loading={deleting}
      />
    </div>
  )
}
