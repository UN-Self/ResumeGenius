import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import UploadDialog from '@/components/intake/UploadDialog'

describe('UploadDialog', () => {
  it('shows a file type badge after selecting a supported file', async () => {
    const user = userEvent.setup()

    render(
      <UploadDialog
        open
        onClose={vi.fn()}
        onUpload={vi.fn().mockResolvedValue(undefined)}
      />,
    )

    const input = screen.getByLabelText('Upload file')
    const file = new File(['resume'], 'sample_resume.docx', {
      type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    })

    await user.upload(input, file)

    expect(screen.getByText('sample_resume')).toBeInTheDocument()
    expect(screen.getByText('DOCX')).toBeInTheDocument()
  })

  it('shows a png badge for png files', async () => {
    const user = userEvent.setup()

    render(
      <UploadDialog
        open
        onClose={vi.fn()}
        onUpload={vi.fn().mockResolvedValue(undefined)}
      />,
    )

    const input = screen.getByLabelText('Upload file')
    const file = new File(['image'], 'cover.png', { type: 'image/png' })

    await user.upload(input, file)

    expect(screen.getByText('cover')).toBeInTheDocument()
    expect(screen.getByText('PNG')).toBeInTheDocument()
  })

  it('shows a validation error for unsupported files', async () => {
    const user = userEvent.setup()

    render(
      <UploadDialog
        open
        onClose={vi.fn()}
        onUpload={vi.fn().mockResolvedValue(undefined)}
      />,
    )

    const input = screen.getByLabelText('Upload file')
    const file = new File(['resume'], 'resume.txt', { type: 'text/plain' })

    fireEvent.change(input, { target: { files: [file] } })

    expect(
      screen.getByText('\u4e0d\u652f\u6301\u7684\u6587\u4ef6\u683c\u5f0f\uff0c\u8bf7\u4e0a\u4f20 PDF\u3001DOCX\u3001PNG \u6216 JPG \u6587\u4ef6'),
    ).toBeInTheDocument()
  })

  it('asks for confirmation before replacing a same-name file in the current project', async () => {
    const user = userEvent.setup()
    const onUpload = vi.fn().mockResolvedValue(undefined)

    render(
      <UploadDialog
        open
        onClose={vi.fn()}
        onUpload={onUpload}
        existingAssets={[
          {
            id: 7,
            project_id: 1,
            type: 'resume_docx',
            uri: undefined,
            content: 'old parsed content',
            label: 'DOCX',
            metadata: {
              parsing: {
                original_filename: 'sample_resume.docx',
              },
            },
            created_at: '2026-05-05T01:00:00Z',
          },
        ]}
      />,
    )

    const input = screen.getByLabelText('Upload file')
    const file = new File(['resume'], 'sample_resume.docx', {
      type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    })

    await user.upload(input, file)
    await user.click(screen.getByRole('button', { name: '上传' }))

    expect(screen.getByText('替换同名文件')).toBeInTheDocument()
    expect(screen.getByText(/已存在同名文件/)).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '确认替换' }))

    expect(onUpload).toHaveBeenCalledWith(file, 7)
  })
})
