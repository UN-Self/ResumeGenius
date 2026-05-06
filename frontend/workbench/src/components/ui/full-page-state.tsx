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
        "app-shell h-screen flex items-center justify-center",
        className
      )}
    >
      <div className="glass-panel relative z-10 rounded-2xl px-8 py-6 text-center">
        {variant === "loading" && (
          <div className="mx-auto mb-3 h-8 w-8 rounded-full border-2 border-primary/20 border-t-primary animate-spin" />
        )}
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
    </div>
  )
}
