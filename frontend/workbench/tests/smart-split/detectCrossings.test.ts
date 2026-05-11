import { describe, it, expect, vi } from 'vitest'
import {
  getBreakerPositions,
  elementCrossesBreaker,
  findCrossingPositions,
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
