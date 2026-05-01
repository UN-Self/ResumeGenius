import { useEffect, useRef, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useEditor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import TextAlign from '@tiptap/extension-text-align'
import { TextStyleKit } from '@tiptap/extension-text-style'
import { A4Canvas } from '@/components/editor/A4Canvas'
import { ActionBar } from '@/components/editor/ActionBar'
import { FormatToolbar } from '@/components/editor/FormatToolbar'
import { SaveIndicator } from '@/components/editor/SaveIndicator'
import { ChatPanel } from '@/components/chat/ChatPanel'
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

  const [draftId, setDraftId] = useState<string | null>(null)
  const [projectTitle, setProjectTitle] = useState('')
  const [parsedContents, setParsedContents] = useState<ParsedContent[]>([])
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
        await request(`/drafts/${draftId}`, { method: 'PUT', body: JSON.stringify({ html_content: html }) })
      }
    },
    saveUrl: draftId ? `/api/v1/drafts/${draftId}` : undefined,
  })

  const { exportPdf, status: exportStatus } = useExport()

  const handleExport = () => {
    if (draftId && editor) {
      exportPdf(Number(draftId), editor.getHTML())
    }
  }

  useEffect(() => {
    if (!projectId) return

    let cancelled = false

    intakeApi.getProject(pid)
      .then((project) => {
        if (cancelled) return

        if (!project.current_draft_id) {
          navigate(`/projects/${pid}`, { replace: true })
          return
        }

        setProjectTitle(project.title)
        setDraftId(String(project.current_draft_id))

        return request<Draft>(`/drafts/${project.current_draft_id}`)
      })
      .then((draft) => {
        if (cancelled || !draft) return

        setPendingHtml(draft.html_content || '')

        return parsingApi.parseProject(pid).catch(() => {
          // Keep the editor usable even if parsing fails.
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
            素材
          </h2>
          <button
            type="button"
            onClick={() => setLeftOpen(false)}
            className="panel-collapse-btn"
            aria-label="收起左面板"
          >
            {'<'}
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

      <div className="editor-panel-center">
        {!leftOpen && (
          <button
            type="button"
            onClick={() => setLeftOpen(true)}
            className="panel-expand-btn panel-expand-left"
            aria-label="展开左面板"
          >
            {'>'}
          </button>
        )}
        {!rightOpen && (
          <button
            type="button"
            onClick={() => setRightOpen(true)}
            className="panel-expand-btn panel-expand-right"
            aria-label="展开右面板"
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
            AI 助手
          </h2>
          <button
            type="button"
            onClick={() => setRightOpen(false)}
            className="panel-collapse-btn"
            aria-label="收起右面板"
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
                      version_label: 'AI 修改',
                    }),
                  }).catch((err) => console.error('Failed to save AI changes:', err))
                }
              }}
            />
          ) : (
            <p className="mt-8 text-center text-xs text-[var(--color-text-secondary)]">
              等待草稿加载...
            </p>
          )}
        </div>
      </div>
    </div>
  )
}
