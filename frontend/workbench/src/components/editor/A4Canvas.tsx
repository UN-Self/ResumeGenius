import { useEffect, useRef, useState, type ReactNode } from 'react'
import type { Editor } from '@tiptap/react'
import { TipTapEditor } from './TipTapEditor'
import { WatermarkOverlay } from './WatermarkOverlay'
import { RESUME_DOCUMENT_CLASS } from '@/lib/extract-styles'

interface A4CanvasProps {
  editor?: Editor | null
  children?: ReactNode
  scopedCSS?: string
}

const CANVAS_PADDING_PX = 48 // 24px * 2 container padding
const MIN_ZOOM = 0.5
const MAX_ZOOM = 1.0

// Total canvas width in px: page (794px) + margins (76px * 2) at 96dpi
const CANVAS_TOTAL_WIDTH_PX = 794 + 76 + 76

function computeZoom(containerWidth: number): number {
  const availableWidth = containerWidth - CANVAS_PADDING_PX
  const zoom = availableWidth / CANVAS_TOTAL_WIDTH_PX
  return Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, zoom))
}

export function A4Canvas({ editor, children, scopedCSS }: A4CanvasProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [zoom, setZoom] = useState(0.75)

  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    const updateZoom = () => {
      setZoom(computeZoom(container.clientWidth))
    }

    updateZoom()

    const observer = new ResizeObserver(() => updateZoom())
    observer.observe(container)
    return () => observer.disconnect()
  }, [])

  return (
    <div ref={containerRef} className="canvas-area bg-canvas-bg">
      <div
        data-testid="a4-canvas"
        className={`${RESUME_DOCUMENT_CLASS} relative bg-resume-paper shadow-[0_22px_80px_rgba(2,8,23,0.24)] ring-1 ring-black/5`}
        style={{
          transform: `scale(${zoom})`,
          transformOrigin: 'top center',
        }}
      >
        {scopedCSS && <style dangerouslySetInnerHTML={{ __html: scopedCSS }} />}
        {children || (editor && <TipTapEditor editor={editor} />)}
        <WatermarkOverlay />
      </div>
    </div>
  )
}
