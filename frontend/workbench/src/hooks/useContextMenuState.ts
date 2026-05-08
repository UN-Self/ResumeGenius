import { useState, useCallback, useEffect } from 'react'

interface ContextMenuState {
  isOpen: boolean
  x: number
  y: number
}

export function useContextMenuState() {
  const [contextMenu, setContextMenu] = useState<ContextMenuState>({
    isOpen: false, x: 0, y: 0,
  })

  const closeContextMenu = useCallback(() => {
    setContextMenu((prev) => ({ ...prev, isOpen: false }))
  }, [])

  const handleContextMenu = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    setContextMenu({ isOpen: true, x: e.clientX, y: e.clientY })
  }, [])

  // Close on scroll
  useEffect(() => {
    const close = () => closeContextMenu()
    document.addEventListener('scroll', close, true)
    return () => document.removeEventListener('scroll', close, true)
  }, [closeContextMenu])

  // Close on outside click or Escape
  useEffect(() => {
    if (!contextMenu.isOpen) return

    const handleClick = (e: MouseEvent) => {
      const target = e.target as HTMLElement
      if (!target.closest('[role="menu"]')) {
        closeContextMenu()
      }
    }

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') closeContextMenu()
    }

    document.addEventListener('mousedown', handleClick)
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('mousedown', handleClick)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [contextMenu.isOpen, closeContextMenu])

  return { contextMenu, closeContextMenu, handleContextMenu }
}
