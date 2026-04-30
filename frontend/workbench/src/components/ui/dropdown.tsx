import * as React from "react"
import { cn } from "@/lib/utils"

interface DropdownTriggerProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  children: React.ReactNode
}

const DropdownTrigger = React.forwardRef<
  HTMLButtonElement,
  DropdownTriggerProps
>(({ className, children, ...props }, ref) => (
  <button
    ref={ref}
    type="button"
    className={cn(
      "flex items-center gap-1 px-2 min-h-[44px] rounded-md text-sm",
      "text-muted-foreground hover:bg-surface-hover transition-colors",
      "focus:outline-none focus:ring-2 focus:ring-ring",
      className
    )}
    {...props}
  >
    {children}
  </button>
))
DropdownTrigger.displayName = "DropdownTrigger"

/* ---------- Item ---------- */

interface DropdownItemProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  active?: boolean
}

const DropdownItem = React.forwardRef<
  HTMLButtonElement,
  DropdownItemProps
>(({ className, active, children, ...props }, ref) => (
  <button
    ref={ref}
    type="button"
    className={cn(
      "px-3 py-2 text-sm text-left rounded-md transition-colors",
      active
        ? "bg-primary-50 text-primary"
        : "text-muted-foreground hover:bg-surface-hover",
      className
    )}
    {...props}
  >
    {children}
  </button>
))
DropdownItem.displayName = "DropdownItem"

export { DropdownTrigger, DropdownItem }
