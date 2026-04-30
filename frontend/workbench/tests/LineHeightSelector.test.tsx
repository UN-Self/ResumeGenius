import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { LineHeightSelector } from '@/components/editor/LineHeightSelector'

const createMockEditor = () => {
  const runMock = vi.fn()
  const focusMock = () => ({
    setLineHeight: () => ({ run: runMock }),
    unsetLineHeight: () => ({ run: runMock }),
  })
  const listeners = new Map<string, Set<() => void>>()
  return {
    chain: () => ({ focus: focusMock }),
    getAttributes: vi.fn(() => ({ lineHeight: null })),
    on: vi.fn((event: string, cb: () => void) => {
      if (!listeners.has(event)) listeners.set(event, new Set())
      listeners.get(event)!.add(cb)
    }),
    off: vi.fn(),
    runMock,
  } as any
}

describe('LineHeightSelector', () => {
  it('renders trigger button', () => {
    const mockEditor = createMockEditor()
    render(<LineHeightSelector editor={mockEditor} />)

    const trigger = screen.getByRole('button', { name: /行高/i })
    expect(trigger).toBeInTheDocument()
    expect(trigger).toHaveTextContent('—')
  })

  it('shows line height options when clicked', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor()
    render(<LineHeightSelector editor={mockEditor} />)

    const trigger = screen.getByRole('button', { name: /行高/i })
    await user.click(trigger)

    // Check that line height options are present
    expect(screen.getByText('1.0')).toBeInTheDocument()
    expect(screen.getByText('1.15')).toBeInTheDocument()
    expect(screen.getByText('1.5')).toBeInTheDocument()
    expect(screen.getByText('1.75')).toBeInTheDocument()
    expect(screen.getByText('2.0')).toBeInTheDocument()
    expect(screen.getByText('2.5')).toBeInTheDocument()
  })

  it('calls editor command when an option is selected', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor()
    render(<LineHeightSelector editor={mockEditor} />)

    const trigger = screen.getByRole('button', { name: /行高/i })
    await user.click(trigger)

    const option1_5 = screen.getByText('1.5')
    await user.click(option1_5)

    expect(mockEditor.runMock).toHaveBeenCalled()
  })

  it('displays current line height when set', async () => {
    const mockEditor = createMockEditor()
    mockEditor.getAttributes = vi.fn(() => ({ lineHeight: '1.5' }))

    render(<LineHeightSelector editor={mockEditor} />)

    await waitFor(() => {
      const trigger = screen.getByRole('button', { name: /行高/i })
      expect(trigger).toHaveTextContent('1.5')
    })
  })

  it('highlights active line height option', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor()
    mockEditor.getAttributes = vi.fn(() => ({ lineHeight: '2.0' }))

    render(<LineHeightSelector editor={mockEditor} />)

    const trigger = screen.getByRole('button', { name: /行高/i })
    await user.click(trigger)

    // Use getAllByText and check the button (not the trigger span)
    const allElements = screen.getAllByText('2.0')
    const buttonOption = allElements.find(el => el.tagName === 'BUTTON')
    expect(buttonOption).toBeInTheDocument()
    expect(buttonOption).toHaveClass('bg-[var(--color-primary-bg)]', 'text-[var(--color-primary)]')
  })

  it('updates display when line height changes via transaction', async () => {
    let mockAttrs = { lineHeight: null }
    const mockEditor = createMockEditor()
    mockEditor.getAttributes = vi.fn(() => mockAttrs)

    render(<LineHeightSelector editor={mockEditor} />)

    // Initially should show "—"
    expect(screen.getByRole('button', { name: /行高/i })).toHaveTextContent('—')

    // Simulate transaction updating line height
    mockAttrs = { lineHeight: '1.75' }
    const onCallback = mockEditor.on.mock.calls.find(
      (call: any[]) => call[0] === 'transaction'
    )
    if (onCallback && onCallback[1]) {
      onCallback[1]()
    }

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /行高/i })).toHaveTextContent('1.75')
    })
  })
})
