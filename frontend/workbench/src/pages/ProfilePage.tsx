import { useState, useEffect, useRef } from 'react'
import { Link } from 'react-router-dom'
import { ArrowLeft, Camera, KeyRound, LockKeyhole, ShieldCheck, ShoppingBag, Sparkles, UserRound, TrendingUp, TrendingDown, Clock, BarChart3, PieChart } from 'lucide-react'
import { authApi, ApiError, type AuthUser, type PointsRecord, type PointsDashboard } from '@/lib/api-client'
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, PieChart as RePieChart, Pie, Cell } from 'recharts'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Alert } from '@/components/ui/alert'
import { ThemeSwitcher } from '@/components/ui/theme-switcher'
import { PointsCoin } from '@/components/ui/PointsCoin'

type Tab = 'profile' | 'points' | 'password' | 'shop'

const tabs: { key: Tab; label: string; icon: typeof UserRound }[] = [
  { key: 'profile', label: '个人信息', icon: UserRound },
  { key: 'points', label: '积分面板', icon: Sparkles },
  { key: 'shop', label: '套餐商城', icon: ShoppingBag },
  { key: 'password', label: '密码修改', icon: LockKeyhole },
]

// Client-side image compression: resize to max 256x256, JPEG quality 0.8
function compressImage(file: File): Promise<File> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => {
      const maxDim = 256
      let { width, height } = img
      if (width <= maxDim && height <= maxDim) {
        resolve(file)
        return
      }
      const ratio = width / height
      if (ratio > 1) { width = maxDim; height = Math.round(maxDim / ratio) }
      else { height = maxDim; width = Math.round(maxDim * ratio) }

      const canvas = document.createElement('canvas')
      canvas.width = width
      canvas.height = height
      const ctx = canvas.getContext('2d')!
      ctx.drawImage(img, 0, 0, width, height)
      canvas.toBlob((blob) => {
        if (!blob) { reject(new Error('compress failed')); return }
        resolve(new File([blob], file.name.replace(/\.[^.]+$/, '.jpg'), { type: 'image/jpeg' }))
      }, 'image/jpeg', 0.8)
    }
    img.onerror = () => reject(new Error('load image failed'))
    img.src = URL.createObjectURL(file)
  })
}

