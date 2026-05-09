import { useState } from 'react'
import type { CSSProperties } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { FileText, LockKeyhole, Sparkles, Upload, UserRound, WandSparkles } from 'lucide-react'
import { ApiError, authApi, type User } from '@/lib/api-client'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Alert } from '@/components/ui/alert'
import { ThemeSwitcher } from '@/components/ui/theme-switcher'
import { AnimatedBrandTitle } from '@/components/ui/animated-brand-title'

interface LoginPageProps {
  onSuccess: (user: User) => void
}

export default function LoginPage({ onSuccess }: LoginPageProps) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const [searchParams] = useSearchParams()
  const justRegistered = searchParams.get('registered') === 'true'

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!username.trim() || !password) return
    try {
      setLoading(true)
      setError('')
      const user = await authApi.login(username.trim(), password)
      onSuccess(user)
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '登录失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="app-shell min-h-screen px-5">
      <div className="relative z-10 mx-auto flex min-h-screen max-w-6xl items-center py-10">
        <div className="grid w-full gap-8 lg:grid-cols-[1.1fr_420px] lg:items-center">
          <section className="stagger-in hidden lg:block">
            <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-border bg-card/60 px-3 py-1 text-xs font-medium text-muted-foreground backdrop-blur-xl">
              <Sparkles size={14} className="text-primary" />
              Futuristic resume intelligence
            </div>
            <div className="group">
              <h1 className="gradient-text max-w-2xl text-6xl font-semibold tracking-tight cursor-default select-none transition-all duration-500 group-hover:animate-[brand-title-wobble_680ms_cubic-bezier(0.34,1.56,0.64,1)] group-hover:[filter:drop-shadow(0_0_18px_color-mix(in_srgb,var(--color-primary),transparent_58%))]">
                让 AI 把经历整理成作品。
              </h1>
            </div>
            <p className="mt-6 max-w-xl text-sm leading-7 text-muted-foreground">
              上传资料、生成初稿、边聊边改，最后在白纸画布里得到一份可编辑、可导出的专业简历。
            </p>
            <div className="mt-10 grid max-w-xl grid-cols-3 gap-3">
              {[
                { label: '资料接入', icon: Upload },
                { label: 'AI 生成', icon: WandSparkles },
                { label: '可视化编辑', icon: FileText },
              ].map((item, index) => {
                const Icon = item.icon
                return (
                  <div
                    key={item.label}
                    className="feature-card glass-panel rounded-2xl p-4 stagger-in cursor-default select-none"
                    style={{ '--delay': `${index * 80 + 120}ms` } as CSSProperties}
                  >
                    <div className="feature-card-icon mb-3 inline-flex h-9 w-9 items-center justify-center rounded-xl bg-primary/10 text-primary transition-all duration-300">
                      <Icon className="h-5 w-5" />
                    </div>
                    <p className="text-sm font-semibold text-foreground">{item.label}</p>
                    <div className="mt-3 h-1.5 rounded-full bg-surface-hover">
                      <div className="feature-card-bar h-full rounded-full bg-primary transition-all duration-500" style={{ width: `${52 + index * 18}%` }} />
                    </div>
                  </div>
                )
              })}
            </div>
          </section>

          <div className="glass-panel stagger-in w-full min-w-0 rounded-3xl p-6">
            <div className="mb-6 flex items-start justify-between gap-3">
              <div className="min-w-0">
                <AnimatedBrandTitle className="text-2xl font-semibold" />
                <p className="mt-2 text-sm text-muted-foreground">登录以访问你的工作台</p>
              </div>
              <ThemeSwitcher compact className="shrink-0" />
            </div>

            {justRegistered && (
              <div className="mb-3 rounded-lg border border-green-300/40 bg-green-50/60 px-4 py-2.5 text-sm text-green-800 dark:border-green-500/30 dark:bg-green-950/40 dark:text-green-300">
                注册成功！请登录你的账号。
              </div>
            )}

            <form id="login-form" onSubmit={handleSubmit} className="space-y-3">
              <label className="relative block">
                <UserRound className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder="用户名或邮箱"
                  className="pl-9"
                />
              </label>
              <label className="relative block">
                <LockKeyhole className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  type="password"
                  placeholder="密码（至少 6 位）"
                  className="pl-9"
                />
              </label>

              {error && (
                <Alert>{error}</Alert>
              )}

              <Button
                type="submit"
                form="login-form"
                size="lg"
                className="mt-2 w-full"
                disabled={loading || !username.trim() || !password}
              >
                {loading ? '登录中...' : '进入工作台'}
              </Button>
            </form>

            <div className="mt-5 text-center text-sm text-muted-foreground">
              还没有账号？
              <Link to="/register" className="ml-1 text-primary hover:underline">
                注册
              </Link>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
