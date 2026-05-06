import { useEffect, useRef } from 'react'
import { useLocation } from 'react-router-dom'

const GRID_SIZE = 64
const RIPPLE_RADIUS = 280
const RIPPLE_DURATION = 920
const SAMPLE_STEP = 8
const MIN_INFLUENCE = 0.006
const INFLUENCE_POWER = 1.55

function shouldShowRipple(pathname: string) {
  return pathname === '/' || pathname === '/login' || /^\/projects\/\d+$/.test(pathname)
}

export function GridRippleCanvas() {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const { pathname } = useLocation()
  const enabled = shouldShowRipple(pathname)

  useEffect(() => {
    if (!enabled) return

    const canvas = canvasRef.current
    const context = canvas?.getContext('2d')
    if (!canvas || !context) return

    let animationFrame = 0
    let lastMoveAt = 0
    let pointerX = window.innerWidth / 2
    let pointerY = window.innerHeight / 2
    let dpr = Math.min(window.devicePixelRatio || 1, 2)

    const resize = () => {
      dpr = Math.min(window.devicePixelRatio || 1, 2)
      canvas.width = Math.floor(window.innerWidth * dpr)
      canvas.height = Math.floor(window.innerHeight * dpr)
      canvas.style.width = `${window.innerWidth}px`
      canvas.style.height = `${window.innerHeight}px`
      context.setTransform(dpr, 0, 0, dpr, 0, 0)
    }

    const drawSegment = (
      from: [number, number],
      to: [number, number],
      strokeStyle: string,
      alpha: number,
      lineWidth: number,
    ) => {
      context.globalAlpha = alpha
      context.strokeStyle = strokeStyle
      context.lineWidth = lineWidth
      context.beginPath()
      context.moveTo(from[0], from[1])
      context.lineTo(to[0], to[1])
      context.stroke()
    }

    const draw = (time: number) => {
      const elapsed = time - lastMoveAt
      const fade = Math.max(0, 1 - elapsed / RIPPLE_DURATION)
      context.clearRect(0, 0, canvas.width / dpr, canvas.height / dpr)

      if (fade <= 0) {
        animationFrame = 0
        return
      }

      const styles = getComputedStyle(document.documentElement)
      const isLightTheme = document.documentElement.dataset.theme === 'light'
      const lineColor = styles.getPropertyValue('--color-primary').trim()
        || styles.getPropertyValue('--color-border-glow').trim()
        || '#6bdcff'
      const radius = RIPPLE_RADIUS
      const amplitude = 11 * fade
      const verticalAlpha = isLightTheme ? 0.34 : 0.3
      const horizontalAlpha = isLightTheme ? 0.28 : 0.24
      const left = Math.floor((pointerX - radius) / GRID_SIZE) * GRID_SIZE
      const right = pointerX + radius
      const top = Math.floor((pointerY - radius) / GRID_SIZE) * GRID_SIZE
      const bottom = pointerY + radius

      context.save()
      context.lineCap = 'round'
      context.lineJoin = 'round'

      for (let x = left; x <= right; x += GRID_SIZE) {
        let previous: [number, number, number] | null = null
        for (let y = pointerY - radius; y <= pointerY + radius; y += SAMPLE_STEP) {
          const dx = x - pointerX
          const dy = y - pointerY
          const distance = Math.hypot(dx, dy)
          const influence = Math.max(0, 1 - distance / radius) ** INFLUENCE_POWER
          const bend = Math.sin(dy * 0.055 + time * 0.006) * amplitude * influence
          const current: [number, number, number] = [x + bend, y, influence]
          if (previous && previous[2] > MIN_INFLUENCE && influence > MIN_INFLUENCE) {
            const segmentInfluence = ((previous[2] + influence) / 2) ** 1.08
            drawSegment(
              [previous[0], previous[1]],
              [current[0], current[1]],
              lineColor,
              verticalAlpha * fade * segmentInfluence,
              1.05,
            )
          }
          previous = current
        }
      }

      for (let y = top; y <= bottom; y += GRID_SIZE) {
        let previous: [number, number, number] | null = null
        for (let x = pointerX - radius; x <= pointerX + radius; x += SAMPLE_STEP) {
          const dx = x - pointerX
          const dy = y - pointerY
          const distance = Math.hypot(dx, dy)
          const influence = Math.max(0, 1 - distance / radius) ** INFLUENCE_POWER
          const bend = Math.sin(dx * 0.055 + time * 0.006) * amplitude * influence
          const current: [number, number, number] = [x, y + bend, influence]
          if (previous && previous[2] > MIN_INFLUENCE && influence > MIN_INFLUENCE) {
            const segmentInfluence = ((previous[2] + influence) / 2) ** 1.08
            drawSegment(
              [previous[0], previous[1]],
              [current[0], current[1]],
              lineColor,
              horizontalAlpha * fade * segmentInfluence,
              1,
            )
          }
          previous = current
        }
      }

      context.restore()
      animationFrame = window.requestAnimationFrame(draw)
    }

    const onPointerMove = (event: PointerEvent) => {
      pointerX = event.clientX
      pointerY = event.clientY
      lastMoveAt = performance.now()
      if (!animationFrame) {
        animationFrame = window.requestAnimationFrame(draw)
      }
    }

    resize()
    window.addEventListener('resize', resize)
    window.addEventListener('pointermove', onPointerMove, { passive: true })

    return () => {
      window.removeEventListener('resize', resize)
      window.removeEventListener('pointermove', onPointerMove)
      if (animationFrame) window.cancelAnimationFrame(animationFrame)
    }
  }, [enabled])

  if (!enabled) return null

  return <canvas ref={canvasRef} className="grid-ripple-canvas" aria-hidden="true" />
}
