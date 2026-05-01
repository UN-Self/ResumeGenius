import { useRef, useEffect } from 'react'

interface Props {
  html: string
}

export function HtmlPreview({ html }: Props) {
  const iframeRef = useRef<HTMLIFrameElement>(null)

  useEffect(() => {
    const iframe = iframeRef.current
    if (!iframe) return

    const doc = iframe.contentDocument
    if (!doc) return

    doc.open()
    doc.write(html)
    doc.close()
  }, [html])

  return (
    <iframe
      ref={iframeRef}
      className="w-full border-0"
      style={{ height: 200 }}
      sandbox="allow-same-origin"
      title="AI 生成的简历 HTML 预览"
    />
  )
}
