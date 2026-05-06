import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { useEditor, type Editor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import TextAlign from '@tiptap/extension-text-align'
import { TextStyleKit } from '@tiptap/extension-text-style'
import { TipTapEditor } from '@/components/editor/TipTapEditor'
import { sampleDraftHtml } from '@/mocks/fixtures'

// Test wrapper component that creates an editor with content
function TestEditorWrapper({ content, onEditor }: { content: string; onEditor?: (editor: Editor) => void }) {
  const editor = useEditor({
    extensions: [
      StarterKit,
      TextAlign.configure({ types: ['heading', 'paragraph'] }),
      TextStyleKit,
    ],
    content,
    editorProps: {
      attributes: {
        class: 'resume-content',
      },
    },
  })

  if (editor && onEditor) {
    onEditor(editor)
  }

  if (!editor) return null

  return <TipTapEditor editor={editor} />
}

import { createMockEditor } from './helpers/mock-editor'

describe('TipTapEditor', () => {
  it('renders the sample draft html', async () => {
    render(<TestEditorWrapper content={sampleDraftHtml} />)
    expect(await screen.findByText(/Sample Draft/i)).toBeInTheDocument()
  })

  it('renders with provided content', () => {
    const testContent = '<h1>Test Heading</h1><p>Test paragraph</p>'
    render(<TestEditorWrapper content={testContent} />)
    expect(screen.getByText(/Test Heading/i)).toBeInTheDocument()
    expect(screen.getByText(/Test paragraph/i)).toBeInTheDocument()
  })

  it('applies custom class name for styling', () => {
    const testContent = '<p>Test</p>'
    render(<TestEditorWrapper content={testContent} />)
    const editorEl = document.querySelector('.resume-content')
    expect(editorEl).toBeInTheDocument()
  })
})
