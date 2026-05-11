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
})
