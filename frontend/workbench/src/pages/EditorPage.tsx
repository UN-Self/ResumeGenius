import { useEffect, useState, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useEditor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import TextAlign from '@tiptap/extension-text-align'
import { TextStyleKit } from '@tiptap/extension-text-style'
import { A4Canvas } from '@/components/editor/A4Canvas'
import { ActionBar } from '@/components/editor/ActionBar'
import { FormatToolbar } from '@/components/editor/FormatToolbar'
import { SaveIndicator } from '@/components/editor/SaveIndicator'
import { AiPanelPlaceholder } from '@/components/editor/AiPanelPlaceholder'
import ParsedSidebar from '@/components/intake/ParsedSidebar'
import { request, intakeApi, parsingApi, ApiError, type ParsedContent } from '@/lib/api-client'
import { useAutoSave } from '@/hooks/useAutoSave'
import { useExport } from '@/hooks/useExport'
import { FullPageState } from '@/components/ui/full-page-state'
import type { Draft } from '@/types/editor'


export default function EditorPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const pid = Number(projectId)

  // Route guard + data loading
  const [draftId, setDraftId] = useState<string | null>(null)
  const [projectTitle, setProjectTitle] = useState('')
  const [parsedContents, setParsedContents] = useState<ParsedContent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Panel collapse state — both open by default, user can collapse manually
  const [leftOpen, setLeftOpen] = useState(true)
  const [rightOpen, setRightOpen] = useState(true)

  // TipTap editor
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

  // Bridge state: loaded draft HTML waiting for editor to be ready
  const [pendingHtml, setPendingHtml] = useState<string | null>(null)

  // Auto-save
  const { scheduleSave, flush, retry, status, lastSavedAt } = useAutoSave({
    save: async (html: string) => {
      if (draftId) {
        await request(`/drafts/${draftId}`, { method: 'PUT', body: JSON.stringify({ html_content: html }) })
      }
    },
    saveUrl: draftId ? `/api/v1/drafts/${draftId}` : undefined,
  })

  // Export
  const { exportPdf, status: exportStatus } = useExport()

  const handleExport = () => {
    if (draftId && editor) {
      exportPdf(Number(draftId), editor.getHTML())
    }
  }

  // Effect 1: Load project (route guard) + draft + parsed contents
  useEffect(() => {
    if (!projectId) return

    let cancelled = false

    intakeApi.getProject(pid)
      .then((project) => {
        if (cancelled) return

        // Route guard: redirect if no draft
        if (!project.current_draft_id) {
          navigate(`/projects/${pid}`, { replace: true })
          return
        }

        setProjectTitle(project.title)
        setDraftId(String(project.current_draft_id))

        // Load draft content
        return request<Draft>(`/drafts/${project.current_draft_id}`)
      })
      .then((draft) => {
        if (cancelled || !draft) return
        // Store draft HTML — Effect 2 will sync it to the editor
        setPendingHtml(draft.html_content || '')

        // Load parsed contents for left panel
        return parsingApi.parseProject(pid).catch(() => {
          // Non-blocking: empty sidebar if parsing fails
        })
      })
      .then((result) => {
        if (cancelled || !result) return
        setParsedContents(result.parsed_contents)
      })
      .catch((err) => {
        if (cancelled) return
        setError(err instanceof ApiError ? err.message : '加载失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => { cancelled = true }
  }, [projectId, navigate, pid])

  // Ref guard: prevent re-applying draft content when editor updates
  const hasAppliedRef = useRef(false)

  // Reset guard when projectId changes (navigation between projects)
  useEffect(() => {
    hasAppliedRef.current = false
  }, [projectId])

  // Effect 2: Sync pending draft HTML into editor once it's ready
  useEffect(() => {
    if (editor && pendingHtml !== null && !hasAppliedRef.current) {
      hasAppliedRef.current = true
      editor.commands.setContent(pendingHtml)
    }
  }, [editor, pendingHtml])

  // Connect editor to autosave
  useEffect(() => {
    if (!editor) return
    const handleUpdate = () => scheduleSave(editor.getHTML())
    editor.on('update', handleUpdate)
    return () => { editor.off('update', handleUpdate) }
  }, [editor, scheduleSave])

  // Flush on unmount
  useEffect(() => { return () => { flush() } }, [flush])

  if (loading) {
    return <FullPageState variant="loading" />
  }

  if (error) {
    return <FullPageState variant="error" message={error!} />
  }

  const gridClass = [
    'editor-workspace',
    leftOpen ? 'left-open' : 'left-collapsed',
    rightOpen ? 'right-open' : 'right-collapsed',
  ].join(' ')

  return (
    <div className={gridClass}>
      {/* Left Panel — Parsed Sidebar */}
      <div className="editor-panel-left">
        <div className="panel-header">
          <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            素材
          </h2>
          <button
            onClick={() => setLeftOpen(false)}
            className="panel-collapse-btn"
            aria-label="收起左面板"
          >
            ‹
          </button>
        </div>
        <div className="panel-body">
          <ParsedSidebar
            projectId={pid}
            contents={parsedContents}
            onParsed={setParsedContents}
          />
        </div>
      </div>

      {/* Center — A4 Canvas */}
      <div className="editor-panel-center">
        {!leftOpen && (
          <button
            onClick={() => setLeftOpen(true)}
            className="panel-expand-btn panel-expand-left"
            aria-label="展开左面板"
          >
            ›
          </button>
        )}
        {!rightOpen && (
          <button
            onClick={() => setRightOpen(true)}
            className="panel-expand-btn panel-expand-right"
            aria-label="展开右面板"
          >
            ‹
          </button>
        )}
        <div className="flex flex-col h-full">
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

      {/* Right Panel — AI */}
      <div className="editor-panel-right">
        <div className="panel-header">
          <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            AI 助手
          </h2>
          <button
            onClick={() => setRightOpen(false)}
            className="panel-collapse-btn"
            aria-label="收起右面板"
          >
            ›
          </button>
        </div>
        <div className="panel-body">
          <AiPanelPlaceholder />
        </div>
      </div>
    </div>
  )
}
