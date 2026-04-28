import { useState } from 'react'
import { ApiError, authApi, type User } from '@/lib/api-client'

interface LoginPageProps {
  onSuccess: (user: User) => void
}

export default function LoginPage({ onSuccess }: LoginPageProps) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

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
    <div className="min-h-screen bg-background flex items-center justify-center px-6">
      <div className="w-full max-w-sm rounded-xl border border-border bg-card p-6">
        <h1 className="font-serif text-2xl font-semibold text-foreground mb-2">ResumeGenius</h1>
        <p className="text-sm text-muted-foreground mb-6">输入用户名和密码后继续</p>

        <form id="login-form" onSubmit={handleSubmit} className="space-y-3">
          <input
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="用户名"
            className="w-full h-10 px-3 text-sm rounded-lg border border-border bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
          />
          <input
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            type="password"
            placeholder="密码（至少 6 位）"
            className="w-full h-10 px-3 text-sm rounded-lg border border-border bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
          />

          {error && (
            <div className="px-3 py-2 text-sm rounded-lg bg-destructive/10 text-destructive border border-destructive/20">
              {error}
            </div>
          )}

          <button
            type="submit"
            form="login-form"
            disabled={loading || !username.trim() || !password}
            className="w-full h-10 text-sm font-medium rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors"
          >
            {loading ? '登录中...' : '登录'}
          </button>
        </form>
      </div>
    </div>
  )
}
