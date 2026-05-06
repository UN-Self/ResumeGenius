import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { RollbackConfirmDialog } from '@/components/version/RollbackConfirmDialog'

describe('RollbackConfirmDialog', () => {
  it('renders confirmation message when open', () => {
    render(
      <RollbackConfirmDialog open onClose={vi.fn()} onConfirm={vi.fn()} />,
    )

    expect(screen.getByText(/覆盖当前编辑内容/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '确认回退' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '取消' })).toBeInTheDocument()
  })

  it('renders nothing when closed', () => {
    const { container } = render(
      <RollbackConfirmDialog open={false} onClose={vi.fn()} onConfirm={vi.fn()} />,
    )
    expect(container.innerHTML).toBe('')
  })

  it('calls onConfirm on confirm button click', async () => {
    const user = userEvent.setup()
    const onConfirm = vi.fn()

    render(<RollbackConfirmDialog open onClose={vi.fn()} onConfirm={onConfirm} />)

    await user.click(screen.getByRole('button', { name: '确认回退' }))

    expect(onConfirm).toHaveBeenCalled()
  })

  it('calls onClose on cancel button click', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()

    render(<RollbackConfirmDialog open onClose={onClose} onConfirm={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: '取消' }))

    expect(onClose).toHaveBeenCalled()
  })
})
