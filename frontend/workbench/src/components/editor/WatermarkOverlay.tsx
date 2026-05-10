import { useEffect, useRef, useMemo } from 'react'

interface WatermarkOverlayProps {
  visible?: boolean
}

const SVG_CONTENT = `<svg xmlns="http://www.w3.org/2000/svg" width="280" height="100">
  <text x="140" y="35" text-anchor="middle" font-size="16" font-weight="600" fill="#1a1815">ResumeGenius 预览</text>
  <text x="140" y="60" text-anchor="middle" font-size="10" fill="#1a1815">导出无水印 PDF 即可去除水印</text>
</svg>`

function createWatermarkElement(bgImage: string): HTMLElement {
  const watermark = document.createElement('div')
  watermark.setAttribute('data-testid', 'watermark-overlay')
  watermark.style.cssText = [
    'position: absolute;',
    'inset: 0;',
    'overflow: hidden;',
    'pointer-events: none;',
    'z-index: 5;',
  ].join('')

  const inner = document.createElement('div')
  inner.style.cssText = [
    'position: absolute;',
    'inset: -50%;',
    `background-image: ${bgImage};`,
    'background-repeat: repeat;',
    'background-size: 280px 100px;',
    'transform: rotate(-30deg);',
    'pointer-events: none;',
    'user-select: none;',
    'opacity: 0.12;',
  ].join('')

  watermark.appendChild(inner)
  return watermark
}

export function WatermarkOverlay({ visible = true }: WatermarkOverlayProps) {
  const anchorRef = useRef<HTMLDivElement>(null)

  const bgImage = useMemo(
    () => `url("data:image/svg+xml;charset=utf-8,${encodeURIComponent(SVG_CONTENT)}")`,
    [],
  )

  useEffect(() => {
    if (!visible || !anchorRef.current) return

    const anchor = anchorRef.current

    const ensureWatermarks = () => {
      // Find the editor root that contains the pagination decorations
      const resumeDoc = anchor.parentElement
      if (!resumeDoc) return

      const pagesWrapper = resumeDoc.querySelector('[data-rm-pagination]')
      if (!pagesWrapper) return

      const pageBreaks = pagesWrapper.querySelectorAll(':scope > .rm-page-break')
      pageBreaks.forEach((pageBreak) => {
        const page = pageBreak.querySelector(':scope > .page')
        if (!page) return

        // Skip if this page already has a watermark
        if (page.querySelector('[data-testid="watermark-overlay"]')) return

        page.appendChild(createWatermarkElement(bgImage))
      })
    }

    // Initial injection — wait for decorations to render (rAF)
    const rafId = requestAnimationFrame(() => ensureWatermarks())

    // Watch for page DOM changes
    const observer = new MutationObserver(() => {
      ensureWatermarks()
    })

    const resumeDoc = anchor.parentElement
    if (resumeDoc) {
      observer.observe(resumeDoc, { childList: true, subtree: true })
    }

    return () => {
      cancelAnimationFrame(rafId)
      observer.disconnect()
    }
  }, [visible, bgImage])

  if (!visible) return null

  return <div ref={anchorRef} style={{ display: 'contents' }} />
}
