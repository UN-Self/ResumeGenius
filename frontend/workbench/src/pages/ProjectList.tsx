import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { intakeApi, ApiError, type Project } from '@/lib/api-client'
import ProjectCard from '@/components/intake/ProjectCard'

export default function ProjectList() {
  const navigate = useNavigate()
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

  useEffect(() => { loadProjects() }, [loadProjects])

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
          <h1 className="font-serif text-2xl font-semibold text-foreground">
            ResumeGenius
          </h1>
          <p className="text-sm text-muted-foreground mt-1">
            AI 辅助简历编辑，从项目开始
          </p>
        </div>

        <div className="flex gap-2 mb-6">
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="输入项目名称，按 Enter 创建"
            className="flex-1 h-10 px-4 text-sm rounded-lg border border-border bg-card text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent transition-shadow"
          />
          <button
            onClick={handleCreate}
            disabled={!title.trim()}
            className="h-10 px-5 text-sm font-medium rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors"
          >
            创建
          </button>
        </div>

        {error && (
          <div className="mb-4 px-4 py-2.5 text-sm rounded-lg bg-destructive/10 text-destructive border border-destructive/20">
            {error}
          </div>
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
                onClick={(id) => navigate(`/projects/${id}`)}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
