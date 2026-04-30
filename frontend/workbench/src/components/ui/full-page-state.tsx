import { cn } from "@/lib/utils"

interface FullPageStateProps {
  variant: "loading" | "error"
  message?: string
  className?: string
}

export function FullPageState({
  variant,
  message = variant === "loading" ? "加载中..." : "",
  className,
}: FullPageStateProps) {
  return (
    <div
      className={cn(
        "h-screen bg-background flex items-center justify-center",
        className
      )}
    >
      <p
        className={cn(
          "text-sm",
          variant === "loading"
            ? "text-muted-foreground"
            : "text-destructive"
        )}
      >
        {message}
      </p>
    </div>
  )
}
