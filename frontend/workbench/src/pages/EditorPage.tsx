import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { FileText, X } from 'lucide-react'
import { useEditor } from '@tiptap/react'
import { BubbleMenu } from '@tiptap/react/menus'
import StarterKit from '@tiptap/starter-kit'
import TextAlign from '@tiptap/extension-text-align'
import { TextStyleKit } from '@tiptap/extension-text-style'
import { PaginationPlus } from 'tiptap-pagination-plus'
import { A4Canvas } from '@/components/editor/A4Canvas'
import { Div, PresetAttributes, Span } from '@/components/editor/extensions'
import { ActionBar } from '@/components/editor/ActionBar'
import { SaveIndicator } from '@/components/editor/SaveIndicator'
import { ChatPanel } from '@/components/chat/ChatPanel'
import AssetSidebar from '@/components/intake/AssetSidebar'
import { AssetWorkspace } from '@/components/intake/AssetWorkspace'
import { getDisplayTitle } from '@/components/intake/AssetList'
import { getAssetVisual } from '@/components/intake/fileVisuals'
import { request, intakeApi, workbenchApi, ApiError, type Asset } from '@/lib/api-client'
import { captureCopy, sliceFromJson, getMimeType } from '@/lib/clipboard'
import { useAutoSave } from '@/hooks/useAutoSave'
import { useExport } from '@/hooks/useExport'
import { FullPageState } from '@/components/ui/full-page-state'
import { ContextMenu } from '@/components/editor/ContextMenu'
import { BubbleToolbar } from '@/components/editor/BubbleToolbar'
import { useVersions } from '@/hooks/useVersions'
import { VersionDropdown } from '@/components/version/VersionDropdown'
import { VersionPreviewBanner } from '@/components/version/VersionPreviewBanner'
import { SaveSnapshotDialog } from '@/components/version/SaveSnapshotDialog'
import { RollbackConfirmDialog } from '@/components/version/RollbackConfirmDialog'
import { useToast } from '@/hooks/useToast'
import { ToastContainer } from '@/components/ui/toast'
import { Button } from '@/components/ui/button'
import { Modal, ModalBody, ModalFooter, ModalHeader } from '@/components/ui/modal'
import { extractStyles } from '@/lib/extract-styles'

