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
  return Math.max(0, $pos.pos - $pos.parentOffset - 1)
}

/**
 * Promotes position to the enclosing listItem if the block sits inside one.
 * break-before on a paragraph child of a list-item does not trigger page
 * breaks in Chrome's PDF renderer — the property must live on the list-item
 * element itself (not the list container, which would push all items).
 */
export function promotePastList(doc: PmNode, blockPos: number): number {
  const $pos = doc.resolve(blockPos)
  for (let d = $pos.depth; d >= 1; d--) {
    const type = $pos.node(d).type.name
    if (type === 'listItem') {
      return Math.max(0, $pos.before(d))
    }
  }
  return blockPos
}
