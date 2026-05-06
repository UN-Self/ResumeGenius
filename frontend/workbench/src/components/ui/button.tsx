import * as React from "react"
import { cn } from "@/lib/utils"

type ButtonVariant = "primary" | "secondary" | "ghost" | "danger"
type ButtonSize = "sm" | "md" | "lg"

// Variant styles: each button type's color, border, and hover behavior.
// Extracted from 16+ button instances across the codebase.
const variantStyles: Record<ButtonVariant, string> = {
  primary:
    "relative overflow-hidden bg-primary text-primary-foreground shadow-[0_10px_28px_color-mix(in_srgb,var(--color-primary),transparent_76%)] hover:brightness-110 disabled:pointer-events-none disabled:opacity-50 before:pointer-events-none before:absolute before:inset-y-0 before:-left-1/2 before:w-1/2 before:skew-x-[-18deg] before:bg-white/25 before:opacity-0 hover:before:animate-[shine_700ms_ease]",
  secondary:
    "border border-border bg-card/70 text-foreground backdrop-blur-xl hover:border-border-glow hover:bg-surface-hover disabled:pointer-events-none disabled:opacity-50",
  ghost:
    "text-muted-foreground hover:bg-surface-hover hover:text-foreground disabled:pointer-events-none disabled:opacity-50",
  danger:
    "border border-border bg-card/60 text-muted-foreground hover:border-destructive/50 hover:bg-destructive/10 hover:text-destructive disabled:pointer-events-none disabled:opacity-50",
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
      "font-medium transition-all duration-200 cursor-pointer inline-flex items-center justify-center active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/45 focus-visible:ring-offset-2 focus-visible:ring-offset-background",
      variantStyles[variant],
      sizeStyles[size],
      className
    )}
    {...props}
  />
))
Button.displayName = "Button"

export { Button, type ButtonVariant, type ButtonSize }
