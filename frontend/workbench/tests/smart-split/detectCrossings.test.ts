import { describe, it, expect, vi } from 'vitest'
import {
  getBreakerPositions,
  elementCrossesBreaker,
  findCrossingPositions,
  findPageStartPositions,
} from '@/components/editor/extensions/smart-split/detectCrossings'
import type { BreakerPosition } from '@/components/editor/extensions/smart-split/types'

describe('getBreakerPositions', () => {
  it('returns empty array when no breakers in DOM', () => {
    const div = document.createElement('div')
    div.innerHTML = '<p>no breakers here</p>'
    document.body.appendChild(div)
    expect(getBreakerPositions(div)).toEqual([])
    document.body.removeChild(div)
  })

  it('finds breaker elements and returns their bounding rects', () => {
    const wrapper = document.createElement('div')
    wrapper.innerHTML = `
      <div class="breaker" style="height:100px;position:absolute;top:500px">Br1</div>
      <div class="breaker" style="height:100px;position:absolute;top:1700px">Br2</div>
    `
    // Override getBoundingClientRect because jsdom does not compute CSS layout
    const breakerEls = wrapper.querySelectorAll('.breaker')
    const rects: DOMRect[] = [
      { top: 500, bottom: 600, left: 0, right: 100, width: 100, height: 100, x: 0, y: 500, toJSON: () => '' },
      { top: 1700, bottom: 1800, left: 0, right: 100, width: 100, height: 100, x: 0, y: 1700, toJSON: () => '' },
    ] as DOMRect[]
    breakerEls.forEach((el, i) => {
      el.getBoundingClientRect = () => rects[i]
    })

    document.body.appendChild(wrapper)
    const positions = getBreakerPositions(wrapper)
    expect(positions).toHaveLength(2)
    expect(positions[0].top).toBe(500)
    expect(positions[0].bottom).toBe(600)
    expect(positions[1].top).toBe(1700)
    expect(positions[1].bottom).toBe(1800)
    document.body.removeChild(wrapper)
  })
})

describe('elementCrossesBreaker', () => {
  it('returns false when element is fully above breaker', () => {
    expect(elementCrossesBreaker(
      { top: 100, bottom: 400 },
      { top: 500, bottom: 632 },
      4,
    )).toBe(false)
  })

  it('returns false when element is fully below breaker', () => {
    expect(elementCrossesBreaker(
      { top: 700, bottom: 900 },
      { top: 500, bottom: 632 },
      4,
    )).toBe(false)
  })

  it('returns true when element straddles breaker', () => {
    expect(elementCrossesBreaker(
      { top: 400, bottom: 600 },
      { top: 500, bottom: 632 },
      4,
    )).toBe(true)
  })

  it('element ending well above breaker top minus threshold is not crossing', () => {
    expect(elementCrossesBreaker(
      { top: 100, bottom: 495 },
      { top: 500, bottom: 632 },
      4,
    )).toBe(false)
  })

  it('element ending within threshold above breaker top is crossing', () => {
    expect(elementCrossesBreaker(
      { top: 100, bottom: 498 },
      { top: 500, bottom: 632 },
      4,
    )).toBe(true)
  })

  it('with jitter, element starting near breaker bottom is not crossing', () => {
    expect(elementCrossesBreaker(
      { top: 628, bottom: 900 },
      { top: 500, bottom: 632 },
      0,
      20,
    )).toBe(false)
  })

  it('with jitter, element starting well above breaker bottom is still crossing', () => {
    expect(elementCrossesBreaker(
      { top: 400, bottom: 700 },
      { top: 500, bottom: 632 },
      0,
      20,
    )).toBe(true)
  })
})

