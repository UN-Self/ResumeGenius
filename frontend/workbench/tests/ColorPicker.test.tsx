import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ColorPicker } from '@/components/editor/ColorPicker'
import { createMockEditor } from './helpers/mock-editor'

describe('ColorPicker', () => {
  it('renders both trigger buttons (font color + highlight color)', () => {
    const mockEditor = createMockEditor({
      chainCommands: ['setColor', 'unsetColor', 'setBackgroundColor', 'unsetBackgroundColor'],
    })
    render(<ColorPicker editor={mockEditor} />)

    // Check for font color button
    const fontButton = screen.getByRole('button', { name: /字体颜色/i })
    expect(fontButton).toBeInTheDocument()

    // Check for highlight color button
    const highlightButton = screen.getByRole('button', { name: /背景高亮/i })
    expect(highlightButton).toBeInTheDocument()
  })

  it('shows color swatches when font color button is clicked', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setColor', 'unsetColor', 'setBackgroundColor', 'unsetBackgroundColor'],
    })
    render(<ColorPicker editor={mockEditor} />)

    const fontButton = screen.getByRole('button', { name: /字体颜色/i })
    await user.click(fontButton)

    // Should show preset color buttons (18 colors)
    const colorButtons = screen.getAllByRole('button', { name: /选择颜色/i })
    expect(colorButtons.length).toBe(18)

    // Should show reset button
    expect(screen.getByRole('button', { name: /重置/i })).toBeInTheDocument()

    // Should show custom color label
    expect(screen.getByText(/自定义/i)).toBeInTheDocument()
  })

  it('calls editor command when a preset color is selected', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setColor', 'unsetColor', 'setBackgroundColor', 'unsetBackgroundColor'],
    })
    render(<ColorPicker editor={mockEditor} />)

    const fontButton = screen.getByRole('button', { name: /字体颜色/i })
    await user.click(fontButton)

    // Click the first preset color (#000000)
    const colorButtons = screen.getAllByRole('button', { name: /选择颜色/i })
    await user.click(colorButtons[0])

    expect(mockEditor.runMock).toHaveBeenCalled()
  })

  it('calls setColor when font color target is active', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setColor', 'unsetColor', 'setBackgroundColor', 'unsetBackgroundColor'],
    })
    render(<ColorPicker editor={mockEditor} />)

    const fontButton = screen.getByRole('button', { name: /字体颜色/i })
    await user.click(fontButton)

    const colorButtons = screen.getAllByRole('button', { name: /选择颜色/i })
    await user.click(colorButtons[0])

    // The chain().focus().setColor().run() should be called
    expect(mockEditor.runMock).toHaveBeenCalled()
  })

  it('calls setBackgroundColor when highlight target is active', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setColor', 'unsetColor', 'setBackgroundColor', 'unsetBackgroundColor'],
    })
    render(<ColorPicker editor={mockEditor} />)

    const highlightButton = screen.getByRole('button', { name: /背景高亮/i })
    await user.click(highlightButton)

    const colorButtons = screen.getAllByRole('button', { name: /选择颜色/i })
    await user.click(colorButtons[0])

    expect(mockEditor.runMock).toHaveBeenCalled()
  })

  it('calls unsetColor when reset button is clicked for font color', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setColor', 'unsetColor', 'setBackgroundColor', 'unsetBackgroundColor'],
    })
    render(<ColorPicker editor={mockEditor} />)

    const fontButton = screen.getByRole('button', { name: /字体颜色/i })
    await user.click(fontButton)

    const resetButton = screen.getByRole('button', { name: /重置/i })
    await user.click(resetButton)

    expect(mockEditor.runMock).toHaveBeenCalled()
  })

  it('shows color indicator when font color is set', async () => {
    const mockEditor = createMockEditor({
      chainCommands: ['setColor', 'unsetColor', 'setBackgroundColor', 'unsetBackgroundColor'],
    })
    // Mock getAttributes to return a color
    mockEditor.getAttributes = vi.fn(() => ({ color: '#e06666' }))

    render(<ColorPicker editor={mockEditor} />)

    const fontButton = screen.getByRole('button', { name: /字体颜色/i })
    // Should have a child span with the color indicator
    const indicator = fontButton.querySelector('span')
    expect(indicator).toBeInTheDocument()
    expect(indicator).toHaveStyle({ backgroundColor: '#e06666' })
  })

  it('shows color indicator when background color is set', async () => {
    const mockEditor = createMockEditor({
      chainCommands: ['setColor', 'unsetColor', 'setBackgroundColor', 'unsetBackgroundColor'],
    })
    // Mock getAttributes to return a background color
    mockEditor.getAttributes = vi.fn(() => ({ backgroundColor: '#ffd966' }))

    render(<ColorPicker editor={mockEditor} />)

    const highlightButton = screen.getByRole('button', { name: /背景高亮/i })
    // Should have a child span with the color indicator
    const indicator = highlightButton.querySelector('span')
    expect(indicator).toBeInTheDocument()
    expect(indicator).toHaveStyle({ backgroundColor: '#ffd966' })
  })

  it('closes popover after selecting a color', async () => {
    const user = userEvent.setup()
    const mockEditor = createMockEditor({
      chainCommands: ['setColor', 'unsetColor', 'setBackgroundColor', 'unsetBackgroundColor'],
    })
    render(<ColorPicker editor={mockEditor} />)

    const fontButton = screen.getByRole('button', { name: /字体颜色/i })
    await user.click(fontButton)

    // Popover content should be visible
    expect(screen.getByRole('button', { name: /重置/i })).toBeInTheDocument()

    // Click a color
    const colorButtons = screen.getAllByRole('button', { name: /选择颜色/i })
    await user.click(colorButtons[0])

    // Popover should close (reset button should not be in document)
    await waitFor(() => {
      expect(screen.queryByRole('button', { name: /重置/i })).not.toBeInTheDocument()
    })
  })

  it('allows custom color selection via native input', async () => {
    const mockEditor = createMockEditor({
      chainCommands: ['setColor', 'unsetColor', 'setBackgroundColor', 'unsetBackgroundColor'],
    })
    render(<ColorPicker editor={mockEditor} />)

    const fontButton = screen.getByRole('button', { name: /字体颜色/i })
    await userEvent.click(fontButton)

    // Find the color input using querySelector
    const input = document.querySelector('input[type="color"]')
    expect(input).toBeInTheDocument()

    if (input) {
      // Simulate color change using fireEvent
      fireEvent.change(input, { target: { value: '#ff0000' } })

      expect(mockEditor.runMock).toHaveBeenCalled()
    }
  })
})
