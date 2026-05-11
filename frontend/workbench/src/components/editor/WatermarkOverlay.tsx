import { useMemo } from 'react'

interface WatermarkOverlayProps {
  visible?: boolean
}

const SVG_CONTENT = `<svg xmlns="http://www.w3.org/2000/svg" width="280" height="100">
  <text x="140" y="35" text-anchor="middle" font-size="16" font-weight="600" fill="#1a1815">ResumeGenius 预览</text>
  <text x="140" y="60" text-anchor="middle" font-size="10" fill="#1a1815">导出无水印 PDF 即可去除水印</text>
</svg>`

export function WatermarkOverlay({ visible = true }: WatermarkOverlayProps) {
  const bgImage = useMemo(
    () => `url("data:image/svg+xml;charset=utf-8,${encodeURIComponent(SVG_CONTENT)}")`,
    [],
  )

  if (!visible) return null

  return (
    <div
      data-testid="watermark-anchor"
      style={{
        position: 'absolute',
        inset: 0,
        overflow: 'hidden',
        pointerEvents: 'none',
        zIndex: -1,
      }}
    >
      <div
        style={{
          position: 'absolute',
          inset: '-50%',
          backgroundImage: bgImage,
          backgroundRepeat: 'repeat',
          backgroundSize: '280px 100px',
          transform: 'rotate(-30deg)',
          pointerEvents: 'none',
          userSelect: 'none',
          opacity: 0.12,
        }}
      />
    </div>
  )
}
