import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useEditor } from '@tiptap/react'
import { BubbleMenu } from '@tiptap/react/menus'
import StarterKit from '@tiptap/starter-kit'
import TextAlign from '@tiptap/extension-text-align'
import { TextStyleKit } from '@tiptap/extension-text-style'
import { A4Canvas } from '@/components/editor/A4Canvas'
import { ActionBar } from '@/components/editor/ActionBar'
import { SaveIndicator } from '@/components/editor/SaveIndicator'
import { ChatPanel } from '@/components/chat/ChatPanel'
import AssetSidebar from '@/components/intake/AssetSidebar'
import { request, intakeApi, workbenchApi, ApiError, type Asset } from '@/lib/api-client'
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
import { extractStyles, reconstructHtml } from '@/lib/extract-styles'
import { Div, Span, PresetAttributes } from '@/components/editor/extensions'
import { usePanelState } from '@/hooks/usePanelState'
import { useContextMenuState } from '@/hooks/useContextMenuState'

export default function EditorPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const pid = Number(projectId)

  const [draftId, setDraftId] = useState<string | null>(null)
  const draftIdRef = useRef<string | null>(null)
  const [projectTitle, setProjectTitle] = useState('')
  const [assets, setAssets] = useState<Asset[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const { leftOpen, rightOpen, setLeftOpen, setRightOpen } = usePanelState()
  const { contextMenu, closeContextMenu, handleContextMenu } = useContextMenuState()

  const editor = useEditor({
    extensions: [
      StarterKit.configure({ strike: false }),
      TextAlign.configure({ types: ['heading', 'paragraph', 'div'] }),
      TextStyleKit,
      Div,
      Span,
      PresetAttributes,
    ],
    content: '',
    editorProps: {
      attributes: {
        class: 'resume-content outline-none',
        style: 'min-height: 261mm;',
      },
      handleDOMEvents: {
        copy(_view, event) {
          const { from, to } = _view.state.selection
          const plainText = _view.state.doc.textBetween(from, to, '\n')
          event.preventDefault()
          event.clipboardData?.setData('text/plain', plainText)
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
    reconstruct: (html: string) => reconstructHtml(html, rawCSSRef.current),
    saveUrl: draftId ? `/api/v1/drafts/${draftId}` : undefined,
  })

  const { exportPdf, status: exportStatus } = useExport()

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
  const savedContentBeforePreview = useRef<string | null>(null)
  const savedScopedCSSBeforePreview = useRef<string | null>(null)
  const savedRawCSSBeforePreview = useRef<string>('')
  const didRecenterAfterPreviewRef = useRef(false)
  const restoringContent = useRef(false)
  const rawCSSRef = useRef('')

  // When preview HTML is ready, load it into the editor and make read-only
  useLayoutEffect(() => {
    if (previewMode === 'previewing' && previewHtml && editor) {
      const { bodyHtml, scopedCSS: css, rawCSS } = extractStyles(previewHtml)
      rawCSSRef.current = rawCSS
      setScopedCSS(css)
      restoringContent.current = true
      editor.commands.setContent(bodyHtml)
      restoringContent.current = false
      editor.setEditable(false)
      didRecenterAfterPreviewRef.current = false
    }
  }, [previewMode, previewHtml, editor])

  const handleStartPreview = useCallback(async (version: Parameters<typeof startPreview>[0]) => {
    if (editor) {
      savedContentBeforePreview.current = editor.getHTML()
    }
    savedScopedCSSBeforePreview.current = scopedCSS
    savedRawCSSBeforePreview.current = rawCSSRef.current
    await startPreview(version)
  }, [editor, scopedCSS, startPreview])

  const handleExitPreview = useCallback(() => {
    if (editor) {
      editor.setEditable(true)
      const saved = savedContentBeforePreview.current
      if (saved) {
        restoringContent.current = true
        editor.commands.setContent(saved)
        restoringContent.current = false
      }
      savedContentBeforePreview.current = null
    }
    if (savedScopedCSSBeforePreview.current !== null) {
      setScopedCSS(savedScopedCSSBeforePreview.current)
      savedScopedCSSBeforePreview.current = null
    }
    if (savedRawCSSBeforePreview.current !== undefined) {
      rawCSSRef.current = savedRawCSSBeforePreview.current
      savedRawCSSBeforePreview.current = ''
    }
    exitPreview()
  }, [editor, exitPreview])

  const reloadAssets = useCallback(async () => {
    const nextAssets = await intakeApi.listAssets(pid)
    setAssets(nextAssets)
  }, [pid])

  const handleExport = () => {
    if (draftId) {
      exportPdf(Number(draftId))
    }
  }

  const handleRollback = async () => {
    setRollbacking(true)
    try {
      await flush()
      const html = await rollbackVersion()
      if (editor) {
        const { bodyHtml, scopedCSS: css, rawCSS } = extractStyles(html)
        rawCSSRef.current = rawCSS
        setScopedCSS(css)
        editor.setEditable(true)
        restoringContent.current = true
        editor.commands.setContent(bodyHtml)
        restoringContent.current = false
      }
      setRollbackDialogOpen(false)
    } catch (e) {
      toast(e instanceof Error ? e.message : '回滚失败')
    } finally {
      setRollbacking(false)
    }
  }

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
            const draft = await workbenchApi.createDraft(pid)
            if (cancelled) return
            setDraftId(String(draft.id))
            setPendingHtml(draft.html_content || '')
            setProjectTitle(project.title)

            const nextAssets = await intakeApi.listAssets(pid)
            if (cancelled) return
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
      const { bodyHtml, scopedCSS: css, rawCSS } = extractStyles(pendingHtml)
      rawCSSRef.current = rawCSS
      setScopedCSS(css)
      editor.commands.setContent(bodyHtml)
    }
  }, [editor, pendingHtml])

  useEffect(() => {
    if (!editor) return

    const handleUpdate = () => {
      if (!restoringContent.current) {
        scheduleSave(editor.getHTML())
      }
    }
    editor.on('update', handleUpdate)

    return () => {
      editor.off('update', handleUpdate)
    }
  }, [editor, scheduleSave])

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

  return (
    <div className={gridClass}>
      <div className="editor-panel-left">
        <div className="panel-header">
          <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            {'\u7d20\u6750'}
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
        <div className="panel-body">
          <AssetSidebar
            projectId={pid}
            assets={assets}
            onReload={reloadAssets}
          />
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
          <div className="flex-1 overflow-auto" onContextMenu={handleContextMenu}>
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
                try {
                  const draft = await workbenchApi.getDraft(Number(draftId))
                  restoringContent.current = true
                  const { bodyHtml, scopedCSS: css, rawCSS } = extractStyles(draft.html_content || '')
                  rawCSSRef.current = rawCSS
                  setScopedCSS(css)
                  editor.commands.setContent(bodyHtml)
                  restoringContent.current = false
                } catch {
                  toast('应用 AI 修改失败，请重试')
                }
              }}
              onRestoreHtml={(html) => {
                if (!editor) return
                restoringContent.current = true
                const { bodyHtml, scopedCSS: css, rawCSS } = extractStyles(html)
                rawCSSRef.current = rawCSS
                setScopedCSS(css)
                editor.commands.setContent(bodyHtml)
                restoringContent.current = false
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
      <ToastContainer toasts={toasts} onDismiss={dismissToast} />
    </div>
  )
}
