import type { BreakerPosition, CrossingInfo } from './types'
import { BLOCK_TAGS } from './types'

/** Read breaker Y positions from the editor DOM */
export function getBreakerPositions(editorDom: Element): BreakerPosition[] {
  const breakers = editorDom.querySelectorAll('.breaker')
  return Array.from(breakers).map((el) => {
    const rect = el.getBoundingClientRect()
    return { top: rect.top, bottom: rect.bottom }
  })
}

/** Check if a single element crosses a breaker boundary */
export function elementCrossesBreaker(
  rect: { top: number; bottom: number },
  breaker: BreakerPosition,
  threshold: number,
): boolean {
  return rect.top < breaker.bottom && rect.bottom > breaker.top - threshold
}

/**
 * Walk editor DOM to find block elements that cross page boundaries.
 * Returns CrossingInfo with ProseMirror positions (via view.posAtDOM).
 */
export function findCrossingPositions(
  view: { posAtDOM: (node: Node, offset: number) => number },
  editorDom: Element,
  breakers: BreakerPosition[],
  threshold: number,
): CrossingInfo[] {
  if (breakers.length === 0) return []

  const results: CrossingInfo[] = []
  const walker = document.createTreeWalker(
    editorDom,
    NodeFilter.SHOW_ELEMENT,
    {
      acceptNode: (node: Element) =>
        BLOCK_TAGS.has(node.tagName) ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_SKIP,
    },
  )

  let el = walker.nextNode() as Element | null
  // Skip the root editorDom itself
  if (el === editorDom) el = walker.nextNode() as Element | null

  while (el) {
    const rect = el.getBoundingClientRect()
    if (rect.height === 0 || (el.textContent ?? '').trim() === '') {
      el = walker.nextNode() as Element | null
      continue
    }

    for (let i = 0; i < breakers.length; i++) {
      if (elementCrossesBreaker(rect, breakers[i], threshold)) {
        try {
          results.push({ pos: view.posAtDOM(el, 0), breakerIndex: i })
        } catch { /* element outside ProseMirror view */ }
        break
      }
    }
    el = walker.nextNode() as Element | null
  }

  return results
}
