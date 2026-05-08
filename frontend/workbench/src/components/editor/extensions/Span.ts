import { Mark, mergeAttributes } from '@tiptap/core'
import { nullSafeAttr } from './attributes'

/**
 * Inline span mark that preserves <span> elements with a `class` attribute
 * through ProseMirror parsing. Only matches spans with `class` but NO `style`
 * — spans with `style` are handled by the TextStyle mark (priority 101).
 */
export const Span = Mark.create({
  name: 'span',
  priority: 50,

  parseHTML() {
    return [
      {
        tag: 'span',
        consuming: false,
        getAttrs: (element: HTMLElement) => {
          if (element.hasAttribute('class') && !element.hasAttribute('style')) {
            return {}
          }
          return false
        },
      },
    ]
  },

  renderHTML({ HTMLAttributes }) {
    return ['span', mergeAttributes(HTMLAttributes), 0]
  },

  addAttributes() {
    return {
      class: nullSafeAttr('class'),
    }
  },
})
