import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { SaveSnapshotDialog } from '@/components/version/SaveSnapshotDialog'

describe('SaveSnapshotDialog', () => {
  it('renders input and buttons when open', () => {
    render(<SaveSnapshotDialog open onClose={vi.fn()} onConfirm={vi.fn()} />)

    expect(screen.getByPlaceholderText(/可选/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '确认' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '取消' })).toBeInTheDocument()
  })

  it('renders nothing when closed', () => {
    const { container } = render(
      <SaveSnapshotDialog open={false} onClose={vi.fn()} onConfirm={vi.fn()} />,
    )
    expect(container.innerHTML).toBe('')
  })

  it('calls onConfirm with input value', async () => {
    const user = userEvent.setup()
    const onConfirm = vi.fn()

    render(<SaveSnapshotDialog open onClose={vi.fn()} onConfirm={onConfirm} />)

    await user.type(screen.getByPlaceholderText(/可选/), '校招版')
    await user.click(screen.getByRole('button', { name: '确认' }))

    expect(onConfirm).toHaveBeenCalledWith('校招版')
  })

  it('calls onConfirm with empty string when no input', async () => {
    const user = userEvent.setup()
    const onConfirm = vi.fn()

    render(<SaveSnapshotDialog open onClose={vi.fn()} onConfirm={onConfirm} />)

    await user.click(screen.getByRole('button', { name: '确认' }))

    expect(onConfirm).toHaveBeenCalledWith('')
  })

  it('calls onClose on cancel button click', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()

    render(<SaveSnapshotDialog open onClose={onClose} onConfirm={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: '取消' }))

    expect(onClose).toHaveBeenCalled()
  })
})
