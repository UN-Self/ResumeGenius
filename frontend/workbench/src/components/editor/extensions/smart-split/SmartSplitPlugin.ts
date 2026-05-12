import { Plugin, PluginKey } from '@tiptap/pm/state'
import type { EditorState } from '@tiptap/pm/state'
import type { EditorView } from '@tiptap/pm/view'
import { undo } from 'prosemirror-history'
import { getBreakerPositions, findCrossingPositions } from './detectCrossings'
import { buildSplitTransaction } from './splitTransaction'
import type { SmartSplitOptions } from './types'

const pluginKey = new PluginKey('smartSplit')

interface SmartSplitState {
  isOwnDispatch: boolean
  /** Document snapshot from before the last user edit */
  preEditDoc: EditorState['doc'] | null
}

export function smartSplitPlugin(options: SmartSplitOptions) {
  let timer: ReturnType<typeof setTimeout> | null = null
  const log = options.debug
    ? (...args: any[]) => console.log('[SmartSplit]', ...args)
    : () => {}

  return new Plugin({
    key: pluginKey,

    state: {
      init(): SmartSplitState {
        return { isOwnDispatch: false, preEditDoc: null }
      },
      apply(tr, value: SmartSplitState, oldState: EditorState): SmartSplitState {
        const isOwnDispatch = !!tr.getMeta(pluginKey)?.ownDispatch
        if (isOwnDispatch) {
          return { ...value, isOwnDispatch }
        }
        if (tr.docChanged) {
          return { isOwnDispatch, preEditDoc: oldState.doc }
        }
        return { ...value, isOwnDispatch }
      },
    },

    view(_editorView: EditorView) {
      return {
        update(_view: EditorView) {
          const pluginState = pluginKey.getState(_view.state) as SmartSplitState | undefined
          if (pluginState?.isOwnDispatch) {
            log('skipping re-detection after own dispatch')
            return
          }

          if (timer) clearTimeout(timer)

          timer = setTimeout(() => {
            performDetectionAndSplit(_view, options, log)
            timer = null
          }, options.debounce)
        },
        destroy() {
          if (timer) clearTimeout(timer)
        },
      }
    },
  })
}

function performDetectionAndSplit(
  view: EditorView, options: SmartSplitOptions,
  log: (...args: any[]) => void,
) {
  const editorDom = view.dom
  const breakers = getBreakerPositions(editorDom)
  log('breakers:', breakers.length, breakers)
  if (breakers.length === 0) return

  const crossings = findCrossingPositions(view, editorDom, breakers, options.threshold, options.jitter)
  log('crossings:', crossings.length,
    crossings.map(c => ({ pos: c.pos, breaker: c.breakerIndex })))
  if (crossings.length === 0) return

  crossings.sort((a, b) => a.pos - b.pos)

  const { state } = view
  let crossPos = -1
  for (const c of crossings) {
    const $pos = state.doc.resolve(c.pos)
    const crossIndex = $pos.index($pos.depth - 1)
    log(`crossing pos=${c.pos} depth=${$pos.depth}`,
      `parent(${$pos.depth - 1})=${$pos.node($pos.depth - 1)?.type?.name ?? '?'}`,
      `index=${crossIndex}`)
    if ($pos.depth >= 2 && crossIndex > 0 && crossPos < 0) {
      crossPos = c.pos
    }
  }
  if (crossPos < 0) {
    log('no splittable crossing (depth>=2, index>0), skipping')
    return
  }
  log('selected crossPos:', crossPos)

  const tr = buildSplitTransaction(state, crossPos, options.parentAttr, log)
  if (!tr) {
    log('buildSplitTransaction returned null ✗')
    return
  }

  // If edit + split produces the same doc as before the edit, it's a no-op.
  // Undo the user's edit to roll back to the pre-edit state and skip the split.
  const { preEditDoc } = pluginKey.getState(state) as SmartSplitState
  const resultState = state.apply(tr)
  if (preEditDoc && resultState.doc.eq(preEditDoc)) {
    log('split result identical to pre-edit state → undoing user edit')
    undo(view.state, (t) => view.dispatch(t))
    return
  }

  log('dispatching transaction ✓')
  tr.setMeta(pluginKey, { ownDispatch: true })
  view.dispatch(tr)
}