describe('findCrossingPositions', () => {
  it('returns empty when no breakers', () => {
    const mockView = { posAtDOM: vi.fn() }
    const wrapper = document.createElement('div')
    const results = findCrossingPositions(mockView as any, wrapper, [], 4)
    expect(results).toEqual([])
  })

  it('detects crossing block element and returns its pos', () => {
    const wrapper = document.createElement('div')
    const block = document.createElement('div')
    block.textContent = 'Hello'
    block.getBoundingClientRect = () => ({ top: 0, bottom: 500, height: 500 } as DOMRect)
    wrapper.appendChild(block)
    document.body.appendChild(wrapper)

    const breakers: BreakerPosition[] = [{ top: 250, bottom: 300 }]
    const mockView = { posAtDOM: vi.fn().mockReturnValue(42) }

    const results = findCrossingPositions(mockView as any, wrapper, breakers, 4)
    expect(results).toHaveLength(1)
    expect(results[0].pos).toBe(42)
    expect(results[0].breakerIndex).toBe(0)

    document.body.removeChild(wrapper)
  })

  it('skips empty elements', () => {
    const wrapper = document.createElement('div')
    const empty = document.createElement('div')
    empty.getBoundingClientRect = () => ({ top: 0, bottom: 0, height: 0 } as DOMRect)
    wrapper.appendChild(empty)
    document.body.appendChild(wrapper)

    const breakers: BreakerPosition[] = [{ top: 250, bottom: 300 }]
    const mockView = { posAtDOM: vi.fn() }

    const results = findCrossingPositions(mockView as any, wrapper, breakers, 4)
    expect(results).toHaveLength(0)

    document.body.removeChild(wrapper)
  })

  it('skips inline elements (span)', () => {
    const wrapper = document.createElement('div')
    const span = document.createElement('span')
    span.textContent = 'inline text'
    span.getBoundingClientRect = () => ({ top: 0, bottom: 500, height: 500 } as DOMRect)
    wrapper.appendChild(span)
    document.body.appendChild(wrapper)

    const breakers: BreakerPosition[] = [{ top: 250, bottom: 300 }]
    const mockView = { posAtDOM: vi.fn() }

    const results = findCrossingPositions(mockView as any, wrapper, breakers, 4)
    // SPAN is not in BLOCK_TAGS, should be skipped
    expect(results).toHaveLength(0)

    document.body.removeChild(wrapper)
  })
})

