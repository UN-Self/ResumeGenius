/**
 * extract-styles.ts
 *
 * Extracts <style> blocks from full HTML documents, scopes CSS selectors
 * to `.resume-document` (the editor container), and strips dimension
 * properties from root container elements to avoid conflicts with the
 * A4Canvas page sizing.
 */

const SCOPE_PREFIX = '.resume-document'

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

  const sheet = new CSSStyleSheet()
  try {
    sheet.replaceSync(css)
  } catch {
    // Unparseable CSS should not be injected — risk of unscoped styles leaking out.
    return ''
  }

  const output: string[] = []

  function processRuleList(rules: CSSRuleList): void {
    for (let i = 0; i < rules.length; i++) {
      const rule = rules[i]

      if (rule instanceof CSSStyleRule) {
        output.push(scopeRule(rule))
      } else if (rule instanceof CSSMediaRule) {
        const mediaText = (rule as CSSMediaRule).media.mediaText
        // Skip @media print
        if (/print/i.test(mediaText)) continue

        const inner: string[] = []
        const mediaRule = rule as CSSMediaRule
        for (let j = 0; j < mediaRule.cssRules.length; j++) {
          const innerRule = mediaRule.cssRules[j]
          if (innerRule instanceof CSSStyleRule) {
            inner.push(scopeRule(innerRule))
          }
        }
        if (inner.length > 0) {
          output.push(`@media ${mediaText} {\n${inner.join('\n')}\n}`)
        }
      }
      // Skip @page and other at-rules
    }
  }

  processRuleList(sheet.cssRules)

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

  const sheet = new CSSStyleSheet()
  try {
    sheet.replaceSync(css)
  } catch {
    // If CSS can't be parsed, return as-is — better to show styled content
    // with extra container dimensions than to lose all styling.
    return css
  }

  const output: string[] = []

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

  for (let i = 0; i < sheet.cssRules.length; i++) {
    const rule = sheet.cssRules[i]

    if (rule instanceof CSSStyleRule) {
      const result = stripStyleRule(rule)
      if (result !== null) output.push(result)
    } else if (rule instanceof CSSMediaRule) {
      const mediaText = (rule as CSSMediaRule).media.mediaText
      // Skip @media print blocks (already dropped by scopeSelectors)
      if (/print/i.test(mediaText)) {
        output.push(rule.cssText)
        continue
      }

      const mediaRule = rule as CSSMediaRule
      const inner: string[] = []
      for (let j = 0; j < mediaRule.cssRules.length; j++) {
        const innerRule = mediaRule.cssRules[j]
        if (innerRule instanceof CSSStyleRule) {
          const result = stripStyleRule(innerRule)
          if (result !== null) inner.push(result)
        } else {
          inner.push(innerRule.cssText)
        }
      }
      if (inner.length > 0) {
        output.push(`@media ${mediaText} {\n${inner.join('\n')}\n}`)
      }
    } else {
      // Preserve other at-rules as-is
      output.push(rule.cssText)
    }
  }

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
 */
export function promoteContainerBackground(css: string, containerSelector: string): string {
  if (!css.trim()) return ''

  const sheet = new CSSStyleSheet()
  try {
    sheet.replaceSync(css)
  } catch {
    return css
  }

  const promotedProps: string[] = []
  const output: string[] = []

  function processStyleRule(rule: CSSStyleRule): void {
    const selectors = rule.selectorText.split(',').map((s) => s.trim())
    const matchesContainer = selectors.some(
      (s) => selectorEndsWith(s, containerSelector),
    )

    if (!matchesContainer) {
      output.push(rule.cssText)
      return
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

    promotedProps.push(...bgProps)

    if (keptProps.length > 0) {
      output.push(`${rule.selectorText} {\n  ${keptProps.join('\n  ')}\n}`)
    }
    // If all props were background-related, the rule is dropped (promoted instead)
  }

  for (let i = 0; i < sheet.cssRules.length; i++) {
    const rule = sheet.cssRules[i]

    if (rule instanceof CSSStyleRule) {
      processStyleRule(rule)
    } else if (rule instanceof CSSMediaRule) {
      const mediaText = (rule as CSSMediaRule).media.mediaText
      if (/print/i.test(mediaText)) {
        output.push(rule.cssText)
        continue
      }

      const mediaRule = rule as CSSMediaRule
      const inner: string[] = []
      for (let j = 0; j < mediaRule.cssRules.length; j++) {
        const innerRule = mediaRule.cssRules[j]
        if (innerRule instanceof CSSStyleRule) {
          processStyleRule(innerRule)
        } else {
          inner.push(innerRule.cssText)
        }
      }
      if (inner.length > 0) {
        output.push(`@media ${mediaText} {\n${inner.join('\n')}\n}`)
      }
    } else {
      output.push(rule.cssText)
    }
  }

  if (promotedProps.length > 0) {
    // Deduplicate props (same property may appear from multiple container classes)
    const uniqueProps = [...new Set(promotedProps)]
    const bgRule = `.resume-document {\n  ${uniqueProps.join('\n  ')}\n}`
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
  let scopedCSS = scopeSelectors(rawCSS)

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
