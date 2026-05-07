/**
 * extract-styles.ts
 *
 * Extracts <style> blocks from full HTML documents, scopes CSS selectors
 * to `.resume-document` (the editor container), and strips dimension
 * properties from root container elements to avoid conflicts with the
 * A4Canvas page sizing.
 */

const SCOPE_PREFIX = '.resume-document'

/**
 * Defense-in-depth: strip XSS-adjacent CSS patterns before injection.
 * Modern browsers have disabled `expression()` for decades and `url(javascript:)`
 * does not execute in CSS property values, but explicit removal costs nothing
 * and adds a safety net against future CSS-based attacks or legacy renderers.
 */
const DANGEROUS_CSS_PATTERNS: Array<{ pattern: RegExp; replace: string }> = [
  { pattern: /\bexpression\s*\(/gi, replace: 'blocked(' },
  { pattern: /\burl\s*\(\s*["'\s]*(?:javascript|data\s*:|vbscript)/gi, replace: 'url(blocked:' },
  { pattern: /^\s*behavior\s*:.*$/gim, replace: '/* blocked */' },
]

function sanitizeCSS(css: string): string {
  let sanitized = css
  for (const { pattern, replace } of DANGEROUS_CSS_PATTERNS) {
    sanitized = sanitized.replace(pattern, replace)
  }
  return sanitized
}

const DIMENSION_PROPERTIES = new Set([
  'width',
  'min-width',
  'max-width',
  'height',
  'min-height',
  'max-height',
  'padding',
  'padding-top',
  'padding-right',
  'padding-bottom',
  'padding-left',
  'margin',
  'margin-top',
  'margin-right',
  'margin-bottom',
  'margin-left',
])

/**
 * Default-valued longhands that CSSOM produces when expanding `background` shorthand.
 * When a `background-color` longhand is present, these can be safely removed to
 * keep the promoted output concise.
 */
const DEFAULT_BG_LONGHANDS = new Set([
  'background-image: none;',
  'background-repeat: repeat;',
  'background-attachment: scroll;',
  'background-position: 0% 0%;',
  'background-size: auto;',
  'background-origin: padding-box;',
  'background-clip: border-box;',
])

// ─── Shared CSS helpers ─────────────────────────────────────────────

/**
 * Parse a CSS string into a CSSStyleSheet.
 * Returns null if parsing fails (unparseable CSS).
 */
function parseCss(css: string): CSSStyleSheet | null {
  const sheet = new CSSStyleSheet()
  try {
    sheet.replaceSync(css)
    return sheet
  } catch {
    return null
  }
}

interface WalkOptions {
  /** Skip @media print blocks entirely */
  skipPrintMedia: boolean
  /** Preserve non-CSSStyleRule, non-CSSMediaRule at-rules as-is */
  preserveOtherRules: boolean
}

/**
 * Walk a CSSRuleList, applying onStyleRule to each CSSStyleRule.
 * CSSMediaRule blocks are recursively walked; non-print results are
 * wrapped back into @media blocks.
 *
 * When `preserveOtherRules` is false, non-CSSStyleRule/non-CSSMediaRule
 * at-rules (@supports, @layer, @font-face, @keyframes, etc.) are silently
 * dropped. This is intentional for scoped-selector rewrites where these
 * at-rules' internal selectors would also need rewriting logic that isn't
 * yet implemented. Set `preserveOtherRules: true` to keep them as-is.
 *
 * Returns an array of CSS rule text strings.
 */
function walkRuleList(
  rules: CSSRuleList,
  onStyleRule: (rule: CSSStyleRule) => string | null,
  options: WalkOptions,
): string[] {
  const output: string[] = []

  for (let i = 0; i < rules.length; i++) {
    const rule = rules[i]

    if (rule instanceof CSSStyleRule) {
      const result = onStyleRule(rule)
      if (result !== null) output.push(result)
    } else if (rule instanceof CSSMediaRule) {
      const mediaText = (rule as CSSMediaRule).media.mediaText
      if (/print/i.test(mediaText)) {
        if (options.skipPrintMedia) continue
        output.push(rule.cssText)
        continue
      }

      const inner = walkRuleList(
        (rule as CSSMediaRule).cssRules,
        onStyleRule,
        options,
      )
      if (inner.length > 0) {
        output.push(`@media ${mediaText} {\n${inner.join('\n')}\n}`)
      }
    } else {
      if (options.preserveOtherRules) {
        output.push(rule.cssText)
      }
    }
  }

  return output
}

// ─── scopeSelectors ───────────────────────────────────────────────────

/**
 * Scope a single simple selector (no commas) to `.resume-document`.
 * The scope prefix is prepended once at the beginning of the selector,
 * not to each individual part of a compound selector.
 */
function scopeSingleSelector(sel: string): string {
  const trimmed = sel.trim()
  if (!trimmed) return ''

  // Already scoped — skip
  if (trimmed.startsWith(SCOPE_PREFIX)) return trimmed

  // Universal
  if (trimmed === '*') return `${SCOPE_PREFIX} *`

  // html / body
  if (trimmed === 'html' || trimmed === 'body') return SCOPE_PREFIX

  // Pseudo-only selectors like :root
  if (trimmed.startsWith(':')) return `${SCOPE_PREFIX} ${trimmed}`

  // For everything else, prepend the scope prefix once at the start.
  // This handles compound selectors like `.section > .title` → `.resume-document .section > .title`
  return `${SCOPE_PREFIX} ${trimmed}`
}

/**
 * Scope all selectors in a CSS rule's selector list.
 */
function scopeSelectorList(selectorText: string): string {
  return selectorText
    .split(',')
    .map((s) => scopeSingleSelector(s))
    .join(', ')
}

/**
 * Process a single CSS rule (non-at-rule), scoping its selectors.
 */
function scopeRule(rule: CSSStyleRule): string {
  const selectors = scopeSelectorList(rule.selectorText)
  const props: string[] = []
  for (let i = 0; i < rule.style.length; i++) {
    const prop = rule.style[i]
    const value = rule.style.getPropertyValue(prop)
    const priority = rule.style.getPropertyPriority(prop)
    props.push(`  ${prop}: ${value}${priority ? ` !${priority}` : ''};`)
  }
  return `${selectors} {\n${props.join('\n')}\n}`
}

/**
 * Scope all selectors in a CSS string to `.resume-document`.
 *
 * - Prefixes selectors with `.resume-document`
 * - Rewrites `body`/`html` to `.resume-document`
 * - Rewrites `*` to `.resume-document *`
 * - Handles comma-separated selectors
 * - Handles compound selectors with combinators
 * - Skips `@page` rules entirely
 * - Skips `@media print` blocks
 * - Preserves non-print `@media` blocks with scoped internal selectors
 */
export function scopeSelectors(css: string): string {
  if (!css.trim()) return ''

  const sheet = parseCss(css)
  if (!sheet) {
    // Unparseable CSS should not be injected — risk of unscoped styles leaking out.
    return ''
  }

  const output = walkRuleList(
    sheet.cssRules,
    (rule: CSSStyleRule) => scopeRule(rule),
    { skipPrintMedia: true, preserveOtherRules: false },
  )

  return output.join('\n')
}

// ─── stripContainerDimensions ─────────────────────────────────────────

/**
 * Check if a selector ends with the given container class selector.
 * E.g., ".resume-document .resume" ends with ".resume"
 */
function selectorEndsWith(selector: string, suffix: string): boolean {
  const trimmed = selector.trim()
  // Exact match
  if (trimmed === suffix) return true
  // Ends with the suffix preceded by a space (descendant) or combinators >, +, ~
  if (trimmed.endsWith(' ' + suffix)) return true
  if (trimmed.endsWith('> ' + suffix)) return true
  if (trimmed.endsWith('+ ' + suffix)) return true
  if (trimmed.endsWith('~ ' + suffix)) return true
  // Ends with suffix as a compound (e.g. ".resume.other" starts with ".resume")
  // but we only want to match the class as a standalone token
  return false
}

/**
 * Remove dimension-related CSS properties from rules matching the
 * root container selector. If all properties of a rule are dimension-related,
 * the entire rule is removed.
 *
 * The `containerSelector` is the original unscoped selector (e.g. `.resume`).
 * After scoping, it may appear as `.resume-document .resume`.
 */
export function stripContainerDimensions(css: string, containerSelector: string): string {
  if (!css.trim()) return ''

  const sheet = parseCss(css)
  if (!sheet) {
    // If CSS can't be parsed, return as-is — better to show styled content
    // with extra container dimensions than to lose all styling.
    return css
  }

  /**
   * Strip dimension properties from a CSSStyleRule that matches the container.
   * Returns the cleaned rule text, or null if the rule should be dropped entirely.
   */
  function stripStyleRule(rule: CSSStyleRule): string | null {
    const selectors = rule.selectorText.split(',').map((s) => s.trim())
    const matchesContainer = selectors.some(
      (s) => selectorEndsWith(s, containerSelector),
    )

    if (!matchesContainer) return rule.cssText

    // Filter out dimension properties
    const keptProps: string[] = []
    for (let j = 0; j < rule.style.length; j++) {
      const prop = rule.style[j]
      if (!DIMENSION_PROPERTIES.has(prop)) {
        const value = rule.style.getPropertyValue(prop)
        const priority = rule.style.getPropertyPriority(prop)
        keptProps.push(`${prop}: ${value}${priority ? ` !${priority}` : ''};`)
      }
    }

    if (keptProps.length > 0) {
      return `${rule.selectorText} {\n  ${keptProps.join('\n  ')}\n}`
    }
    // If all props were dimension-related, drop the rule entirely
    return null
  }

  const output = walkRuleList(
    sheet.cssRules,
    (rule: CSSStyleRule) => stripStyleRule(rule),
    { skipPrintMedia: false, preserveOtherRules: true },
  )

  return output.join('\n')
}

// ─── promoteContainerBackground ────────────────────────────────────

/**
 * Check if a CSS property is background-related.
 * Covers both the `background` shorthand and all `background-*` longhands.
 */
function isBackgroundProperty(prop: string): boolean {
  return prop === 'background' || prop.startsWith('background-')
}

/**
 * Promote background properties from root container rules to `.resume-document`.
 *
 * After scoping, root container background ends up on `.resume-document .resume`,
 * which only covers the content area (inside padding). Promoting to `.resume-document`
 * ensures the background covers the full A4 canvas including padding.
 *
 * Non-background properties on the root container are left untouched.
 *
 * NOTE: Relies on stripContainerDimensions being called first to remove
 * dimension properties from the container rule. Without that step, the
 * original scoped rule (more specific selector) could override the
 * promoted background on `.resume-document` (less specific selector).
 */
export function promoteContainerBackground(css: string, containerSelector: string): string {
  if (!css.trim()) return ''

  const sheet = parseCss(css)
  if (!sheet) return css

  const promotedProps: string[] = []

  function processStyleRule(rule: CSSStyleRule): { keptRule: string | null; bgProps: string[] } {
    const selectors = rule.selectorText.split(',').map((s) => s.trim())
    const matchesContainer = selectors.some(
      (s) => selectorEndsWith(s, containerSelector),
    )

    if (!matchesContainer) {
      return { keptRule: rule.cssText, bgProps: [] }
    }

    const bgProps: string[] = []
    const keptProps: string[] = []
    for (let i = 0; i < rule.style.length; i++) {
      const prop = rule.style[i]
      const value = rule.style.getPropertyValue(prop)
      const priority = rule.style.getPropertyPriority(prop)
      const propStr = `${prop}: ${value}${priority ? ` !${priority}` : ''};`

      if (isBackgroundProperty(prop)) {
        bgProps.push(propStr)
      } else {
        keptProps.push(propStr)
      }
    }

    const keptRule =
      keptProps.length > 0
        ? `${rule.selectorText} {\n  ${keptProps.join('\n  ')}\n}`
        : null

    return { keptRule, bgProps }
  }

  const output = walkRuleList(
    sheet.cssRules,
    (rule: CSSStyleRule) => {
      const { keptRule, bgProps } = processStyleRule(rule)
      promotedProps.push(...bgProps)
      return keptRule
    },
    { skipPrintMedia: false, preserveOtherRules: true },
  )

  if (promotedProps.length > 0) {
    // Deduplicate by property name. When the same property appears in
    // multiple container-class rules, the last one wins (CSS cascade).
    // Uses a Map keyed by prop_priority so different !important states
    // are tracked separately.
    const propMap = new Map<string, string>()
    for (const propStr of promotedProps) {
      const colonIdx = propStr.indexOf(':')
      const propName = propStr.slice(0, colonIdx)
      const valuePart = propStr.slice(colonIdx + 1)
      const key = valuePart.includes('!important')
        ? `${propName}!important`
        : propName
      propMap.set(key, propStr)
    }
    const uniqueProps = [...propMap.values()]

    // Clean up CSSOM shorthand expansion artifacts:
    // When `background: #fff` is parsed by CSSOM, it expands to many longhands
    // (background-color, background-image: none, background-repeat: repeat, etc.).
    // If we have a background-color, remove default-valued longhands to keep
    // the output concise.
    const hasBgColor = uniqueProps.some((p) => p.startsWith('background-color:'))
    const cleaned = hasBgColor
      ? uniqueProps.filter((p) => !DEFAULT_BG_LONGHANDS.has(p))
      : uniqueProps

    const bgRule = `.resume-document {\n  ${cleaned.join('\n  ')}\n}`
    output.unshift(bgRule)
  }

  return output.join('\n')
}

// ─── getRootContainerClasses ──────────────────────────────────────────

/**
 * Find the first child element of `<body>` and return its CSS classes.
 * Accepts a pre-parsed Document to avoid redundant DOMParser calls.
 */
export function getRootContainerClasses(doc: Document): string[] {
  const body = doc.body
  if (!body) return []

  const firstChild = body.firstElementChild
  if (!firstChild) return []

  const classList = firstChild.classList
  return classList ? Array.from(classList) : []
}

// ─── extractStyles (main entry point) ─────────────────────────────────

export interface ExtractedStyles {
  bodyHtml: string
  scopedCSS: string
}

/**
 * Parse a full HTML document, extract `<style>` blocks and `<body>` content,
 * scope all CSS selectors to `.resume-document`, and strip dimension
 * properties from root container elements.
 */
export function extractStyles(html: string): ExtractedStyles {
  if (!html.trim()) {
    return { bodyHtml: '', scopedCSS: '' }
  }

  const parser = new DOMParser()
  const doc = parser.parseFromString(html, 'text/html')

  // Extract <style> content
  const styleElements = doc.querySelectorAll('style')
  const rawCSS = Array.from(styleElements)
    .map((el) => el.textContent ?? '')
    .join('\n')

  // Extract body innerHTML
  const bodyHtml = doc.body?.innerHTML ?? ''

  // If no styles found, return body only
  if (!rawCSS.trim()) {
    return { bodyHtml, scopedCSS: '' }
  }

  // Scope selectors
  let scopedCSS = scopeSelectors(sanitizeCSS(rawCSS))

  // Strip container dimensions for root container classes
  const containerClasses = getRootContainerClasses(doc)
  for (const cls of containerClasses) {
    scopedCSS = stripContainerDimensions(scopedCSS, `.${cls}`)
  }

  // Promote root container background to .resume-document for full A4 canvas coverage
  for (const cls of containerClasses) {
    scopedCSS = promoteContainerBackground(scopedCSS, `.${cls}`)
  }

  return { bodyHtml, scopedCSS }
}
