import type { ToolCallEntry } from '@/lib/api-client'

const TOOL_LABELS: Record<string, string> = {
  get_draft: '读取简历',
  apply_edits: '应用修改',
  search_assets: '搜索资料',
}

interface Props {
  calls: ToolCallEntry[]
}

export function ToolCallLog({ calls }: Props) {
  if (calls.length === 0) return null
  return (
    <div className="mt-1 space-y-1 text-xs text-[var(--color-text-secondary)]">
      {calls.map((call, i) => (
        <details key={i} className="group">
          <summary className="flex cursor-pointer items-center gap-1 hover:text-[var(--color-text-main)]">
            <span>{call.status === 'running' ? '...' : call.status === 'completed' ? 'OK' : 'FAIL'}</span>
            <span>{TOOL_LABELS[call.name] || call.name}</span>
          </summary>
          <pre className="mt-1 max-h-32 overflow-auto rounded border border-[var(--color-divider)] bg-[var(--color-card)] p-2 text-[10px] text-[var(--color-text-main)]">
            {JSON.stringify({ params: call.params, result: call.result }, null, 2)}
          </pre>
        </details>
      ))}
    </div>
  )
}