export default function EditorPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const pid = Number(projectId)

  const [draftId, setDraftId] = useState<string | null>(null)
  const draftIdRef = useRef<string | null>(null)
  const [projectTitle, setProjectTitle] = useState('')
  const [assets, setAssets] = useState<Asset[]>([])
  const [openAssetTabIds, setOpenAssetTabIds] = useState<number[]>([])
  const [activeTab, setActiveTab] = useState<'resume' | number>('resume')
  const [assetDrafts, setAssetDrafts] = useState<Record<number, string>>({})
  const [assetSavingId, setAssetSavingId] = useState<number | null>(null)
  const [pendingCloseAsset, setPendingCloseAsset] = useState<Asset | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [leftOpen, setLeftOpen] = useState(true)
  const [rightOpen, setRightOpen] = useState(true)
  const [contextMenu, setContextMenu] = useState<{ isOpen: boolean; x: number; y: number }>({
    isOpen: false, x: 0, y: 0,
  })

  const editor = useEditor({
    extensions: [
      StarterKit.configure({ strike: false }),
      TextAlign.configure({ types: ['heading', 'paragraph'] }),
      TextStyleKit,
      PresetAttributes,
      Div,
      Span,
      PaginationPlus.configure({
        // A4 at 96dpi: 210×297mm ≈ 794×1123px
        pageHeight: 1123,
        pageWidth: 794,
        // Margins: 18mm top/bottom ≈ 68px, 20mm left/right ≈ 76px
        marginTop: 68,
        marginBottom: 68,
        marginLeft: 76,
        marginRight: 76,
        pageGap: 32,
        contentMarginTop: 0,
        contentMarginBottom: 0,
        pageBreakBackground: '#ede8e0',
        pageGapBorderSize: 0,
        // No headers/footers
        headerLeft: '',
        headerRight: '',
        footerLeft: '',
        footerRight: '',
      }),
    ],
    content: '',
    editorProps: {
      attributes: {
        class: 'resume-content outline-none',
      },
      handleDOMEvents: {
        copy(_view, event) {
          const { from, to } = _view.state.selection
          const { text, json } = captureCopy(_view.state, from, to)
          event.preventDefault()
          event.clipboardData?.setData('text/plain', text)
          event.clipboardData?.setData(getMimeType(), json)
          return true
        },
        cut(_view, event) {
          const { from, to } = _view.state.selection
          const { text, json } = captureCopy(_view.state, from, to)
          event.preventDefault()
          event.clipboardData?.setData('text/plain', text)
          event.clipboardData?.setData(getMimeType(), json)
          _view.dispatch(_view.state.tr.deleteSelection())
          return true
        },
        paste(_view, event) {
          event.preventDefault()
          const raw = event.clipboardData?.getData(getMimeType())
          if (raw) {
            const slice = sliceFromJson(_view.state.schema, raw)
            _view.dispatch(_view.state.tr.replaceSelection(slice))
            return true
          }
          const text = event.clipboardData?.getData('text/plain') || ''
          if (text) {
            _view.dispatch(_view.state.tr.insertText(text))
          }
          return true
        },
      },
    },
  })

  const [pendingHtml, setPendingHtml] = useState<string | null>(null)
  const [scopedCSS, setScopedCSS] = useState('')

  const { scheduleSave, flush, retry, status, lastSavedAt } = useAutoSave({
    save: async (html: string) => {
      const currentDraftId = draftIdRef.current
      if (currentDraftId) {
        await request(`/drafts/${currentDraftId}`, {
          method: 'PUT',
          body: JSON.stringify({ html_content: html }),
        })
      }
    },
    saveUrl: draftId ? `/api/v1/drafts/${draftId}` : undefined,
  })

  const {
    exportPdf,
    status: exportStatus,
    error: exportError,
    progress: exportProgress,
    message: exportMessage,
  } = useExport()

  const {
    versions,
    loading: versionsLoading,
    error: versionsError,
    previewMode,
    previewVersion,
    previewHtml,
    startPreview,
    exitPreview,
    createSnapshot,
    rollback: rollbackVersion,
    refreshList,
  } = useVersions(draftId ? Number(draftId) : null)

  const [saveDialogOpen, setSaveDialogOpen] = useState(false)
  const [rollbackDialogOpen, setRollbackDialogOpen] = useState(false)
  const [savingSnapshot, setSavingSnapshot] = useState(false)
  const [rollbacking, setRollbacking] = useState(false)
  const { toasts, toast, dismissToast } = useToast()

  // Save editor content before preview so we can restore it on exit
  const savedContentBeforePreview = useRef<{ html: string; scopedCSS: string } | null>(null)
  const didRecenterAfterPreviewRef = useRef(false)
  const restoringContent = useRef(false)

  const applyHtmlToEditor = useCallback((html: string) => {
    if (!editor) return

    const { bodyHtml, scopedCSS: nextScopedCSS } = extractStyles(html)
    restoringContent.current = true
    setScopedCSS(nextScopedCSS)
    editor.commands.setContent(bodyHtml || html)
    restoringContent.current = false
  }, [editor])

  const getPersistableHTML = useCallback(() => {
    if (!editor) return ''

    const bodyHtml = editor.getHTML()
    if (!scopedCSS.trim()) return bodyHtml

    return [
      '<!DOCTYPE html>',
      '<html>',
      '<head>',
      '<meta charset="utf-8">',
      `<style>${scopedCSS}</style>`,
      '</head>',
      '<body>',
      bodyHtml,
      '</body>',
      '</html>',
    ].join('')
  }, [editor, scopedCSS])

  // When preview HTML is ready, load it into the editor and make read-only
  useLayoutEffect(() => {
    if (previewMode === 'previewing' && previewHtml && editor) {
      applyHtmlToEditor(previewHtml)
      editor.setEditable(false)
      didRecenterAfterPreviewRef.current = false
    }
  }, [previewMode, previewHtml, editor, applyHtmlToEditor])

  const handleStartPreview = useCallback(async (version: Parameters<typeof startPreview>[0]) => {
    if (editor) {
      savedContentBeforePreview.current = {
        html: editor.getHTML(),
        scopedCSS,
      }
    }
    await startPreview(version)
  }, [editor, scopedCSS, startPreview])

  const handleExitPreview = useCallback(() => {
    if (editor) {
      editor.setEditable(true)
      const saved = savedContentBeforePreview.current
      if (saved) {
        restoringContent.current = true
        setScopedCSS(saved.scopedCSS)
        editor.commands.setContent(saved.html)
        restoringContent.current = false
      }
      savedContentBeforePreview.current = null
    }
    exitPreview()
  }, [editor, exitPreview])

  const reloadAssets = useCallback(async () => {
    const nextAssets = await intakeApi.listAssets(pid)
    setAssets(nextAssets)
  }, [pid])

  useEffect(() => {
    const assetIds = new Set(assets.map((asset) => asset.id))
    setOpenAssetTabIds((current) => current.filter((id) => assetIds.has(id)))
    setAssetDrafts((current) => {
      const next: Record<number, string> = {}
      for (const asset of assets) {
        if (current[asset.id] !== undefined) {
          next[asset.id] = current[asset.id]
        }
      }
      return next
    })
    setActiveTab((current) => {
      if (current === 'resume' || assetIds.has(current)) return current
      return 'resume'
    })
  }, [assets])

  const handleExport = () => {
    if (draftId) {
      exportPdf(Number(draftId))
    }
  }

  const openAssetTab = (asset: Asset) => {
    setOpenAssetTabIds((current) => current.includes(asset.id) ? current : [...current, asset.id])
    setAssetDrafts((current) => ({
      ...current,
      [asset.id]: current[asset.id] ?? (asset.content ?? ''),
    }))
    setActiveTab(asset.id)
  }

  const saveAssetTab = async (asset: Asset) => {
    const content = assetDrafts[asset.id] ?? asset.content ?? ''
    setAssetSavingId(asset.id)
    try {
      const updated = await intakeApi.updateAsset(asset.id, { content })
      setAssets((current) => current.map((item) => item.id === updated.id ? updated : item))
      setAssetDrafts((current) => ({ ...current, [asset.id]: updated.content ?? '' }))
    } catch (saveError) {
      toast(saveError instanceof Error ? saveError.message : '保存素材失败')
    } finally {
      setAssetSavingId(null)
    }
  }

  const finalizeCloseAssetTab = (asset: Asset) => {
    setOpenAssetTabIds((current) => {
      const index = current.indexOf(asset.id)
      const next = current.filter((id) => id !== asset.id)
      if (activeTab === asset.id) {
        setActiveTab(next[index] ?? next[index - 1] ?? 'resume')
      }
      return next
    })
    setPendingCloseAsset(null)
  }

  const requestCloseAssetTab = (asset: Asset) => {
    const draft = assetDrafts[asset.id] ?? asset.content ?? ''
    const dirty = draft !== (asset.content ?? '')
    if (dirty) {
      setPendingCloseAsset(asset)
      return
    }

    finalizeCloseAssetTab(asset)
  }

  const saveAndClosePendingAsset = async () => {
    if (!pendingCloseAsset) return
    await saveAssetTab(pendingCloseAsset)
    finalizeCloseAssetTab(pendingCloseAsset)
  }

  const handleRollback = async () => {
    setRollbacking(true)
    try {
      await flush()
      const html = await rollbackVersion()
      applyHtmlToEditor(html)
      if (editor) {
        editor.setEditable(true)
      }
      savedContentBeforePreview.current = null
      setRollbackDialogOpen(false)
    } catch (e) {
      toast(e instanceof Error ? e.message : '回滚失败')
    } finally {
      setRollbacking(false)
    }
  }

  const handleContextMenu = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    setContextMenu({ isOpen: true, x: e.clientX, y: e.clientY })
  }, [])

  const closeContextMenu = useCallback(() => {
    setContextMenu((prev) => ({ ...prev, isOpen: false }))
  }, [])

  useEffect(() => {
    const close = () => closeContextMenu()
    document.addEventListener('scroll', close, true)
    return () => document.removeEventListener('scroll', close, true)
  }, [closeContextMenu])

  useEffect(() => {
    if (!contextMenu.isOpen) return

    const handleClick = (e: MouseEvent) => {
      const target = e.target as HTMLElement
      if (!target.closest('[role="menu"]')) {
        closeContextMenu()
      }
    }

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') closeContextMenu()
    }

    document.addEventListener('mousedown', handleClick)
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('mousedown', handleClick)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [contextMenu.isOpen, closeContextMenu])

  useEffect(() => {
    draftIdRef.current = draftId
  }, [draftId])

  useEffect(() => {
    if (!projectId || Number.isNaN(pid)) return

    let cancelled = false

    const load = async () => {
      try {
        setLoading(true)

        const project = await intakeApi.getProject(pid)
        if (cancelled) return

        if (!project.current_draft_id) {
          try {
            const [draft, nextAssets] = await Promise.all([
              workbenchApi.createDraft(pid),
              intakeApi.listAssets(pid),
            ])
            if (cancelled) return
            setProjectTitle(project.title)
            setDraftId(String(draft.id))
            setPendingHtml(draft.html_content || '')
            setAssets(nextAssets)
            setError(null)
          } catch (createErr) {
            if (cancelled) return
            setError(createErr instanceof ApiError ? createErr.message : '创建草稿失败')
          }
          setLoading(false)
          return
        }

        const currentDraftId = String(project.current_draft_id)
        const [draft, nextAssets] = await Promise.all([
          workbenchApi.getDraft(project.current_draft_id),
          intakeApi.listAssets(pid),
        ])
        if (cancelled) return

        setProjectTitle(project.title)
        setDraftId(currentDraftId)
        setPendingHtml(draft.html_content || '')
        setAssets(nextAssets)
        setError(null)
      } catch (loadError) {
        if (cancelled) return
        setError(loadError instanceof ApiError ? loadError.message : '\u52a0\u8f7d\u5931\u8d25')
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    load()

    return () => {
      cancelled = true
    }
  }, [navigate, pid, projectId])

  const hasAppliedRef = useRef(false)

  useEffect(() => {
    hasAppliedRef.current = false
  }, [projectId])

  useEffect(() => {
    if (editor && pendingHtml !== null && !hasAppliedRef.current) {
      hasAppliedRef.current = true
      applyHtmlToEditor(pendingHtml)
    }
  }, [editor, pendingHtml, applyHtmlToEditor])

  useEffect(() => {
    if (!editor) return

    const handleUpdate = () => {
      if (!restoringContent.current) {
        scheduleSave(getPersistableHTML())
      }
    }
    editor.on('update', handleUpdate)

    return () => {
      editor.off('update', handleUpdate)
    }
  }, [editor, getPersistableHTML, scheduleSave])

  useEffect(() => {
    return () => {
      flush()
    }
  }, [flush])

  if (loading) {
    return <FullPageState variant="loading" />
  }

  if (error) {
    return <FullPageState variant="error" message={error} />
  }

  const gridClass = [
    'editor-workspace',
    leftOpen ? 'left-open' : 'left-collapsed',
    rightOpen ? 'right-open' : 'right-collapsed',
  ].join(' ')

  const openAssetTabs = openAssetTabIds
    .map((id) => assets.find((asset) => asset.id === id))
    .filter((asset): asset is Asset => Boolean(asset))
  const activeAsset = typeof activeTab === 'number'
    ? assets.find((asset) => asset.id === activeTab) ?? null
    : null
  return (
    <div className={gridClass}>
      <div className="editor-panel-left">
        <div className="panel-header">
          <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            文件
          </h2>
          <button
            type="button"
            onClick={() => setLeftOpen(false)}
            className="panel-collapse-btn"
            aria-label="收起左侧栏"
          >
            {'<'}
          </button>
        </div>
        <div className="panel-body left-workbench-body">
          <nav className="left-activity-bar" aria-label="左侧功能">
            <button
              type="button"
              aria-label="文件"
              aria-pressed="true"
              className="left-activity-button active"
            >
              <FileText className="h-5 w-5" />
            </button>
          </nav>
          <div className="left-workbench-content">
            <AssetSidebar
              projectId={pid}
              assets={assets}
              onReload={reloadAssets}
              onOpenAsset={openAssetTab}
              selectedAssetId={typeof activeTab === 'number' ? activeTab : null}
            />
          </div>
        </div>
      </div>

      <div className="editor-panel-center">
        {!leftOpen && (
          <button
            type="button"
            onClick={() => setLeftOpen(true)}
            className="panel-expand-btn panel-expand-left"
            aria-label="展开左侧栏"
          >
            {'>'}
          </button>
        )}
        {!rightOpen && (
          <button
            type="button"
            onClick={() => setRightOpen(true)}
            className="panel-expand-btn panel-expand-right"
            aria-label="展开右侧栏"
          >
            {'<'}
          </button>
        )}
        <div className="flex h-full flex-col">
          <div className="flex items-center gap-2">
            <ActionBar
              projectName={projectTitle}
              saveIndicator={<SaveIndicator status={status} lastSavedAt={lastSavedAt} onRetry={retry} />}
              draftId={draftId}
              exportStatus={exportStatus}
              onExport={handleExport}
              onBack={() => navigate(`/projects/${pid}`)}
            >
              <VersionDropdown
                versions={versions}
                loading={versionsLoading}
                error={versionsError}
                onPreview={handleStartPreview}
                onSaveSnapshot={() => setSaveDialogOpen(true)}
                onRetry={refreshList}
              />
            </ActionBar>
          </div>
          <div className="editor-tab-strip" role="tablist" aria-label="打开的栏目">
            <button
              type="button"
              role="tab"
              aria-selected={activeTab === 'resume'}
              onClick={() => setActiveTab('resume')}
              className={activeTab === 'resume' ? 'editor-tab active' : 'editor-tab'}
            >
              <FileText className="h-3.5 w-3.5 text-primary" />
              <span className="truncate">简历</span>
            </button>
            {openAssetTabs.map((asset) => {
              const visual = getAssetVisual(asset.type, asset.uri)
              const title = getDisplayTitle(asset, visual.chipLabel)
              const Icon = visual.icon
              const dirty = (assetDrafts[asset.id] ?? asset.content ?? '') !== (asset.content ?? '')
              return (
                <button
                  key={asset.id}
                  type="button"
                  role="tab"
                  aria-selected={activeTab === asset.id}
                  onClick={() => setActiveTab(asset.id)}
                  className={activeTab === asset.id ? 'editor-tab active' : 'editor-tab'}
                >
                  <Icon className="h-3.5 w-3.5 shrink-0" />
                  <span className="truncate">{title}</span>
                  {dirty && <span className="editor-tab-dirty" aria-label="未保存" />}
                  <span
                    role="button"
                    tabIndex={0}
                    aria-label={`关闭 ${title}`}
                    className="editor-tab-close"
                    onClick={(event) => {
                      event.stopPropagation()
                      requestCloseAssetTab(asset)
                    }}
                    onKeyDown={(event) => {
                      if (event.key !== 'Enter' && event.key !== ' ') return
                      event.preventDefault()
                      event.stopPropagation()
                      requestCloseAssetTab(asset)
                    }}
                  >
                    <X className="h-3.5 w-3.5" />
                  </span>
                </button>
              )
            })}
          </div>
          <div
            className="flex-1 overflow-auto"
            onContextMenu={activeTab === 'resume' ? handleContextMenu : undefined}
          >
            {activeTab === 'resume' ? (
              <>
                {previewMode === 'previewing' && previewVersion && (
                  <VersionPreviewBanner
                    version={previewVersion}
                    onRollback={() => setRollbackDialogOpen(true)}
                    onClose={handleExitPreview}
                  />
                )}
                <A4Canvas editor={editor} scopedCSS={scopedCSS} />
                {editor && previewMode !== 'previewing' && (
                  <BubbleMenu
                    editor={editor}
                    options={{
                      placement: 'top',
                      arrow: false,
                    }}
                    shouldShow={({ editor: e }) => {
                      const { from, to } = e.state.selection
                      return from !== to
                    }}
                  >
                    <BubbleToolbar editor={editor} />
                  </BubbleMenu>
                )}
              </>
            ) : activeAsset ? (
              <AssetWorkspace
                asset={activeAsset}
                value={assetDrafts[activeAsset.id] ?? activeAsset.content ?? ''}
                dirty={(assetDrafts[activeAsset.id] ?? activeAsset.content ?? '') !== (activeAsset.content ?? '')}
                saving={assetSavingId === activeAsset.id}
                onChange={(value) => setAssetDrafts((current) => ({ ...current, [activeAsset.id]: value }))}
                onSave={() => void saveAssetTab(activeAsset)}
              />
            ) : (
              <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
                该栏目已经关闭。
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="editor-panel-right">
        <div className="panel-header">
          <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            {'AI \u52a9\u624b'}
          </h2>
          <button
            type="button"
            onClick={() => setRightOpen(false)}
            className="panel-collapse-btn"
            aria-label="收起右侧栏"
          >
            {'>'}
          </button>
        </div>
        <div className="panel-body">
          {draftId ? (
            <ChatPanel
              draftId={Number(draftId)}
              onApplyEdits={async () => {
                if (!editor || !draftId) return
                const draft = await workbenchApi.getDraft(Number(draftId))
                applyHtmlToEditor(draft.html_content || '')
              }}
              onRestoreHtml={(html) => {
                if (!editor) return
                applyHtmlToEditor(html)
              }}
            />
          ) : (
            <p className="mt-8 text-center text-xs text-muted-foreground">
              {'\u7b49\u5f85\u8349\u7a3f\u52a0\u8f7d...'}
            </p>
          )}
        </div>
      </div>

      <ContextMenu
        editor={editor}
        isOpen={contextMenu.isOpen}
        x={contextMenu.x}
        y={contextMenu.y}
        onClose={closeContextMenu}
      />
      <SaveSnapshotDialog
        open={saveDialogOpen}
        saving={savingSnapshot}
        onClose={() => setSaveDialogOpen(false)}
        onConfirm={async (label) => {
          setSavingSnapshot(true)
          try {
            await flush()
            await createSnapshot(label)
            setSaveDialogOpen(false)
          } catch (e) {
            toast(e instanceof Error ? e.message : '保存快照失败')
          } finally {
            setSavingSnapshot(false)
          }
        }}
      />
      <RollbackConfirmDialog
        open={rollbackDialogOpen}
        rollbacking={rollbacking}
        onClose={() => setRollbackDialogOpen(false)}
        onConfirm={handleRollback}
      />
      <Modal
        open={exportStatus === 'exporting' || exportStatus === 'completed' || exportStatus === 'failed'}
        onClose={() => undefined}
        maxWidth="max-w-md"
      >
        <ModalHeader>导出 PDF</ModalHeader>
        <ModalBody>
          <div className="mt-4 rounded-2xl border border-border bg-background/70 p-4">
            <div className="flex items-center justify-between gap-4">
              <p className="text-sm font-medium text-foreground">
                {exportStatus === 'completed'
                  ? '导出完成'
                  : exportStatus === 'failed'
                    ? '导出失败'
                    : '正在导出 PDF...'}
              </p>
              <span className="text-xs font-semibold text-primary">
                {Math.round(exportProgress)}%
              </span>
            </div>
            <div className="mt-3 h-2 overflow-hidden rounded-full bg-muted">
              <div
                className="h-full rounded-full bg-primary transition-all duration-300"
                style={{ width: `${Math.max(0, Math.min(exportProgress, 100))}%` }}
              />
            </div>
            <p className="mt-3 text-xs leading-5 text-muted-foreground">
              {exportError || exportMessage || '正在准备导出任务，请稍候。'}
            </p>
          </div>
        </ModalBody>
      </Modal>
      <Modal
        open={pendingCloseAsset !== null}
        onClose={() => setPendingCloseAsset(null)}
        maxWidth="max-w-lg"
        className="asset-unsaved-dialog"
      >
        <ModalHeader>
          <div className="flex items-center gap-3">
            <span className="asset-unsaved-dialog-icon">
              <FileText className="h-4 w-4" />
            </span>
            <span>保存素材改动？</span>
          </div>
        </ModalHeader>
        <ModalBody>
          <p className="text-sm leading-6 text-muted-foreground">
            这个栏目里有尚未保存的解析文本。关闭前可以保存到素材库，也可以放弃这次修改。
          </p>
        </ModalBody>
        <ModalFooter className="items-center justify-between">
          <Button
            variant="ghost"
            type="button"
            onClick={() => setPendingCloseAsset(null)}
          >
            继续编辑
          </Button>
          <div className="flex gap-2">
            <Button
              variant="secondary"
              type="button"
              disabled={assetSavingId === pendingCloseAsset?.id}
              onClick={() => {
                if (!pendingCloseAsset) return
                finalizeCloseAssetTab(pendingCloseAsset)
              }}
            >
              放弃修改
            </Button>
            <Button
              type="button"
              disabled={assetSavingId === pendingCloseAsset?.id}
              onClick={() => void saveAndClosePendingAsset()}
            >
              {assetSavingId === pendingCloseAsset?.id ? '保存中...' : '保存并关闭'}
            </Button>
          </div>
        </ModalFooter>
      </Modal>
      <ToastContainer toasts={toasts} onDismiss={dismissToast} />
    </div>
  )
}
