export type ThemePreset = {
  id: string
  label: string
  mode: 'light' | 'dark'
  palette: 'futuristic-apple' | 'warm-editorial' | 'quiet-luxury' | 'professional-blue'
}

export const THEME_PRESETS: ThemePreset[] = [
  { id: 'futuristic-dark', label: '科技夜', mode: 'dark', palette: 'futuristic-apple' },
  { id: 'futuristic-light', label: '银白', mode: 'light', palette: 'futuristic-apple' },
  { id: 'warm-editorial', label: '暖驼', mode: 'light', palette: 'warm-editorial' },
  { id: 'quiet-luxury', label: '静奢', mode: 'dark', palette: 'quiet-luxury' },
  { id: 'professional-blue', label: '企业蓝', mode: 'light', palette: 'professional-blue' },
]

export const THEME_STORAGE_KEY = 'resume-genius-theme-preset'
const THEME_COOKIE_KEY = 'resume_theme_preset'

function readThemeCookie() {
  if (typeof document === 'undefined') return null
  const entry = document.cookie
    .split('; ')
    .find((item) => item.startsWith(`${THEME_COOKIE_KEY}=`))
  return entry ? decodeURIComponent(entry.split('=').slice(1).join('=')) : null
}

function writeThemeCookie(id: string) {
  if (typeof document === 'undefined') return
  document.cookie = `${THEME_COOKIE_KEY}=${encodeURIComponent(id)}; path=/; max-age=31536000; SameSite=Lax`
}

export function getPresetById(id: string | null) {
  return THEME_PRESETS.find((preset) => preset.id === id) ?? THEME_PRESETS[0]
}

export function applyPreset(preset: ThemePreset) {
  document.documentElement.dataset.theme = preset.mode
  document.documentElement.dataset.palette = preset.palette
  localStorage.setItem(THEME_STORAGE_KEY, preset.id)
  writeThemeCookie(preset.id)
}

export function getInitialPreset() {
  if (typeof window === 'undefined') return THEME_PRESETS[0]
  const saved = localStorage.getItem(THEME_STORAGE_KEY) ?? readThemeCookie()
  return getPresetById(saved)
}
