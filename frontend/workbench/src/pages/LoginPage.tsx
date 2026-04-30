import { useState } from 'react'
import { ApiError, authApi, type User } from '@/lib/api-client'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Alert } from '@/components/ui/alert'

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
          <Input
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="用户名"
            className="bg-background px-3"
          />
          <Input
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            type="password"
            placeholder="密码（至少 6 位）"
            className="bg-background px-3"
          />

          {error && (
            <Alert>{error}</Alert>
          )}

          <Button
            type="submit"
            form="login-form"
            size="lg"
            className="w-full"
            disabled={loading || !username.trim() || !password}
          >
            {loading ? '登录中...' : '登录'}
          </Button>
        </form>
      </div>
    </div>
  )
}
