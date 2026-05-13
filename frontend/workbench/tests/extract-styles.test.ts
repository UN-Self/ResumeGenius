import { describe, expect, it } from 'vitest'
import {
  extractStyles,
  scopeSelectors,
  stripContainerDimensions,
  getRootContainerClasses,
  promoteContainerBackground,
  RESUME_DOCUMENT_CLASS,
  SCOPE_PREFIX,
  processScopedCSS,
  reconstructHtml,
  stripTrailingEditorEmptyParagraphs,
} from '@/lib/extract-styles'

// ─── Test helpers ──────────────────────────────────────────────────

/** Regex matching ".resume-document {" (used by promoteContainerBackground) */
const SCOPE_PREFIX_BLOCK_RE = new RegExp(`\\${SCOPE_PREFIX}\\s*\\{[^}]*`)
/** Regex matching ".resume-document .resume {" (scoped container rule) */
const SCOPE_PREFIX_CONTAINER_RE = new RegExp(`\\${SCOPE_PREFIX}\\s+\\.resume\\s*\\{([^}]*)\\}`)

// ─── Shared test fixtures ─────────────────────────────────────────────

const SAMPLE_HTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <style>
    @page { size: A4; margin: 0; }
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: 'Noto Sans SC', sans-serif; font-size: 10.5pt; line-height: 1.4; color: #333; }
    .resume { width: 210mm; min-height: 297mm; padding: 18mm 20mm; }
    .section { margin-bottom: 10pt; }
    .tag { background: #f0f0f0; padding: 2pt 8pt; border-radius: 3pt; }
  </style>
</head>
<body>
  <div class="resume">
    <section class="section"><h2>标题</h2></section>
    <span class="tag">标签</span>
  </div>
</body>
</html>`

// ─── scopeSelectors ───────────────────────────────────────────────────

describe('scopeSelectors', () => {
  it('prefixes simple class selectors', () => {
    const css = '.section { margin-bottom: 10pt; }'
    const result = scopeSelectors(css)
    expect(result).toContain(`${SCOPE_PREFIX} .section`)
    expect(result).toContain('margin-bottom: 10pt')
  })

  it('rewrites body selector to scope', () => {
    const css = 'body { color: #333; }'
    const result = scopeSelectors(css)
    expect(result).toContain(SCOPE_PREFIX)
    expect(result).not.toContain('body')
    // CSSOM normalizes #333 to rgb(51, 51, 51) — match either form
    expect(result).toMatch(/color:\s*(#333|rgb\(51,\s*51,\s*51\))/)
  })

  it('rewrites html selector to scope', () => {
    const css = 'html { font-size: 16px; }'
    const result = scopeSelectors(css)
    expect(result).toContain(SCOPE_PREFIX)
    expect(result).not.toContain('html')
    expect(result).toContain('font-size: 16px')
  })

  it('rewrites * selector to scope *', () => {
    const css = '* { margin: 0; padding: 0; }'
    const result = scopeSelectors(css)
    expect(result).toContain(`${SCOPE_PREFIX} *`)
    expect(result).toContain('margin: 0')
  })

  it('handles comma-separated selectors', () => {
    const css = 'h1, h2, h3 { font-weight: bold; }'
    const result = scopeSelectors(css)
    expect(result).toContain(`${SCOPE_PREFIX} h1`)
    expect(result).toContain(`${SCOPE_PREFIX} h2`)
    expect(result).toContain(`${SCOPE_PREFIX} h3`)
    expect(result).toContain('font-weight: bold')
  })

  it('handles compound selectors with combinators', () => {
    const css = '.section > .title { font-size: 14pt; }'
    const result = scopeSelectors(css)
    expect(result).toContain(`${SCOPE_PREFIX} .section > .title`)
  })

  it('handles child combinator', () => {
    const css = '.section .title { color: blue; }'
    const result = scopeSelectors(css)
    expect(result).toContain(`${SCOPE_PREFIX} .section .title`)
  })

  it('skips @page rules', () => {
    const css = '@page { size: A4; margin: 0; }'
    const result = scopeSelectors(css)
    expect(result).not.toContain('@page')
    expect(result).not.toContain('size: A4')
    // Should return empty or whitespace-only
    expect(result.trim()).toBe('')
  })

  it('skips @media print blocks', () => {
    const css = '@media print { .no-print { display: none; } }'
    const result = scopeSelectors(css)
    expect(result).not.toContain('@media print')
    expect(result).not.toContain('.no-print')
    expect(result.trim()).toBe('')
  })

  it('preserves non-print @media blocks with scoped internal selectors', () => {
    const css = '@media screen and (max-width: 600px) { .section { font-size: 12pt; } }'
    const result = scopeSelectors(css)
    expect(result).toContain('@media screen and (max-width: 600px)')
    expect(result).toContain(`${SCOPE_PREFIX} .section`)
    expect(result).toContain('font-size: 12pt')
  })

  it('handles sibling combinator +', () => {
    const css = '.section + .section { margin-top: 5pt; }'
    const result = scopeSelectors(css)
    expect(result).toContain(`${SCOPE_PREFIX} .section + .section`)
  })

  it('handles general sibling combinator ~', () => {
    const css = '.item ~ .item { color: red; }'
    const result = scopeSelectors(css)
    expect(result).toContain(`${SCOPE_PREFIX} .item ~ .item`)
  })

  it('handles already-scoped selectors without double-prefixing', () => {
    const css = `${SCOPE_PREFIX} .section { color: blue; }`
    const result = scopeSelectors(css)
    expect(result).not.toContain(`${SCOPE_PREFIX} ${SCOPE_PREFIX}`)
    expect(result).toContain(`${SCOPE_PREFIX} .section`)
  })
})

// ─── stripContainerDimensions ─────────────────────────────────────────

describe('stripContainerDimensions', () => {
  it('strips dimension properties from root container', () => {
    const css = '.resume { width: 210mm; min-height: 297mm; padding: 18mm 20mm; background: white; }'
    const result = stripContainerDimensions(css, '.resume')
    expect(result).not.toContain('width: 210mm')
    expect(result).not.toContain('min-height: 297mm')
    expect(result).not.toContain('padding: 18mm 20mm')
    expect(result).toContain('background: white')
  })

  it('preserves non-dimension properties on root container', () => {
    const css = '.resume { color: #333; font-size: 12pt; background: white; }'
    const result = stripContainerDimensions(css, '.resume')
    expect(result).toContain('font-size: 12pt')
    // CSSOM may normalize colors; check property is present
    expect(result).toMatch(/color:/)
    expect(result).toMatch(/background/)
  })

  it('removes rule entirely if all properties are dimension-related', () => {
    const css = '.resume { width: 210mm; height: 297mm; padding: 18mm 20mm; margin: 0; }'
    const result = stripContainerDimensions(css, '.resume')
    // The entire .resume rule should be gone
    expect(result.trim()).toBe('')
  })

  it('handles margin-top/right/bottom/left individually', () => {
    const css = '.resume { margin-top: 10px; margin-bottom: 10px; color: red; }'
    const result = stripContainerDimensions(css, '.resume')
    expect(result).not.toContain('margin-top')
    expect(result).not.toContain('margin-bottom')
    expect(result).toContain('color: red')
  })

  it('handles padding-top/right/bottom/left individually', () => {
    const css = '.resume { padding-left: 20mm; padding-right: 20mm; color: red; }'
    const result = stripContainerDimensions(css, '.resume')
    expect(result).not.toContain('padding-left')
    expect(result).not.toContain('padding-right')
    expect(result).toContain('color: red')
  })

  it('handles min-width and max-width', () => {
    const css = '.resume { min-width: 100px; max-width: 500px; color: blue; }'
    const result = stripContainerDimensions(css, '.resume')
    expect(result).not.toContain('min-width')
    expect(result).not.toContain('max-width')
    expect(result).toContain('color: blue')
  })

  it('does not affect other selectors', () => {
    const css = '.section { width: 100%; padding: 10px; } .resume { width: 210mm; color: red; }'
    const result = stripContainerDimensions(css, '.resume')
    expect(result).toContain('.section { width: 100%; padding: 10px; }')
    expect(result).not.toContain('.resume { width: 210mm')
    expect(result).toContain('color: red')
  })
})

// ─── getRootContainerClasses ──────────────────────────────────────────

describe('getRootContainerClasses', () => {
  it('returns classes of first child element of body', () => {
    const html = '<body><div class="resume container">content</div></body>'
    const doc = new DOMParser().parseFromString(html, 'text/html')
    const result = getRootContainerClasses(doc)
    expect(result).toEqual(['resume', 'container'])
  })

  it('returns empty array when body has no children', () => {
    const html = '<body></body>'
    const doc = new DOMParser().parseFromString(html, 'text/html')
    const result = getRootContainerClasses(doc)
    expect(result).toEqual([])
  })

  it('returns empty array when first child has no classes', () => {
    const html = '<body><div>content</div></body>'
    const doc = new DOMParser().parseFromString(html, 'text/html')
    const result = getRootContainerClasses(doc)
    expect(result).toEqual([])
  })
})

// ─── promoteContainerBackground ────────────────────────────────────

describe('promoteContainerBackground', () => {
  it('promotes background-color from root container to .resume-document', () => {
    const css = `${SCOPE_PREFIX} .resume { background-color: #f0f0f0; color: #333; }`
    const result = promoteContainerBackground(css, '.resume')
    // Background should be on .resume-document
    expect(SCOPE_PREFIX_BLOCK_RE.test(result)).toBe(true)
    // Root container should still have non-background properties
    expect(result).toContain('color:')
    // Root container should NOT have background-color anymore
    const resumeRuleMatch = result.match(SCOPE_PREFIX_CONTAINER_RE)
    expect(resumeRuleMatch).not.toBeNull()
    expect(resumeRuleMatch![1]).not.toContain('background')
  })

  it('promotes background shorthand (expanded by CSSOM)', () => {
    const css = `${SCOPE_PREFIX} .resume { background: #f0f0f0; color: #333; }`
    const result = promoteContainerBackground(css, '.resume')
    // CSSOM expands shorthand — background-color should be promoted
    expect(SCOPE_PREFIX_BLOCK_RE.test(result)).toBe(true)
    // Should NOT include default-valued longhands like background-image: none
    // that CSSOM expands from the shorthand
    expect(result).not.toContain('background-image: none')
  })

  it('does not affect non-container selectors', () => {
    const css = `${SCOPE_PREFIX} .section { background: #eee; } ${SCOPE_PREFIX} .resume { background: #f0f0f0; }`
    const result = promoteContainerBackground(css, '.resume')
    // .section background should remain untouched
    expect(result).toContain(`${SCOPE_PREFIX} .section`)
    expect(result).toMatch(new RegExp(`\\${SCOPE_PREFIX}\\s+\\.section[^{]*\\{[^}]*background`))
  })

  it('returns unchanged CSS when no background on container', () => {
    const css = `${SCOPE_PREFIX} .resume { color: #333; font-size: 12pt; }`
    const result = promoteContainerBackground(css, '.resume')
    expect(result).not.toContain(`${SCOPE_PREFIX} {`)
    expect(result).toContain('color:')
  })

  it('returns empty for empty input', () => {
    expect(promoteContainerBackground('', '.resume')).toBe('')
  })

  it('preserves non-container rules inside @media screen when promoting container background', () => {
    const css = `@media screen and (max-width: 600px) { ${SCOPE_PREFIX} .resume { background: #f5f5f5; } ${SCOPE_PREFIX} .section { font-size: 12pt; } }`
    const result = promoteContainerBackground(css, '.resume')
    // @media block should still exist (non-container .section rule preserves it)
    expect(result).toContain('@media screen and (max-width: 600px)')
    // .section should remain INSIDE the @media block, not leaked to top level
    expect(result).toMatch(new RegExp(`@media[\\s\\S]*\\${SCOPE_PREFIX}\\s+\\.section[\\s\\S]*font-size:\\s*12pt`))
    // Background should be promoted to .resume-document at top level
    expect(SCOPE_PREFIX_BLOCK_RE.test(result)).toBe(true)
  })

  it('drops empty @media block when all inner rules are fully promoted', () => {
    const css = `@media screen and (max-width: 600px) { ${SCOPE_PREFIX} .resume { background: #f5f5f5; } }`
    const result = promoteContainerBackground(css, '.resume')
    // All container props are background → rule dropped → @media block empty → dropped
    expect(result).not.toContain('@media')
    // But background is still promoted
    expect(SCOPE_PREFIX_BLOCK_RE.test(result)).toBe(true)
  })

  it('preserves !important priority on promoted background', () => {
    const css = `${SCOPE_PREFIX} .resume { background: #fff !important; }`
    const result = promoteContainerBackground(css, '.resume')
    expect(result).toContain('!important')
  })
})

