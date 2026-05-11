import type { Transaction, EditorState } from '@tiptap/pm/state'
import { Fragment } from '@tiptap/pm/model'
import { Step, StepResult } from '@tiptap/pm/transform'

class ReplaceDocStep extends Step {
  private newDoc: any
  constructor(newDoc: any) { super(); this.newDoc = newDoc }
  apply(_doc: any) { return StepResult.ok(this.newDoc) }
  invert(doc: any) { return new ReplaceDocStep(doc) }
  map(_mapping: any) { return this }
  merge(_other: any) { return null }
  toJSON() { return { stepType: 'replaceDoc' } }
}

function rebuildAncestors(
  doc: any, $pos: any, parentDepth: number, frontNode: any, padNode: any, backNode: any,
): any {
  let replacement: any[] = [frontNode, padNode, backNode]
  for (let d = parentDepth; d >= 1; d--) {
    const ancestor = $pos.node(d - 1)
    const idx = $pos.index(d - 1)
    const children: any[] = []
    ancestor.forEach((child: any, _o: number, i: number) => {
      if (i === idx) children.push(...replacement)
      else children.push(child)
    })
    if (d === 1) return doc.copy(Fragment.fromArray(children))
    replacement = [ancestor.type.create(ancestor.attrs, Fragment.fromArray(children))]
  }
  return doc
}

export function buildSplitTransaction(
  state: EditorState,
  crossPos: number,
  parentAttr: string,
): Transaction | null {
  const { doc, tr, schema } = state
  const $pos = doc.resolve(crossPos)
  if ($pos.depth < 2) return null

  const parentDepth = $pos.depth - 1
  if (parentDepth < 1) return null

  const parent = $pos.node(parentDepth)
  const crossIndex = $pos.index(parentDepth)
  console.log(`[SmartSplit.buildTr] depth=${$pos.depth} parentDepth=${parentDepth}`,
    `parentType=${parent.type.name} parentClass=${parent.attrs.class}`,
    `crossIndex=${crossIndex} childCount=${parent.childCount}`)
  if (crossIndex === 0) return null
  if (parent.childCount < 2) return null

  const front: any[] = []
  const back: any[] = []
  parent.forEach((child: any, _o: number, i: number) => {
    if (i < crossIndex) front.push(child)
    else back.push(child)
  })
  console.log(`[SmartSplit.buildTr] front=${front.length} back=${back.length}`,
    `frontClasses=${front.map((c: any) => c.attrs.class)}`,
    `backClasses=${back.map((c: any) => c.attrs.class)}`)
  if (front.length === 0 || back.length === 0) return null

  let parentId: string = parent.attrs[parentAttr]
  if (!parentId) parentId = Math.random().toString(36).slice(2, 10)

  const attrs: Record<string, any> = { ...parent.attrs, [parentAttr]: parentId }
  delete attrs.id

  const frontNode = parent.type.create(attrs, Fragment.fromArray(front))
  const backNode = parent.type.create(attrs, Fragment.fromArray(back))
  if (frontNode.nodeSize <= 2 || backNode.nodeSize <= 2) return null

  const padNode = schema.nodes.paragraph.create()

  console.log(`[SmartSplit.buildTr] splitting into front(${front.length}) + <p> + back(${back.length})`)
  const newDoc = rebuildAncestors(doc, $pos, parentDepth, frontNode, padNode, backNode)
  tr.step(new ReplaceDocStep(newDoc))
  tr.setMeta('skipTrailingNode', true)
  return tr
}
