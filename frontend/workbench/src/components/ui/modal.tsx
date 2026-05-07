import { useEffect, type HTMLAttributes, type ReactNode } from "react"
import { createPortal } from "react-dom"
import { cn } from "@/lib/utils"

/* ---------- Subcomponents ---------- */

function ModalHeader({
  className,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("text-base font-serif font-semibold text-foreground", className)}
      {...props}
    />
  )
}

function ModalBody({
  className,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("mt-2", className)} {...props} />
}

function ModalFooter({
  className,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("flex justify-end gap-2 mt-5", className)}
      {...props}
    />
  )
}

/* ---------- Root ---------- */

interface ModalProps {
  open: boolean
  onClose: () => void
  children: ReactNode
  className?: string
  /** Override the max-width. Defaults to max-w-md. */
  maxWidth?: string
}

function Modal({
  open,
  onClose,
  children,
  className,
  maxWidth = "max-w-md",
}: ModalProps) {
  useEffect(() => {
    if (!open) return

    const prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.body.style.overflow = prevOverflow
    }
  }, [open, onClose])

  if (!open) return null

  return createPortal(
    <div className="fixed inset-0 z-[9998] flex items-center justify-center p-4">
      {/* Backdrop */}
      <div className="fixed inset-0 bg-black/55 backdrop-blur-md" onClick={onClose} />
      {/* Content */}
      <div
        className={cn(
          "relative z-[9999] w-full rounded-2xl border border-border bg-popover p-6 text-popover-foreground shadow-[0_24px_90px_rgba(2,8,23,0.48)]",
          maxWidth,
          className
        )}
      >
        {children}
      </div>
    </div>,
    document.body
  )
}

export { Modal, ModalHeader, ModalBody, ModalFooter }
