import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import EditorPage from '@/pages/EditorPage'
import './setup'

function renderWithRouter(element: React.ReactElement) {
  return render(<MemoryRouter initialEntries={['/projects/1/edit']}>{element}</MemoryRouter>)
}

describe('EditorPage', () => {
  it('renders the editor page shell with a 210mm by 297mm canvas', async () => {
    renderWithRouter(<EditorPage />)

    // Wait for the AI panel text to appear
    expect(await screen.findByText('AI 助手')).toBeInTheDocument()

    // Check for A4 canvas with correct dimensions
    const canvas = screen.getByTestId('a4-canvas')
    expect(canvas).toBeInTheDocument()
    expect(canvas).toHaveStyle({ width: '210mm', minHeight: '297mm' })
  })
})