// ─── extractStyles (integration) ──────────────────────────────────────

describe('extractStyles', () => {
  it('extracts style CSS and body HTML from full document', () => {
    const { bodyHtml, scopedCSS } = extractStyles(SAMPLE_HTML)
    expect(bodyHtml).toContain('<div class="resume">')
    expect(bodyHtml).toContain('<section class="section">')
    expect(bodyHtml).toContain('<span class="tag">')
    // bodyHtml should NOT contain <style> tags
    expect(bodyHtml).not.toContain('<style')
  })

  it('scopes CSS selectors with prefix', () => {
    const { scopedCSS } = extractStyles(SAMPLE_HTML)
    // @page should be removed
    expect(scopedCSS).not.toContain('@page')
    // * should be scoped
    expect(scopedCSS).toContain(`${SCOPE_PREFIX} *`)
    // body should be rewritten
    expect(scopedCSS).not.toMatch(/\bbody\b/)
    // .section should be scoped
    expect(scopedCSS).toContain(`${SCOPE_PREFIX} .section`)
  })

  it('returns empty for empty input', () => {
    const { bodyHtml, scopedCSS } = extractStyles('')
    expect(bodyHtml).toBe('')
    expect(scopedCSS).toBe('')
  })

  it('handles plain HTML fragment without style blocks', () => {
    const html = '<div class="resume"><p>Hello</p></div>'
    const { bodyHtml, scopedCSS } = extractStyles(html)
    expect(bodyHtml).toContain('<div class="resume">')
    expect(scopedCSS).toBe('')
  })

  it('full integration: strips container dimensions from sample HTML', () => {
    const { scopedCSS, bodyHtml } = extractStyles(SAMPLE_HTML)
    // .resume width/min-height/padding should be stripped
    expect(scopedCSS).not.toContain('width: 210mm')
    expect(scopedCSS).not.toContain('min-height: 297mm')
    expect(scopedCSS).not.toContain('padding: 18mm 20mm')
    // .tag padding should NOT be stripped (it is not a root container)
    expect(scopedCSS).toContain('padding: 2pt 8pt')
    // body should be in the output
    expect(bodyHtml).toContain('class="resume"')
  })

  it('merges CSS from multiple <style> blocks', () => {
    const html = `<!DOCTYPE html>
<html><head>
  <style>
    .section { margin-bottom: 10pt; }
  </style>
  <style>
    .tag { background: #f0f0f0; }
  </style>
</head>
<body>
  <div class="resume">
    <section class="section"><span class="tag">标签</span></section>
  </div>
</body></html>`
    const { scopedCSS, bodyHtml } = extractStyles(html)
    // Both style blocks should be merged and scoped
    expect(scopedCSS).toContain(`${SCOPE_PREFIX} .section`)
    expect(scopedCSS).toContain('margin-bottom: 10pt')
    expect(scopedCSS).toContain(`${SCOPE_PREFIX} .tag`)
    expect(scopedCSS).toContain('background')
    expect(bodyHtml).toContain('class="resume"')
  })

  it('promotes root container background to .resume-document', () => {
    const html = `<!DOCTYPE html>
<html><head>
  <style>
    .resume { width: 210mm; min-height: 297mm; padding: 18mm 20mm; background: #f5f5f5; }
    .section { margin-bottom: 10pt; }
  </style>
</head>
<body>
  <div class="resume">
    <section class="section"><h2>标题</h2></section>
  </div>
</body></html>`
    const { scopedCSS } = extractStyles(html)
    // Dimensions should be stripped
    expect(scopedCSS).not.toContain('width: 210mm')
    expect(scopedCSS).not.toContain('min-height: 297mm')
    expect(scopedCSS).not.toContain('padding: 18mm 20mm')
    // Background should be promoted to .resume-document (full canvas coverage)
    expect(SCOPE_PREFIX_BLOCK_RE.test(scopedCSS)).toBe(true)
    // .section should remain scoped normally
    expect(scopedCSS).toContain(`${SCOPE_PREFIX} .section`)
  })

  it('returns rawCSS (un-scoped, un-processed) alongside scopedCSS', () => {
    const { rawCSS, scopedCSS } = extractStyles(SAMPLE_HTML)
    // rawCSS should contain the original CSS before scoping
    expect(rawCSS).toContain('.section {')
    expect(rawCSS).toContain('.tag {')
    expect(rawCSS).toContain('.resume {')
    // rawCSS should NOT be scoped (no .resume-document prefix)
    expect(rawCSS).not.toContain('.resume-document')
    // rawCSS should still have @page (not stripped)
    expect(rawCSS).toContain('@page')
    // rawCSS should still have container dimensions (not stripped)
    expect(rawCSS).toContain('width: 210mm')
  })

  it('returns empty rawCSS for HTML without style blocks', () => {
    const html = '<div class="resume"><p>Hello</p></div>'
    const { rawCSS } = extractStyles(html)
    expect(rawCSS).toBe('')
  })

  it('returns empty rawCSS for empty input', () => {
    const { rawCSS } = extractStyles('')
    expect(rawCSS).toBe('')
  })

  it('rawCSS can round-trip through extractStyles', () => {
    const { bodyHtml, rawCSS } = extractStyles(SAMPLE_HTML)
    // Reconstruct a full HTML document from bodyHtml + rawCSS
    const reconstructed = `<!DOCTYPE html><html><head><style>${rawCSS}</style></head><body>${bodyHtml}</body></html>`
    const result = extractStyles(reconstructed)
    // The second extraction should produce the same rawCSS
    expect(result.rawCSS).toContain('.section {')
    expect(result.rawCSS).toContain('.tag {')
    expect(result.bodyHtml).toContain('<div class="resume">')
  })

  it('strips trailing empty editor paragraphs from extracted body HTML', () => {
    const html = '<div class="page"><p>内容</p></div><p></p>'

    const { bodyHtml } = extractStyles(html)

    expect(bodyHtml).toBe('<div class="page"><p>内容</p></div>')
  })
})

