export const sampleDraftHtml = `
<h1>Sample Draft</h1>
<p>This is a sample resume draft for testing.</p>
`

export const sampleDraft = {
  id: 1,
  project_id: 1,
  html_content: sampleDraftHtml.trim(),
  updated_at: '2026-04-28T12:00:00Z',
}

export const updatedDraftHtml = `
<h1>Sample Draft</h1>
<p>This is an updated resume draft for testing.</p>
`

export const updatedDraft = {
  id: 1,
  project_id: 1,
  html_content: updatedDraftHtml.trim(),
  updated_at: '2026-04-28T12:05:00Z',
}

export const sampleProject = {
  id: 1,
  title: 'Test Project',
  status: 'active',
  current_draft_id: 1,
  created_at: '2026-04-28T12:00:00Z',
}
