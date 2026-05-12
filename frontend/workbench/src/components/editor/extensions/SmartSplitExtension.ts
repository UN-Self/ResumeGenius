import { Extension } from '@tiptap/core'
import { smartSplitPlugin } from './smart-split/SmartSplitPlugin'
import { DEFAULT_OPTIONS, type SmartSplitOptions } from './smart-split/types'

export const SmartSplitExtension = Extension.create<SmartSplitOptions>({
  name: 'smartSplit',

  addOptions() {
    return { ...DEFAULT_OPTIONS }
  },

  addProseMirrorPlugins() {
    return [smartSplitPlugin(this.options)]
  },

  onCreate() {
    const editor = this.editor

    // PaginationPlus may dispatch async updates (via rAF) during init.
    // Retry until .breaker elements exist or max attempts exhausted.
    // Dispatch an empty transaction to trigger the plugin's view.update,
    // which runs performDetectionAndSplit → syncPageBreaks.
    // We use an empty tr (no doc steps) as the wake-up signal — the plugin's
    // view.update fires on every dispatch, not just doc-changing ones.
    const trySync = (attempts: number) => {
      if (editor.isDestroyed) return
      const breakers = editor.view.dom.querySelectorAll('.breaker')
      if (breakers.length > 0 || attempts <= 0) {
        // Dispatch even when attempts exhausted (single-page doc) so
        // syncPageBreaks can clean up stale break-before styles from a
        // previous multi-page state.
        editor.view.dispatch(
          editor.state.tr.setMeta('addToHistory', false),
        )
        return
      }
      setTimeout(() => trySync(attempts - 1), 100)
    }
    requestAnimationFrame(() => trySync(5))
  },
})
