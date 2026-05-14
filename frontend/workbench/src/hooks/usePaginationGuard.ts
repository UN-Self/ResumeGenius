import { useEffect, useRef } from 'react'
import type { Editor } from '@tiptap/react'
import { A4_LAYOUT } from '@/lib/pagination-plus/layout'
import { SMART_SPLIT_PLUGIN_KEY } from '@/components/editor/extensions/smart-split/SmartSplitPlugin'

/** Per-page content area height (usable space after margins) */
const PAGE_CONTENT_HEIGHT =
  A4_LAYOUT.pageHeight - A4_LAYOUT.marginTop - A4_LAYOUT.marginBottom

/** Extra tolerance pages on top of the calculated maximum */
const TOLERANCE_PAGES = 2

/**
 * Computes the maximum reasonable page count by summing content element heights.
 * Returns 0 if there's no measurable content.
 */
function computeMaxReasonablePages(editorDom: HTMLElement): number {
  const children = Array.from(editorDom.children).filter((child) => {
    if (child instanceof HTMLElement && child.dataset.rmPagination) return false
    return true
  })
  if (children.length === 0) return 0

  let contentHeight = 0
  for (const child of children) {
    const rect = child.getBoundingClientRect()
    if (rect.height === 0) continue
    const cs = window.getComputedStyle(child)
    contentHeight +=
      rect.height +
      (parseFloat(cs.marginTop) || 0) +
      (parseFloat(cs.marginBottom) || 0)
  }

  if (contentHeight === 0) return 0
  return Math.ceil(contentHeight / PAGE_CONTENT_HEIGHT) + TOLERANCE_PAGES
}

/**
 * Guard against PaginationPlus infinite pagination.
 *
 * After every editor transaction, checks whether the current page count
 * exceeds a calculated maximum. If so, disables both PaginationPlus
 * and SmartSplit to prevent runaway page creation.
 */
export function usePaginationGuard(editor: Editor | null) {
  const disabledRef = useRef(false)

  useEffect(() => {
    if (!editor) return

    const check = () => {
      const editorDom = editor.view.dom
      const paginationEl = editorDom.querySelector('[data-rm-pagination]')
      if (!paginationEl) return

      const pageCount = paginationEl.children.length
      const maxPages = computeMaxReasonablePages(editorDom)

      if (maxPages > 0 && pageCount > maxPages) {
        if (!disabledRef.current) {
          disabledRef.current = true
          editor.commands.disablePagination()
          // Freeze SmartSplit: set plugin state so view.update skips detection
          const tr = editor.state.tr.setMeta(SMART_SPLIT_PLUGIN_KEY, { disabled: true })
          editor.view.dispatch(tr)
        }
      }
    }

    editor.on('transaction', check)
    return () => {
      editor.off('transaction', check)
    }
  }, [editor])
}
