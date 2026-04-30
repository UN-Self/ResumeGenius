import { useEffect, useState } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import ProjectList from '@/pages/ProjectList'
import ProjectDetail from '@/pages/ProjectDetail'
import LoginPage from '@/pages/LoginPage'
import EditorPage from '@/pages/EditorPage'
import { authApi } from '@/lib/api-client'
import { FullPageState } from '@/components/ui/full-page-state'

type AuthState = 'checking' | 'authed' | 'guest'

export default function App() {
  const [authState, setAuthState] = useState<AuthState>('checking')

  useEffect(() => {
    let cancelled = false
    authApi.me()
      .then(() => {
        if (cancelled) return
        setAuthState('authed')
      })
      .catch(() => {
        if (cancelled) return
        setAuthState('guest')
      })

    return () => {
      cancelled = true
    }
  }, [])

  if (authState === 'checking') {
    return <FullPageState variant="loading" className="min-h-screen" />
  }

  return (
    <BrowserRouter>
      <Routes>
        <Route
          path="/login"
          element={authState === 'authed'
            ? <Navigate to="/" replace />
            : <LoginPage onSuccess={() => { setAuthState('authed') }} />}
        />
        <Route
          path="/"
          element={authState === 'authed' ? <ProjectList /> : <Navigate to="/login" replace />}
        />
        <Route
          path="/projects/:projectId"
          element={authState === 'authed' ? <ProjectDetail /> : <Navigate to="/login" replace />}
        />
        <Route
          path="/projects/:projectId/edit"
          element={authState === 'authed' ? <EditorPage /> : <Navigate to="/login" replace />}
        />
        <Route path="*" element={<Navigate to={authState === 'authed' ? '/' : '/login'} replace />} />
      </Routes>
    </BrowserRouter>
  )
}
