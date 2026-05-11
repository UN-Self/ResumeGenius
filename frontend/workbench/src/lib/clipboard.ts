import { Fragment, Slice } from 'prosemirror-model'

const MIME_INTERNAL = 'application/x-resume-genius'

let lastCopy: { json: string; text: string } | null = null

export function captureCopy(
  state: { selection: { content(): any }; doc: any },
  from: number,
  to: number,
) {
  const slice = state.selection.content()
  const text = state.doc.textBetween(from, to, '\n')
  const json = JSON.stringify({
    content: slice.content.toJSON(),
    openStart: slice.openStart,
    openEnd: slice.openEnd,
  })
  lastCopy = { json, text }
  return { text, json }
}

export function sliceFromJson(schema: any, json: string) {
  const { content, openStart, openEnd } = JSON.parse(json)
  return new Slice(Fragment.fromJSON(schema, content), openStart, openEnd)
}

export function getMimeType() {
  return MIME_INTERNAL
}

export function getLastCopy() {
  return lastCopy
}
