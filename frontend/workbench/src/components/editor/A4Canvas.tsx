import { useEffect, useRef, useState, type ReactNode } from 'react'
import type { Editor } from '@tiptap/react'
import { TipTapEditor } from './TipTapEditor'

interface A4CanvasProps {
  editor?: Editor | null
  children?: ReactNode
}

const CANVAS_WIDTH_MM = 210
const CANVAS_PADDING_PX = 48 // 24px * 2 page margin
const MIN_ZOOM = 0.5
const MAX_ZOOM = 1.0

function computeZoom(containerWidth: number): number {
  const availableWidth = containerWidth - CANVAS_PADDING_PX
  // 1mm ≈ 3.7795px at 96dpi
  const canvasPx = CANVAS_WIDTH_MM * 3.7795
  const zoom = availableWidth / canvasPx
  return Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, zoom))
}

export function A4Canvas({ editor, children }: A4CanvasProps) {
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
        className="bg-white shadow-[0_2px_12px_rgba(0,0,0,0.08)] p-[18mm_20mm]"
        style={{
          width: '210mm',
          minHeight: '297mm',
          transform: `scale(${zoom})`,
          transformOrigin: 'top center',
        }}
      >
        {children || (editor && <TipTapEditor editor={editor} />)}
      </div>
    </div>
  )
}
