import { useState, useEffect } from 'react'
import { intakeApi, type SSHKey } from '@/lib/api-client'

interface SSHKeySelectorProps {
  value: number | null
  onChange: (keyId: number | null) => void
}

export default function SSHKeySelector({ value, onChange }: SSHKeySelectorProps) {
  const [keys, setKeys] = useState<SSHKey[]>([])
  const [showNew, setShowNew] = useState(false)
  const [alias, setAlias] = useState('')
  const [privateKey, setPrivateKey] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    intakeApi.listSSHKeys().then(setKeys).catch(() => {})
  }, [])

  const handleCreate = async () => {
    if (!alias.trim() || !privateKey.trim()) return
    try {
      setLoading(true)
      setError('')
      const newKey = await intakeApi.createSSHKey(alias.trim(), privateKey.trim())
      setKeys(prev => [newKey, ...prev])
      onChange(newKey.id)
      setShowNew(false)
      setAlias('')
      setPrivateKey('')
    } catch (e) {
      setError(e instanceof Error ? e.message : '创建失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-2">
      <label className="text-sm font-medium">SSH 密钥</label>

      {!showNew ? (
        <div className="flex gap-2">
          <select
            className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm"
            value={value ?? ''}
            onChange={(e) => onChange(e.target.value ? Number(e.target.value) : null)}
          >
            <option value="">选择已有密钥...</option>
            {keys.map(k => (
              <option key={k.id} value={k.id}>{k.alias}</option>
            ))}
          </select>
          <button
            type="button"
            className="text-sm text-primary hover:underline whitespace-nowrap"
            onClick={() => setShowNew(true)}
          >
            上传新密钥
          </button>
        </div>
      ) : (
        <div className="space-y-2 border rounded-md p-3">
          <input
            type="text"
            placeholder="密钥别名（如：github-deploy-key）"
            value={alias}
            onChange={(e) => setAlias(e.target.value)}
            className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
          />
          <textarea
            placeholder="粘贴 SSH 私钥内容（-----BEGIN OPENSSH PRIVATE KEY----- ...）"
            value={privateKey}
            onChange={(e) => setPrivateKey(e.target.value)}
            className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm min-h-[100px] font-mono text-xs"
          />
          <p className="text-xs text-amber-600">
            安全提示：请使用专用只读密钥，切勿使用生产环境密钥或有写权限的密钥。
          </p>
          <div className="flex gap-2">
            <button
              type="button"
              className="text-sm text-muted-foreground hover:underline"
              onClick={() => { setShowNew(false); setAlias(''); setPrivateKey('') }}
            >
              取消
            </button>
            <button
              type="button"
              className="text-sm bg-primary text-primary-foreground px-3 py-1 rounded-md disabled:opacity-50"
              onClick={handleCreate}
              disabled={!alias.trim() || !privateKey.trim() || loading}
            >
              {loading ? '保存中...' : '保存密钥'}
            </button>
          </div>
          {error && <p className="text-xs text-destructive">{error}</p>}
        </div>
      )}
    </div>
  )
}