export default function ProfilePage() {
  const [user, setUser] = useState<AuthUser | null>(null)
  const [tab, setTab] = useState<Tab>('profile')
  const [loading, setLoading] = useState(true)
  const [avatarUploading, setAvatarUploading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    authApi.me().then((u) => setUser(u as AuthUser)).catch(() => {}).finally(() => setLoading(false))
  }, [])

  const refreshUser = async () => {
    const u = await authApi.me()
    setUser(u as AuthUser)
  }

  const handleAvatarClick = () => {
    if (avatarUploading) return
    fileInputRef.current?.click()
  }

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    try {
      setAvatarUploading(true)
      const compressed = await compressImage(file)
      const updated = await authApi.uploadAvatar(compressed)
      setUser(updated as AuthUser)
    } catch (err) {
      console.error('Avatar upload failed:', err)
    } finally {
      setAvatarUploading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  const avatarUrl = (user as any)?.avatar_url || ''
  const displayInitial = user?.username?.charAt(0).toUpperCase() || '?'

  if (loading) {
    return (
      <div className="app-shell flex min-h-screen items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    )
  }

  return (
    <div className="app-shell min-h-screen">
      <div className="relative z-10 mx-auto w-full max-w-[1400px] px-5 py-6 sm:px-8 lg:px-10">
        {/* Header */}
        <header className="stagger-in mb-8 flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Link to="/" className="flex h-9 w-9 items-center justify-center rounded-lg border border-border bg-card/60 text-muted-foreground transition-colors hover:bg-surface-hover hover:text-foreground">
              <ArrowLeft size={18} />
            </Link>
            <div>
              <h1 className="text-2xl font-semibold text-foreground">个人中心</h1>
              <p className="text-sm text-muted-foreground">管理你的账号信息和设置</p>
            </div>
          </div>
          <ThemeSwitcher compact />
        </header>

        <div className="grid gap-6 lg:grid-cols-[280px_1fr]">
          {/* ── Left sidebar ── */}
          <aside className="stagger-in space-y-4">
            {/* Avatar card */}
            <div className="glass-panel rounded-2xl p-6 text-center">
              <input
                ref={fileInputRef}
                type="file"
                accept="image/png,image/jpeg,image/webp"
                className="hidden"
                onChange={handleFileChange}
              />
              <button
                type="button"
                onClick={handleAvatarClick}
                disabled={avatarUploading}
                className="group relative mx-auto mb-3 flex h-20 w-20 items-center justify-center rounded-full transition-transform duration-200 hover:scale-105 active:scale-95 disabled:opacity-60"
                title="点击上传头像"
              >
                {avatarUrl ? (
                  <img
                    src={avatarUrl}
                    alt={user?.username || ''}
                    className="h-full w-full rounded-full object-cover shadow-[0_0_24px_color-mix(in_srgb,var(--color-primary),transparent_55%)]"
                  />
                ) : (
                  <div className="flex h-full w-full items-center justify-center rounded-full bg-gradient-to-br from-primary to-accent text-3xl font-bold text-white shadow-[0_0_24px_color-mix(in_srgb,var(--color-primary),transparent_55%)]">
                    {displayInitial}
                  </div>
                )}
                {/* Hover overlay */}
                <div className="absolute inset-0 flex items-center justify-center rounded-full bg-black/40 opacity-0 transition-opacity group-hover:opacity-100">
                  {avatarUploading ? (
                    <div className="h-5 w-5 animate-spin rounded-full border-2 border-white border-t-transparent" />
                  ) : (
                    <Camera size={20} className="text-white" />
                  )}
                </div>
              </button>
              <p className="text-base font-semibold text-foreground">{user?.username || '...'}</p>
              {(user as any)?.email && (
                <p className="mt-0.5 text-xs text-muted-foreground truncate">{(user as any).email}</p>
              )}
            </div>

            {/* Plan card */}
            <PlanCard user={user} />

            {/* Nav */}
            <nav className="glass-panel rounded-2xl p-1.5">
              {tabs.map(({ key, label, icon: Icon }) => (
                <button
                  key={key}
                  onClick={() => setTab(key)}
                  className={`flex w-full items-center gap-2.5 rounded-xl px-3 py-2.5 text-sm transition-colors ${
                    tab === key
                      ? 'bg-primary text-primary-foreground font-medium shadow-sm'
                      : 'text-muted-foreground hover:bg-surface-hover hover:text-foreground'
                  }`}
                >
                  <Icon size={16} />
                  {label}
                </button>
              ))}
            </nav>

          </aside>

          {/* ── Right content area ── */}
          <main className="stagger-in min-h-[400px]">
            {tab === 'profile' && (
              <ProfilePanel user={user} onUpdated={refreshUser} />
            )}
            {tab === 'points' && (
              <PointsPanel />
            )}
            {tab === 'shop' && (
              <ShopPanel />
            )}
            {tab === 'password' && (
              <PasswordPanel />
            )}
          </main>
        </div>
      </div>
    </div>
  )
}

// ── Plan Card ──

function PlanCard({ user }: { user: AuthUser | null }) {
  const plan = (user as any)?.plan || 'free'
  const isPro = plan === 'pro'
  const startedAt = (user as any)?.plan_started_at
  const expiresAt = (user as any)?.plan_expires_at

  const formatDate = (d?: string) => {
    if (!d) return '—'
    return new Date(d).toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })
  }

  return (
    <div className={`glass-panel rounded-2xl p-5 text-center relative overflow-hidden ${
      isPro ? 'border-primary/30' : ''
    }`}>
      {/* Decorative glow */}
      {isPro && (
        <div className="absolute -top-6 left-1/2 -translate-x-1/2 w-20 h-20 rounded-full bg-primary/15 blur-xl" />
      )}
      <p className="relative z-10 text-xs font-medium text-muted-foreground tracking-widest uppercase mb-1">
        当前套餐
      </p>
      <p className={`relative z-10 font-serif italic text-2xl font-semibold mb-3 cursor-default select-none ${
        isPro ? 'gradient-text' : 'text-foreground animate-plan-title'
      }`}>
        {isPro ? 'Pro' : 'Free'}
      </p>
      <div className="relative z-10 space-y-1.5 text-xs text-muted-foreground">
        <div className="flex items-center justify-between gap-2">
          <span>开通时间</span>
          <span className="text-foreground font-medium">{formatDate(startedAt)}</span>
        </div>
        <div className="flex items-center justify-between gap-2">
          <span>到期时间</span>
          <span className={`font-medium ${isPro && expiresAt ? 'text-amber-500' : 'text-emerald-500'}`}>
            {isPro && expiresAt ? formatDate(expiresAt) : '永久有效'}
          </span>
        </div>
      </div>
    </div>
  )
}

