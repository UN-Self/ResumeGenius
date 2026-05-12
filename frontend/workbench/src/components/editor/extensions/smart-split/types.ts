/** Y-position range of a .breaker element in the editor viewport */
export interface BreakerPosition {
  top: number
  bottom: number
}

/** A block element detected to cross a page boundary */
export interface CrossingInfo {
  /** ProseMirror position of the crossing element's start */
  pos: number
  /** Index of the breaker this element crosses */
  breakerIndex: number
}

export interface SmartSplitOptions {
  /** Debounce delay in ms after document change */
  debounce: number
  /** Pixel threshold for crossing detection */
  threshold: number
  /** Shrink detection zone from breaker bottom (avoids false positives on new-page content) */
  jitter: number
  /** Attribute name for tracing split siblings */
  parentAttr: string
  /** Auto-sync break-before: page after split so PDF export matches canvas pagination */
  insertPageBreaks: boolean
  /** Enable debug logging */
  debug: boolean
}

export const DEFAULT_OPTIONS: SmartSplitOptions = {
  debounce: 300,
  threshold: 4,
  jitter: 0,
  parentAttr: 'data-ss-parent',
  insertPageBreaks: true,
  debug: false,
}

/** Block-level HTML tags that can cross page boundaries */
const EXTRA_BLOCK_TAGS = [
  'TABLE', 'BLOCKQUOTE', 'FIGURE',
  'P', 'H1', 'H2', 'H3', 'H4', 'H5', 'H6', 'TR',
] as const

/** Container tags from Div extension (must stay in sync) */
const CONTAINER_TAGS = ['div', 'section', 'header', 'footer', 'main', 'article', 'nav', 'aside'] as const

export const BLOCK_TAGS = new Set([
  ...CONTAINER_TAGS.map(t => t.toUpperCase()),
  ...EXTRA_BLOCK_TAGS,
])
