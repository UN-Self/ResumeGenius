import { useEffect, useRef, useMemo } from 'react'

interface WatermarkOverlayProps {
  visible?: boolean
}

const SVG_CONTENT = `<svg xmlns="http://www.w3.org/2000/svg" width="280" height="100">
  <text x="140" y="35" text-anchor="middle" font-size="16" font-weight="600" fill="#1a1815">ResumeGenius 预览</text>
  <text x="140" y="60" text-anchor="middle" font-size="10" fill="#1a1815">导出无水印 PDF 即可去除水印</text>
</svg>`

export function WatermarkOverlay({ visible = true }: WatermarkOverlayProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const watermarkRef = useRef<HTMLDivElement>(null)

  const bgImage = useMemo(
    () => `url("data:image/svg+xml;charset=utf-8,${encodeURIComponent(SVG_CONTENT)}")`,
    [],
  )

  useEffect(() => {
    if (!visible || !containerRef.current) return

    const container = containerRef.current

    const observer = new MutationObserver((mutations) => {
      for (const mutation of mutations) {
        for (const removed of mutation.removedNodes) {
          if (!(removed instanceof HTMLElement)) continue
          if (removed.dataset?.testid === 'watermark-overlay') {
            const existing = container.querySelector('[data-testid="watermark-overlay"]')
            if (!existing && watermarkRef.current) {
              container.appendChild(watermarkRef.current)
            }
          }
        }
      }
    })

    observer.observe(container, { childList: true })
    return () => observer.disconnect()
  }, [visible])

  if (!visible) return null

  return (
    <div ref={containerRef} className="watermark-container">
      <div
        ref={watermarkRef}
        data-testid="watermark-overlay"
        className="watermark-overlay"
        style={{ backgroundImage: bgImage }}
      />
    </div>
  )
}
