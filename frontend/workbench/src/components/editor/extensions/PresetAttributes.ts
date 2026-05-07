import { Extension } from '@tiptap/core'
import { nullSafeAttr } from './attributes'

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
          class: nullSafeAttr('class'),
          style: nullSafeAttr('style'),
        },
      },
    ]
  },
})
