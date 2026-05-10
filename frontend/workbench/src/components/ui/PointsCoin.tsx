export function PointsCoin({ size = 24, className = '' }: { size?: number; className?: string }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 64 64"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      aria-hidden="true"
    >
      <defs>
        <linearGradient id="coin-grad" x1="0" y1="0" x2="64" y2="64">
          <stop offset="0%" stopColor="#fbbf24" />
          <stop offset="35%" stopColor="#f59e0b" />
          <stop offset="65%" stopColor="#d97706" />
          <stop offset="100%" stopColor="#b45309" />
        </linearGradient>
        <linearGradient id="coin-shine" x1="12" y1="12" x2="52" y2="52">
          <stop offset="0%" stopColor="rgba(255,255,255,0.5)" />
          <stop offset="50%" stopColor="rgba(255,255,255,0.05)" />
          <stop offset="100%" stopColor="rgba(255,255,255,0)" />
        </linearGradient>
        <radialGradient id="coin-glow" cx="32" cy="32" r="32">
          <stop offset="60%" stopColor="transparent" />
          <stop offset="100%" stopColor="var(--color-primary)" stopOpacity="0.25" />
        </radialGradient>
        <filter id="coin-shadow">
          <feDropShadow dx="0" dy="0" stdDeviation="2" floodColor="#f59e0b" floodOpacity="0.5" />
        </filter>
      </defs>

      {/* Outer glow ring */}
      <circle cx="32" cy="32" r="28" fill="url(#coin-glow)" className="animate-pulse" />

      {/* Coin body */}
      <circle cx="32" cy="32" r="22" fill="url(#coin-grad)" filter="url(#coin-shadow)" />

      {/* Rim highlight */}
      <circle
        cx="32" cy="32" r="21"
        fill="none"
        stroke="rgba(255,255,255,0.15)"
        strokeWidth="2"
      />
      <circle
        cx="32" cy="32" r="20"
        fill="none"
        stroke="rgba(255,255,255,0.08)"
        strokeWidth="1"
      />

      {/* Shine */}
      <circle cx="32" cy="32" r="19" fill="url(#coin-shine)" />

      {/* Star / sparkle icon */}
      <path
        d="M32 14l2.5 7.5L42 24l-7.5 2.5L32 34l-2.5-7.5L22 24l7.5-2.5L32 14z"
        fill="rgba(255,255,255,0.85)"
      />
      {/* Small sparkles */}
      <circle cx="22" cy="20" r="1.2" fill="rgba(255,255,255,0.7)" className="animate-ping" style={{ animationDuration: '2.5s' }} />
      <circle cx="44" cy="44" r="1" fill="rgba(255,255,255,0.6)" className="animate-ping" style={{ animationDuration: '3s', animationDelay: '0.5s' }} />
      <circle cx="40" cy="18" r="0.8" fill="rgba(255,255,255,0.5)" className="animate-ping" style={{ animationDuration: '2s', animationDelay: '1s' }} />
    </svg>
  )
}
