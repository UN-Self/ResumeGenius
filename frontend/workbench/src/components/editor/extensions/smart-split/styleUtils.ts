export function appendBreakBefore(style: string): string {
  const parts = style.split(';').map(s => s.trim()).filter(Boolean)
  if (parts.some(p => p.startsWith('break-before'))) return style
  parts.push('break-before: page')
  return parts.join('; ')
}

export function removeBreakBefore(style: string): string | null {
  const parts = style.split(';').map(s => s.trim()).filter(p => p && !p.startsWith('break-before'))
  return parts.length > 0 ? parts.join('; ') : null
}
