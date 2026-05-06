import { ArrowUpRight, CheckCircle2, Clock3, FileText, MoreHorizontal, Plus } from 'lucide-react'
import type { CSSProperties } from 'react'

interface ProjectCardProps {
  project: {
    id: number
    title: string
    status: string
    created_at: string
    current_draft_id?: number | null
    asset_count?: number
  }
  onClick: (id: number) => void
}

interface NewResumeCardProps {
  onClick: () => void
}

const TEMPLATES = ['classic-blue', 'compact-black', 'modern-sidebar', 'warm-editorial', 'minimal-apple'] as const

function formatDate(value: string) {
  return new Date(value).toLocaleDateString('zh-CN', {
    month: 'short',
    day: 'numeric',
  })
}

function getTemplate(projectId: number) {
  return TEMPLATES[projectId % TEMPLATES.length]
}

function ResumePreview({ template }: { template: string }) {
  const black = template === 'compact-black'
  const warm = template === 'warm-editorial'
  const minimal = template === 'minimal-apple'
  const accent = 'var(--color-primary)'
  const strongAccent = black ? 'var(--color-foreground)' : 'var(--color-primary)'
  const softLine = 'color-mix(in srgb, var(--color-primary), var(--color-muted-foreground) 62%)'
  const paleLine = 'color-mix(in srgb, var(--color-muted-foreground), transparent 62%)'

  return (
    <div
      className="relative h-full overflow-hidden rounded-xl bg-resume-paper p-3 text-[6px] shadow-[inset_0_0_0_1px_color-mix(in_srgb,var(--color-border),transparent_28%)]"
      style={{ color: 'color-mix(in srgb, var(--color-foreground), #111827 28%)' }}
    >
      <div className="absolute inset-0 bg-[linear-gradient(90deg,color-mix(in_srgb,var(--color-primary),transparent_94%)_1px,transparent_1px),linear-gradient(color-mix(in_srgb,var(--color-primary),transparent_94%)_1px,transparent_1px)] bg-[length:16px_16px] opacity-70" />
      <div className="relative z-10 flex h-full gap-2">
        {template === 'modern-sidebar' && (
          <div
            className="w-[30%] rounded-md p-2"
            style={{
              background: 'linear-gradient(180deg, var(--color-primary), color-mix(in srgb, var(--color-accent), var(--color-primary) 36%))',
              color: 'var(--color-primary-foreground)',
            }}
          >
            <div className="mb-2 h-8 w-8 rounded-full bg-white/80" />
            <div className="mb-3 h-1.5 w-10 rounded bg-white/85" />
            {Array.from({ length: 7 }).map((_, index) => (
              <div key={index} className="mb-1 h-1 rounded bg-white/45" style={{ width: `${70 + (index % 3) * 10}%` }} />
            ))}
          </div>
        )}

        <div className="min-w-0 flex-1">
          <div className="mb-2 flex items-start justify-between gap-2 border-b pb-2" style={{ borderColor: 'color-mix(in srgb, var(--color-primary), transparent 64%)' }}>
            <div className="min-w-0">
              <div className="mb-1 h-2.5 w-16 rounded" style={{ background: accent }} />
              <div className="h-1.5 w-28 rounded" style={{ background: paleLine }} />
            </div>
            {!minimal && <div className="h-9 w-7 rounded" style={{ background: 'color-mix(in srgb, var(--color-muted), var(--color-resume-paper) 38%)' }} />}
          </div>

          {Array.from({ length: minimal ? 5 : 6 }).map((_, section) => (
            <div key={section} className="mb-2">
              <div className="mb-1.5 flex items-center gap-1">
                <span className="h-2 w-2 rounded-full" style={{ background: accent }} />
                <span className="h-1.5 w-14 rounded" style={{ background: black ? strongAccent : softLine }} />
              </div>
              <div className="space-y-1">
                {Array.from({ length: section % 2 === 0 ? 3 : 2 }).map((_, line) => (
                  <div
                    key={line}
                    className="h-1 rounded"
                    style={{
                      width: `${line === 0 ? 92 : 58 + ((line + section) % 3) * 12}%`,
                      background: paleLine,
                    }}
                  />
                ))}
              </div>
            </div>
          ))}

          {warm && (
            <div className="absolute bottom-3 right-3 h-6 w-6 rounded-full bg-[color-mix(in_srgb,var(--color-primary),transparent_82%)]" />
          )}
        </div>
      </div>
    </div>
  )
}

export function NewResumeCard({ onClick }: NewResumeCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="resume-card-hover group relative flex min-h-[320px] flex-col overflow-hidden rounded-2xl border border-dashed border-border bg-card/45 p-3 text-left backdrop-blur-xl"
      style={{ '--delay': '80ms' } as CSSProperties}
    >
      <div className="flex flex-1 items-center justify-center rounded-xl border border-border/70 bg-[linear-gradient(135deg,var(--color-surface-hover),transparent)]">
        <div className="text-center">
          <div className="soft-pulse mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full border border-border-glow bg-primary/10 text-primary shadow-[0_0_28px_color-mix(in_srgb,var(--color-primary),transparent_70%)]">
            <Plus size={26} />
          </div>
          <p className="text-sm font-semibold text-foreground">新建简历</p>
          <p className="mt-1 text-xs text-muted-foreground">上传文件或从零开始</p>
        </div>
      </div>
    </button>
  )
}

export default function ProjectCard({ project, onClick }: ProjectCardProps) {
  const date = formatDate(project.created_at)
  const template = getTemplate(project.id)
  const ready = Boolean(project.current_draft_id)

  return (
    <button
      onClick={() => onClick(project.id)}
      className="resume-card-hover group relative flex min-h-[320px] flex-col overflow-hidden rounded-2xl border border-border bg-card/70 p-3 text-left backdrop-blur-xl"
    >
      <div className="relative flex-1 rounded-xl bg-canvas-bg p-3">
        <ResumePreview template={template} />
        <div className="absolute right-5 top-5 z-20 flex items-center gap-1 rounded-full border border-border bg-popover/92 px-2.5 py-1 text-[11px] font-medium text-foreground opacity-0 shadow-[0_12px_28px_rgba(2,8,23,0.18)] backdrop-blur-xl transition-opacity group-hover:opacity-100">
          <ArrowUpRight size={12} />
          打开
        </div>
      </div>

      <div className="flex items-center justify-between gap-3 px-1 pb-1 pt-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <FileText className="h-4 w-4 shrink-0 text-primary" />
            <h3 className="truncate text-sm font-semibold text-foreground">
              {project.title}
            </h3>
          </div>
          <div className="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
            {ready ? <CheckCircle2 className="h-3.5 w-3.5 text-accent" /> : <Clock3 className="h-3.5 w-3.5" />}
            <span>{ready ? '可编辑' : '待生成'}</span>
            <span>·</span>
            <span>{date}</span>
          </div>
        </div>
        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-muted-foreground transition-colors group-hover:bg-surface-hover group-hover:text-foreground">
          <MoreHorizontal size={16} />
        </span>
      </div>
    </button>
  )
}
