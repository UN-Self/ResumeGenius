import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useEffect } from 'react'
import { useEditor, type Editor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import TextAlign from '@tiptap/extension-text-align'
import { TextStyleKit } from '@tiptap/extension-text-style'
import { TipTapEditor } from '@/components/editor/TipTapEditor'
import { FormatToolbar } from '@/components/editor/FormatToolbar'
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

// Mock editor instance for FormatToolbar tests
const createMockEditor = () => {
  const runMock = vi.fn()
  const listeners = new Map<string, Set<() => void>>()
  const focusMock = () => ({
    toggleBold: () => ({ run: runMock }),
    toggleItalic: () => ({ run: runMock }),
    toggleUnderline: () => ({ run: runMock }),
    toggleBulletList: () => ({ run: runMock }),
    toggleOrderedList: () => ({ run: runMock }),
    setTextAlign: () => ({ run: runMock }),
    setFontFamily: () => ({ run: runMock }),
    setFontSize: () => ({ run: runMock }),
    setColor: () => ({ run: runMock }),
    setBackgroundColor: () => ({ run: runMock }),
    setLineHeight: () => ({ run: runMock }),
    unsetFontFamily: () => ({ run: runMock }),
    unsetFontSize: () => ({ run: runMock }),
    unsetColor: () => ({ run: runMock }),
    unsetBackgroundColor: () => ({ run: runMock }),
    unsetLineHeight: () => ({ run: runMock }),
  })
  const mock = {
    chain: () => ({ focus: focusMock }),
    isActive: (name: string, attrs?: Record<string, unknown>) => {
      if (name === 'bold') return false
      if (name === 'italic') return false
      if (name === 'underline') return false
      if (name === 'bulletList') return false
      if (name === 'orderedList') return false
      if (name === 'textAlign') {
        if (attrs?.textAlign === 'left') return false
        if (attrs?.textAlign === 'center') return false
        if (attrs?.textAlign === 'right') return false
        if (attrs?.textAlign === 'justify') return false
        return false
      }
      return false
    },
    on: vi.fn((event: string, cb: () => void) => {
      if (!listeners.has(event)) listeners.set(event, new Set())
      listeners.get(event)!.add(cb)
    }),
    off: vi.fn((event: string, cb: () => void) => {
      listeners.get(event)?.delete(cb)
    }),
    getAttributes: vi.fn(() => ({})),
  }
  // Attach runMock for testing
  ;(mock as any).runMock = runMock
  return mock as any
}

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

describe('FormatToolbar', () => {
  it('renders format toolbar buttons', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor()
    render(<FormatToolbar editor={mockEditor} />)

    // Check for bold button
    const boldButton = screen.getByRole('button', { name: /粗体/i })
    expect(boldButton).toBeInTheDocument()

    // Check for italic button
    const italicButton = screen.getByRole('button', { name: /斜体/i })
    expect(italicButton).toBeInTheDocument()
  })

  it('toggles bold when bold button is clicked', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor()

    render(<FormatToolbar editor={mockEditor} />)

    const boldButton = screen.getByRole('button', { name: /粗体/i })
    await user.click(boldButton)

    expect(mockEditor.runMock).toHaveBeenCalled()
  })

  it('toggles italic when italic button is clicked', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor()

    render(<FormatToolbar editor={mockEditor} />)

    const italicButton = screen.getByRole('button', { name: /斜体/i })
    await user.click(italicButton)

    expect(mockEditor.runMock).toHaveBeenCalled()
  })

  it('renders all toolbar buttons', () => {
    const mockEditor = createMockEditor()
    render(<FormatToolbar editor={mockEditor} />)

    // Check all expected buttons exist
    expect(screen.getByRole('button', { name: /粗体/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /斜体/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /下划线/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /无序列表/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /有序列表/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /左对齐/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /居中/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /右对齐/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /两端对齐/i })).toBeInTheDocument()
  })

  it('renders all typography selectors and toolbar buttons', () => {
    const mockEditor = createMockEditor()
    render(<FormatToolbar editor={mockEditor} />)

    // Font selector - shows "字体" text
    expect(screen.getByRole('button', { name: /^字体$/ })).toBeInTheDocument()

    // Font size selector - shows the current size number (default "12")
    expect(screen.getByRole('button', { name: '12' })).toBeInTheDocument()

    // Color picker buttons
    expect(screen.getByRole('button', { name: /字体颜色/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /背景高亮/ })).toBeInTheDocument()

    // Line height selector - has aria-label
    expect(screen.getByRole('button', { name: /行高/ })).toBeInTheDocument()

    // Alignment (including right-align)
    expect(screen.getByRole('button', { name: /右对齐/i })).toBeInTheDocument()
  })

  it('returns null when editor is not provided', () => {
    render(<FormatToolbar editor={null} />)
    expect(screen.queryByRole('group')).not.toBeInTheDocument()
  })
})

// Integration test wrapper: real editor + FormatToolbar
function TestToolbarWrapper({ content, onEditor }: { content: string; onEditor?: (editor: Editor) => void }) {
  const editor = useEditor({
    extensions: [
      StarterKit,
      TextAlign.configure({ types: ['heading', 'paragraph'] }),
      TextStyleKit,
    ],
    content,
  })

  useEffect(() => {
    if (editor && onEditor) onEditor(editor)
  }, [editor, onEditor])

  if (!editor) return null

  return <FormatToolbar editor={editor} />
}

describe('FormatToolbar integration (real editor)', () => {
  it('updates bullet list active state when list is toggled', async () => {
    let editorRef: Editor | null = null

    render(
      <TestToolbarWrapper
        content="<p>List item</p>"
        onEditor={(e) => { editorRef = e }}
      />,
    )

    await screen.findByRole('button', { name: /无序列表/i })

    act(() => {
      editorRef!.chain().focus().setTextSelection(1).toggleBulletList().run()
    })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /无序列表/i })).toHaveAttribute('aria-pressed', 'true')
    })
  })

})
