import { useState, useEffect, useCallback } from 'react'
import type { CSSProperties } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { LogOut, Search, Sparkles } from 'lucide-react'
import { intakeApi, ApiError, authApi, type Project } from '@/lib/api-client'
import ProjectCard, { NewResumeCard } from '@/components/intake/ProjectCard'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Alert } from '@/components/ui/alert'
import { AnimatedBrandTitle } from '@/components/ui/animated-brand-title'
import { Modal, ModalBody, ModalFooter, ModalHeader } from '@/components/ui/modal'
import { ThemeSwitcher } from '@/components/ui/theme-switcher'
import DeleteConfirm from '@/components/intake/DeleteConfirm'

export default function ProjectList() {
  const navigate = useNavigate()
  const location = useLocation()
  const [projects, setProjects] = useState<Project[]>([])
  const [title, setTitle] = useState('')
  const [query, setQuery] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<Project | null>(null)
  const [deleteLoading, setDeleteLoading] = useState(false)
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

  useEffect(() => {
    const timeout = window.setTimeout(() => {
      void loadProjects()
    }, 0)

    return () => window.clearTimeout(timeout)
  }, [loadProjects, location.key])

  const handleCreate = async () => {
    const trimmed = title.trim()
    if (!trimmed) return
    try {
      setError('')
      await intakeApi.createProject(trimmed)
      setTitle('')
      setCreateOpen(false)
      await loadProjects()
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '创建失败')
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') handleCreate()
  }

  const handleDeleteProject = async () => {
    if (!deleteTarget) return
    try {
      setDeleteLoading(true)
      setError('')
      await intakeApi.deleteProject(deleteTarget.id)
      setProjects((current) => current.filter((project) => project.id !== deleteTarget.id))
      setDeleteTarget(null)
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '删除失败')
    } finally {
      setDeleteLoading(false)
    }
  }

  const visibleProjects = projects.filter((project) =>
    project.title.toLowerCase().includes(query.trim().toLowerCase())
  )

  return (
    <div className="app-shell min-h-screen">
      <div className="relative z-10 mx-auto max-w-7xl px-5 py-6 sm:px-8 lg:px-10">
        <header className="stagger-in relative z-50 mb-10 flex flex-col gap-5 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <div className="brand-kicker mb-3 flex w-fit items-center gap-2 rounded-full px-3 py-1 text-xs font-medium backdrop-blur-xl">
              <Sparkles size={14} className="text-primary" />
              AI resume workspace
            </div>
            <AnimatedBrandTitle className="text-4xl font-semibold sm:text-5xl" />
            <p className="mt-3 max-w-2xl text-sm leading-6 text-muted-foreground">
              从资料接入、AI 生成到可视化编辑，把每一份简历整理成可直接交付的作品。
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <ThemeSwitcher />
            <Button
              variant="secondary"
              size="sm"
              onClick={async () => {
                try {
                  await authApi.logout()
                } finally {
                  window.location.assign('/app/login')
                }
              }}
            >
              <LogOut size={14} className="mr-1.5" />
              退出
            </Button>
          </div>
        </header>

        {error && (
          <Alert className="stagger-in mb-5">{error}</Alert>
        )}

        <section className="glass-panel stagger-in relative z-10 mb-6 rounded-3xl p-4" style={{ '--delay': '80ms' } as CSSProperties}>
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div>
              <h2 className="text-lg font-semibold text-foreground">所有简历</h2>
              <p className="mt-1 text-xs text-muted-foreground">
                {projects.length} 个项目 · 模板预览式工作区
              </p>
            </div>
            <label className="relative block w-full md:w-80">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="搜索简历"
                className="pl-9"
              />
            </label>
          </div>
        </section>

        {loading ? (
          <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {Array.from({ length: 4 }).map((_, index) => (
              <div key={index} className="glass-panel min-h-[320px] animate-pulse rounded-2xl" />
            ))}
          </div>
        ) : (
          <div className="stagger-in grid gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4" style={{ '--delay': '140ms' } as CSSProperties}>
            <NewResumeCard onClick={() => setCreateOpen(true)} />
            {visibleProjects.map((project) => (
              <ProjectCard
                key={project.id}
                project={project}
                onClick={(id) => navigate(
                  project.current_draft_id ? `/projects/${id}/edit` : `/projects/${id}`
                )}
                onDelete={setDeleteTarget}
              />
            ))}
          </div>
        )}

        {!loading && visibleProjects.length === 0 && projects.length > 0 && (
          <p className="mt-8 text-center text-sm text-muted-foreground">
            没有找到匹配的简历。
          </p>
        )}
      </div>

      <Modal open={createOpen} onClose={() => setCreateOpen(false)}>
        <ModalHeader>新建简历</ModalHeader>
        <ModalBody>
          <p className="mb-4 text-sm text-muted-foreground">
            先给这份简历起一个项目名，之后可以上传文件、接入 Git 或直接补充经历。
          </p>
          <Input
            autoFocus
            value={title}
            onChange={(event) => setTitle(event.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="例如：MEOwj 的前端简历"
          />
        </ModalBody>
        <ModalFooter>
          <Button variant="secondary" onClick={() => setCreateOpen(false)}>
            取消
          </Button>
          <Button onClick={handleCreate} disabled={!title.trim()}>
            创建简历
          </Button>
        </ModalFooter>
      </Modal>

      <DeleteConfirm
        open={deleteTarget !== null}
        title="删除简历"
        message={`确定要删除「${deleteTarget?.title ?? ''}」吗？相关资料和草稿也会一起删除，此操作不可撤销。`}
        loading={deleteLoading}
        onConfirm={handleDeleteProject}
        onCancel={() => setDeleteTarget(null)}
      />
    </div>
  )
}
