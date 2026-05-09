import {
  CheckCircle2,
  ChevronDown,
  FileText,
  LoaderCircle,
  Palette,
  PencilLine,
  Search,
  XCircle,
} from 'lucide-react'
import type { ComponentType } from 'react'
import type { ToolCallEntry } from '@/lib/api-client'

const TOOL_META: Record<string, { label: string; icon: ComponentType<{ className?: string }> }> = {
  get_draft: { label: '读取简历', icon: FileText },
  apply_edits: { label: '应用修改', icon: PencilLine },
  search_assets: { label: '搜索资料', icon: Search },
  'resume-design': { label: '加载设计技能', icon: Palette },
  'resume-interview': { label: '加载面试技能', icon: Search },
  get_skill_reference: { label: '获取参考内容', icon: FileText },
}

interface Props {
  calls: ToolCallEntry[]
  compact?: boolean
}

interface ToolCallGroup {
  name: string
  calls: ToolCallEntry[]
  status: ToolCallEntry['status']
}

function groupCalls(calls: ToolCallEntry[]): ToolCallGroup[] {
  const groups = new Map<string, ToolCallEntry[]>()
  for (const call of calls) {
    const current = groups.get(call.name) ?? []
    current.push(call)
    groups.set(call.name, current)
  }

  return Array.from(groups.entries()).map(([name, groupCalls]) => {
    const status = groupCalls.some((call) => call.status === 'running')
      ? 'running'
      : groupCalls.some((call) => call.status === 'failed')
        ? 'failed'
        : 'completed'

    return { name, calls: groupCalls, status }
  })
}

function getGroupPayload(group: ToolCallGroup) {
  if (group.calls.length === 1) {
    return getPayload(group.calls[0])
  }

  const attempts = group.calls.map((call, index) => ({
    index: index + 1,
    status: call.status,
    params: call.params ?? {},
    result: formatResult(call.result),
  }))
  const latest = group.calls[group.calls.length - 1]

  return JSON.stringify({
    total: group.calls.length,
    running: group.calls.filter((call) => call.status === 'running').length,
    completed: group.calls.filter((call) => call.status === 'completed').length,
    failed: group.calls.filter((call) => call.status === 'failed').length,
    latest: {
      params: latest?.params ?? {},
      result: formatResult(latest?.result),
    },
    attempts,
  }, null, 2)
}

function getPayload(call: ToolCallEntry) {
  const result = formatResult(call.result)
  return JSON.stringify({ params: call.params ?? {}, result }, null, 2)
}

function formatResult(result?: string) {
  if (typeof result !== 'string') return result
  try {
    return JSON.parse(result)
  } catch {
    return result
  }
}

function StatusIcon({ status }: { status: ToolCallEntry['status'] }) {
  if (status === 'running') return <LoaderCircle className="h-3.5 w-3.5 animate-spin" />
  if (status === 'failed') return <XCircle className="h-3.5 w-3.5" />
  return <CheckCircle2 className="h-3.5 w-3.5" />
}

export function ToolCallLog({ calls, compact = false }: Props) {
  if (calls.length === 0) return null
  const groups = groupCalls(calls)

  return (
    <div className={`ai-tool-log ${compact ? 'is-compact' : ''}`}>
      {groups.map((group) => {
        const meta = TOOL_META[group.name] ?? { label: group.name, icon: FileText }
        const Icon = meta.icon
        const failedCount = group.calls.filter((call) => call.status === 'failed').length

        return (
          <details key={group.name} className="ai-tool-item">
            <summary>
              <span className="ai-tool-left">
                <span className="ai-tool-glyph">
                  <Icon className="h-3.5 w-3.5" />
                </span>
                <span className="ai-tool-name">{meta.label}</span>
                {group.calls.length > 1 && (
                  <span className="ai-tool-count">×{group.calls.length}</span>
                )}
              </span>
              <span className={`ai-tool-status is-${group.status}`}>
                <StatusIcon status={group.status} />
                <span>
                  {group.status === 'running'
                    ? '处理中'
                    : failedCount > 0
                      ? `${failedCount} 次失败`
                      : '完成'}
                </span>
              </span>
              <ChevronDown className="ai-tool-chevron h-3.5 w-3.5" />
            </summary>
            <pre>{getGroupPayload(group)}</pre>
          </details>
        )
      })}
    </div>
  )
}
