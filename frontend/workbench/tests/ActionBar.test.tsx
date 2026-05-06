import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it } from 'vitest'
import { ActionBar } from '@/components/editor/ActionBar'
import { THEME_MANUAL_STORAGE_KEY, THEME_STORAGE_KEY } from '@/lib/theme'

function renderActionBar() {
  return render(
    <MemoryRouter>
      <ActionBar
        projectName="MEOwj 的简历"
        draftId="1"
        exportStatus="idle"
        onExport={() => {}}
      />
    </MemoryRouter>
  )
}

describe('ActionBar', () => {
  beforeEach(() => {
    localStorage.removeItem(THEME_STORAGE_KEY)
    localStorage.removeItem(THEME_MANUAL_STORAGE_KEY)
  })

  it('keeps the home link in the editor header', () => {
    renderActionBar()

    expect(screen.getByRole('link', { name: '返回简历首页' })).toHaveAttribute('href', '/')
  })

  it('keeps the theme switcher available in the editor header', () => {
    renderActionBar()

    expect(screen.getByRole('button', { name: '选择主题' })).toBeInTheDocument()
  })
})
