import { Extension } from '@tiptap/core'

/**
 * Adds `class` and `style` attributes to standard block nodes
 * so that AI-generated HTML class attributes survive ProseMirror parsing.
 * Uses the same addGlobalAttributes pattern as TextAlign.
 */
export const PresetAttributes = Extension.create({
  name: 'presetAttributes',

  addGlobalAttributes() {
    return [
      {
        types: [
          'paragraph',
          'heading',
          'listItem',
          'bulletList',
          'orderedList',
          'blockquote',
          'codeBlock',
        ],
        attributes: {
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
        },
      },
    ]
  },
})