// ── Profile Panel (edit nickname) ──

function ProfilePanel({ user, onUpdated }: { user: AuthUser | null; onUpdated: () => void }) {
  const [nickname, setNickname] = useState(user?.username || '')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)

  const handleSave = async () => {
    if (!nickname.trim() || nickname.trim().length < 2) {
      setError('昵称至少 2 个字符')
      return
    }
    try {
      setSaving(true)
      setError('')
      setSuccess(false)
      await authApi.updateProfile(nickname.trim())
      setSuccess(true)
      onUpdated()
      setTimeout(() => setSuccess(false), 2000)
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="glass-panel rounded-2xl p-6">
      <h2 className="mb-4 text-lg font-semibold text-foreground">个人信息</h2>
      <div className="space-y-4 max-w-md">
        <div>
          <label className="mb-1.5 flex items-center gap-1.5 text-sm text-muted-foreground"><svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" className="animate-icon-float text-primary/70"><circle cx="12" cy="8" r="5"/><path d="M20 21a8 8 0 1 0-16 0"/></svg>昵称</label>
          <Input
            value={nickname}
            onChange={(e) => setNickname(e.target.value)}
            placeholder="输入新的昵称"
          />
        </div>
        <div>
          <label className="mb-1.5 flex items-center gap-1.5 text-sm text-muted-foreground"><svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" className="animate-icon-float text-primary/70"><rect x="2" y="4" width="20" height="16" rx="2"/><path d="m22 4-10 7.5L2 4"/></svg>邮箱</label>
          <Input
            value={(user as any)?.email || '未绑定'}
            disabled
            className="opacity-60"
          />
        </div>
        {error && <Alert>{error}</Alert>}
        {success && (
          <div className="rounded-lg border border-green-300/40 bg-green-50/60 px-4 py-2.5 text-sm text-green-800 dark:border-green-500/30 dark:bg-green-950/40 dark:text-green-300">
            保存成功
          </div>
        )}
        <Button onClick={handleSave} disabled={saving || nickname === user?.username}>
          {saving ? '保存中...' : '保存修改'}
        </Button>
      </div>
    </div>
  )
}

// ── Points Panel (NewAPI-style visual dashboard) ──

const CATEGORY_COLORS: Record<string, string> = {
  ai_usage: '#f59e0b',
  pdf_export: '#a855f7',
  register_bonus: '#22c55e',
  daily_login: '#3b82f6',
  invite_bonus: '#06b6d4',
  admin_grant: '#10b981',
}
const CATEGORY_FALLBACK_COLORS = ['#f59e0b', '#a855f7', '#22c55e', '#3b82f6', '#06b6d4', '#10b981', '#ef4444', '#ec4899']

