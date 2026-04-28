export interface Draft {
  id: number
  project_id: number
  html_content: string
  updated_at: string
}

export type EditorState = 'loading' | 'empty' | 'ready' | 'error'
