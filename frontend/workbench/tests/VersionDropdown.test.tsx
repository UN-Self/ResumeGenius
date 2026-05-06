import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { VersionDropdown } from '@/components/version/VersionDropdown'
import type { Version } from '@/lib/api-client'

const sampleVersions: Version[] = [
  { id: 3, label: 'AI 修改：精简项目经历', created_at: '2026-05-06T10:15:00Z' },
  { id: 2, label: '手动保存', created_at: '2026-05-06T10:10:00Z' },
  { id: 1, label: 'AI 初始生成', created_at: '2026-05-06T10:00:00Z' },
]

describe('VersionDropdown', () => {
  it('shows version history button', () => {
    render(
      <VersionDropdown
        versions={sampleVersions}
        loading={false}
        onPreview={vi.fn()}
        onSaveSnapshot={vi.fn()}
      />,
    )

    expect(screen.getByRole('button', { name: '版本历史' })).toBeInTheDocument()
  })

  it('shows loading state', async () => {
    const user = userEvent.setup()

    render(
      <VersionDropdown
        versions={[]}
        loading={true}
        onPreview={vi.fn()}
        onSaveSnapshot={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('button', { name: '版本历史' }))

    expect(screen.getByText('加载中...')).toBeInTheDocument()
  })

  it('lists versions when opened', async () => {
    const user = userEvent.setup()

    render(
      <VersionDropdown
        versions={sampleVersions}
        loading={false}
        onPreview={vi.fn()}
        onSaveSnapshot={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('button', { name: '版本历史' }))

    expect(screen.getByText('AI 修改：精简项目经历')).toBeInTheDocument()
    expect(screen.getByText('手动保存')).toBeInTheDocument()
    expect(screen.getByText('AI 初始生成')).toBeInTheDocument()
  })

  it('calls onPreview when clicking a version', async () => {
    const user = userEvent.setup()
    const onPreview = vi.fn()

    render(
      <VersionDropdown
        versions={sampleVersions}
        loading={false}
        onPreview={onPreview}
        onSaveSnapshot={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('button', { name: '版本历史' }))
    await user.click(screen.getByText('手动保存'))

    expect(onPreview).toHaveBeenCalledWith(sampleVersions[1])
  })

  it('shows save snapshot button', async () => {
    const user = userEvent.setup()
    const onSaveSnapshot = vi.fn()

    render(
      <VersionDropdown
        versions={sampleVersions}
        loading={false}
        onPreview={vi.fn()}
        onSaveSnapshot={onSaveSnapshot}
      />,
    )

    await user.click(screen.getByRole('button', { name: '版本历史' }))
    await user.click(screen.getByRole('button', { name: '保存快照' }))

    expect(onSaveSnapshot).toHaveBeenCalled()
  })

  it('shows empty state', async () => {
    const user = userEvent.setup()

    render(
      <VersionDropdown
        versions={[]}
        loading={false}
        onPreview={vi.fn()}
        onSaveSnapshot={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('button', { name: '版本历史' }))

    expect(screen.getByText('暂无版本记录')).toBeInTheDocument()
  })
})
