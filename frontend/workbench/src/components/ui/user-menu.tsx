import { useState, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { LogOut, UserRound } from 'lucide-react'
import { authApi, type AuthUser } from '@/lib/api-client'

export function UserMenu() {
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [user, setUser] = useState<AuthUser | null>(null)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    authApi.me().then((u) => setUser(u as AuthUser)).catch(() => {})
  }, [])

  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    if (open) document.addEventListener('click', handleClick)
    return () => document.removeEventListener('click', handleClick)
  }, [open])

  const handleLogout = async () => {
    try {
      await authApi.logout()
    } catch {
      // ignore — navigate anyway
    }
    window.location.replace('/app/login')
  }

  const displayName = user?.username || ''
  const email = (user as any)?.email || ''
  const avatarUrl = (user as any)?.avatar_url || ''
  const initial = displayName ? displayName.charAt(0).toUpperCase() : '?'

  return (
    <div ref={ref} className="relative shrink-0">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="relative flex h-9 w-9 items-center justify-center rounded-full bg-gradient-to-br from-primary to-accent text-sm font-bold text-white shadow-[0_0_14px_color-mix(in_srgb,var(--color-primary),transparent_55%)] transition-all duration-200 hover:scale-105 hover:shadow-[0_0_22px_color-mix(in_srgb,var(--color-primary),transparent_40%)] active:scale-95 overflow-hidden"
        aria-label="用户菜单"
        title={displayName}
      >
        {avatarUrl ? (
          <img src={avatarUrl} alt={displayName} className="h-full w-full object-cover" />
        ) : (
          initial
        )}
      </button>

      {open && (
        <div className="absolute right-0 top-full z-50 mt-2 w-56 origin-top-right rounded-2xl border border-border bg-popover/95 p-1.5 shadow-[0_20px_60px_rgba(2,8,23,0.28)] backdrop-blur-2xl animate-in fade-in zoom-in-95">
          <div className="px-3 py-2">
            <div className="text-sm font-semibold text-foreground truncate">
              {displayName || '...'}
            </div>
            {email && (
              <div className="mt-0.5 text-xs text-muted-foreground truncate">
                {email}
              </div>
            )}
          </div>
          <div className="my-1 h-px bg-border" />

          <button
            type="button"
            onClick={() => { setOpen(false); navigate('/profile') }}
            className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-surface-hover hover:text-foreground"
          >
            <UserRound size={16} />
            个人中心
          </button>

          <button
            type="button"
            onClick={handleLogout}
            className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
          >
            <LogOut size={16} />
            退出登录
          </button>
        </div>
      )}
    </div>
  )
}
