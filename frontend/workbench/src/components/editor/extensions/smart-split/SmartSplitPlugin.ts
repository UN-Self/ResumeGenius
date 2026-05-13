import { Plugin, PluginKey } from '@tiptap/pm/state'
import type { EditorState } from '@tiptap/pm/state'
import type { EditorView } from '@tiptap/pm/view'
import { undo } from 'prosemirror-history'
import { getBreakerPositions, findCrossingPositions, findPageStartPositions } from './detectCrossings'
import { buildSplitTransaction } from './splitTransaction'
import { appendBreakBefore, removeBreakBefore, resolveToBlockPos, promotePastList } from './styleUtils'
import type { SmartSplitOptions, BreakerPosition } from './types'

const pluginKey = new PluginKey('smartSplit')

interface SmartSplitState {
  isOwnDispatch: boolean
}

export function smartSplitPlugin(options: SmartSplitOptions) {
  let timer: ReturnType<typeof setTimeout> | null = null
  // Suppression counter: incremented before each SmartSplit dispatch, decremented
  // in view.update. Prevents cascade loops from PaginationPlus (meta-only rAF),
  // trailing-node (paragraph append), or any other reactive plugin.
  let suppressCount = 0
  const suppress = {
    preSplitDoc: null as EditorState['doc'] | null,
  }
  const log = options.debug
    ? (...args: any[]) => console.log('[SmartSplit]', ...args)
    : () => {}

  return new Plugin({
    key: pluginKey,

    state: {
      init(): SmartSplitState {
        return { isOwnDispatch: false }
      },
      apply(tr, value: SmartSplitState): SmartSplitState {
        const isOwnDispatch = !!tr.getMeta(pluginKey)?.ownDispatch
        if (isOwnDispatch !== value.isOwnDispatch) {
          return { ...value, isOwnDispatch }
        }
        return value
      },
    },

    view(_editorView: EditorView) {
      return {
        update(_view: EditorView) {
          // Drain one suppress slot per view.update. When suppressCount > 0,
          // the update was triggered (directly or indirectly) by our own
          // dispatches — skip re-detection.
          if (suppressCount > 0) {
            suppressCount--
            log('skipping re-detection (suppressCount)', suppressCount)
            return
          }

          const pluginState = pluginKey.getState(_view.state) as SmartSplitState | undefined
          if (pluginState?.isOwnDispatch) {
            log('skipping re-detection after own dispatch')
            return
          }

          if (suppress.preSplitDoc && _view.state.doc.eq(suppress.preSplitDoc)) {
            suppress.preSplitDoc = null
            log('split undone by user → undoing user edit')
            suppressCount++
            undo(_view.state, (t) => {
              t.setMeta(pluginKey, { ownDispatch: true })
              _view.dispatch(t)
            })
            return
          }

          if (timer) clearTimeout(timer)

          timer = setTimeout(() => {
            // Final guard: re-check suppression in case a late-arriving
            // plugin dispatch (e.g. trailing-node via appendTransaction)
            // set isOwnDispatch after the view.update that started this timer.
            const current = pluginKey.getState(_view.state) as SmartSplitState | undefined
            if (current?.isOwnDispatch || suppressCount > 0) {
              log('timer fired but suppressed, skipping')
              timer = null
              return
            }
            performDetectionAndSplit(_view, options, log, suppress, () => { suppressCount++ })
            timer = null
          }, options.debounce)
        },
        destroy() {
          if (timer) clearTimeout(timer)
          suppress.preSplitDoc = null
          suppressCount = 0
        },
      }
    },
  })
}

function performDetectionAndSplit(
  view: EditorView, options: SmartSplitOptions,
  log: (...args: any[]) => void,
  suppress: { preSplitDoc: EditorState['doc'] | null },
  onDispatch: () => void,
) {
  const editorDom = view.dom
  const breakers = getBreakerPositions(editorDom)
  log('breakers:', breakers.length, breakers)
  if (breakers.length === 0) {
    if (options.insertPageBreaks) {
      syncPageBreaks(view, breakers, log, onDispatch)
    }
    return
  }

  const crossings = findCrossingPositions(view, editorDom, breakers, options.threshold, options.jitter)
  log('crossings:', crossings.length,
    crossings.map(c => ({ pos: c.pos, breaker: c.breakerIndex })))

  let didSplit = false
  if (crossings.length > 0) {
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
    if (crossPos >= 0) {
      log('selected crossPos:', crossPos)

      const tr = buildSplitTransaction(state, crossPos, options.parentAttr, log)
      if (!tr) {
        log('buildSplitTransaction returned null ✗')
      } else {
        const preSplitDoc = state.doc
        const resultState = state.apply(tr)
        if (resultState.doc.eq(preSplitDoc)) {
          log('split result identical to pre-split state, skipping')
        } else {
          log('dispatching transaction ✓')
          suppress.preSplitDoc = preSplitDoc
          tr.setMeta(pluginKey, { ownDispatch: true })
          onDispatch()
          view.dispatch(tr)
          didSplit = true
        }
      }
    } else {
      log('no splittable crossing (depth>=2, index>0), skipping')
    }
  }

  if (options.insertPageBreaks) {
    const currentBreakers = didSplit ? getBreakerPositions(editorDom) : breakers
    syncPageBreaks(view, currentBreakers, log, onDispatch)
  }
}

function syncPageBreaks(
  view: EditorView,
  breakers: BreakerPosition[],
  log: (...args: any[]) => void,
  onDispatch: () => void,
) {
  const { state } = view
  const { tr, doc } = state
  if (!doc?.descendants) return

  // Clean up all existing break-before styles
  doc.descendants((node, pos) => {
    if (!node.isBlock) return false
    const style = node.attrs.style as string | null
    if (style && style.includes('break-before')) {
      const cleaned = removeBreakBefore(style)
      tr.setNodeMarkup(pos, undefined, { ...node.attrs, style: cleaned })
    }
    return true
  })

  // Find page-start nodes
  const pageStarts = findPageStartPositions(view, view.dom, breakers)
  log('pageStarts:', pageStarts.length, pageStarts)

  // Add break-before: page to page-start nodes
  for (const pos of pageStarts) {
    const blockPos = resolveToBlockPos(tr.doc, pos)
    // Promote past list wrappers — break-before on a paragraph child
    // of listItem is ignored by Chrome's PDF renderer.
    const targetPos = promotePastList(tr.doc, blockPos)
    const node = tr.doc.nodeAt(targetPos)
    if (!node) continue
    const currentStyle = (node.attrs.style as string) || ''
    const newStyle = appendBreakBefore(currentStyle)
    tr.setNodeMarkup(targetPos, undefined, { ...node.attrs, style: newStyle })
  }

  if (tr.docChanged) {
    log('syncPageBreaks dispatching ✓')
    tr.setMeta(pluginKey, { ownDispatch: true })
    tr.setMeta('addToHistory', false)
    tr.setMeta('skipTrailingNode', true)
    onDispatch()
    view.dispatch(tr)
  }
}
