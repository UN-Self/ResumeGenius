import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { FontSelector } from '@/components/editor/FontSelector'

const createMockEditor = () => {
  const runMock = vi.fn()
  const listeners = new Map<string, Set<() => void>>()
  let currentFontFamily: string | null = null

  const focusMock = () => ({
    setFontFamily: () => ({ run: runMock }),
    unsetFontFamily: () => ({ run: runMock }),
  })

  const mock = {
    chain: () => ({ focus: focusMock }),
    getAttributes: vi.fn(() => ({ fontFamily: currentFontFamily })),
    on: vi.fn((event: string, cb: () => void) => {
      if (!listeners.has(event)) listeners.set(event, new Set())
      listeners.get(event)!.add(cb)
    }),
    off: vi.fn((event: string, cb: () => void) => {
      listeners.get(event)?.delete(cb)
    }),
    runMock,
  }

  // Helper to simulate transaction event
  const simulateTransaction = (fontFamily: string | null) => {
    currentFontFamily = fontFamily
    const callbacks = listeners.get('transaction')
    if (callbacks) {
      callbacks.forEach((cb) => cb())
    }
  }

  return { editor: mock as any, simulateTransaction }
}

describe('FontSelector', () => {
  it('renders trigger button with default text "字体"', () => {
    const { editor } = createMockEditor()
    render(<FontSelector editor={editor} />)

    const trigger = screen.getByRole('button', { name: /字体/ })
    expect(trigger).toBeInTheDocument()
    expect(trigger).toHaveTextContent('字体')
  })

  it('shows font list when clicked', async () => {
    const user = userEvent.setup()
    const { editor } = createMockEditor()
    render(<FontSelector editor={editor} />)

    const trigger = screen.getByRole('button', { name: /字体/ })
    await user.click(trigger)

    // Check for specific fonts in the list
    expect(screen.getByText('宋体')).toBeInTheDocument()
    expect(screen.getByText('Arial')).toBeInTheDocument()
    expect(screen.getByText('默认字体')).toBeInTheDocument()
  })

  it('calls editor command when a font is selected', async () => {
    const user = userEvent.setup()
    const { editor } = createMockEditor()
    render(<FontSelector editor={editor} />)

    const trigger = screen.getByRole('button', { name: /字体/ })
    await user.click(trigger)

    const arialOption = screen.getByText('Arial')
    await user.click(arialOption)

    expect(editor.runMock).toHaveBeenCalled()
  })

  it('calls unsetFontFamily when "默认字体" is selected', async () => {
    const user = userEvent.setup()
    const { editor } = createMockEditor()
    render(<FontSelector editor={editor} />)

    const trigger = screen.getByRole('button', { name: /字体/ })
    await user.click(trigger)

    const defaultFontOption = screen.getByText('默认字体')
    await user.click(defaultFontOption)

    expect(editor.runMock).toHaveBeenCalled()
  })

  it('displays current font name when set', async () => {
    const { editor, simulateTransaction } = createMockEditor()
    render(<FontSelector editor={editor} />)

    // Initially should show "字体"
    expect(screen.getByRole('button', { name: /字体/ })).toHaveTextContent('字体')

    // Simulate a transaction that sets the font to SimSun (宋体)
    await waitFor(() => {
      simulateTransaction('SimSun, serif')
    })

    // Should now display "宋体"
    await waitFor(() => {
      expect(screen.getByRole('button')).toHaveTextContent('宋体')
    })
  })

  it('displays Arial label when Arial font is set', async () => {
    const { editor, simulateTransaction } = createMockEditor()
    render(<FontSelector editor={editor} />)

    // Simulate setting Arial font
    await waitFor(() => {
      simulateTransaction('Arial, sans-serif')
    })

    await waitFor(() => {
      expect(screen.getByRole('button')).toHaveTextContent('Arial')
    })
  })

  it('closes popover after selecting a font', async () => {
    const user = userEvent.setup()
    const { editor } = createMockEditor()
    render(<FontSelector editor={editor} />)

    const trigger = screen.getByRole('button', { name: /字体/ })
    await user.click(trigger)

    // Popover content should be visible
    expect(screen.getByText('宋体')).toBeInTheDocument()

    const arialOption = screen.getByText('Arial')
    await user.click(arialOption)

    // After selection, the popover should close and font list should not be visible
    await waitFor(() => {
      expect(screen.queryByText('黑体')).not.toBeInTheDocument()
    })
  })

  it('renders all 8 font options', async () => {
    const user = userEvent.setup()
    const { editor } = createMockEditor()
    render(<FontSelector editor={editor} />)

    const trigger = screen.getByRole('button', { name: /字体/ })
    await user.click(trigger)

    expect(screen.getByText('默认字体')).toBeInTheDocument()
    expect(screen.getByText('宋体')).toBeInTheDocument()
    expect(screen.getByText('黑体')).toBeInTheDocument()
    expect(screen.getByText('楷体')).toBeInTheDocument()
    expect(screen.getByText('仿宋')).toBeInTheDocument()
    expect(screen.getByText('Times New Roman')).toBeInTheDocument()
    expect(screen.getByText('Arial')).toBeInTheDocument()
    expect(screen.getByText('Georgia')).toBeInTheDocument()
  })
})
