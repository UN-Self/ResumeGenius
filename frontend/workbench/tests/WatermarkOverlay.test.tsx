import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { WatermarkOverlay } from '@/components/editor/WatermarkOverlay'

describe('WatermarkOverlay', () => {
  it('renders a watermark overlay when visible=true (default)', () => {
    const { container } = render(<WatermarkOverlay />)
    const overlay = container.firstElementChild as HTMLElement
    expect(overlay).toBeInTheDocument()
    expect(overlay.style.position).toBe('absolute')
    expect(overlay.style.zIndex).toBe('5')
  })

  it('does not render when visible=false', () => {
    const { container } = render(<WatermarkOverlay visible={false} />)
    expect(container.innerHTML).toBe('')
  })

  it('renders inner div with rotated SVG watermark background', () => {
    render(<WatermarkOverlay />)
    const outer = screen.getByTestId('watermark-anchor')
    const inner = outer.firstElementChild as HTMLElement
    expect(inner).toBeInTheDocument()
    const style = inner.getAttribute('style')!
    expect(style).toContain('data:image/svg+xml')
    expect(style).toContain('ResumeGenius')
    expect(style).toContain('opacity: 0.12')
    expect(style).toContain('rotate(-30deg)')
  })
})