const POINTS_TYPE_LABELS: Record<string, string> = {
  register_bonus: '注册奖励',
  daily_login: '每日签到',
  ai_usage: 'AI 消耗',
  pdf_export: 'PDF 导出',
  invite_bonus: '邀请奖励',
  admin_grant: '系统发放',
}

function PointsPanel() {
  const [dashboard, setDashboard] = useState<PointsDashboard | null>(null)
  const [records, setRecords] = useState<PointsRecord[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      authApi.getPointsDashboard().catch(() => null),
      authApi.getPointsRecords().catch(() => ({ items: [] })),
    ]).then(([d, r]) => {
      setDashboard(d)
      setRecords(r?.items || [])
    }).finally(() => setLoading(false))
  }, [])

  if (loading) {
    return (
      <div className="glass-panel rounded-2xl p-6 flex items-center justify-center min-h-[400px]">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    )
  }

  const d = dashboard || { balance: 0, month_used: 0, total_earned: 0, daily_usage: [], categories: [] }
  const hasData = d.daily_usage.some((x) => x.used > 0 || x.earned > 0)

  return (
    <div className="space-y-5">
      {/* ── Stats cards ── */}
      <div className="grid gap-4 sm:grid-cols-3">
        <div className="glass-panel rounded-2xl p-5 text-center relative overflow-hidden">
          <div className="absolute -top-3 -right-3 w-16 h-16 rounded-full bg-primary/10" />
          <div className="relative z-10 mb-2 flex items-center justify-center">
            <PointsCoin size={28} />
          </div>
          <div className="relative z-10 text-2xl font-bold gradient-text">{d.balance}</div>
          <p className="relative z-10 mt-0.5 text-xs text-muted-foreground">可用积分</p>
        </div>
        <div className="glass-panel rounded-2xl p-5 text-center relative overflow-hidden">
          <div className="absolute -top-3 -right-3 w-16 h-16 rounded-full bg-amber-500/10" />
          <div className="relative z-10 mb-2 flex items-center justify-center">
            <TrendingDown size={20} className="text-amber-500" />
          </div>
          <div className="relative z-10 text-2xl font-bold text-amber-600 dark:text-amber-400">{d.month_used}</div>
          <p className="relative z-10 mt-0.5 text-xs text-muted-foreground">本月已用</p>
        </div>
        <div className="glass-panel rounded-2xl p-5 text-center relative overflow-hidden">
          <div className="absolute -top-3 -right-3 w-16 h-16 rounded-full bg-emerald-500/10" />
          <div className="relative z-10 mb-2 flex items-center justify-center">
            <TrendingUp size={20} className="text-emerald-500" />
          </div>
          <div className="relative z-10 text-2xl font-bold text-emerald-600 dark:text-emerald-400">{d.total_earned}</div>
          <p className="relative z-10 mt-0.5 text-xs text-muted-foreground">累计获得</p>
        </div>
      </div>

      {/* ── Charts row ── */}
      <div className="grid gap-5 lg:grid-cols-[1fr_300px]">
        {/* 30-day usage area chart */}
        <div className="glass-panel rounded-2xl p-5">
          <div className="flex items-center gap-2 mb-4">
            <BarChart3 size={16} className="text-muted-foreground" />
            <h3 className="text-sm font-semibold text-foreground">近30天积分使用趋势</h3>
          </div>
          {hasData ? (
            <div className="h-[260px] -mx-2">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={d.daily_usage} margin={{ top: 4, right: 8, left: -8, bottom: 0 }}>
                  <defs>
                    <linearGradient id="colorUsed" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="#f59e0b" stopOpacity={0.35} />
                      <stop offset="100%" stopColor="#f59e0b" stopOpacity={0.02} />
                    </linearGradient>
                    <linearGradient id="colorEarned" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="#22c55e" stopOpacity={0.3} />
                      <stop offset="100%" stopColor="#22c55e" stopOpacity={0.02} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" opacity={0.4} />
                  <XAxis
                    dataKey="date"
                    tick={{ fontSize: 11, fill: 'var(--color-muted-foreground)' }}
                    tickLine={false}
                    axisLine={false}
                    interval="preserveStartEnd"
                  />
                  <YAxis
                    tick={{ fontSize: 11, fill: 'var(--color-muted-foreground)' }}
                    tickLine={false}
                    axisLine={false}
                    width={36}
                  />
                  <Tooltip
                    contentStyle={{
                      background: 'var(--color-card)',
                      border: '1px solid var(--color-border)',
                      borderRadius: '12px',
                      fontSize: '13px',
                      boxShadow: '0 8px 32px rgba(2,8,23,0.18)',
                    }}
                    labelStyle={{ fontWeight: 600, marginBottom: 4 }}
                  />
                  <Area
                    type="monotone"
                    dataKey="earned"
                    stroke="#22c55e"
                    strokeWidth={2}
                    fill="url(#colorEarned)"
                    name="获得"
                  />
                  <Area
                    type="monotone"
                    dataKey="used"
                    stroke="#f59e0b"
                    strokeWidth={2}
                    fill="url(#colorUsed)"
                    name="消耗"
                  />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          ) : (
            <div className="h-[240px] flex items-center justify-center text-muted-foreground">
              <div className="text-center">
                <BarChart3 size={40} className="mx-auto mb-3 opacity-30" />
                <p className="text-sm">暂无使用数据</p>
              </div>
            </div>
          )}
        </div>

        {/* Category donut chart */}
        <div className="glass-panel rounded-2xl p-5">
          <div className="flex items-center gap-2 mb-4">
            <PieChart size={16} className="text-muted-foreground" />
            <h3 className="text-sm font-semibold text-foreground">消耗分布</h3>
          </div>
          {d.categories.length > 0 ? (
            <>
              <div className="h-[200px] -mx-4">
                <ResponsiveContainer width="100%" height="100%">
                  <RePieChart>
                    <Pie
                      data={d.categories}
                      dataKey="total"
                      nameKey="type"
                      cx="50%"
                      cy="50%"
                      innerRadius={52}
                      outerRadius={82}
                      paddingAngle={3}
                      stroke="transparent"
                    >
                      {d.categories.map((entry, i) => (
                        <Cell
                          key={entry.type}
                          fill={CATEGORY_COLORS[entry.type] || CATEGORY_FALLBACK_COLORS[i % CATEGORY_FALLBACK_COLORS.length]}
                        />
                      ))}
                    </Pie>
                    <Tooltip
                      contentStyle={{
                        background: 'var(--color-card)',
                        border: '1px solid var(--color-border)',
                        borderRadius: '10px',
                        fontSize: '13px',
                      }}
                      formatter={((value: any, _name: any, _props: any) => [`${value} 积分`, POINTS_TYPE_LABELS[_name] || _name]) as any}
                    />
                  </RePieChart>
                </ResponsiveContainer>
              </div>
              <div className="flex flex-wrap gap-x-3 gap-y-1 mt-2 justify-center">
                {d.categories.map((entry, i) => (
                  <div key={entry.type} className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <span
                      className="w-2 h-2 rounded-full shrink-0"
                      style={{ backgroundColor: CATEGORY_COLORS[entry.type] || CATEGORY_FALLBACK_COLORS[i % CATEGORY_FALLBACK_COLORS.length] }}
                    />
                    {POINTS_TYPE_LABELS[entry.type] || entry.type}
                  </div>
                ))}
              </div>
            </>
          ) : (
            <div className="h-[240px] flex items-center justify-center text-muted-foreground">
              <div className="text-center">
                <PieChart size={40} className="mx-auto mb-3 opacity-30" />
                <p className="text-sm">暂无消耗记录</p>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* ── Transaction table ── */}
      <div className="glass-panel rounded-2xl overflow-hidden">
        <div className="flex items-center gap-2 px-6 pt-5 pb-3">
          <Clock size={16} className="text-muted-foreground" />
          <h3 className="text-sm font-semibold text-foreground">积分流水</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-xs text-muted-foreground">
                <th className="py-2.5 pl-6 pr-3 text-left font-medium">时间</th>
                <th className="py-2.5 px-3 text-left font-medium">类型</th>
                <th className="py-2.5 px-3 text-right font-medium">数量</th>
                <th className="py-2.5 px-3 text-right font-medium">余额</th>
                <th className="py-2.5 pl-3 pr-6 text-left font-medium hidden sm:table-cell">备注</th>
              </tr>
            </thead>
            <tbody>
              {records.length === 0 ? (
                <tr>
                  <td colSpan={5} className="py-16 text-center text-muted-foreground">
                    <Sparkles size={32} className="mx-auto mb-3 text-primary opacity-30" />
                    <p className="text-sm">暂无积分记录</p>
                    <p className="mt-1 text-xs">使用 AI 功能后将在此显示消耗明细</p>
                  </td>
                </tr>
              ) : (
                records.map((r) => {
                  const typeLabel = POINTS_TYPE_LABELS[r.type] || r.type
                  const isPositive = r.amount > 0
                  return (
                    <tr key={r.id} className="border-b border-border/50 last:border-0 hover:bg-surface-hover/50 transition-colors">
                      <td className="py-2.5 pl-6 pr-3 whitespace-nowrap text-muted-foreground text-xs">
                        {new Date(r.created_at).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}
                      </td>
                      <td className="py-2.5 px-3 whitespace-nowrap text-xs font-medium text-foreground">
                        {typeLabel}
                      </td>
                      <td className={`py-2.5 px-3 text-right whitespace-nowrap font-mono text-xs font-medium ${isPositive ? 'text-emerald-500' : 'text-amber-500'}`}>
                        {isPositive ? '+' : ''}{r.amount}
                      </td>
                      <td className="py-2.5 px-3 text-right whitespace-nowrap font-mono text-xs text-foreground">
                        {r.balance}
                      </td>
                      <td className="py-2.5 pl-3 pr-6 text-xs text-muted-foreground hidden sm:table-cell max-w-[180px] truncate">
                        {r.note || '-'}
                      </td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}

// ── Shop Panel (point packages) ──

const packages = [
  {
    name: '尝鲜包',
    points: 100,
    price: '¥1',
    period: '',
    highlight: false,
    badge: '',
    features: ['100 积分到账', '适合轻度体验', '有效期 2 年'],
  },
  {
    name: '进阶包',
    points: 600,
    price: '¥5',
    period: '',
    highlight: false,
    badge: '省 ¥1',
    features: ['600 积分到账', '送 100 积分 (省 ¥1)', '约 60 次 AI 对话', '有效期 2 年'],
  },
  {
    name: '专业包',
    points: 1200,
    price: '¥10',
    period: '',
    highlight: true,
    badge: '推荐 · 省 ¥2',
    features: ['1200 积分到账', '送 200 积分 (省 ¥2)', '约 120 次 AI 对话', '有效期 2 年'],
  },
  {
    name: '旗舰包',
    points: 3400,
    price: '¥29',
    period: '',
    highlight: false,
    premium: true,
    badge: '超值 · 省 ¥5',
    features: ['3400 积分到账', '送 500 积分 (省 ¥5)', '约 340 次 AI 对话', '有效期 2 年'],
  },
]

const memberships = [
  {
    name: 'Free',
    price: '¥0',
    period: '',
    desc: '免费开始，按需使用',
    features: ['所有功能开放', '注册即送 100 积分', 'AI 对话消耗积分', '带水印 PDF 导出 10 积分/次'],
    cta: '当前方案',
    active: true,
  },
  {
    name: 'Pro',
    price: '¥19',
    period: '/ 月',
    desc: '专业用户，无限导出',
    features: ['所有功能开放', '每月赠送 2,000 积分', '无限次无水印 PDF 导出', 'AI / 导出优先使用，无需排队', '积分充值享 9 折'],
    cta: '升级 Pro',
    active: false,
    highlight: true,
  },
]

function ShopPanel() {
  return (
    <div className="space-y-6">
      {/* ── Membership plans ── */}
      <div className="glass-panel rounded-2xl p-6">
        <h2 className="mb-1 text-lg font-semibold text-foreground">会员方案</h2>
        <p className="mb-6 text-sm text-muted-foreground">选择适合你的计划，随时升级</p>
        <div className="grid gap-5 sm:grid-cols-2">
          {memberships.map((m) => (
            <div
              key={m.name}
              className={`card-hover relative flex flex-col rounded-lg border px-6 py-7 transition-all duration-200 ${
                m.highlight
                  ? 'border-primary/40 ring-1 ring-primary/20 bg-primary/[0.03] shadow-[0_0_28px_color-mix(in_srgb,var(--color-primary),transparent_90%)]'
                  : 'border-border bg-card'
              }`}
            >
              <h3 className="font-sans font-semibold text-base text-foreground mb-1">{m.name}</h3>
              <p className="text-xs text-muted-foreground mb-3">{m.desc}</p>
              <div className="flex items-baseline gap-1 mb-5">
                <span className="font-serif font-semibold text-[2.25rem] leading-none text-foreground">{m.price}</span>
                {m.period && <span className="text-sm text-muted-foreground">{m.period}</span>}
              </div>
              <ul className="flex-1 space-y-2.5 mb-7">
                {m.features.map((f) => (
                  <li key={f} className="flex items-start gap-2 text-sm text-muted-foreground">
                    <svg className="w-4 h-4 mt-0.5 shrink-0 text-primary" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
                    {f}
                  </li>
                ))}
              </ul>
              <button
                disabled={m.active}
                className={`inline-flex items-center justify-center h-11 w-full rounded-lg text-sm font-medium transition-colors ${
                  m.active
                    ? 'border border-border bg-transparent text-muted-foreground cursor-default'
                    : m.highlight
                      ? 'bg-primary text-primary-foreground border border-primary/30 hover:brightness-110 active:brightness-90'
                      : 'border border-border bg-card text-foreground hover:bg-surface-hover'
                }`}
              >
                {m.cta}
              </button>
            </div>
          ))}
        </div>
      </div>

      {/* ── Point packages ── */}
      <div className="glass-panel rounded-2xl p-6">
        <h2 className="mb-1 text-lg font-semibold text-foreground">积分充值</h2>
        <p className="mb-6 text-sm text-muted-foreground">
          基准汇率 1 元 = 100 积分，大额套餐享额外赠送。购买后即时到账，有效期 2 年。
        </p>
        <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-4">
          {packages.map((pkg) => {
            let borderClass = 'border-border bg-card'
            if (pkg.highlight) {
              borderClass = 'border-primary/40 ring-1 ring-primary/20 bg-primary/[0.03] shadow-[0_0_28px_color-mix(in_srgb,var(--color-primary),transparent_90%)]'
            } else if ((pkg as any).premium) {
              borderClass = 'border-amber-400/50 ring-1 ring-amber-400/20 bg-amber-500/[0.03] shadow-[0_0_28px_color-mix(in_srgb,#f59e0b,transparent_92%)]'
            }
            return (
            <div
              key={pkg.name}
              className={`card-hover relative flex flex-col rounded-lg border px-6 py-7 transition-all duration-200 ${borderClass}`}
            >
              {pkg.badge && (
                <span className={`absolute -top-2.5 right-4 rounded-full px-3 py-0.5 text-xs font-semibold ${
                  pkg.badge.includes('推荐')
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-amber-500 text-white'
                }`}>
                  {pkg.badge}
                </span>
              )}
              <h3 className="font-sans font-semibold text-base text-foreground mb-2">{pkg.name}</h3>
              <div className="flex items-baseline gap-1 mb-5">
                <span className="font-serif font-semibold text-[2.25rem] leading-none text-foreground">{pkg.price}</span>
                {pkg.period && <span className="text-sm text-muted-foreground">{pkg.period}</span>}
              </div>
              <ul className="flex-1 space-y-2.5 mb-7">
                {pkg.features.map((f) => (
                  <li key={f} className="flex items-start gap-2 text-sm text-muted-foreground">
                    <svg className="w-4 h-4 mt-0.5 shrink-0 text-primary" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
                    {f}
                  </li>
                ))}
              </ul>
              <button
                className={`inline-flex items-center justify-center h-11 w-full rounded-lg text-sm font-medium transition-colors ${
                  pkg.highlight
                    ? 'bg-primary text-primary-foreground border border-primary/30 hover:brightness-110 active:brightness-90'
                    : 'border border-border bg-card text-foreground hover:bg-surface-hover'
                }`}
                onClick={() => alert(`购买 ${pkg.name}：${pkg.price} / ${pkg.points} 积分\n\n支付功能即将上线，敬请期待！`)}
              >
                立即购买
              </button>
            </div>
          )})}
        </div>
      </div>

      <div className="text-center text-xs text-muted-foreground space-y-1">
        <p>积分不可兑换会员、转赠或提现，充值后不支持退款。</p>
        <p>Pro 会员充值享 <span className="text-primary font-medium">9 折</span> 优惠。</p>
      </div>
    </div>
  )
}

// ── Password Panel ──

function PasswordPanel() {
  const [oldPwd, setOldPwd] = useState('')
  const [newPwd, setNewPwd] = useState('')
  const [confirmPwd, setConfirmPwd] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)

  const handleChange = async () => {
    if (!oldPwd || !newPwd || !confirmPwd) return
    if (newPwd !== confirmPwd) {
      setError('两次密码不一致')
      return
    }
    if (newPwd.length < 6) {
      setError('新密码需 6 位以上')
      return
    }
    try {
      setSaving(true)
      setError('')
      setSuccess(false)
      await authApi.changePassword(oldPwd, newPwd)
      setSuccess(true)
      setOldPwd('')
      setNewPwd('')
      setConfirmPwd('')
      setTimeout(() => setSuccess(false), 2000)
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '修改失败')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="glass-panel rounded-2xl p-6">
      <h2 className="mb-4 text-lg font-semibold text-foreground">密码修改</h2>
      <div className="space-y-4 max-w-md">
        <label className="relative block">
          <KeyRound className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="password"
            value={oldPwd}
            onChange={(e) => setOldPwd(e.target.value)}
            placeholder="原密码"
            className="pl-9"
          />
        </label>
        <label className="relative block">
          <LockKeyhole className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="password"
            value={newPwd}
            onChange={(e) => setNewPwd(e.target.value)}
            placeholder="新密码（至少 6 位）"
            className="pl-9"
          />
        </label>
        <label className="relative block">
          <ShieldCheck className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="password"
            value={confirmPwd}
            onChange={(e) => setConfirmPwd(e.target.value)}
            placeholder="确认新密码"
            className="pl-9"
          />
        </label>

        {error && <Alert>{error}</Alert>}
        {success && (
          <div className="rounded-lg border border-green-300/40 bg-green-50/60 px-4 py-2.5 text-sm text-green-800 dark:border-green-500/30 dark:bg-green-950/40 dark:text-green-300">
            密码修改成功
          </div>
        )}

        <Button
          onClick={handleChange}
          disabled={saving || !oldPwd || !newPwd || !confirmPwd}
        >
          {saving ? '修改中...' : '修改密码'}
        </Button>
      </div>
    </div>
  )
}
