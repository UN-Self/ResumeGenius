import { http } from 'msw'
import { sampleProject } from '../fixtures'

export const projectHandlers = [
  http.get('/api/v1/projects/:projectId', async ({ params }) => {
    await new Promise((resolve) => setTimeout(resolve, 200))
    return Response.json({
      code: 0,
      data: {
        ...sampleProject,
        id: Number(params.projectId),
      },
      message: 'ok',
    })
  }),
]
