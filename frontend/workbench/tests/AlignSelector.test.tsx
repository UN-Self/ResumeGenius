import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AlignSelector } from '@/components/editor/AlignSelector'
import { createMockEditor } from './helpers/mock-editor'

describe('AlignSelector', () => {
  it('renders trigger button with default icon', () => {
    const mockEditor = createMockEditor({
      chainCommands: ['setTextAlign'],
    })
    render(<AlignSelector editor={mockEditor} />)

    const trigger = screen.getByRole('button', { name: /对齐/ })
    expect(trigger).toBeInTheDocument()
  })

  it('shows alignment options when clicked', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setTextAlign'],
    })
    render(<AlignSelector editor={mockEditor} />)

    await user.click(screen.getByRole('button', { name: /对齐/ }))

    expect(screen.getByText('左对齐')).toBeInTheDocument()
    expect(screen.getByText('居中')).toBeInTheDocument()
    expect(screen.getByText('右对齐')).toBeInTheDocument()
    expect(screen.getByText('两端对齐')).toBeInTheDocument()
  })

  it('calls setTextAlign when an option is selected', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setTextAlign'],
    })
    render(<AlignSelector editor={mockEditor} />)

    await user.click(screen.getByRole('button', { name: /对齐/ }))
    await user.click(screen.getByText('居中'))

    expect(mockEditor.runMock).toHaveBeenCalled()
  })

  it('closes popover after selecting an option', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setTextAlign'],
    })
    render(<AlignSelector editor={mockEditor} />)

    await user.click(screen.getByRole('button', { name: /对齐/ }))
    expect(screen.getByText('左对齐')).toBeInTheDocument()

    await user.click(screen.getByText('居中'))

    await waitFor(() => {
      expect(screen.queryByText('右对齐')).not.toBeInTheDocument()
    })
  })

  it('highlights active alignment option', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setTextAlign'],
      isActive: (name: string, attrs?: Record<string, unknown>) => {
        if (name === 'textAlign' && attrs?.textAlign === 'center') return true
        return false
      },
    })
    render(<AlignSelector editor={mockEditor} />)

    await user.click(screen.getByRole('button', { name: /对齐/ }))

    // The "居中" item should receive the `active` prop, which applies
    // "bg-primary-50 text-primary" via cn(). We check className because
    // DropdownItem does not expose a data attribute for the active state.
    const centerItem = screen.getByText('居中').closest('button')
    expect(centerItem?.className).toContain('bg-primary-50')
    // Verify the non-active item does NOT have the active style
    const leftItem = screen.getByText('左对齐').closest('button')
    expect(leftItem?.className).not.toContain('bg-primary-50')
  })

  it('updates trigger icon after transaction changes alignment', async () => {
    const mockEditor = createMockEditor({
      chainCommands: ['setTextAlign'],
      isActive: () => false,
    })
    render(<AlignSelector editor={mockEditor} />)

    // Default state: no alignment active, should render AlignLeft icon
    const trigger = screen.getByRole('button', { name: /对齐/ })
    const triggerSvg = trigger.querySelector('svg')
    expect(triggerSvg).toBeInTheDocument()
    const svgBefore = triggerSvg?.innerHTML

    // Simulate editor switching to center alignment
    mockEditor.isActive = (name: string, attrs?: Record<string, unknown>) => {
      if (name === 'textAlign' && attrs?.textAlign === 'center') return true
      return false
    }
    act(() => {
      mockEditor.simulateTransaction()
    })

    // After transaction, the trigger SVG should have changed (different icon rendered)
    const triggerSvgAfter = trigger.querySelector('svg')
    const svgAfter = triggerSvgAfter?.innerHTML
    expect(svgAfter).not.toBe(svgBefore)
  })
})
