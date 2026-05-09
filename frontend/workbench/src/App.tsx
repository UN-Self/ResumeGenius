import { useEffect, useState } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import ProjectList from '@/pages/ProjectList'
import ProjectDetail from '@/pages/ProjectDetail'
import LoginPage from '@/pages/LoginPage'
import RegisterPage from '@/pages/RegisterPage'
import EditorPage from '@/pages/EditorPage'
import { authApi } from '@/lib/api-client'
import { FullPageState } from '@/components/ui/full-page-state'
import { GridRippleCanvas } from '@/components/ui/GridRippleCanvas'
import { applyPreset, getInitialPreset, hasStoredPreset, THEME_MANUAL_STORAGE_KEY, THEME_STORAGE_KEY } from '@/lib/theme'

type AuthState = 'checking' | 'authed' | 'guest'

export default function App() {
  const [authState, setAuthState] = useState<AuthState>('checking')

  useEffect(() => {
    const syncTheme = () => {
      applyPreset(getInitialPreset(), { persist: false })
    }

    syncTheme()

    const handleStorage = (event: StorageEvent) => {
      if (event.key !== THEME_STORAGE_KEY && event.key !== THEME_MANUAL_STORAGE_KEY) return
      syncTheme()
    }
    const handleSystemThemeChange = () => {
      if (hasStoredPreset()) return
      syncTheme()
    }
    const media = window.matchMedia?.('(prefers-color-scheme: dark)')

    window.addEventListener('storage', handleStorage)
    media?.addEventListener('change', handleSystemThemeChange)
    return () => {
      window.removeEventListener('storage', handleStorage)
      media?.removeEventListener('change', handleSystemThemeChange)
    }
  }, [])

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
    <BrowserRouter basename="/app">
      <GridRippleCanvas />
      <Routes>
        <Route
          path="/login"
          element={authState === 'authed'
            ? <Navigate to="/" replace />
            : <LoginPage onSuccess={() => { setAuthState('authed') }} />}
        />
        <Route
          path="/register"
          element={authState === 'authed'
            ? <Navigate to="/" replace />
            : <RegisterPage />}
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
