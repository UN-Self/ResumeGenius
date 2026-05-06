import { http } from 'msw'
import { sampleVersions, sampleDraftHtml } from '../fixtures'

export const versionHandlers = [
  http.get('/api/v1/drafts/:draftId/versions', async () => {
    await new Promise((resolve) => setTimeout(resolve, 100))
    return Response.json({
      code: 0,
      data: { items: sampleVersions, total: sampleVersions.length },
      message: 'ok',
    })
  }),

  http.get('/api/v1/drafts/:draftId/versions/:versionId', async ({ params }) => {
    const versionId = Number(params.versionId)
    const found = sampleVersions.find((v) => v.id === versionId)
    if (!found) {
      return Response.json({ code: 5004, data: null, message: 'version not found' }, { status: 404 })
    }
    return Response.json({
      code: 0,
      data: { ...found, html_snapshot: sampleDraftHtml.trim() },
      message: 'ok',
    })
  }),

  http.post('/api/v1/drafts/:draftId/versions', async ({ request }) => {
    const body = (await request.json()) as { label?: string }
    return Response.json({
      code: 0,
      data: {
        id: Date.now(),
        label: body.label || '手动保存',
        created_at: new Date().toISOString(),
      },
      message: 'ok',
    })
  }),

  http.post('/api/v1/drafts/:draftId/rollback', async ({ params }) => {
    const draftId = Number(params.draftId)
    return Response.json({
      code: 0,
      data: {
        draft_id: draftId,
        updated_at: new Date().toISOString(),
        new_version_id: Date.now(),
        new_version_label: '回退到版本 2026-05-06 10:00:00',
        new_version_created_at: new Date().toISOString(),
      },
      message: 'ok',
    })
  }),
]
