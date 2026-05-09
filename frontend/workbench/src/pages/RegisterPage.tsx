import { useState } from 'react'
import type { CSSProperties } from 'react'
import { useNavigate } from 'react-router-dom'
import { FileText, LockKeyhole, Mail, ShieldCheck, Sparkles, UserRound } from 'lucide-react'
import { ApiError, authApi } from '@/lib/api-client'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Alert } from '@/components/ui/alert'
import { ThemeSwitcher } from '@/components/ui/theme-switcher'
import { AnimatedBrandTitle } from '@/components/ui/animated-brand-title'

type Step = 'form' | 'verify'

export default function RegisterPage() {
  const navigate = useNavigate()

  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [code, setCode] = useState('')
  const [step, setStep] = useState<Step>('form')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [devCode, setDevCode] = useState('')
  const [codeSent, setCodeSent] = useState(false)

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!username.trim() || !email.trim() || !password || !confirmPassword) return

    if (password !== confirmPassword) {
      setError('两次密码不一致')
      return
    }

    try {
      setLoading(true)
      setError('')
      const res = await authApi.register(username.trim(), password, email.trim())
      if ((res as any).dev_code) {
        setDevCode((res as any).dev_code)
      }
      setCodeSent(true)
      setStep('verify')
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '注册失败')
    } finally {
      setLoading(false)
    }
  }

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!code.trim()) return

    try {
      setLoading(true)
      setError('')
      await authApi.verifyEmail(email.trim(), code.trim())
      navigate('/login?registered=true')
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '验证失败')
    } finally {
      setLoading(false)
    }
  }

  const handleResendCode = async () => {
    try {
      setLoading(true)
      setError('')
      await authApi.sendCode(email.trim())
      setCodeSent(false)
      setTimeout(() => setCodeSent(true), 200)
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '发送失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="app-shell min-h-screen px-5">
      <div className="relative z-10 mx-auto flex min-h-screen max-w-6xl items-center py-10">
        <div className="grid w-full gap-8 lg:grid-cols-[1.1fr_420px] lg:items-center">

          {/* ── Left brand section ── */}
          <section className="stagger-in hidden lg:block">
            <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-border bg-card/60 px-3 py-1 text-xs font-medium text-muted-foreground backdrop-blur-xl">
              <Sparkles size={14} className="text-primary" />
              Create your account
            </div>
            <div className="group">
              <h1 className="gradient-text max-w-2xl text-6xl font-semibold tracking-tight cursor-default select-none transition-all duration-500 group-hover:animate-[brand-title-wobble_680ms_cubic-bezier(0.34,1.56,0.64,1)] group-hover:[filter:drop-shadow(0_0_18px_color-mix(in_srgb,var(--color-primary),transparent_58%))]">
                创建账号，开始你的简历之旅。
              </h1>
            </div>
            <p className="mt-6 max-w-xl text-sm leading-7 text-muted-foreground">
              注册后将收到邮箱验证码，验证完成后即可登录工作台，体验 AI 辅助简历编辑。
            </p>
            <div className="mt-10 grid max-w-xl grid-cols-3 gap-3">
              {[
                { label: '安全注册', icon: ShieldCheck },
                { label: '邮箱验证', icon: Mail },
                { label: '即刻使用', icon: FileText },
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

          {/* ── Right form panel ── */}
          <div className="glass-panel stagger-in w-full min-w-0 rounded-3xl p-6">
            <div className="mb-6 flex items-start justify-between gap-3">
              <div className="min-w-0">
                <AnimatedBrandTitle className="text-2xl font-semibold" />
                <p className="mt-2 text-sm text-muted-foreground">
                  {step === 'form' ? '创建你的账号' : '验证你的邮箱'}
                </p>
              </div>
              <ThemeSwitcher compact className="shrink-0" />
            </div>

            {step === 'form' && (
              <form key="form" id="register-form" onSubmit={handleRegister} className="space-y-3 stagger-in">
                <label className="relative block">
                  <UserRound className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    placeholder="用户名（3-64 个字符）"
                    className="pl-9"
                  />
                </label>
                <label className="relative block">
                  <Mail className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    type="email"
                    placeholder="邮箱地址"
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
                <label className="relative block">
                  <ShieldCheck className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    type="password"
                    placeholder="确认密码"
                    className="pl-9"
                  />
                </label>

                {error && <Alert>{error}</Alert>}

                <Button
                  type="submit"
                  form="register-form"
                  size="lg"
                  className="mt-2 w-full"
                  disabled={loading || !username.trim() || !email.trim() || !password || !confirmPassword}
                >
                  {loading ? '注册中...' : '注册'}
                </Button>
              </form>
            )}

            {step === 'verify' && (
              <form key="verify" id="verify-form" onSubmit={handleVerify} className="space-y-3 stagger-in">
                <div className="rounded-lg border border-border bg-card/50 px-4 py-3 text-sm text-muted-foreground">
                  验证码已发送至 <span className="font-medium text-foreground">{email}</span>，
                  请查收邮件并输入 6 位验证码。
                </div>

                {devCode && (
                  <div className="rounded-lg border border-amber-300/40 bg-amber-50/60 px-4 py-3 text-sm text-amber-800 dark:border-amber-500/30 dark:bg-amber-950/40 dark:text-amber-300">
                    <span className="font-medium">开发模式</span> — 验证码：
                    <span className="ml-2 text-lg font-bold tracking-[0.3em]">{devCode}</span>
                  </div>
                )}

                <label className="relative block">
                  <ShieldCheck className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    value={code}
                    onChange={(e) => setCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                    placeholder="6 位验证码"
                    className="pl-9 text-center text-lg tracking-[0.5em]"
                    maxLength={6}
                  />
                </label>

                {error && <Alert>{error}</Alert>}

                <Button
                  type="submit"
                  form="verify-form"
                  size="lg"
                  className="mt-2 w-full"
                  disabled={loading || code.length !== 6}
                >
                  {loading ? '验证中...' : '验证邮箱'}
                </Button>

                <button
                  type="button"
                  onClick={handleResendCode}
                  disabled={loading}
                  className="mt-3 w-full text-center text-sm text-muted-foreground hover:text-primary transition-colors disabled:opacity-50"
                >
                  {codeSent ? '验证码已重新发送' : '重新发送验证码'}
                </button>
              </form>
            )}

            <div className="mt-5 text-center text-sm text-muted-foreground">
              已有账号？
              <button
                onClick={() => navigate('/login')}
                className="ml-1 text-primary hover:underline"
              >
                登录
              </button>
            </div>
          </div>

        </div>
      </div>
    </div>
  )
}
