import { useEffect, useRef, useState } from 'react'
import type { KeyboardEvent } from 'react'
import { ChevronDown, Folder, Loader2, MoreHorizontal, PencilLine, RefreshCcw, Trash2 } from 'lucide-react'
import type { Asset } from '@/lib/api-client'
import { getAssetVisual, getDisplayAssetTitle, getDisplayFileName, getOriginalFilenameFromAsset } from './fileVisuals'

type AssetItem = Asset & { label?: string; content?: string; uri?: string }

interface AssetListProps {
  assets: AssetItem[]
  onDelete: (id: number) => void
  onOpenAsset?: (asset: AssetItem) => void
  selectedAssetId?: number | null
  onRenameAsset?: (asset: AssetItem, label: string) => Promise<void> | void
  onReparseAsset?: (asset: AssetItem) => void
  canReparseAsset?: (asset: AssetItem) => boolean
  reparseLoadingAssetId?: number | null
  onSelectFolder?: (folderId: number | null) => void
  selectedFolderId?: number | null
  onMoveAsset?: (assetId: number, targetFolderId: number | null) => Promise<void>
}

interface AssetActionButtonProps {
  label: string
  onClick: () => void
  icon: typeof RefreshCcw
  danger?: boolean
  disabled?: boolean
}

function AssetActionButton({
  label,
  onClick,
  icon: Icon,
  danger = false,
  disabled = false,
}: AssetActionButtonProps) {
  return (
    <button
      type="button"
      onClick={(event) => {
        event.stopPropagation()
        onClick()
      }}
      disabled={disabled}
      className={[
        'inline-flex h-7 items-center gap-1 rounded-full border px-2.5 text-[11px] font-medium shadow-sm transition-colors',
        'disabled:pointer-events-none disabled:opacity-50',
        danger
          ? 'border-red-200 bg-white text-muted-foreground hover:bg-red-50 hover:text-red-600'
          : 'border-border bg-white text-muted-foreground hover:border-primary-200 hover:bg-primary-50 hover:text-foreground',
      ].join(' ')}
    >
      <Icon className="h-3.5 w-3.5" />
      <span>{label}</span>
    </button>
  )
}

interface AssetMoreMenuProps {
  asset: AssetItem
  renameable: boolean
  onRename: (asset: AssetItem) => void
  onDelete: (id: number) => void
}

function AssetMoreMenu({
  asset,
  renameable,
  onRename,
  onDelete,
}: AssetMoreMenuProps) {
  const [open, setOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return

    const handlePointerDown = (event: PointerEvent) => {
      const target = event.target as Node
      if (menuRef.current?.contains(target)) return
      setOpen(false)
    }

    document.addEventListener('pointerdown', handlePointerDown)
    return () => document.removeEventListener('pointerdown', handlePointerDown)
  }, [open])

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key !== 'Escape') return
    setOpen(false)
  }

  return (
    <div ref={menuRef} className="relative shrink-0" onKeyDown={handleKeyDown}>
      <button
        type="button"
        aria-label="更多素材操作"
        aria-expanded={open}
        aria-haspopup="menu"
        onClick={(event) => {
          event.stopPropagation()
          setOpen((value) => !value)
        }}
        className={[
          'flex h-8 w-8 items-center justify-center rounded-full border border-transparent text-muted-foreground transition-all',
          'hover:border-border hover:bg-popover/80 hover:text-foreground hover:shadow-[0_10px_28px_rgba(2,8,23,0.18)]',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/45',
          open ? 'border-border bg-popover/90 text-foreground shadow-[0_10px_28px_rgba(2,8,23,0.18)]' : '',
        ].join(' ')}
      >
        <MoreHorizontal className="h-4 w-4" />
      </button>

      {open && (
        <div
          role="menu"
          className="absolute right-0 top-10 z-30 w-36 overflow-hidden rounded-2xl border border-border bg-popover/96 p-1.5 text-popover-foreground shadow-[0_18px_50px_rgba(2,8,23,0.28)] backdrop-blur-2xl"
        >
          <button
            type="button"
            role="menuitem"
            disabled={!renameable}
            onClick={(event) => {
              event.stopPropagation()
              if (!renameable) return
              setOpen(false)
              onRename(asset)
            }}
            className="flex w-full items-center gap-2 rounded-xl px-3 py-2 text-left text-xs font-medium text-muted-foreground transition-colors hover:bg-primary/10 hover:text-foreground disabled:pointer-events-none disabled:opacity-45"
          >
            <PencilLine className="h-3.5 w-3.5" />
            重命名
          </button>

          <button
            type="button"
            role="menuitem"
            onClick={(event) => {
              event.stopPropagation()
              setOpen(false)
              onDelete(asset.id)
            }}
            className="flex w-full items-center gap-2 rounded-xl px-3 py-2 text-left text-xs font-medium text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
          >
            <Trash2 className="h-3.5 w-3.5" />
            删除
          </button>
        </div>
      )}
    </div>
  )
}

