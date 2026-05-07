import { Node, mergeAttributes } from '@tiptap/core'

const CONTAINER_TAGS = ['div', 'section', 'header', 'footer', 'main', 'article', 'nav', 'aside'] as const

/**
 * Block container node that preserves <div>, <section>, <header>, etc.
 * with their class and style attributes through ProseMirror parsing.
 * Renders as the original tag name (e.g., <section> stays <section>).
 */
export const Div = Node.create({
  name: 'div',
  group: 'block',
  content: 'block*',
  selectable: false,
  draggable: false,

  parseHTML() {
    return CONTAINER_TAGS.map((tag) => ({ tag }))
  },

  renderHTML({ node, HTMLAttributes }) {
    const tag = node.attrs.originalTag || 'div'
    // mergeAttributes calls each attribute's renderHTML callback:
    // - class/style → rendered normally
    // - originalTag → renderHTML returns {} (suppressed from DOM)
    return [tag, mergeAttributes(HTMLAttributes), 0]
  },

  addAttributes() {
    return {
      originalTag: {
        default: 'div',
        parseHTML: (element: HTMLElement) => element.tagName.toLowerCase(),
        renderHTML: () => ({}),
      },
      class: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('class'),
        renderHTML: (attributes: Record<string, string>) => {
          if (!attributes.class) return {}
          return { class: attributes.class }
        },
      },
      style: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('style'),
        renderHTML: (attributes: Record<string, string>) => {
          if (!attributes.style) return {}
          return { style: attributes.style }
        },
      },
    }
  },
})