describe('stripTrailingEditorEmptyParagraphs', () => {
  it('removes empty paragraphs at the end only', () => {
    const html = '<p></p><div class="page"><p>内容</p></div><p></p><p><br></p>'

    expect(stripTrailingEditorEmptyParagraphs(html)).toBe('<p></p><div class="page"><p>内容</p></div>')
  })

  it('keeps trailing paragraphs with attributes', () => {
    const html = '<div class="page"><p>内容</p></div><p style="break-before: page"></p>'

    expect(stripTrailingEditorEmptyParagraphs(html)).toBe(html)
  })

  it('is applied when reconstructing full HTML', () => {
    const html = reconstructHtml('<div class="page">内容</div><p></p>', '.page { color: red; }')

    expect(html).toContain('<body><div class="page">内容</div></body>')
  })
})

// ─── SCOPE_PREFIX consistency ───────────────────────────────────────

describe('RESUME_DOCUMENT_CLASS and SCOPE_PREFIX', () => {
  it('exports RESUME_DOCUMENT_CLASS as the bare class name (no dot)', () => {
    expect(RESUME_DOCUMENT_CLASS).toBe('resume-document')
  })

  it('derives SCOPE_PREFIX from RESUME_DOCUMENT_CLASS with dot prefix', () => {
    expect(SCOPE_PREFIX).toBe(`.${RESUME_DOCUMENT_CLASS}`)
  })

  it('promoteContainerBackground uses SCOPE_PREFIX for promoted background rule', () => {
    const css = `${SCOPE_PREFIX} .resume { background: #f0f0f0; }`
    const result = promoteContainerBackground(css, '.resume')
    expect(result).toContain(`${SCOPE_PREFIX} {`)
  })
})

