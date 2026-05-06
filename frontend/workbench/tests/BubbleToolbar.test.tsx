import { describe, it, expect, vi } from 'vitest'
import { render, screen, act, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BubbleToolbar } from '@/components/editor/BubbleToolbar'
import { createMockEditor } from './helpers/mock-editor'

const createBubbleToolbarMockEditor = (overrides: {
  isActive?: (name: string, attrs?: Record<string, unknown>) => boolean
} = {}) => {
  return createMockEditor({
    chainCommands: [
      'toggleBold', 'toggleItalic', 'toggleUnderline',
      'toggleBulletList', 'toggleOrderedList', 'setTextAlign',
      'setFontFamily', 'setFontSize', 'setColor', 'setBackgroundColor',
      'setLineHeight', 'unsetFontFamily', 'unsetColor', 'unsetBackgroundColor',
    ],
    isActive: overrides.isActive ?? (() => false),
  })
}

describe('BubbleToolbar', () => {
  it('renders all toolbar buttons', () => {
    const mockEditor = createBubbleToolbarMockEditor()
    render(<BubbleToolbar editor={mockEditor} />)

    expect(screen.getByRole('button', { name: /粗体/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /斜体/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /下划线/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /无序列表/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /有序列表/i })).toBeInTheDocument()
  })

  it('renders typography selectors', () => {
    const mockEditor = createBubbleToolbarMockEditor()
    render(<BubbleToolbar editor={mockEditor} />)

    expect(screen.getByRole('button', { name: /^字体$/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '12' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /字体颜色/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /背景高亮/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /行距/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /对齐方式/ })).toBeInTheDocument()
  })

  it('calls toggleBold when bold button is clicked', async () => {
    const user = userEvent.setup()
    const mockEditor = createBubbleToolbarMockEditor()
    render(<BubbleToolbar editor={mockEditor} />)

    await user.click(screen.getByRole('button', { name: /粗体/i }))
    expect(mockEditor.runMock).toHaveBeenCalled()
  })

  it('calls toggleBulletList when list button is clicked', async () => {
    const user = userEvent.setup()
    const mockEditor = createBubbleToolbarMockEditor()
    render(<BubbleToolbar editor={mockEditor} />)

    await user.click(screen.getByRole('button', { name: /无序列表/i }))
    expect(mockEditor.runMock).toHaveBeenCalled()
  })

  it('has proper ARIA groups with labels', () => {
    const mockEditor = createBubbleToolbarMockEditor()
    render(<BubbleToolbar editor={mockEditor} />)

    expect(screen.getByRole('group', { name: '字体和字号' })).toBeInTheDocument()
    expect(screen.getByRole('group', { name: '文本格式' })).toBeInTheDocument()
    expect(screen.getByRole('group', { name: '列表格式' })).toBeInTheDocument()
    expect(screen.getByRole('group', { name: '行距和对齐' })).toBeInTheDocument()
  })

  it('initializes activeStates eagerly (no null flash)', () => {
    // BubbleToolbar uses lazy initializer: useState(() => getActiveStates(editor))
    // so it should render immediately with button states, unlike FormatToolbar
    // which starts with null and returns null on first render.
    const mockEditor = createBubbleToolbarMockEditor({
      isActive: (name: string) => name === 'bold',
    })
    const { container } = render(<BubbleToolbar editor={mockEditor} />)

    // Should render content immediately (not empty from null guard)
    expect(container.innerHTML).not.toBe('')
    expect(screen.getByRole('button', { name: /粗体/i })).toBeInTheDocument()
  })

  it('registers transaction listener on mount and cleans up on unmount', () => {
    const mockEditor = createBubbleToolbarMockEditor()
    const { unmount } = render(<BubbleToolbar editor={mockEditor} />)

    expect(mockEditor.on).toHaveBeenCalledWith('transaction', expect.any(Function))
    const registeredCb = mockEditor.on.mock.calls[0][1]

    unmount()

    expect(mockEditor.off).toHaveBeenCalledWith('transaction', registeredCb)
  })

  describe('activeStates synchronization', () => {
    it('updates bold aria-pressed when editor transaction changes active state', async () => {
      const mockEditor = createBubbleToolbarMockEditor({
        isActive: () => false,
      })
      render(<BubbleToolbar editor={mockEditor} />)

      const boldButton = screen.getByRole('button', { name: /粗体/i })
      expect(boldButton).toHaveAttribute('aria-pressed', 'false')

      // Simulate user toggling bold on
      mockEditor.isActive = (name: string) => name === 'bold'
      act(() => {
        mockEditor.simulateTransaction()
      })

      await waitFor(() => {
        expect(boldButton).toHaveAttribute('aria-pressed', 'true')
      })
    })

    it('updates multiple states simultaneously on transaction', async () => {
      const mockEditor = createBubbleToolbarMockEditor({
        isActive: () => false,
      })
      render(<BubbleToolbar editor={mockEditor} />)

      const boldButton = screen.getByRole('button', { name: /粗体/i })
      const listButton = screen.getByRole('button', { name: /无序列表/i })
      expect(boldButton).toHaveAttribute('aria-pressed', 'false')
      expect(listButton).toHaveAttribute('aria-pressed', 'false')

      // Simulate both bold and bulletList becoming active
      mockEditor.isActive = (name: string) =>
        name === 'bold' || name === 'bulletList'
      act(() => {
        mockEditor.simulateTransaction()
      })

      await waitFor(() => {
        expect(boldButton).toHaveAttribute('aria-pressed', 'true')
        expect(listButton).toHaveAttribute('aria-pressed', 'true')
      })
    })
  })
})
