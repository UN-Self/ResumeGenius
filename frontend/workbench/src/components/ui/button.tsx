import * as React from "react"
import { cn } from "@/lib/utils"

type ButtonVariant = "primary" | "secondary" | "ghost" | "danger"
type ButtonSize = "sm" | "md" | "lg"

// Variant styles: each button type's color, border, and hover behavior.
// Extracted from 16+ button instances across the codebase.
const variantStyles: Record<ButtonVariant, string> = {
  primary:
    "bg-primary text-white hover:bg-primary-500 disabled:pointer-events-none disabled:opacity-50",
  secondary:
    "border border-border bg-white text-foreground hover:bg-gray-50",
  ghost:
    "text-muted-foreground hover:bg-surface-hover",
  danger:
    "border border-border text-muted-foreground hover:text-red-500 hover:border-red-300 disabled:pointer-events-none disabled:opacity-50",
}

// Size styles: sm for toolbars, md for dialogs (most common), lg for page actions.
const sizeStyles: Record<ButtonSize, string> = {
  sm: "px-3 py-1.5 text-sm rounded-lg",
  md: "px-4 py-2 text-sm rounded-lg",
  lg: "h-10 px-5 text-sm rounded-lg",
}

const Button = React.forwardRef<
  HTMLButtonElement,
  React.ButtonHTMLAttributes<HTMLButtonElement> & {
    variant?: ButtonVariant
    size?: ButtonSize
  }
>(({ className, variant = "primary", size = "md", ...props }, ref) => (
  <button
    ref={ref}
    className={cn(
      "font-medium transition-colors cursor-pointer inline-flex items-center justify-center",
      variantStyles[variant],
      sizeStyles[size],
      className
    )}
    {...props}
  />
))
Button.displayName = "Button"

export { Button, type ButtonVariant, type ButtonSize }
