import { cn } from "@/lib/utils"

interface AlertProps {
  variant?: "error"
  children: React.ReactNode
  className?: string
}

export function Alert({ variant = "error", children, className }: AlertProps) {
  return (
    <div
      className={cn(
        "px-4 py-2.5 text-sm rounded-lg",
        variant === "error" &&
          "bg-destructive/10 text-destructive border border-destructive/20",
        className
      )}
    >
      {children}
    </div>
  )
}
