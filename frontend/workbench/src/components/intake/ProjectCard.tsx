interface ProjectCardProps {
  project: {
    id: number
    title: string
    status: string
    created_at: string
    asset_count?: number
  }
  onClick: (id: number) => void
}

export default function ProjectCard({ project, onClick }: ProjectCardProps) {
  const date = new Date(project.created_at).toLocaleDateString('zh-CN', {
    month: 'short',
    day: 'numeric',
  })

  return (
    <button
      onClick={() => onClick(project.id)}
      className="w-full text-left px-5 py-4 rounded-lg border border-border bg-card hover:bg-accent transition-colors group"
    >
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0 flex-1">
          <h3 className="font-serif text-base font-semibold text-foreground truncate group-hover:text-accent transition-colors">
            {project.title}
          </h3>
          <div className="flex items-center gap-3 mt-1 text-sm text-muted-foreground">
            <span>{date}</span>
            {project.asset_count !== undefined && (
              <span>{project.asset_count} 份资料</span>
            )}
          </div>
        </div>
        <svg
          className="w-4 h-4 text-muted-foreground group-hover:text-accent transition-colors shrink-0"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          strokeWidth={2}
        >
          <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
        </svg>
      </div>
    </button>
  )
}