export function getDisplayTitle(asset: AssetItem, fallbackLabel: string) {
  if (asset.label?.trim() && !isGenericAssetLabel(asset, asset.label.trim())) {
    return getDisplayAssetTitle(asset.type, asset.label) || asset.label.trim()
  }

  const originalFilename = getOriginalFilenameFromAsset(asset)
  if (isFileAsset(asset.type) && originalFilename) {
    return getDisplayFileName(originalFilename) || originalFilename
  }

  if (asset.uri && asset.type.startsWith('resume_')) {
    const fileName = asset.uri.split('/').pop() ?? asset.uri
    return getDisplayFileName(fileName) || fallbackLabel
  }

  if (asset.uri && asset.type === 'git_repo') {
    const normalized = asset.uri.replace(/\/+$/, '')
    return normalized.split('/').pop()?.replace(/\.git$/i, '') || normalized
  }

  if (asset.type === 'note') {
    return '未命名备注'
  }

  return fallbackLabel
}

function isFileAsset(assetType: string) {
  return assetType === 'resume_pdf' || assetType === 'resume_docx' || assetType === 'resume_image'
}

type ParseStatus = 'parsing' | 'success' | 'failed' | 'skipped' | null

function getParseStatus(asset: AssetItem): ParseStatus {
  const parsing = asset.metadata?.parsing as Record<string, unknown> | undefined
  const status = parsing?.status
  if (status === 'parsing' || status === 'success' || status === 'failed' || status === 'skipped') {
    return status
  }
  return null
}

function isGenericAssetLabel(asset: AssetItem, label: string) {
  const visual = getAssetVisual(asset.type, asset.uri)
  const normalized = label.trim().toLowerCase()
  return normalized === visual.chipLabel.toLowerCase() || normalized === visual.typeLabel.toLowerCase()
}

