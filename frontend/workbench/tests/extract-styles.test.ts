import { describe, expect, it } from 'vitest'
import {
  extractStyles,
  scopeSelectors,
  stripContainerDimensions,
  getRootContainerClasses,
} from '@/lib/extract-styles'

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
    expect(result).toContain('.resume-document .section')
    expect(result).toContain('margin-bottom: 10pt')
  })

  it('rewrites body selector to scope', () => {
    const css = 'body { color: #333; }'
    const result = scopeSelectors(css)
    expect(result).toContain('.resume-document')
    expect(result).not.toContain('body')
    // CSSOM normalizes #333 to rgb(51, 51, 51) — match either form
    expect(result).toMatch(/color:\s*(#333|rgb\(51,\s*51,\s*51\))/)
  })

  it('rewrites html selector to scope', () => {
    const css = 'html { font-size: 16px; }'
    const result = scopeSelectors(css)
    expect(result).toContain('.resume-document')
    expect(result).not.toContain('html')
    expect(result).toContain('font-size: 16px')
  })

  it('rewrites * selector to scope *', () => {
    const css = '* { margin: 0; padding: 0; }'
    const result = scopeSelectors(css)
    expect(result).toContain('.resume-document *')
    expect(result).toContain('margin: 0')
  })

  it('handles comma-separated selectors', () => {
    const css = 'h1, h2, h3 { font-weight: bold; }'
    const result = scopeSelectors(css)
    expect(result).toContain('.resume-document h1')
    expect(result).toContain('.resume-document h2')
    expect(result).toContain('.resume-document h3')
    expect(result).toContain('font-weight: bold')
  })

  it('handles compound selectors with combinators', () => {
    const css = '.section > .title { font-size: 14pt; }'
    const result = scopeSelectors(css)
    expect(result).toContain('.resume-document .section > .title')
  })

  it('handles child combinator', () => {
    const css = '.section .title { color: blue; }'
    const result = scopeSelectors(css)
    expect(result).toContain('.resume-document .section .title')
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
    expect(result).toContain('.resume-document .section')
    expect(result).toContain('font-size: 12pt')
  })

  it('handles sibling combinator +', () => {
    const css = '.section + .section { margin-top: 5pt; }'
    const result = scopeSelectors(css)
    expect(result).toContain('.resume-document .section + .section')
  })

  it('handles general sibling combinator ~', () => {
    const css = '.item ~ .item { color: red; }'
    const result = scopeSelectors(css)
    expect(result).toContain('.resume-document .item ~ .item')
  })

  it('handles already-scoped selectors without double-prefixing', () => {
    const css = '.resume-document .section { color: blue; }'
    const result = scopeSelectors(css)
    expect(result).not.toContain('.resume-document .resume-document')
    expect(result).toContain('.resume-document .section')
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
    expect(scopedCSS).toContain('.resume-document *')
    // body should be rewritten
    expect(scopedCSS).not.toMatch(/\bbody\b/)
    // .section should be scoped
    expect(scopedCSS).toContain('.resume-document .section')
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
    expect(scopedCSS).toContain('.resume-document .section')
    expect(scopedCSS).toContain('margin-bottom: 10pt')
    expect(scopedCSS).toContain('.resume-document .tag')
    expect(scopedCSS).toContain('background')
    expect(bodyHtml).toContain('class="resume"')
  })
})
