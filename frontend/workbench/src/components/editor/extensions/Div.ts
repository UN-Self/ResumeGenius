import { Node } from '@tiptap/core'

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
    return CONTAINER_TAGS.map((tag) => ({
      tag,
      getAttrs: (element: HTMLElement) => ({
        originalTag: element.tagName.toLowerCase(),
      }),
    }))
  },

  renderHTML({ node }) {
    const tag = node.attrs.originalTag || 'div'
    const attrs: Record<string, string> = {}
    if (node.attrs.class) attrs.class = node.attrs.class
    if (node.attrs.style) attrs.style = node.attrs.style
    return [tag, attrs, 0]
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
