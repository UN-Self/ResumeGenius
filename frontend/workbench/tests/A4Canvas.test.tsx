import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import EditorPage from '@/pages/EditorPage'
import { server } from './setup'

describe('A4Canvas', () => {
  it('renders the editor page with a 210mm by 297mm canvas', async () => {
    // Add assets mock handler required by the asset-driven editor sidebar.
    server.use(
      http.get('/api/v1/assets', () => {
        return HttpResponse.json({
          code: 0,
          data: [],
          message: 'ok',
        })
      })
    )

    render(
      <MemoryRouter initialEntries={['/projects/1/edit']}>
        <Routes>
          <Route path="/projects/:projectId" element={<div>Project Detail</div>} />
          <Route path="/projects/:projectId/edit" element={<EditorPage />} />
        </Routes>
      </MemoryRouter>
    )

    // Wait for the canvas to appear (after project + draft + parsing load)
    const canvas = await screen.findByTestId('a4-canvas')
    expect(canvas).toBeInTheDocument()
    expect(canvas).toHaveStyle({ width: '210mm', minHeight: '297mm' })
  })
})