describe('findPageStartPositions', () => {
  it('returns empty when no breakers', () => {
    const mockView = { posAtDOM: vi.fn() }
    const wrapper = document.createElement('div')
    const results = findPageStartPositions(mockView as any, wrapper, [])
    expect(results).toEqual([])
  })

  it('finds first block element after each breaker', () => {
    const wrapper = document.createElement('div')

    const block1 = document.createElement('div')
    block1.textContent = 'Page 1 content'
    block1.getBoundingClientRect = () => ({ top: 0, bottom: 400, height: 400 } as DOMRect)

    const block2 = document.createElement('div')
    block2.textContent = 'Page 2 content'
    block2.getBoundingClientRect = () => ({ top: 650, bottom: 1000, height: 350 } as DOMRect)

    wrapper.appendChild(block1)
    wrapper.appendChild(block2)
    document.body.appendChild(wrapper)

    const breakers: BreakerPosition[] = [{ top: 500, bottom: 600 }]
    const mockView = { posAtDOM: vi.fn().mockReturnValue(10) }

    const results = findPageStartPositions(mockView as any, wrapper, breakers)
    expect(results).toHaveLength(1)
    expect(results[0]).toBe(10)
    expect(mockView.posAtDOM).toHaveBeenCalledWith(block2, 0)

    document.body.removeChild(wrapper)
  })

  it('skips empty/zero-height elements', () => {
    const wrapper = document.createElement('div')

    const emptyBlock = document.createElement('div')
    emptyBlock.getBoundingClientRect = () => ({ top: 0, bottom: 0, height: 0 } as DOMRect)

    const realBlock = document.createElement('div')
    realBlock.textContent = 'Content'
    realBlock.getBoundingClientRect = () => ({ top: 650, bottom: 900, height: 250 } as DOMRect)

    wrapper.appendChild(emptyBlock)
    wrapper.appendChild(realBlock)
    document.body.appendChild(wrapper)

    const breakers: BreakerPosition[] = [{ top: 500, bottom: 600 }]
    const mockView = { posAtDOM: vi.fn().mockReturnValue(5) }

    const results = findPageStartPositions(mockView as any, wrapper, breakers)
    expect(results).toHaveLength(1)
    expect(results[0]).toBe(5)

    document.body.removeChild(wrapper)
  })

  it('handles multiple breakers in single pass', () => {
    const wrapper = document.createElement('div')

    const block1 = document.createElement('div')
    block1.textContent = 'Page 1'
    block1.getBoundingClientRect = () => ({ top: 0, bottom: 400, height: 400 } as DOMRect)

    const block2 = document.createElement('div')
    block2.textContent = 'Page 2'
    block2.getBoundingClientRect = () => ({ top: 650, bottom: 1000, height: 350 } as DOMRect)

    const block3 = document.createElement('div')
    block3.textContent = 'Page 3'
    block3.getBoundingClientRect = () => ({ top: 1250, bottom: 1600, height: 350 } as DOMRect)

    wrapper.appendChild(block1)
    wrapper.appendChild(block2)
    wrapper.appendChild(block3)
    document.body.appendChild(wrapper)

    const breakers: BreakerPosition[] = [
      { top: 500, bottom: 600 },
      { top: 1100, bottom: 1200 },
    ]
    const mockView = { posAtDOM: vi.fn().mockReturnValueOnce(10).mockReturnValueOnce(20) }

    const results = findPageStartPositions(mockView as any, wrapper, breakers)
    expect(results).toHaveLength(2)
    expect(results[0]).toBe(10)
    expect(results[1]).toBe(20)

    document.body.removeChild(wrapper)
  })

  it('gracefully handles posAtDOM failure for decoration elements', () => {
    const wrapper = document.createElement('div')

    const decoBlock = document.createElement('div')
    decoBlock.getBoundingClientRect = () => ({ top: 650, bottom: 900, height: 250 } as DOMRect)

    wrapper.appendChild(decoBlock)
    document.body.appendChild(wrapper)

    const breakers: BreakerPosition[] = [{ top: 500, bottom: 600 }]
    const mockView = {
      posAtDOM: vi.fn().mockImplementation(() => { throw new Error('not in doc') }),
    }

    const results = findPageStartPositions(mockView as any, wrapper, breakers)
    expect(results).toEqual([])

    document.body.removeChild(wrapper)
  })

  it('detects UL element as page start (not in BLOCK_TAGS but in PAGE_START_TAGS)', () => {
    const wrapper = document.createElement('div')

    const block1 = document.createElement('div')
    block1.textContent = 'Page 1 content'
    block1.getBoundingClientRect = () => ({ top: 0, bottom: 400, height: 400 } as DOMRect)

    const ul = document.createElement('ul')
    const li = document.createElement('li')
    li.textContent = 'List item on page 2'
    ul.appendChild(li)
    ul.getBoundingClientRect = () => ({ top: 650, bottom: 900, height: 250 } as DOMRect)

    wrapper.appendChild(block1)
    wrapper.appendChild(ul)
    document.body.appendChild(wrapper)

    const breakers: BreakerPosition[] = [{ top: 500, bottom: 600 }]
    const mockView = { posAtDOM: vi.fn().mockReturnValue(15) }

    const results = findPageStartPositions(mockView as any, wrapper, breakers)
    expect(results).toHaveLength(1)
    expect(results[0]).toBe(15)
    expect(mockView.posAtDOM).toHaveBeenCalledWith(ul, 0)

    document.body.removeChild(wrapper)
  })

  it('detects OL element as page start', () => {
    const wrapper = document.createElement('div')

    const block1 = document.createElement('div')
    block1.textContent = 'Page 1'
    block1.getBoundingClientRect = () => ({ top: 0, bottom: 400, height: 400 } as DOMRect)

    const ol = document.createElement('ol')
    const li = document.createElement('li')
    li.textContent = 'Ordered item'
    ol.appendChild(li)
    ol.getBoundingClientRect = () => ({ top: 650, bottom: 900, height: 250 } as DOMRect)

    wrapper.appendChild(block1)
    wrapper.appendChild(ol)
    document.body.appendChild(wrapper)

    const breakers: BreakerPosition[] = [{ top: 500, bottom: 600 }]
    const mockView = { posAtDOM: vi.fn().mockReturnValue(20) }

    const results = findPageStartPositions(mockView as any, wrapper, breakers)
    expect(results).toHaveLength(1)
    expect(mockView.posAtDOM).toHaveBeenCalledWith(ol, 0)

    document.body.removeChild(wrapper)
  })

  it('detects LI element as page start when LI is direct child of wrapper', () => {
    const wrapper = document.createElement('div')

    const block1 = document.createElement('div')
    block1.textContent = 'Page 1'
    block1.getBoundingClientRect = () => ({ top: 0, bottom: 400, height: 400 } as DOMRect)

    const li = document.createElement('li')
    li.textContent = 'Standalone list item on page 2'
    li.getBoundingClientRect = () => ({ top: 650, bottom: 900, height: 250 } as DOMRect)

    wrapper.appendChild(block1)
    wrapper.appendChild(li)
    document.body.appendChild(wrapper)

    const breakers: BreakerPosition[] = [{ top: 500, bottom: 600 }]
    const mockView = { posAtDOM: vi.fn().mockReturnValue(25) }

    const results = findPageStartPositions(mockView as any, wrapper, breakers)
    expect(results).toHaveLength(1)
    expect(mockView.posAtDOM).toHaveBeenCalledWith(li, 0)

    document.body.removeChild(wrapper)
  })

  it('returns first matching element in document order after breaker', () => {
    const wrapper = document.createElement('div')

    const block1 = document.createElement('div')
    block1.textContent = 'Page 1'
    block1.getBoundingClientRect = () => ({ top: 0, bottom: 400, height: 400 } as DOMRect)

    const ul = document.createElement('ul')
    const li = document.createElement('li')
    li.textContent = 'List at page start'
    ul.appendChild(li)
    ul.getBoundingClientRect = () => ({ top: 650, bottom: 800, height: 150 } as DOMRect)

    const p = document.createElement('p')
    p.textContent = 'Paragraph after list'
    p.getBoundingClientRect = () => ({ top: 820, bottom: 950, height: 130 } as DOMRect)

    wrapper.appendChild(block1)
    wrapper.appendChild(ul)
    wrapper.appendChild(p)
    document.body.appendChild(wrapper)

    const breakers: BreakerPosition[] = [{ top: 500, bottom: 600 }]
    const mockView = { posAtDOM: vi.fn().mockReturnValue(30) }

    const results = findPageStartPositions(mockView as any, wrapper, breakers)
    expect(results).toHaveLength(1)
    expect(mockView.posAtDOM).toHaveBeenCalledWith(ul, 0)

    document.body.removeChild(wrapper)
  })

  it('detects crossing element that straddles the breaker boundary', () => {
    const wrapper = document.createElement('div')

    const block1 = document.createElement('div')
    block1.textContent = 'Page 1'
    block1.getBoundingClientRect = () => ({ top: 0, bottom: 400, height: 400 } as DOMRect)

    // Starts before breaker but extends past it
    const crossing = document.createElement('div')
    crossing.textContent = 'I cross the page boundary'
    crossing.getBoundingClientRect = () => ({ top: 350, bottom: 700, height: 350 } as DOMRect)

    const block3 = document.createElement('div')
    block3.textContent = 'Page 2 after crossing'
    block3.getBoundingClientRect = () => ({ top: 750, bottom: 900, height: 150 } as DOMRect)

    wrapper.appendChild(block1)
    wrapper.appendChild(crossing)
    wrapper.appendChild(block3)
    document.body.appendChild(wrapper)

    const breakers: BreakerPosition[] = [{ top: 500, bottom: 600 }]
    const mockView = { posAtDOM: vi.fn().mockReturnValue(15) }

    const results = findPageStartPositions(mockView as any, wrapper, breakers)
    expect(results).toHaveLength(1)
    expect(mockView.posAtDOM).toHaveBeenCalledWith(crossing, 0)

    document.body.removeChild(wrapper)
  })
})
