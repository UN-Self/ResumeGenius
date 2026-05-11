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
  /** Enable debug logging */
  debug: boolean
}

export const DEFAULT_OPTIONS: SmartSplitOptions = {
  debounce: 300,
  threshold: 0,
  jitter: 0,
  parentAttr: 'data-ss-parent',
  debug: false,
}

/** Block-level HTML tags that can cross page boundaries */
export const BLOCK_TAGS = new Set([
  'DIV', 'SECTION', 'HEADER', 'FOOTER', 'MAIN', 'ARTICLE',
  'NAV', 'ASIDE', 'UL', 'OL', 'TABLE', 'BLOCKQUOTE', 'FIGURE',
  'P', 'H1', 'H2', 'H3', 'H4', 'H5', 'H6', 'LI', 'TR',
])
