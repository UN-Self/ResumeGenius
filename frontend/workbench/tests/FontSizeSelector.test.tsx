import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { FontSizeSelector } from '@/components/editor/FontSizeSelector'
import { createMockEditor } from './helpers/mock-editor'

describe('FontSizeSelector', () => {
  it('renders trigger button with default text', () => {
    const mockEditor = createMockEditor({
      chainCommands: ['setFontSize', 'unsetFontSize'],
      getAttributes: () => ({ fontSize: null }),
    })
    render(<FontSizeSelector editor={mockEditor} />)

    const trigger = screen.getByRole('button')
    expect(trigger).toBeInTheDocument()
    expect(trigger).toHaveTextContent('12')
  })

  it('shows size list when clicked', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setFontSize', 'unsetFontSize'],
      getAttributes: () => ({ fontSize: null }),
    })
    render(<FontSizeSelector editor={mockEditor} />)

    const trigger = screen.getByRole('button')
    await user.click(trigger)

    // Check for specific sizes
    expect(screen.getByText('12pt')).toBeInTheDocument()
    expect(screen.getByText('24pt')).toBeInTheDocument()
  })

  it('calls editor command when a size is selected', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setFontSize', 'unsetFontSize'],
      getAttributes: () => ({ fontSize: null }),
    })
    render(<FontSizeSelector editor={mockEditor} />)

    const trigger = screen.getByRole('button')
    await user.click(trigger)

    const sizeOption = screen.getByText('14pt')
    await user.click(sizeOption)

    expect(mockEditor.runMock).toHaveBeenCalled()
  })

  it('displays current size when set to 14pt', async () => {
    const mockEditor = createMockEditor({
      chainCommands: ['setFontSize', 'unsetFontSize'],
      getAttributes: () => ({ fontSize: null }),
    })
    // Mock getAttributes to return fontSize
    mockEditor.getAttributes = vi.fn(() => ({ fontSize: '14pt' }))

    render(<FontSizeSelector editor={mockEditor} />)

    await waitFor(() => {
      expect(screen.getByRole('button')).toHaveTextContent('14')
    })
  })

  it('highlights the current size in the dropdown', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setFontSize', 'unsetFontSize'],
      getAttributes: () => ({ fontSize: null }),
    })
    // Mock getAttributes to return fontSize
    mockEditor.getAttributes = vi.fn(() => ({ fontSize: '16pt' }))

    render(<FontSizeSelector editor={mockEditor} />)

    // Wait for initial state to update
    await waitFor(() => {
      expect(screen.getByRole('button')).toHaveTextContent('16')
    })

    const trigger = screen.getByRole('button')
    await user.click(trigger)

    // The 16pt option should have the active styling
    const size16pt = screen.getByText('16pt')
    expect(size16pt).toBeInTheDocument()
    expect(size16pt).toHaveClass('bg-primary-50', 'text-primary')
  })
})
