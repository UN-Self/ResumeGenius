import { useState } from 'react'

export function usePanelState() {
  const [leftOpen, setLeftOpen] = useState(true)
  const [rightOpen, setRightOpen] = useState(true)

  return { leftOpen, rightOpen, setLeftOpen, setRightOpen }
}
