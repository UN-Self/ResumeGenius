import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { VersionPreviewBanner } from '@/components/version/VersionPreviewBanner'

describe('VersionPreviewBanner', () => {
  const version = { id: 3, label: 'AI 修改', created_at: '2026-05-06T10:15:00Z' }

  it('displays version label', () => {
    render(
      <VersionPreviewBanner
        version={version}
        onRollback={vi.fn()}
        onClose={vi.fn()}
      />,
    )

    expect(screen.getByText(/AI 修改/)).toBeInTheDocument()
    expect(screen.getByText(/正在预览/)).toBeInTheDocument()
  })

  it('calls onRollback when rollback button clicked', async () => {
    const user = userEvent.setup()
    const onRollback = vi.fn()

    render(
      <VersionPreviewBanner
        version={version}
        onRollback={onRollback}
        onClose={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('button', { name: '回退到此版本' }))

    expect(onRollback).toHaveBeenCalled()
  })

  it('calls onClose when close button clicked', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()

    render(
      <VersionPreviewBanner
        version={version}
        onRollback={vi.fn()}
        onClose={onClose}
      />,
    )

    await user.click(screen.getByRole('button', { name: '关闭预览' }))

    expect(onClose).toHaveBeenCalled()
  })
})
