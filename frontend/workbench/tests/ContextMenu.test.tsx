import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMockEditor } from './helpers/mock-editor'

// Import will fail until the component is created — that is expected in TDD red phase.
import { ContextMenu } from '@/components/editor/ContextMenu'

/**
 * Helper: create a mock editor suitable for ContextMenu tests.
 *
 * ContextMenu needs:
 *   - editor.can().undo() / editor.can().redo()
 *   - editor.chain().focus().undo().run() / redo().run() / insertContent().run()
 *   - editor.state.selection.from / .to
 *   - editor.view.state.doc.textBetween(from, to, '\n')
 */
function createContextMenuMockEditor(overrides: {
  canUndo?: boolean
  canRedo?: boolean
  selectionFrom?: number
  selectionTo?: number
} = {}) {
  const {
    canUndo = true,
    canRedo = true,
    selectionFrom = 0,
    selectionTo = 5,
  } = overrides

  const runMock = vi.fn()

  const editor = createMockEditor({
    chainCommands: ['undo', 'redo', 'insertContent', 'selectAll', 'deleteSelection'],
  })

  // editor.can() returns an object with undo() and redo() returning booleans
  ;(editor as any).can = vi.fn(() => ({
    undo: vi.fn(() => canUndo),
    redo: vi.fn(() => canRedo),
  }))

  // editor.state.selection
  ;(editor as any).state = {
    selection: {
      from: selectionFrom,
      to: selectionTo,
    },
  }

  // editor.view.state.doc.textBetween for plain-text extraction
  ;(editor as any).view = {
    state: {
      doc: {
        textBetween: vi.fn(() => 'selected text'),
      },
    },
  }

  return editor as any
}

describe('ContextMenu', () => {
  const onClose = vi.fn()

  beforeEach(() => {
    onClose.mockClear()
    vi.stubGlobal('navigator', {
      clipboard: {
        writeText: vi.fn(() => Promise.resolve()),
        readText: vi.fn(() => Promise.resolve('clipboard text')),
      },
    })
  })

  it('renders nothing when isOpen=false', () => {
    const editor = createContextMenuMockEditor()
    const { container } = render(
      <ContextMenu editor={editor} isOpen={false} x={100} y={200} onClose={onClose} />,
    )
    expect(container.innerHTML).toBe('')
  })

  it('renders all 6 menu items when isOpen=true', () => {
    const editor = createContextMenuMockEditor()
    render(
      <ContextMenu editor={editor} isOpen={true} x={100} y={200} onClose={onClose} />,
    )

    expect(screen.getByText('撤销')).toBeInTheDocument()
    expect(screen.getByText('重做')).toBeInTheDocument()
    expect(screen.getByText('剪切')).toBeInTheDocument()
    expect(screen.getByText('复制')).toBeInTheDocument()
    expect(screen.getByText('粘贴')).toBeInTheDocument()
    expect(screen.getByText('全选')).toBeInTheDocument()
  })

  it('shows shortcut hints (Ctrl+Z, Ctrl+Y, etc.)', () => {
    const editor = createContextMenuMockEditor()
    render(
      <ContextMenu editor={editor} isOpen={true} x={100} y={200} onClose={onClose} />,
    )

    expect(screen.getByText('Ctrl+Z')).toBeInTheDocument()
    expect(screen.getByText('Ctrl+Y')).toBeInTheDocument()
    expect(screen.getByText('Ctrl+X')).toBeInTheDocument()
    expect(screen.getByText('Ctrl+C')).toBeInTheDocument()
    expect(screen.getByText('Ctrl+V')).toBeInTheDocument()
    expect(screen.getByText('Ctrl+A')).toBeInTheDocument()
  })

  it('calls onClose when a menu item is clicked', async () => {
    const user = userEvent.setup()
    const editor = createContextMenuMockEditor()
    render(
      <ContextMenu editor={editor} isOpen={true} x={100} y={200} onClose={onClose} />,
    )

    // Click 全选 (Select All) — it's always enabled
    const selectAllItem = screen.getByRole('menuitem', { name: /全选/ })
    await user.click(selectAllItem)

    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('calls editor.chain().focus().undo().run() when 撤销 is clicked', async () => {
    const user = userEvent.setup()
    const editor = createContextMenuMockEditor()
    render(
      <ContextMenu editor={editor} isOpen={true} x={100} y={200} onClose={onClose} />,
    )

    const undoItem = screen.getByRole('menuitem', { name: /撤销/ })
    await user.click(undoItem)

    expect(editor.runMock).toHaveBeenCalled()
  })

  it('positions menu at given coordinates', () => {
    const editor = createContextMenuMockEditor()
    render(
      <ContextMenu editor={editor} isOpen={true} x={150} y={300} onClose={onClose} />,
    )

    const menu = screen.getByRole('menu')
    expect(menu).toHaveStyle({ left: '150px', top: '300px' })
  })

  it('has separators between groups', () => {
    const editor = createContextMenuMockEditor()
    render(
      <ContextMenu editor={editor} isOpen={true} x={100} y={200} onClose={onClose} />,
    )

    // There should be 2 separators (between undo/redo group and cut/copy/paste group,
    // and between cut/copy/paste group and select all)
    const menu = screen.getByRole('menu')
    const separators = menu.querySelectorAll('[role="separator"]')
    expect(separators.length).toBe(2)
  })

  it('disables 撤销 when editor.can().undo() returns false', () => {
    const editor = createContextMenuMockEditor({ canUndo: false })
    render(
      <ContextMenu editor={editor} isOpen={true} x={100} y={200} onClose={onClose} />,
    )

    const undoItem = screen.getByRole('menuitem', { name: /撤销/ })
    // Disabled items should have the disabled attribute or aria-disabled
    expect(undoItem).toHaveAttribute('aria-disabled', 'true')
  })

  it('disables 剪切 and 复制 when there is no selection', () => {
    const editor = createContextMenuMockEditor({ selectionFrom: 5, selectionTo: 5 })
    render(
      <ContextMenu editor={editor} isOpen={true} x={100} y={200} onClose={onClose} />,
    )

    const cutItem = screen.getByRole('menuitem', { name: /剪切/ })
    const copyItem = screen.getByRole('menuitem', { name: /复制/ })

    expect(cutItem).toHaveAttribute('aria-disabled', 'true')
    expect(copyItem).toHaveAttribute('aria-disabled', 'true')
  })

  it('has role="menu" on the container and role="menuitem" on items', () => {
    const editor = createContextMenuMockEditor()
    render(
      <ContextMenu editor={editor} isOpen={true} x={100} y={200} onClose={onClose} />,
    )

    const menu = screen.getByRole('menu')
    expect(menu).toBeInTheDocument()

    const menuItems = screen.getAllByRole('menuitem')
    expect(menuItems.length).toBe(6)
  })
})
