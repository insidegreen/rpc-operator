import { useState } from 'react'
import type { PipelineProjectSpec } from '../types'

interface Props {
  onCreate: (name: string, spec: PipelineProjectSpec) => Promise<void>
  onClose: () => void
}

export function ProjectForm({ onCreate, onClose }: Props) {
  const [name, setName] = useState('')
  const [instances, setInstances] = useState('')
  const [storage, setStorage] = useState('')
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)

  async function handleCreate() {
    if (!/^[a-z]([-a-z0-9]*[a-z0-9])?$/.test(name)) {
      setError('Project name must be a DNS-1123 label (lower-case, digits, dashes).'); return
    }
    const spec: PipelineProjectSpec = {}
    if (instances.trim()) {
      const n = Number(instances)
      if (!Number.isInteger(n) || n < 1) { setError('Cluster instances must be a positive integer.'); return }
      spec.cluster = { instances: n }
    }
    if (storage.trim()) spec.nats = { storage: storage.trim() }

    setBusy(true)
    try {
      await onCreate(name, spec)
    } catch (e) {
      setError((e as Error).message)
      setBusy(false)
    }
  }

  return (
    <div style={overlayStyle} onClick={onClose}>
      <div style={dialogStyle} onClick={e => e.stopPropagation()}>
        <h3 style={{ margin: '0 0 16px', fontSize: 16 }}>New Project</h3>
        <label style={labelStyle}>
          Project name
          <input value={name} onChange={e => setName(e.target.value)} style={inputStyle} placeholder="orders" />
        </label>
        <label style={labelStyle}>
          Cluster instances (optional, default 1)
          <input value={instances} onChange={e => setInstances(e.target.value)} style={inputStyle} placeholder="1" />
        </label>
        <label style={labelStyle}>
          NATS storage (optional, default 10Gi)
          <input value={storage} onChange={e => setStorage(e.target.value)} style={inputStyle} placeholder="10Gi" />
        </label>
        {error && <div style={{ color: '#dc2626', fontSize: 13, marginTop: 12 }}>{error}</div>}
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, marginTop: 20 }}>
          <button onClick={onClose} style={cancelBtnStyle}>Cancel</button>
          <button onClick={handleCreate} disabled={busy} style={createBtnStyle}>
            {busy ? 'Creating…' : 'Create project'}
          </button>
        </div>
      </div>
    </div>
  )
}

const overlayStyle: React.CSSProperties = {
  position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.25)', zIndex: 50,
  display: 'flex', alignItems: 'center', justifyContent: 'center',
}
const dialogStyle: React.CSSProperties = {
  width: 420, maxWidth: '90vw', background: '#fff', borderRadius: 8, padding: 24,
  boxShadow: '0 8px 24px rgba(0,0,0,0.15)',
}
const labelStyle: React.CSSProperties = {
  display: 'flex', flexDirection: 'column', gap: 4, fontSize: 13, color: '#444', marginTop: 10,
}
const inputStyle: React.CSSProperties = {
  padding: '6px 8px', border: '1px solid #ccc', borderRadius: 4, fontSize: 13,
}
const createBtnStyle: React.CSSProperties = {
  padding: '7px 14px', fontSize: 13, background: '#1d4ed8', color: '#fff',
  border: 'none', borderRadius: 6, cursor: 'pointer',
}
const cancelBtnStyle: React.CSSProperties = {
  padding: '7px 14px', fontSize: 13, background: '#fff', color: '#444',
  border: '1px solid #ccc', borderRadius: 6, cursor: 'pointer',
}
