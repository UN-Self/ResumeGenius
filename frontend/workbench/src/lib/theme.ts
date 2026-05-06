export type ThemePreset = {
  id: string
  label: string
  mode: 'light' | 'dark'
  palette: 'futuristic-apple' | 'warm-editorial' | 'quiet-luxury'
}

export const THEME_PRESETS: ThemePreset[] = [
  { id: 'futuristic-dark', label: '月光', mode: 'dark', palette: 'futuristic-apple' },
  { id: 'futuristic-light', label: '银白', mode: 'light', palette: 'futuristic-apple' },
  { id: 'warm-editorial', label: '暖驼', mode: 'light', palette: 'warm-editorial' },
  { id: 'quiet-luxury', label: '静奢', mode: 'dark', palette: 'quiet-luxury' },
]

export const SYSTEM_THEME_ID = 'system'
export const THEME_CHOICES = [
  { id: SYSTEM_THEME_ID, label: '跟随系统' },
  ...THEME_PRESETS,
]

export const THEME_STORAGE_KEY = 'resume-genius-theme-preset'
export const THEME_MANUAL_STORAGE_KEY = 'resume-genius-theme-manual'
const THEME_COOKIE_KEY = 'resume_theme_preset'
const THEME_MANUAL_COOKIE_KEY = 'resume_theme_manual'
const SYSTEM_DARK_PRESET_ID = 'futuristic-dark'
const SYSTEM_LIGHT_PRESET_ID = 'futuristic-light'

function readThemeCookie(key: string) {
  if (typeof document === 'undefined') return null
  const entry = document.cookie
    .split('; ')
    .find((item) => item.startsWith(`${key}=`))
  return entry ? decodeURIComponent(entry.split('=').slice(1).join('=')) : null
}

function writeThemeCookie(key: string, value: string) {
  if (typeof document === 'undefined') return
  document.cookie = `${key}=${encodeURIComponent(value)}; path=/; max-age=31536000; SameSite=Lax`
}

function findPresetById(id: string | null) {
  return THEME_PRESETS.find((preset) => preset.id === id) ?? null
}

function normalizePresetId(id: string | null) {
  if (!id) return null
  return findPresetById(id)?.id ?? null
}

export function hasStoredPreset() {
  if (typeof window === 'undefined') return false
  return localStorage.getItem(THEME_MANUAL_STORAGE_KEY) === 'true'
    || readThemeCookie(THEME_MANUAL_COOKIE_KEY) === 'true'
}

export function getSystemPreset() {
  const systemPresetId = typeof window !== 'undefined'
    && window.matchMedia?.('(prefers-color-scheme: dark)').matches
    ? SYSTEM_DARK_PRESET_ID
    : SYSTEM_LIGHT_PRESET_ID
  return findPresetById(systemPresetId) ?? THEME_PRESETS[0]
}

export function getStoredPresetId() {
  if (typeof window === 'undefined' || !hasStoredPreset()) return null
  return normalizePresetId(localStorage.getItem(THEME_STORAGE_KEY) ?? readThemeCookie(THEME_COOKIE_KEY))
}

export function getInitialThemeChoiceId() {
  if (typeof window === 'undefined' || !hasStoredPreset()) return SYSTEM_THEME_ID
  return getStoredPresetId() ?? SYSTEM_THEME_ID
}

export function getPresetById(id: string | null) {
  return findPresetById(normalizePresetId(id)) ?? getSystemPreset()
}

export function applyPreset(preset: ThemePreset, options: { persist?: boolean } = {}) {
  const { persist = true } = options
  document.documentElement.dataset.theme = preset.mode
  document.documentElement.dataset.palette = preset.palette
  if (!persist) return
  localStorage.setItem(THEME_STORAGE_KEY, preset.id)
  localStorage.setItem(THEME_MANUAL_STORAGE_KEY, 'true')
  writeThemeCookie(THEME_COOKIE_KEY, preset.id)
  writeThemeCookie(THEME_MANUAL_COOKIE_KEY, 'true')
}

export function applyThemeChoice(id: string) {
  if (id === SYSTEM_THEME_ID) {
    applyPreset(getSystemPreset(), { persist: false })
    localStorage.setItem(THEME_STORAGE_KEY, SYSTEM_THEME_ID)
    localStorage.setItem(THEME_MANUAL_STORAGE_KEY, 'false')
    writeThemeCookie(THEME_COOKIE_KEY, SYSTEM_THEME_ID)
    writeThemeCookie(THEME_MANUAL_COOKIE_KEY, 'false')
    return
  }

  applyPreset(getPresetById(id))
}

export function getInitialPreset() {
  if (typeof window === 'undefined') return getSystemPreset()
  const saved = getStoredPresetId()
  return saved ? getPresetById(saved) : getSystemPreset()
}
