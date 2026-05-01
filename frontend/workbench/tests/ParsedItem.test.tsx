import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import ParsedItem from '@/components/intake/ParsedItem'

describe('ParsedItem', () => {
  it('renders a cleaned file title with a file-type badge', () => {
    render(
      <ParsedItem
        content={{
          asset_id: 1,
          type: 'resume_pdf',
          label: '71e10333-d8fe-4e67-ab5d-938bf04fd5dc_sample_resume.pdf',
          text: 'resume body',
        }}
      />,
    )

    expect(screen.getByText('sample_resume')).toBeInTheDocument()
    expect(screen.getByText('PDF')).toBeInTheDocument()
    expect(screen.queryByText('71e10333-d8fe-4e67-ab5d-938bf04fd5dc_sample_resume.pdf')).not.toBeInTheDocument()
  })

  it('removes the duplicated note label from the body text', () => {
    render(
      <ParsedItem
        content={{
          asset_id: 2,
          type: 'note',
          label: '\u6c42\u804c\u65b9\u5411',
          text: '\u6c42\u804c\u65b9\u5411\n\u5e0c\u671b\u7a81\u51fa React \u548c TypeScript \u7ecf\u9a8c',
        }}
      />,
    )

    expect(screen.getByText('\u6c42\u804c\u65b9\u5411')).toBeInTheDocument()
    expect(screen.getByText('\u5e0c\u671b\u7a81\u51fa React \u548c TypeScript \u7ecf\u9a8c')).toBeInTheDocument()
  })
})