export default function AssetList({
  assets,
  onDelete,
  onOpenAsset,
  selectedAssetId,
  onRenameAsset,
  onReparseAsset,
  canReparseAsset,
  reparseLoadingAssetId,
  onSelectFolder,
  selectedFolderId,
  onMoveAsset,
}: AssetListProps) {
  const [renamingAssetId, setRenamingAssetId] = useState<number | null>(null)
  const [renameValue, setRenameValue] = useState('')
  const [renameSavingAssetId, setRenameSavingAssetId] = useState<number | null>(null)
  const [collapsedFolderIds, setCollapsedFolderIds] = useState<Set<number>>(() => new Set())
  const [rootCollapsed, setRootCollapsed] = useState(false)
  const [draggedAssetId, setDraggedAssetId] = useState<number | null>(null)
  const [dragOverFolderId, setDragOverFolderId] = useState<number | null>(null)
  const renameInputRef = useRef<HTMLInputElement>(null)

  const handleDragStart = (assetId: number, event: React.DragEvent) => {
    if (!onMoveAsset) return
    event.dataTransfer.setData('application/asset-id', String(assetId))
    event.dataTransfer.effectAllowed = 'move'
    setDraggedAssetId(assetId)
  }

  const handleDragEnd = () => {
    setDraggedAssetId(null)
    setDragOverFolderId(null)
  }

  const handleFolderDragEnter = (folderId: number, event: React.DragEvent) => {
    if (!onMoveAsset || draggedAssetId === null || draggedAssetId === folderId) return
    event.preventDefault()
    event.stopPropagation()
    event.dataTransfer.dropEffect = 'move'
    setDragOverFolderId(folderId)
  }

  const handleFolderDragOver = (folderId: number, event: React.DragEvent) => {
    if (!onMoveAsset || draggedAssetId === null || draggedAssetId === folderId) return
    event.preventDefault()
    event.stopPropagation()
    event.dataTransfer.dropEffect = 'move'
  }

  const handleFolderDragLeave = (folderId: number, event: React.DragEvent) => {
    // Only clear when truly leaving the folder, not when entering a child element
    if (event.currentTarget.contains(event.relatedTarget as Node | null)) return
    setDragOverFolderId((prev) => (prev === folderId ? null : prev))
  }

  const handleFolderDrop = async (targetFolderId: number, event: React.DragEvent) => {
    event.preventDefault()
    event.stopPropagation()
    if (!onMoveAsset || draggedAssetId === null || draggedAssetId === targetFolderId) return
    const assetId = draggedAssetId
    setDraggedAssetId(null)
    setDragOverFolderId(null)
    await onMoveAsset(assetId, targetFolderId)
  }

  const handleRootDragEnter = (event: React.DragEvent) => {
    if (!onMoveAsset || draggedAssetId === null) return
    event.preventDefault()
    event.dataTransfer.dropEffect = 'move'
    setDragOverFolderId(-1)
  }

  const handleRootDragOver = (event: React.DragEvent) => {
    if (!onMoveAsset || draggedAssetId === null) return
    event.preventDefault()
    event.dataTransfer.dropEffect = 'move'
  }

  const handleRootDragLeave = (event: React.DragEvent) => {
    if (event.currentTarget.contains(event.relatedTarget as Node | null)) return
    setDragOverFolderId((prev) => (prev === -1 ? null : prev))
  }

  const handleRootDrop = async (event: React.DragEvent) => {
    event.preventDefault()
    event.stopPropagation()
    if (!onMoveAsset || draggedAssetId === null) return
    const assetId = draggedAssetId
    setDraggedAssetId(null)
    setDragOverFolderId(null)
    await onMoveAsset(assetId, null)
  }

  useEffect(() => {
    if (renamingAssetId === null) return
    renameInputRef.current?.focus()
    renameInputRef.current?.select()
  }, [renamingAssetId])

  const sanitizeAssetLabel = (value: string) => value.replace(/\s+/g, ' ').trim()

  const beginRename = (asset: AssetItem, title: string) => {
    setRenamingAssetId(asset.id)
    setRenameValue(sanitizeAssetLabel(title))
  }

  const cancelRename = () => {
    setRenamingAssetId(null)
    setRenameValue('')
  }

  const commitRename = async (asset: AssetItem) => {
    if (!onRenameAsset) {
      cancelRename()
      return
    }

    const nextLabel = sanitizeAssetLabel(renameValue)
    const currentLabel = (asset.label?.trim() || '').trim()
    if (!nextLabel) {
      cancelRename()
      return
    }

    if (nextLabel === currentLabel) {
      cancelRename()
      return
    }

    try {
      setRenameSavingAssetId(asset.id)
      await onRenameAsset(asset, nextLabel)
      cancelRename()
    } finally {
      setRenameSavingAssetId(null)
    }
  }

  const folders = assets.filter((asset) => asset.type === 'folder')
  const files = assets.filter((asset) => asset.type !== 'folder')
  const folderIdSet = new Set(folders.map((folder) => folder.id))
  const folderIdForAsset = (asset: AssetItem) => {
    const raw = asset.metadata?.folder_id
    if (typeof raw !== 'number' || raw === asset.id || !folderIdSet.has(raw)) {
      return null
    }
    return raw
  }
  const childFoldersOf = (parentId: number | null) =>
    folders.filter((folder) => folderIdForAsset(folder) === parentId)
  const childFilesOf = (parentId: number | null) =>
    files.filter((asset) => folderIdForAsset(asset) === parentId)
  const rootFiles = files.filter((asset) => folderIdForAsset(asset) === null)

  useEffect(() => {
    const nextFolderIds = new Set(folders.map((folder) => folder.id))
    setCollapsedFolderIds((current) => {
      const next = new Set([...current].filter((id) => nextFolderIds.has(id)))
      return next.size === current.size ? current : next
    })
  }, [folders])

  if (assets.length === 0) {
    return (
      <div className="py-12 text-center text-muted-foreground">
        <p className="text-sm">还没有添加任何资料</p>
        <p className="mt-1 text-xs">点击上方按钮上传文件、接入 Git 或添加备注</p>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <div className={["rounded-2xl border border-border bg-card/70 p-2", dragOverFolderId === -1 ? "ring-2 ring-primary/30 border-primary" : ""].join(" ")}
        onDragEnter={handleRootDragEnter}
        onDragOver={handleRootDragOver}
        onDragLeave={(event) => handleRootDragLeave(event)}
        onDrop={(event) => handleRootDrop(event)}>
        <button
          type="button"
          onClick={() => {
            onSelectFolder?.(null)
            setRootCollapsed((value) => !value)
          }}
          className={[
            'flex w-full items-center gap-2 rounded-xl px-2 py-2 text-left transition-colors hover:bg-surface-hover',
            selectedFolderId === null ? 'bg-primary/10 text-foreground' : 'text-muted-foreground',
          ].join(' ')}
        >
          <ChevronDown className={`h-4 w-4 transition-transform ${rootCollapsed ? '-rotate-90' : ''}`} />
          <Folder className="h-4 w-4 text-cyan-500" />
          <span className="min-w-0 flex-1 truncate text-sm font-semibold">根目录</span>
          <span className="text-[11px]">{childFoldersOf(null).length + rootFiles.length}</span>
        </button>
        {!rootCollapsed && (
          <div className="mt-2 space-y-2">
            {childFoldersOf(null).length > 0 && (
              <div className="space-y-2 pl-5">
                {childFoldersOf(null).map((folder) => renderFolder(folder))}
              </div>
            )}
            {rootFiles.map((asset) => renderAsset(asset))}
          </div>
        )}
      </div>
    </div>
  )

  function toggleFolder(folderId: number) {
    onSelectFolder?.(folderId)
    setCollapsedFolderIds((current) => {
      const next = new Set(current)
      if (next.has(folderId)) {
        next.delete(folderId)
      } else {
        next.add(folderId)
      }
      return next
    })
  }

  function renderFolder(folder: AssetItem) {
    const childFolders = childFoldersOf(folder.id)
    const childFiles = childFilesOf(folder.id)
    const expanded = !collapsedFolderIds.has(folder.id)
    const visual = getAssetVisual(folder.type, folder.uri)
    const Icon = visual.icon
    const title = getDisplayTitle(folder, visual.chipLabel)
    const renaming = renamingAssetId === folder.id
    const renameSaving = renameSavingAssetId === folder.id

    return (
      <div key={folder.id} className={["rounded-xl border border-border/80 bg-card/70 p-2", dragOverFolderId === folder.id ? "ring-2 ring-primary/30 border-primary" : ""].join(" ")}
        onDragEnter={(event) => handleFolderDragEnter(folder.id, event)}
        onDragOver={(event) => handleFolderDragOver(folder.id, event)}
        onDragLeave={(event) => handleFolderDragLeave(folder.id, event)}
        onDrop={(event) => handleFolderDrop(folder.id, event)}>
        <div
          className={[
            'flex w-full items-center gap-2 rounded-lg px-2 py-1.5 transition-colors hover:bg-surface-hover',
            selectedFolderId === folder.id ? 'bg-primary/10 text-foreground' : 'text-muted-foreground',
          ].join(' ')}
        >
          <button
            type="button"
            onClick={() => toggleFolder(folder.id)}
            className="flex min-w-0 flex-1 items-center gap-2 text-left"
          >
            <ChevronDown className={`h-4 w-4 shrink-0 transition-transform ${expanded ? '' : '-rotate-90'}`} />
            <Icon className={`h-4 w-4 shrink-0 ${visual.iconClassName}`} />
            {renaming ? (
              <input
                ref={renameInputRef}
                value={renameValue}
                disabled={renameSaving}
                onClick={(event) => event.stopPropagation()}
                onChange={(event) => setRenameValue(event.target.value.replace(/[\r\n]+/g, ' '))}
                onBlur={() => void commitRename(folder)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') {
                    event.preventDefault()
                    event.currentTarget.blur()
                  }
                  if (event.key === 'Escape') {
                    event.preventDefault()
                    cancelRename()
                  }
                }}
                className="h-7 min-w-0 flex-1 rounded-lg border border-primary/35 bg-background/80 px-2 text-sm font-semibold text-foreground outline-none transition focus:border-primary focus:ring-2 focus:ring-ring/35 disabled:opacity-60"
              />
            ) : (
              <span className="min-w-0 flex-1 truncate text-sm font-semibold" title={title}>{title}</span>
            )}
          </button>
          <span className="text-[11px]">{childFolders.length + childFiles.length}</span>
          <AssetMoreMenu
            asset={folder}
            renameable={onRenameAsset !== undefined}
            onRename={() => beginRename(folder, title)}
            onDelete={onDelete}
          />
        </div>
        {expanded && (
          <div className="mt-2 space-y-2 pl-5">
            {childFolders.map((child) => renderFolder(child))}
            {childFiles.map((asset) => renderAsset(asset))}
          </div>
        )}
      </div>
    )
  }

  function renderAsset(asset: AssetItem) {
        const visual = getAssetVisual(asset.type, asset.uri)
        const Icon = visual.icon
        const title = getDisplayTitle(asset, visual.chipLabel)
        const renameable = onRenameAsset !== undefined
        const renaming = renamingAssetId === asset.id
        const renameSaving = renameSavingAssetId === asset.id
        const reparsable = onReparseAsset !== undefined && (canReparseAsset ? canReparseAsset(asset) : false)
        const showGitSource = asset.type === 'git_repo' && asset.uri

        return (
          <div
            key={asset.id}
            role={onOpenAsset ? 'button' : undefined}
            tabIndex={onOpenAsset ? 0 : undefined}
            draggable={onMoveAsset !== undefined}
            onClick={() => onOpenAsset?.(asset)}
            onDragStart={(event) => handleDragStart(asset.id, event)}
            onDragEnd={handleDragEnd}
            onKeyDown={(event) => {
              if (!onOpenAsset) return
              if (event.key !== 'Enter' && event.key !== ' ') return
              event.preventDefault()
              onOpenAsset(asset)
            }}
            className={[
              'rounded-xl border bg-card/85 px-2 py-1.5 shadow-sm transition-colors',
              onOpenAsset ? 'cursor-pointer hover:border-primary-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/35' : 'hover:border-primary-200',
              selectedAssetId === asset.id ? 'border-primary/60 bg-primary/10' : 'border-border',
			  draggedAssetId === asset.id ? 'opacity-50 scale-[0.97]' : '',
            ].join(' ')}
          >
            <div className="flex items-center gap-2">
              <div
                className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border ${visual.iconWrapperClassName}`}
              >
                <Icon className={`h-4 w-4 ${visual.iconClassName}`} />
              </div>

              <div className="min-w-0 flex-1">
                <div className="flex flex-col gap-2 xl:flex-row xl:items-center xl:justify-between">
                  <div className="min-w-0 flex-1">
                    <div className="flex min-h-6 min-w-0 items-center gap-2">
                      {renaming ? (
                        <input
                          ref={renameInputRef}
                          value={renameValue}
                          disabled={renameSaving}
                          onClick={(event) => event.stopPropagation()}
                          onChange={(event) => setRenameValue(event.target.value.replace(/[\r\n]+/g, ' '))}
                          onBlur={() => void commitRename(asset)}
                          onKeyDown={(event) => {
                            if (event.key === 'Enter') {
                              event.preventDefault()
                              event.currentTarget.blur()
                            }
                            if (event.key === 'Escape') {
                              event.preventDefault()
                              cancelRename()
                            }
                          }}
                          className="h-7 min-w-0 flex-1 rounded-lg border border-primary/35 bg-background/80 px-2 text-sm font-semibold text-foreground shadow-[0_8px_24px_rgba(2,8,23,0.16)] outline-none transition focus:border-primary focus:ring-2 focus:ring-ring/35 disabled:opacity-60"
                        />
                      ) : (
                        <p className="min-w-0 flex-1 truncate text-sm font-semibold text-foreground" title={title}>{title}</p>
                      )}
                      <span
                        className={`inline-flex shrink-0 items-center rounded-full border px-2 py-0.5 text-[11px] font-semibold tracking-wide ${visual.chipClassName}`}
                      >
                        {visual.chipLabel}
                      </span>
                      {getParseStatus(asset) === 'parsing' && (
                        <Loader2 className="h-3.5 w-3.5 shrink-0 animate-spin text-amber-500" />
                      )}
                      {getParseStatus(asset) === 'success' && (
                        <span className="inline-flex h-3.5 w-3.5 shrink-0 items-center justify-center rounded-full bg-emerald-100 text-emerald-600" title="解析成功">
                          <svg className="h-2.5 w-2.5" fill="currentColor" viewBox="0 0 20 20"><path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd"/></svg>
                        </span>
                      )}
                      {getParseStatus(asset) === 'failed' && (
                        <span className="inline-flex h-3.5 w-3.5 shrink-0 items-center justify-center rounded-full bg-red-100 text-red-500" title="解析失败">
                          <svg className="h-2.5 w-2.5" fill="currentColor" viewBox="0 0 20 20"><path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd"/></svg>
                        </span>
                      )}
                    </div>

                    {showGitSource && (
                      <p className="mt-1 text-xs text-muted-foreground break-all">{asset.uri}</p>
                    )}
                  </div>

                  <div className="flex shrink-0 flex-wrap items-center justify-end gap-1.5">
                    {reparsable && (
                      <AssetActionButton
                        label={reparseLoadingAssetId === asset.id ? '解析中...' : '重新解析'}
                        onClick={() => onReparseAsset?.(asset)}
                        icon={RefreshCcw}
                        disabled={reparseLoadingAssetId === asset.id}
                      />
                    )}

                    <AssetMoreMenu
                      asset={asset}
                      renameable={renameable}
                      onRename={() => beginRename(asset, title)}
                      onDelete={onDelete}
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>
        )
  }
}
