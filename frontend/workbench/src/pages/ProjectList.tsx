import { useState, useEffect, useCallback } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { intakeApi, ApiError, authApi, type Project } from '@/lib/api-client'
import ProjectCard from '@/components/intake/ProjectCard'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Alert } from '@/components/ui/alert'

export default function ProjectList() {
  const navigate = useNavigate()
  const location = useLocation()
  const [projects, setProjects] = useState<Project[]>([])
  const [title, setTitle] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const loadProjects = useCallback(async () => {
    try {
      setLoading(true)
      const data = await intakeApi.listProjects()
      setProjects(data)
      setError('')
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { loadProjects() }, [loadProjects, location.key])

  const handleCreate = async () => {
    const trimmed = title.trim()
    if (!trimmed) return
    try {
      setError('')
      await intakeApi.createProject(trimmed)
      setTitle('')
      await loadProjects()
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '创建失败')
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') handleCreate()
  }

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-2xl mx-auto px-6 py-10">
        <div className="mb-8">
          <div className="flex items-start justify-between gap-3">
            <h1 className="font-serif text-2xl font-semibold text-foreground">
              ResumeGenius
            </h1>
            <button
              onClick={async () => {
                try {
                  await authApi.logout()
                } finally {
                  window.location.assign('/login')
                }
              }}
              className="text-xs text-muted-foreground hover:text-foreground transition-colors"
            >
              退出登录
            </button>
          </div>
          <p className="text-sm text-muted-foreground mt-1">
            AI 辅助简历编辑，从项目开始
          </p>
        </div>

        <div className="flex gap-2 mb-6">
          <Input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="输入项目名称，按 Enter 创建"
            className="flex-1"
          />
          <Button
            size="lg"
            onClick={handleCreate}
            disabled={!title.trim()}
          >
            创建
          </Button>
        </div>

        {error && (
          <Alert className="mb-4">{error}</Alert>
        )}

        {loading ? (
          <div className="text-center py-12 text-muted-foreground text-sm">
            加载中...
          </div>
        ) : projects.length === 0 ? (
          <div className="text-center py-16">
            <p className="text-muted-foreground text-sm">还没有项目</p>
            <p className="text-muted-foreground text-xs mt-1">
              在上方输入框创建你的第一个简历项目
            </p>
          </div>
        ) : (
          <div className="flex flex-col gap-2">
            {projects.map((project) => (
              <ProjectCard
                key={project.id}
                project={project}
                onClick={(id) => navigate(
                  project.current_draft_id ? `/projects/${id}/edit` : `/projects/${id}`
                )}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
