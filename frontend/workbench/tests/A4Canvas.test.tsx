import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import EditorPage from '@/pages/EditorPage'
import { server } from './setup'

describe('A4Canvas', () => {
  beforeEach(() => {
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
  })

  it('renders the editor page with an A4 canvas using zoom transform', async () => {
    // Wait for the canvas to appear (after project + draft + parsing load)
    const canvas = await screen.findByTestId('a4-canvas')
    expect(canvas).toBeInTheDocument()
    // Canvas has the resume-document class and uses a scale() transform for zoom
    expect(canvas).toHaveClass('resume-document')
    expect(canvas.style.transform).toMatch(/scale\(/)
  })

  it('renders watermark overlay on the canvas', async () => {
    await screen.findByTestId('a4-canvas')
    expect(screen.getByTestId('watermark-anchor')).toBeInTheDocument()
  })

  it('prevents context menu on the canvas area', async () => {
    const canvas = await screen.findByTestId('a4-canvas')
    const container = canvas.closest('.canvas-area')!

    const contextEvent = new MouseEvent('contextmenu', { bubbles: true, cancelable: true })
    container.dispatchEvent(contextEvent)

    expect(contextEvent.defaultPrevented).toBe(true)
  })
})
