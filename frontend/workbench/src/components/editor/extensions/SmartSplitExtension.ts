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
    // Retry until .breaker elements exist or max attempts exhausted,
    // then trigger syncPageBreaks to ensure break-before: page is always
    // present in exported HTML — even without user edits.
    const trySync = (attempts: number) => {
      if (editor.isDestroyed) return
      if (attempts <= 0) return
      const breakers = editor.view.dom.querySelectorAll('.breaker')
      if (breakers.length === 0) {
        setTimeout(() => trySync(attempts - 1), 100)
        return
      }
      editor.view.dispatch(
        editor.state.tr.setMeta('addToHistory', false),
      )
    }
    requestAnimationFrame(() => trySync(5))
  },
})
