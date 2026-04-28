export function EditorSkeleton() {
  return (
    <div className="a4-canvas">
      <div className="animate-pulse">
        {/* Header skeleton */}
        <div className="skeleton-line h-8 w-48 mb-6" />
        <div className="skeleton-line h-4 w-32 mb-4" />

        {/* Content skeleton */}
        <div className="space-y-3">
          <div className="skeleton-line h-4 w-full" />
          <div className="skeleton-line h-4 w-full" />
          <div className="skeleton-line h-4 w-3/4" />
        </div>

        <div className="mt-6 space-y-3">
          <div className="skeleton-line h-4 w-full" />
          <div className="skeleton-line h-4 w-5/6" />
        </div>
      </div>
    </div>
  )
}
