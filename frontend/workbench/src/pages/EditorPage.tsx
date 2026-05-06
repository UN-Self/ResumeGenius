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
import { Deletion, Insertion } from '@/components/editor/extensions/ai-diff'
import { request, intakeApi, workbenchApi, ApiError, type Asset, type PendingEdit } from '@/lib/api-client'
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
      Deletion,
      Insertion,
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

  // Save editor content before preview so we can restore it on exit
  const savedContentBeforePreview = useRef<string | null>(null)
  const didRecenterAfterPreviewRef = useRef(false)

  // When preview HTML is ready, load it into the editor and make read-only
  useLayoutEffect(() => {
    if (previewMode === 'previewing' && previewHtml && editor) {
      editor.commands.setContent(previewHtml)
      editor.setEditable(false)
      didRecenterAfterPreviewRef.current = false
    }
  }, [previewMode, previewHtml, editor])

  const handleStartPreview = useCallback(async (version: Parameters<typeof startPreview>[0]) => {
    if (editor) {
      savedContentBeforePreview.current = editor.getHTML()
    }
    await startPreview(version)
  }, [editor, startPreview])

  const handleExitPreview = useCallback(() => {
    if (editor) {
      editor.setEditable(true)
      const saved = savedContentBeforePreview.current
      if (saved) {
        editor.commands.setContent(saved)
      }
      savedContentBeforePreview.current = null
    }
    exitPreview()
  }, [editor, exitPreview])

  const reloadAssets = useCallback(async () => {
    const nextAssets = await intakeApi.listAssets(pid)
    setAssets(nextAssets)
  }, [pid])

  const handleExport = () => {
    if (draftId && editor) {
      exportPdf(Number(draftId), editor.getHTML())
    }
  }

  const handleRollback = async () => {
    try {
      const html = await rollbackVersion()
      editor?.commands.setContent(html)
      setRollbackDialogOpen(false)
    } catch (e) {
      console.error('Rollback failed:', e)
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
          navigate(`/projects/${pid}`, { replace: true })
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
      editor.commands.setContent(pendingHtml)
    }
  }, [editor, pendingHtml])

  useEffect(() => {
    if (!editor) return

    const handleUpdate = () => scheduleSave(editor.getHTML())
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
            <A4Canvas editor={editor} />
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
              onApplyDiffHTML={(edits: PendingEdit[]) => {
                if (!editor) return
                const currentHTML = editor.getHTML()
                let diffHTML = currentHTML
                for (const edit of edits) {
                  diffHTML = diffHTML.replace(
                    edit.old_string,
                    `<del>${edit.old_string}</del><ins>${edit.new_string}</ins>`
                  )
                }
                editor.commands.setContent(diffHTML)
              }}
              onRestoreHtml={(html) => editor?.commands.setContent(html)}
            />
          ) : (
            <p className="mt-8 text-center text-xs text-[var(--color-text-secondary)]">
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
          } catch {
            // error is handled by useVersions
          } finally {
            setSavingSnapshot(false)
          }
        }}
      />
      <RollbackConfirmDialog
        open={rollbackDialogOpen}
        onClose={() => setRollbackDialogOpen(false)}
        onConfirm={handleRollback}
      />
    </div>
  )
}
