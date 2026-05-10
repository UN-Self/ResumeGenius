import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { WatermarkOverlay } from '@/components/editor/WatermarkOverlay'

describe('WatermarkOverlay', () => {
  it('renders a display:contents anchor element when visible=true (default)', () => {
    const { container } = render(<WatermarkOverlay />)
    // The component renders a div with display:contents as MutationObserver anchor
    const anchor = container.firstElementChild as HTMLElement
    expect(anchor).toBeInTheDocument()
    expect(anchor.style.display).toBe('contents')
    // Watermarks are injected into extension DOM, not React tree
    expect(screen.queryByTestId('watermark-overlay')).toBeNull()
  })

  it('does not render when visible=false', () => {
    const { container } = render(<WatermarkOverlay visible={false} />)
    expect(container.innerHTML).toBe('')
  })

  it('injects watermarks into extension page elements when pagination DOM is present', async () => {
    // Render with a mock pagination DOM structure (simulating extension output)
    // Use vanilla DOM to create the extension-mimicking structure since these are
    // not React-rendered elements (they come from ProseMirror decorations).
    const { container } = render(
      <div>
        <WatermarkOverlay />
        <div id="host" ref={(el) => {
          if (!el) return
          // Build pagination DOM structure via vanilla APIs
          const pagesWrapper = document.createElement('div')
          pagesWrapper.setAttribute('data-rm-pagination', 'true')
          pagesWrapper.id = 'pages'

          for (let i = 0; i < 2; i++) {
            const pageBreak = document.createElement('div')
            pageBreak.className = 'rm-page-break'

            const page = document.createElement('div')
            page.className = 'page'
            page.style.position = 'relative'

            const breaker = document.createElement('div')
            breaker.className = 'breaker'

            pageBreak.appendChild(page)
            pageBreak.appendChild(breaker)
            pagesWrapper.appendChild(pageBreak)
          }

          el.appendChild(pagesWrapper)
        }} />
      </div>
    )

    // Wait for MutationObserver + rAF to inject watermarks
    await waitFor(() => {
      const watermarks = container.querySelectorAll('[data-testid="watermark-overlay"]')
      expect(watermarks.length).toBe(2)
    })

    // Verify watermark content
    const firstWatermark = container.querySelector('[data-testid="watermark-overlay"]')!
    const inner = firstWatermark.firstElementChild as HTMLElement
    expect(inner.style.backgroundImage).toContain('data:image/svg+xml')
    expect(inner.style.backgroundImage).toContain('ResumeGenius')
    expect(inner.style.opacity).toBe('0.12')
  })

  it('does not duplicate watermarks on re-render', async () => {
    const { container, rerender } = render(
      <div>
        <WatermarkOverlay />
        <div id="host" ref={(el) => {
          if (!el) return
          const pagesWrapper = document.createElement('div')
          pagesWrapper.setAttribute('data-rm-pagination', 'true')
          pagesWrapper.id = 'pages'

          const pageBreak = document.createElement('div')
          pageBreak.className = 'rm-page-break'

          const page = document.createElement('div')
          page.className = 'page'
          page.style.position = 'relative'

          const breaker = document.createElement('div')
          breaker.className = 'breaker'

          pageBreak.appendChild(page)
          pageBreak.appendChild(breaker)
          pagesWrapper.appendChild(pageBreak)
          el.appendChild(pagesWrapper)
        }} />
      </div>
    )

    await waitFor(() => {
      expect(container.querySelectorAll('[data-testid="watermark-overlay"]').length).toBe(1)
    })

    // Re-render should not add duplicate watermarks
    rerender(
      <div>
        <WatermarkOverlay />
        <div id="host" ref={(el) => {
          if (!el) return
          const pagesWrapper = document.createElement('div')
          pagesWrapper.setAttribute('data-rm-pagination', 'true')
          pagesWrapper.id = 'pages'

          const pageBreak = document.createElement('div')
          pageBreak.className = 'rm-page-break'

          const page = document.createElement('div')
          page.className = 'page'
          page.style.position = 'relative'

          const breaker = document.createElement('div')
          breaker.className = 'breaker'

          pageBreak.appendChild(page)
          pageBreak.appendChild(breaker)
          pagesWrapper.appendChild(pageBreak)
          el.appendChild(pagesWrapper)
        }} />
      </div>
    )

    // Wait a tick for observer to fire — count should still be 1
    await vi.waitFor(() => {
      expect(container.querySelectorAll('[data-testid="watermark-overlay"]').length).toBe(1)
    })
  })
})
