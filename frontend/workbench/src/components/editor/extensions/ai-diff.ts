import { Mark, markInputRule, markPasteRule } from '@tiptap/core'

/**
 * Mark for `<del>` tags -- represents AI-deleted text in diff view.
 * Input rule: ~~text~~  |  Paste rule: same.
 */
export const Deletion = Mark.create({
  name: 'deletion',

  parseHTML() {
    return [{ tag: 'del' }]
  },

  renderHTML({ HTMLAttributes }) {
    return ['del', HTMLAttributes, 0]
  },

  addInputRules() {
    return [
      markInputRule({
        find: /(?:^|\s)~~((?:[^~]+))~~$/,
        type: this.type,
      }),
    ]
  },

  addPasteRules() {
    return [
      markPasteRule({
        find: /(?:^|\s)~~((?:[^~]+))~~$/g,
        type: this.type,
      }),
    ]
  },
})

/**
 * Mark for `<ins>` tags -- represents AI-inserted text in diff view.
 * Input rule: ++text++  |  Paste rule: same.
 */
export const Insertion = Mark.create({
  name: 'insertion',

  parseHTML() {
    return [{ tag: 'ins' }]
  },

  renderHTML({ HTMLAttributes }) {
    return ['ins', HTMLAttributes, 0]
  },

  addInputRules() {
    return [
      markInputRule({
        find: /\+\+((?:[^+]+))\+\+$/,
        type: this.type,
      }),
    ]
  },

  addPasteRules() {
    return [
      markPasteRule({
        find: /\+\+((?:[^+]+))\+\+$/g,
        type: this.type,
      }),
    ]
  },
})
