import { useEffect, useState, useCallback } from 'react'
import { useParams } from 'react-router-dom'
import { useEditor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Underline from '@tiptap/extension-underline'
import TextAlign from '@tiptap/extension-text-align'
import { WorkbenchLayout } from '@/components/editor/WorkbenchLayout'
import { A4Canvas } from '@/components/editor/A4Canvas'
import { FormatToolbar } from '@/components/editor/FormatToolbar'
import { SaveIndicator } from '@/components/editor/SaveIndicator'
import { EditorErrorState } from '@/components/editor/EditorErrorState'
import { EditorEmptyState } from '@/components/editor/EditorEmptyState'
import { EditorSkeleton } from '@/components/editor/EditorSkeleton'
import { request } from '@/lib/api-client'
import { useAutoSave } from '@/hooks/useAutoSave'
import type { Draft, EditorState } from '@/types/editor'

const DRAFT_ID = '1' // For now, use a fixed draft ID

export default function EditorPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const [state, setState] = useState<EditorState>('loading')
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  const editor = useEditor({
    extensions: [
      StarterKit,
      Underline,
      TextAlign.configure({ types: ['heading', 'paragraph'] }),
    ],
    content: '',
    editorProps: {
      attributes: {
        class: 'resume-content outline-none',
        style: 'min-height: 261mm;',
      },
    },
  })

  // Auto-save hook
  const { scheduleSave, flush, retry, status, lastSavedAt } = useAutoSave({
    save: async (html: string) => {
      await request(`/drafts/${DRAFT_ID}`, { method: 'PUT', body: JSON.stringify({ html_content: html }) })
    },
    saveUrl: `/api/v1/drafts/${DRAFT_ID}`,
  })

  // Connect editor onUpdate to autosave
  useEffect(() => {
    if (!editor) return

    const handleUpdate = () => {
      scheduleSave(editor.getHTML())
    }

    editor.on('update', handleUpdate)
    return () => {
      editor.off('update', handleUpdate)
    }
  }, [editor, scheduleSave])

  // Cleanup: flush pending saves on unmount
  useEffect(() => {
    return () => {
      flush()
    }
  }, [flush])

  const loadDraft = useCallback(() => {
    setState('loading')
    request<Draft>(`/drafts/${DRAFT_ID}`)
      .then((data) => {
        if (editor && data.html_content) {
          editor.commands.setContent(data.html_content)
        }
        if (!data.html_content || data.html_content.trim() === '') {
          setState('empty')
        } else {
          setState('ready')
        }
      })
      .catch((err) => {
        console.error('Failed to load draft:', err)
        setErrorMessage(err instanceof Error ? err.message : 'Failed to load draft')
        setState('error')
      })
  }, [editor])

  useEffect(() => {
    loadDraft()
  }, [loadDraft])

  const renderContent = () => {
    switch (state) {
      case 'loading':
        return (
          <A4Canvas>
            <EditorSkeleton />
          </A4Canvas>
        )
      case 'empty':
        return (
          <A4Canvas>
            <EditorEmptyState />
          </A4Canvas>
        )
      case 'ready':
        return <A4Canvas editor={editor} />
      case 'error':
        return (
          <A4Canvas>
            <EditorErrorState
              message="加载失败"
              detail={errorMessage || undefined}
              onRetry={loadDraft}
            />
          </A4Canvas>
        )
      default:
        return null
    }
  }

  return (
    <WorkbenchLayout
      projectName={`Project ${projectId}`}
      toolbar={editor ? <FormatToolbar editor={editor} /> : null}
      saveIndicator={<SaveIndicator status={status} lastSavedAt={lastSavedAt} onRetry={retry} />}
    >
      {renderContent()}
    </WorkbenchLayout>
  )
}
