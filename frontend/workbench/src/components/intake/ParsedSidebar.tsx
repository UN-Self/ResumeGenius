import { useState } from 'react'
import { intakeApi, parsingApi, type ParsedContent } from '@/lib/api-client'
import ParsedItem from './ParsedItem'
import UploadDialog from './UploadDialog'

interface ParsedSidebarProps {
  projectId: number
  contents: ParsedContent[]
  onParsed: (contents: ParsedContent[]) => void
}

export default function ParsedSidebar({ projectId, contents, onParsed }: ParsedSidebarProps) {
  const [uploadOpen, setUploadOpen] = useState(false)

  const handleUpload = async (file: File) => {
    await intakeApi.uploadFile(projectId, file)
    const result = await parsingApi.parseProject(projectId)
    onParsed(result.parsed_contents)
  }

  return (
    <div className="h-full overflow-y-auto p-4">
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          素材
        </h2>
        <button
          onClick={() => setUploadOpen(true)}
          className="text-xs text-primary hover:underline cursor-pointer"
        >
          上传文件
        </button>
      </div>
      <h3 className="mb-2 text-xs font-semibold text-muted-foreground">
        解析结果
      </h3>
      <div className="flex flex-col gap-2">
        {contents.map((c) => (
          <ParsedItem key={c.asset_id} content={c} />
        ))}
      </div>
      <UploadDialog open={uploadOpen} onClose={() => setUploadOpen(false)} onUpload={handleUpload} />
    </div>
  )
}
