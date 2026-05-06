import * as React from "react"
import { cn } from "@/lib/utils"

const Input = React.forwardRef<
  HTMLInputElement,
  React.InputHTMLAttributes<HTMLInputElement>
>(({ className, type, ...props }, ref) => (
  <input
    type={type}
    ref={ref}
    className={cn(
      "flex h-10 w-full px-4 text-sm rounded-lg border border-border bg-card text-foreground",
      "placeholder:text-muted-foreground",
      "focus:outline-none focus:ring-4 focus:ring-ring/20 focus:border-border-glow",
      "transition-all duration-200 backdrop-blur-xl",
      className
    )}
    {...props}
  />
))
Input.displayName = "Input"

export { Input }
