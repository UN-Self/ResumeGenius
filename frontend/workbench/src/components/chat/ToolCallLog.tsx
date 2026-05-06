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
    <div className="space-y-1 text-xs text-gray-500 mt-1">
      {calls.map((call, i) => (
        <details key={i} className="group">
          <summary className="cursor-pointer hover:text-gray-700 flex items-center gap-1">
            <span>{call.status === 'running' ? '...' : call.status === 'completed' ? 'OK' : 'FAIL'}</span>
            <span>{TOOL_LABELS[call.name] || call.name}</span>
          </summary>
          <pre className="mt-1 p-2 bg-gray-50 rounded text-[10px] overflow-auto max-h-32">
            {JSON.stringify({ params: call.params, result: call.result }, null, 2)}
          </pre>
        </details>
      ))}
    </div>
  )
}
