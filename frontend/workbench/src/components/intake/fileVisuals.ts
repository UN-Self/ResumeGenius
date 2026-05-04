import { FileImage, FileText, GitBranch, type LucideIcon } from 'lucide-react'
import type { Asset } from '@/lib/api-client'

type FileVisualKey = 'pdf' | 'docx' | 'png' | 'jpg' | 'jpeg' | 'git' | 'note' | 'generic'

export interface FileVisual {
  key: FileVisualKey
  chipLabel: string
  typeLabel: string
  icon: LucideIcon
  iconWrapperClassName: string
  iconClassName: string
  chipClassName: string
}

const visuals: Record<FileVisualKey, FileVisual> = {
  pdf: {
    key: 'pdf',
    chipLabel: 'PDF',
    typeLabel: 'PDF \u7b80\u5386',
    icon: FileText,
    iconWrapperClassName: 'border-red-200 bg-red-50',
    iconClassName: 'text-red-600',
    chipClassName: 'border-red-200 bg-red-100 text-red-700',
  },
  docx: {
    key: 'docx',
    chipLabel: 'DOCX',
    typeLabel: 'DOCX \u7b80\u5386',
    icon: FileText,
    iconWrapperClassName: 'border-sky-200 bg-sky-50',
    iconClassName: 'text-sky-600',
    chipClassName: 'border-sky-200 bg-sky-100 text-sky-700',
  },
  png: {
    key: 'png',
    chipLabel: 'PNG',
    typeLabel: 'PNG \u56fe\u7247',
    icon: FileImage,
    iconWrapperClassName: 'border-emerald-200 bg-emerald-50',
    iconClassName: 'text-emerald-600',
    chipClassName: 'border-emerald-200 bg-emerald-100 text-emerald-700',
  },
  jpg: {
    key: 'jpg',
    chipLabel: 'JPG',
    typeLabel: 'JPG \u56fe\u7247',
    icon: FileImage,
    iconWrapperClassName: 'border-emerald-200 bg-emerald-50',
    iconClassName: 'text-emerald-600',
    chipClassName: 'border-emerald-200 bg-emerald-100 text-emerald-700',
  },
  jpeg: {
    key: 'jpeg',
    chipLabel: 'JPEG',
    typeLabel: 'JPEG \u56fe\u7247',
    icon: FileImage,
    iconWrapperClassName: 'border-emerald-200 bg-emerald-50',
    iconClassName: 'text-emerald-600',
    chipClassName: 'border-emerald-200 bg-emerald-100 text-emerald-700',
  },
  git: {
    key: 'git',
    chipLabel: 'GIT',
    typeLabel: 'Git \u4ed3\u5e93',
    icon: GitBranch,
    iconWrapperClassName: 'border-slate-200 bg-slate-50',
    iconClassName: 'text-slate-700',
    chipClassName: 'border-slate-200 bg-slate-100 text-slate-700',
  },
  note: {
    key: 'note',
    chipLabel: '\u5907\u6ce8',
    typeLabel: '\u5907\u6ce8',
    icon: FileText,
    iconWrapperClassName: 'border-amber-200 bg-amber-50',
    iconClassName: 'text-amber-700',
    chipClassName: 'border-amber-200 bg-amber-100 text-amber-700',
  },
  generic: {
    key: 'generic',
    chipLabel: 'FILE',
    typeLabel: '\u6587\u4ef6',
    icon: FileText,
    iconWrapperClassName: 'border-primary-200 bg-primary-50',
    iconClassName: 'text-primary-700',
    chipClassName: 'border-primary-200 bg-primary-100 text-primary-700',
  },
}

const STORED_FILE_PREFIX_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}_/i

export function getExt(name: string) {
  const extIndex = name.lastIndexOf('.')
  return extIndex >= 0 ? name.substring(extIndex).toLowerCase() : ''
}

export function stripStoredFilePrefix(name: string) {
  return name.replace(STORED_FILE_PREFIX_PATTERN, '')
}

export function getDisplayFileName(name: string) {
  const cleanedName = stripStoredFilePrefix(name.trim())
  const extIndex = cleanedName.lastIndexOf('.')
  if (extIndex <= 0) {
    return cleanedName
  }

  return cleanedName.substring(0, extIndex)
}

export function getStoredFileName(reference?: string | null) {
  if (!reference) {
    return ''
  }

  const fileName = reference.split('/').pop() ?? reference
  return stripStoredFilePrefix(fileName.trim())
}

export function getUploadFileVisual(name: string) {
  switch (getExt(name)) {
    case '.pdf':
      return visuals.pdf
    case '.docx':
      return visuals.docx
    case '.png':
      return visuals.png
    case '.jpg':
      return visuals.jpg
    case '.jpeg':
      return visuals.jpeg
    default:
      return visuals.generic
  }
}

export function getAssetVisual(type: string, uri?: string | null) {
  switch (type) {
    case 'resume_pdf':
      return visuals.pdf
    case 'resume_docx':
      return visuals.docx
    case 'resume_image':
      switch (getExt(uri ?? '')) {
        case '.png':
          return visuals.png
        case '.jpg':
          return visuals.jpg
        case '.jpeg':
          return visuals.jpeg
        default:
          return visuals.png
      }
    case 'git_repo':
      return visuals.git
    case 'note':
      return visuals.note
    default:
      return visuals.generic
  }
}

export function getAssetBadgeText(type: string, reference?: string | null) {
  switch (type) {
    case 'resume_pdf':
      return 'PDF'
    case 'resume_docx':
      return 'DOCX'
    case 'resume_image':
      return getUploadFileVisual(reference ?? '').chipLabel
    case 'git_repo':
      return 'GIT'
    case 'note':
      return '\u5907\u6ce8'
    default:
      return '\u7d20\u6750'
  }
}

export function getDisplayAssetTitle(type: string, label: string) {
  const trimmedLabel = label.trim()
  if (!trimmedLabel) {
    return ''
  }

  switch (type) {
    case 'resume_pdf':
    case 'resume_docx':
    case 'resume_image':
      return getDisplayFileName(trimmedLabel)
    case 'git_repo': {
      const normalized = trimmedLabel.replace(/\/+$/, '')
      const repoName = normalized.split('/').pop() ?? normalized
      return repoName.replace(/\.git$/i, '')
    }
    default:
      return trimmedLabel
  }
}

export function getOriginalFilenameFromAsset(asset: Pick<Asset, 'metadata' | 'uri'>) {
  if (asset.metadata && typeof asset.metadata === 'object') {
    const parsing = (asset.metadata as Record<string, unknown>).parsing
    if (parsing && typeof parsing === 'object') {
      const originalFilename = (parsing as Record<string, unknown>).original_filename
      if (typeof originalFilename === 'string' && originalFilename.trim()) {
        return originalFilename.trim()
      }
    }
  }

  return getStoredFileName(asset.uri)
}

export function getUploadAssetType(name: string) {
  switch (getExt(name)) {
    case '.pdf':
      return 'resume_pdf'
    case '.docx':
      return 'resume_docx'
    case '.png':
    case '.jpg':
    case '.jpeg':
      return 'resume_image'
    default:
      return ''
  }
}

export function formatFileSize(size: number) {
  if (size >= 1024 * 1024) {
    return `${(size / (1024 * 1024)).toFixed(1)} MB`
  }

  return `${(size / 1024).toFixed(1)} KB`
}
