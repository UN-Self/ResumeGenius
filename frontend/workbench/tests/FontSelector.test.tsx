import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { FontSelector } from '@/components/editor/FontSelector'
import { createMockEditor } from './helpers/mock-editor'

describe('FontSelector', () => {
  it('renders trigger button with default text "字体"', () => {
    const mockEditor = createMockEditor({
      chainCommands: ['setFontFamily', 'unsetFontFamily'],
    })
    render(<FontSelector editor={mockEditor} />)

    const trigger = screen.getByRole('button', { name: /字体/ })
    expect(trigger).toBeInTheDocument()
    expect(trigger).toHaveTextContent('字体')
  })

  it('shows font list when clicked', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setFontFamily', 'unsetFontFamily'],
    })
    render(<FontSelector editor={mockEditor} />)

    const trigger = screen.getByRole('button', { name: /字体/ })
    await user.click(trigger)

    // Check for specific fonts in the list
    expect(screen.getByText('宋体')).toBeInTheDocument()
    expect(screen.getByText('Arial')).toBeInTheDocument()
    expect(screen.getByText('默认字体')).toBeInTheDocument()
  })

  it('calls editor command when a font is selected', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setFontFamily', 'unsetFontFamily'],
    })
    render(<FontSelector editor={mockEditor} />)

    const trigger = screen.getByRole('button', { name: /字体/ })
    await user.click(trigger)

    const arialOption = screen.getByText('Arial')
    await user.click(arialOption)

    expect(mockEditor.runMock).toHaveBeenCalled()
  })

  it('calls unsetFontFamily when "默认字体" is selected', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setFontFamily', 'unsetFontFamily'],
    })
    render(<FontSelector editor={mockEditor} />)

    const trigger = screen.getByRole('button', { name: /字体/ })
    await user.click(trigger)

    const defaultFontOption = screen.getByText('默认字体')
    await user.click(defaultFontOption)

    expect(mockEditor.runMock).toHaveBeenCalled()
  })

  it('displays current font name when set', async () => {
    const mockEditor = createMockEditor({
      chainCommands: ['setFontFamily', 'unsetFontFamily'],
    })
    render(<FontSelector editor={mockEditor} />)

    // Initially should show "字体"
    expect(screen.getByRole('button', { name: /字体/ })).toHaveTextContent('字体')

    // Simulate a transaction that sets the font to SimSun (宋体)
    mockEditor.getAttributes = vi.fn(() => ({ fontFamily: 'SimSun, serif' }))
    mockEditor.simulateTransaction()

    // Should now display "宋体"
    await waitFor(() => {
      expect(screen.getByRole('button')).toHaveTextContent('宋体')
    })
  })

  it('displays Arial label when Arial font is set', async () => {
    const mockEditor = createMockEditor({
      chainCommands: ['setFontFamily', 'unsetFontFamily'],
    })
    render(<FontSelector editor={mockEditor} />)

    // Simulate setting Arial font
    mockEditor.getAttributes = vi.fn(() => ({ fontFamily: 'Arial, sans-serif' }))
    mockEditor.simulateTransaction()

    await waitFor(() => {
      expect(screen.getByRole('button')).toHaveTextContent('Arial')
    })
  })

  it('closes popover after selecting a font', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setFontFamily', 'unsetFontFamily'],
    })
    render(<FontSelector editor={mockEditor} />)

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
    const mockEditor = createMockEditor({
      chainCommands: ['setFontFamily', 'unsetFontFamily'],
    })
    render(<FontSelector editor={mockEditor} />)

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