// ─── processScopedCSS ──────────────────────────────────────────────

describe('processScopedCSS', () => {
  it('scopes, strips container dimensions, and promotes background in one call', () => {
    const rawCSS = `
      body { font-family: sans-serif; }
      .resume { width: 210mm; padding: 18mm; background: #f5f5f5; }
      .section { margin-bottom: 10pt; }
    `
    const containerClasses = ['resume']
    const result = processScopedCSS(rawCSS, containerClasses)

    // body should be scoped away
    expect(result).not.toContain('body')
    // Container dimensions should be stripped
    expect(result).not.toContain('width: 210mm')
    expect(result).not.toContain('padding: 18mm')
    // Background should be promoted to .resume-document
    expect(SCOPE_PREFIX_BLOCK_RE.test(result)).toBe(true)
    // .section should be scoped normally
    expect(result).toContain(`${SCOPE_PREFIX} .section`)
  })

  it('returns empty string for empty CSS input', () => {
    expect(processScopedCSS('', [])).toBe('')
  })

  it('returns empty string when no container classes provided', () => {
    const css = '.section { color: red; }'
    const result = processScopedCSS(css, [])
    // Should still scope selectors but skip dimension/background steps
    expect(result).toContain(`${SCOPE_PREFIX} .section`)
    expect(result).toContain('color: red')
  })
})
