import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { WatermarkOverlay } from '@/components/editor/WatermarkOverlay'

describe('WatermarkOverlay', () => {
  it('renders watermark overlay element when visible=true (default)', () => {
    render(<WatermarkOverlay />)
    expect(screen.getByTestId('watermark-overlay')).toBeInTheDocument()
  })

  it('watermark has SVG background-image for tiling', () => {
    render(<WatermarkOverlay />)
    const overlay = screen.getByTestId('watermark-overlay')
    const bg = overlay.style.backgroundImage
    expect(bg).toContain('data:image/svg+xml')
    expect(bg).toContain('ResumeGenius')
  })

  it('does not render when visible=false', () => {
    const { container } = render(<WatermarkOverlay visible={false} />)
    expect(container.innerHTML).toBe('')
  })

  it('re-creates the watermark DOM node when removed externally', () => {
    const { container } = render(<WatermarkOverlay />)
    const watermark = container.querySelector('[data-testid="watermark-overlay"]')
    expect(watermark).toBeInTheDocument()

    // Simulate external removal (e.g. DevTools delete)
    watermark!.remove()

    // MutationObserver should re-create it
    return vi.waitFor(() => {
      expect(container.querySelector('[data-testid="watermark-overlay"]')).toBeInTheDocument()
    })
  })
})
