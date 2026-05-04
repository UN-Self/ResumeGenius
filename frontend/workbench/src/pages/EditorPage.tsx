import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useEditor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import TextAlign from '@tiptap/extension-text-align'
import { TextStyleKit } from '@tiptap/extension-text-style'
import { A4Canvas } from '@/components/editor/A4Canvas'
import { ActionBar } from '@/components/editor/ActionBar'
import { FormatToolbar } from '@/components/editor/FormatToolbar'
import { SaveIndicator } from '@/components/editor/SaveIndicator'
import { ChatPanel } from '@/components/chat/ChatPanel'
import AssetSidebar from '@/components/intake/AssetSidebar'
import { request, intakeApi, workbenchApi, ApiError, type Asset } from '@/lib/api-client'
import { useAutoSave } from '@/hooks/useAutoSave'
import { useExport } from '@/hooks/useExport'
import { FullPageState } from '@/components/ui/full-page-state'

export default function EditorPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const pid = Number(projectId)

  const [draftId, setDraftId] = useState<string | null>(null)
  const [projectTitle, setProjectTitle] = useState('')
  const [assets, setAssets] = useState<Asset[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [leftOpen, setLeftOpen] = useState(true)
  const [rightOpen, setRightOpen] = useState(true)

  const editor = useEditor({
    extensions: [
      StarterKit,
      TextAlign.configure({ types: ['heading', 'paragraph'] }),
      TextStyleKit,
    ],
    content: '',
    editorProps: {
      attributes: {
        class: 'resume-content outline-none',
        style: 'min-height: 261mm;',
      },
    },
  })

  const [pendingHtml, setPendingHtml] = useState<string | null>(null)

  const { scheduleSave, flush, retry, status, lastSavedAt } = useAutoSave({
    save: async (html: string) => {
      if (draftId) {
        await request(`/drafts/${draftId}`, {
          method: 'PUT',
          body: JSON.stringify({ html_content: html }),
        })
      }
    },
    saveUrl: draftId ? `/api/v1/drafts/${draftId}` : undefined,
  })

  const { exportPdf, status: exportStatus } = useExport()

  const reloadAssets = useCallback(async () => {
    const nextAssets = await intakeApi.listAssets(pid)
    setAssets(nextAssets)
  }, [pid])

  const handleExport = () => {
    if (draftId && editor) {
      exportPdf(Number(draftId), editor.getHTML())
    }
  }

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
          <ActionBar
            projectName={projectTitle}
            saveIndicator={<SaveIndicator status={status} lastSavedAt={lastSavedAt} onRetry={retry} />}
            draftId={draftId}
            exportStatus={exportStatus}
            onExport={handleExport}
          />
          <div className="flex-1 overflow-auto">
            <A4Canvas editor={editor} />
          </div>
          <div className="format-toolbar">
            <FormatToolbar editor={editor} />
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
              onApplyHTML={(html) => {
                editor?.commands.setContent(html)

                if (draftId) {
                  request(`/drafts/${draftId}`, {
                    method: 'PUT',
                    body: JSON.stringify({
                      html_content: html,
                      create_version: true,
                      version_label: 'AI \u4fee\u6539',
                    }),
                  }).catch((saveError) => console.error('Failed to save AI changes:', saveError))
                }
              }}
            />
          ) : (
            <p className="mt-8 text-center text-xs text-[var(--color-text-secondary)]">
              {'\u7b49\u5f85\u8349\u7a3f\u52a0\u8f7d...'}
            </p>
          )}
        </div>
      </div>
    </div>
  )
}
