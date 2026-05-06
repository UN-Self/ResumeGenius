import type { CSSProperties } from 'react'
import { cn } from '@/lib/utils'

const letters = [
  { char: 'R', motion: 'rise', delay: '-0.1s' },
  { char: 'e' },
  { char: 's', motion: 'sway', delay: '-0.7s' },
  { char: 'u' },
  { char: 'm', motion: 'pulse', delay: '-1.3s' },
  { char: 'e' },
  { char: 'G', motion: 'roll', delay: '-0.35s' },
  { char: 'e' },
  { char: 'n', motion: 'nod', delay: '-1s' },
  { char: 'i', motion: 'dot', delay: '-0.55s' },
  { char: 'u' },
  { char: 's', motion: 'sway', delay: '-1.6s' },
]

interface AnimatedBrandTitleProps {
  className?: string
}

export function AnimatedBrandTitle({ className }: AnimatedBrandTitleProps) {
  return (
    <h1 className={cn('animated-brand-title', className)} aria-label="ResumeGenius">
      {letters.map((letter, index) => (
        letter.motion === 'dot' ? (
          <span
            key={`${letter.char}-${index}`}
            aria-hidden="true"
            className="brand-letter brand-letter-i"
            style={{ '--motion-delay': letter.delay } as CSSProperties}
          >
            <span className="brand-i-dot" />
            <span className="brand-i-stem" />
          </span>
        ) : (
          <span
            key={`${letter.char}-${index}`}
            aria-hidden="true"
            className={cn('brand-letter', letter.motion && `brand-letter-${letter.motion}`)}
            style={{ '--motion-delay': letter.delay ?? '0ms' } as CSSProperties}
          >
            {letter.char}
          </span>
        )
      ))}
    </h1>
  )
}
