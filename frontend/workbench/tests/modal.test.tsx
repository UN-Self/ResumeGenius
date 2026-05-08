import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { Modal } from '@/components/ui/modal'

describe('Modal', () => {
  const onClose = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    document.body.style.overflow = ''
  })

  it('calls onClose when Escape key is pressed', () => {
    render(
      <Modal open={true} onClose={onClose}>
        <p>Modal content</p>
      </Modal>,
    )

    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('does not call onClose when other keys are pressed', () => {
    render(
      <Modal open={true} onClose={onClose}>
        <p>Modal content</p>
      </Modal>,
    )

    fireEvent.keyDown(document, { key: 'Enter' })
    expect(onClose).not.toHaveBeenCalled()
  })

  it('does not call onClose on Escape when modal is closed', () => {
    render(
      <Modal open={false} onClose={onClose}>
        <p>Modal content</p>
      </Modal>,
    )

    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).not.toHaveBeenCalled()
  })

  it('sets body overflow to hidden when modal opens', () => {
    const { rerender } = render(
      <Modal open={false} onClose={onClose}>
        <p>Modal content</p>
      </Modal>,
    )

    expect(document.body.style.overflow).not.toBe('hidden')

    rerender(
      <Modal open={true} onClose={onClose}>
        <p>Modal content</p>
      </Modal>,
    )

    expect(document.body.style.overflow).toBe('hidden')
  })

  it('restores body overflow when modal closes', () => {
    const { rerender } = render(
      <Modal open={true} onClose={onClose}>
        <p>Modal content</p>
      </Modal>,
    )

    expect(document.body.style.overflow).toBe('hidden')

    rerender(
      <Modal open={false} onClose={onClose}>
        <p>Modal content</p>
      </Modal>,
    )

    expect(document.body.style.overflow).not.toBe('hidden')
  })

  it('calls onClose when clicking the backdrop', () => {
    render(
      <Modal open={true} onClose={onClose}>
        <p>Modal content</p>
      </Modal>,
    )

    // The backdrop is the fixed inset-0 div; find it by its role or position
    // Since there's no role, click the first fixed inset-0 element (backdrop comes first)
    const portal = document.querySelector('.fixed.inset-0')
    expect(portal).toBeTruthy()
    fireEvent.click(portal!.firstElementChild as HTMLElement) // backdrop is first child
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
