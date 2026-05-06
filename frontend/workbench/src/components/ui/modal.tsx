import * as React from "react"
import { cn } from "@/lib/utils"

/* ---------- Subcomponents ---------- */

function ModalHeader({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
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
}: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("mt-2", className)} {...props} />
}

function ModalFooter({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
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
  children: React.ReactNode
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
  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="fixed inset-0 bg-black/45 backdrop-blur-md" onClick={onClose} />
      {/* Content */}
      <div
        className={cn(
          "relative glass-panel rounded-2xl p-6 w-full mx-4 stagger-in",
          maxWidth,
          className
        )}
      >
        {children}
      </div>
    </div>
  )
}

export { Modal, ModalHeader, ModalBody, ModalFooter }
