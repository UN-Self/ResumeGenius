import type { ReactNode } from 'react'
import { ActionBar } from './ActionBar'
import { AiPanelPlaceholder } from './AiPanelPlaceholder'
import '@/styles/editor.css'

interface WorkbenchLayoutProps {
  projectName: string
  children: ReactNode
  toolbar?: ReactNode
  saveIndicator?: ReactNode
}

export function WorkbenchLayout({ projectName, children, toolbar, saveIndicator }: WorkbenchLayoutProps) {
  return (
    <div className="workbench-layout">
      <ActionBar projectName={projectName} saveIndicator={saveIndicator} />
      {children}
      <AiPanelPlaceholder />
      <div className="format-toolbar">
        {toolbar}
      </div>
    </div>
  )
}