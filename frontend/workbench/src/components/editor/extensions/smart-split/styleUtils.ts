import type { Node as PmNode } from '@tiptap/pm/model'

// NOTE: these utilities split on ';' and assume CSS property values
// contain no semicolons. The input comes from ProseMirror style attrs where
// we only set break-before, so this holds in practice.
export function appendBreakBefore(style: string): string {
  const parts = style.split(';').map(s => s.trim()).filter(Boolean)
  if (parts.some(p => p.startsWith('break-before: page'))) return style
  parts.push('break-before: page')
  return parts.join('; ')
}

export function removeBreakBefore(style: string): string | null {
  const parts = style.split(';').map(s => s.trim()).filter(p => p && !p.startsWith('break-before: page'))
  return parts.length > 0 ? parts.join('; ') : null
}

export function resolveToBlockPos(doc: PmNode, pos: number): number {
  const $pos = doc.resolve(pos)
  return $pos.pos - $pos.parentOffset - 1
}
