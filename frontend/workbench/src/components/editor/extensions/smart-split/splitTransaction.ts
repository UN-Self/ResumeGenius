import type { Transaction, EditorState } from '@tiptap/pm/state'
import { Node as PmNode, Fragment } from '@tiptap/pm/model'
import { Step, StepResult, type Mapping } from '@tiptap/pm/transform'

class ReplaceDocStep extends Step {
  private newDoc: PmNode
  constructor(newDoc: PmNode) { super(); this.newDoc = newDoc }
  apply(_doc: PmNode) { return StepResult.ok(this.newDoc) }
  invert(doc: PmNode) { return new ReplaceDocStep(doc) }
  map(_mapping: Mapping) { return this }
  merge(_other: Step) { return null }
  toJSON() { return { stepType: 'replaceDoc', doc: this.newDoc.toJSON() } }
}

// NOTE: Step.jsonID registration deferred — @tiptap/pm/transform's bundled Step
// may not expose jsonID reliably in all environments. The improved toJSON above
// is sufficient for inspection/logging; full Step.fromJSON round-trip can be
// added when collaborative editing or state persistence is implemented.

function rebuildAncestors(
  doc: PmNode,
  $pos: ReturnType<PmNode['resolve']>,
  parentDepth: number,
  frontNode: PmNode,
  padNode: PmNode,
  backNode: PmNode,
): PmNode {
  let replacement: PmNode[] = [frontNode, padNode, backNode]
  for (let d = parentDepth; d >= 1; d--) {
    const ancestor = $pos.node(d - 1)
    const idx = $pos.index(d - 1)
    const children: PmNode[] = []
    ancestor.forEach((child: PmNode, _o: number, i: number) => {
      if (i === idx) children.push(...replacement)
      else children.push(child)
    })
    if (d === 1) return doc.copy(Fragment.fromArray(children))
    replacement = [ancestor.type.create(ancestor.attrs, Fragment.fromArray(children))]
  }
  return doc
}

function isProtectedRootContainer(node: PmNode, depth: number): boolean {
  if (depth !== 1) return false
  const className = typeof node.attrs.class === 'string' ? node.attrs.class : ''
  return className.split(/\s+/).some((cls) => cls === 'resume' || cls === 'resume-document')
}

export function buildSplitTransaction(
  state: EditorState,
  crossPos: number,
  parentAttr: string,
  log: (...args: unknown[]) => void = () => {},
): Transaction | null {
  const { doc, tr, schema } = state
  const $pos = doc.resolve(crossPos)

  if ($pos.depth < 2) return null

  const parentDepth = $pos.depth - 1
  if (parentDepth < 1) return null

  const parent = $pos.node(parentDepth)
  const crossIndex = $pos.index(parentDepth)
  log(`depth=${$pos.depth} parentDepth=${parentDepth}`,
    `parentType=${parent.type.name} parentClass=${parent.attrs.class}`,
    `crossIndex=${crossIndex} childCount=${parent.childCount}`)
  if (isProtectedRootContainer(parent, parentDepth)) {
    log('skip split: protected root resume container')
    return null
  }
  if (crossIndex === 0) return null
  if (parent.childCount < 2) return null

  const front: PmNode[] = []
  const back: PmNode[] = []
  parent.forEach((child: PmNode, _o: number, i: number) => {
    if (i < crossIndex) front.push(child)
    else back.push(child)
  })
  log(`front=${front.length} back=${back.length}`,
    `frontClasses=${front.map((c) => c.attrs.class)}`,
    `backClasses=${back.map((c) => c.attrs.class)}`)
  if (front.length === 0 || back.length === 0) return null

  let parentId: string = parent.attrs[parentAttr]
  if (!parentId) parentId = Math.random().toString(36).slice(2, 10)

  const attrs: Record<string, unknown> = { ...parent.attrs, [parentAttr]: parentId }
  delete attrs.id

  const frontNode = parent.type.create(attrs, Fragment.fromArray(front))
  const backNode = parent.type.create(attrs, Fragment.fromArray(back))
  if (frontNode.nodeSize <= 2 || backNode.nodeSize <= 2) return null

  const padNode = schema.nodes.paragraph.create()

  log(`splitting into front(${front.length}) + <p> + back(${back.length})`)
  const newDoc = rebuildAncestors(doc, $pos, parentDepth, frontNode, padNode, backNode)
  tr.step(new ReplaceDocStep(newDoc))
  // TipTap's trailing-node plugin auto-appends a paragraph when the doc ends
  // with a non-textblock node (like our back section). This meta prevents it.
  tr.setMeta('skipTrailingNode', true)
  return tr
}
