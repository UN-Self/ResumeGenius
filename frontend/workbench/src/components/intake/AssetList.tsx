import { useEffect, useRef, useState } from 'react'
import type { KeyboardEvent } from 'react'
import { MoreHorizontal, PencilLine, RefreshCcw, Trash2 } from 'lucide-react'
import type { Asset } from '@/lib/api-client'
import { getAssetVisual, getDisplayAssetTitle, getDisplayFileName, getOriginalFilenameFromAsset } from './fileVisuals'

type AssetItem = Asset & { label?: string; content?: string; uri?: string }

interface AssetListProps {
  assets: AssetItem[]
  onDelete: (id: number) => void
  onRenameAsset?: (asset: AssetItem, label: string) => Promise<void> | void
  onReparseAsset?: (asset: AssetItem) => void
  canReparseAsset?: (asset: AssetItem) => boolean
  reparseLoadingAssetId?: number | null
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
      onClick={onClick}
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
        onClick={() => setOpen((value) => !value)}
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

function getDisplayTitle(asset: AssetItem, fallbackLabel: string) {
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

function isGenericAssetLabel(asset: AssetItem, label: string) {
  const visual = getAssetVisual(asset.type, asset.uri)
  const normalized = label.trim().toLowerCase()
  return normalized === visual.chipLabel.toLowerCase() || normalized === visual.typeLabel.toLowerCase()
}

function getContentPreview(asset: AssetItem) {
  const raw = asset.content?.replace(/\r\n/g, '\n').replace(/\r/g, '\n').trim() ?? ''
  if (!raw) {
    return ''
  }

  if (asset.type === 'note' && asset.label?.trim() && raw.startsWith(asset.label.trim())) {
    return raw.slice(asset.label.trim().length).replace(/^\s+/, '').replace(/\n{3,}/g, '\n\n')
  }

  return raw.replace(/\n{3,}/g, '\n\n')
}

export default function AssetList({
  assets,
  onDelete,
  onRenameAsset,
  onReparseAsset,
  canReparseAsset,
  reparseLoadingAssetId,
}: AssetListProps) {
  const [renamingAssetId, setRenamingAssetId] = useState<number | null>(null)
  const [renameValue, setRenameValue] = useState('')
  const [renameSavingAssetId, setRenameSavingAssetId] = useState<number | null>(null)
  const renameInputRef = useRef<HTMLInputElement>(null)

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
      {assets.map((asset) => {
        const visual = getAssetVisual(asset.type, asset.uri)
        const Icon = visual.icon
        const title = getDisplayTitle(asset, visual.chipLabel)
        const contentPreview = getContentPreview(asset)
        const renameable = onRenameAsset !== undefined
        const renaming = renamingAssetId === asset.id
        const renameSaving = renameSavingAssetId === asset.id
        const reparsable = onReparseAsset !== undefined && (canReparseAsset ? canReparseAsset(asset) : false)
        const showGitSource = asset.type === 'git_repo' && asset.uri

        return (
          <div
            key={asset.id}
            className="rounded-2xl border border-border bg-card/90 p-3.5 shadow-sm transition-colors hover:border-primary-200"
          >
            <div className="flex items-start gap-3">
              <div
                className={`mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border ${visual.iconWrapperClassName}`}
              >
                <Icon className={`h-5 w-5 ${visual.iconClassName}`} />
              </div>

              <div className="min-w-0 flex-1">
                <div className="flex flex-col gap-3 xl:flex-row xl:items-start xl:justify-between">
                  <div className="min-w-0 flex-1">
                    <div className="flex min-h-8 min-w-0 items-center gap-2">
                      {renaming ? (
                        <input
                          ref={renameInputRef}
                          value={renameValue}
                          disabled={renameSaving}
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
                          className="h-8 min-w-0 flex-1 rounded-lg border border-primary/35 bg-background/80 px-2.5 text-sm font-semibold text-foreground shadow-[0_8px_24px_rgba(2,8,23,0.16)] outline-none transition focus:border-primary focus:ring-2 focus:ring-ring/35 disabled:opacity-60"
                        />
                      ) : (
                        <p className="min-w-0 flex-1 truncate text-sm font-semibold text-foreground" title={title}>{title}</p>
                      )}
                      <span
                        className={`inline-flex shrink-0 items-center rounded-full border px-2 py-0.5 text-[11px] font-semibold tracking-wide ${visual.chipClassName}`}
                      >
                        {visual.chipLabel}
                      </span>
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

                {contentPreview ? (
                  <div className="mt-3 max-h-36 overflow-y-auto rounded-xl border border-border/80 bg-background px-3 py-2.5 text-[13px] leading-relaxed text-muted-foreground whitespace-pre-wrap break-words">
                    {contentPreview}
                  </div>
                ) : (
                  <div className="mt-3 rounded-xl border border-dashed border-border/80 bg-background/60 px-3 py-2.5 text-xs text-muted-foreground">
                    该素材当前没有可展示的正文内容。
                  </div>
                )}
              </div>
            </div>
          </div>
        )
      })}
    </div>
  )
}
