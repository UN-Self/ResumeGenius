import { FileImage, FileText, GitBranch, type LucideIcon } from 'lucide-react'

type FileVisualKey = 'pdf' | 'docx' | 'image' | 'git' | 'note' | 'generic'

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
  image: {
    key: 'image',
    chipLabel: 'IMAGE',
    typeLabel: '\u56fe\u7247',
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
    chipLabel: 'NOTE',
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

export function getExt(name: string) {
  const extIndex = name.lastIndexOf('.')
  return extIndex >= 0 ? name.substring(extIndex).toLowerCase() : ''
}

export function getUploadFileVisual(name: string) {
  switch (getExt(name)) {
    case '.pdf':
      return visuals.pdf
    case '.docx':
      return visuals.docx
    case '.png':
    case '.jpg':
    case '.jpeg':
      return visuals.image
    default:
      return visuals.generic
  }
}

export function getAssetVisual(type: string) {
  switch (type) {
    case 'resume_pdf':
      return visuals.pdf
    case 'resume_docx':
      return visuals.docx
    case 'resume_image':
      return visuals.image
    case 'git_repo':
      return visuals.git
    case 'note':
      return visuals.note
    default:
      return visuals.generic
  }
}

export function formatFileSize(size: number) {
  if (size >= 1024 * 1024) {
    return `${(size / (1024 * 1024)).toFixed(1)} MB`
  }

  return `${(size / 1024).toFixed(1)} KB`
}
