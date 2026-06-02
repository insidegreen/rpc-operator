import { useEffect, useState } from 'react'
import { listProjects } from '../api'
import type { PipelineProject } from '../types'

interface Props {
  namespace: string
  onViewDetail: (name: string) => void
  /** Hidden in Mode C (read-only) when undefined. */
  onNew?: () => void
}

export function ProjectList({ namespace, onViewDetail, onNew }: Props) {
  const [projects, setProjects] = useState<PipelineProject[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string>()

  useEffect(() => {
    let cancelled = false
    function load() {
      listProjects(namespace)
        .then(items => { if (!cancelled) { setProjects(items); setError(undefined) } })
        .catch(e => {
          if (!cancelled) {
            if ((e as { status?: number }).status === 403) {
              setProjects([])
              setError(undefined)
            } else {
              setError((e as Error).message)
            }
          }
        })
        .finally(() => { if (!cancelled) setLoading(false) })
    }
    load()
    const id = setInterval(load, 15_000)
    return () => { cancelled = true; clearInterval(id) }
  }, [namespace])

  if (loading) return <p style={{ color: '#888' }}>Loading projects…</p>
  if (error)   return <p style={{ color: 'red' }}>Error: {error}</p>

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ margin: 0, fontSize: 18 }}>Projects — {namespace}</h2>
        {onNew && (
          <button onClick={onNew} style={newBtnStyle}>+ New Project</button>
        )}
      </div>
      {projects.length === 0 ? (
        <p style={{ color: '#888' }}>No projects in this namespace.</p>
      ) : (
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 14 }}>
          <thead>
            <tr style={{ background: '#f5f5f5', textAlign: 'left' }}>
              <th style={thStyle}>Name</th>
              <th style={thStyle}>Phase</th>
              <th style={thStyle}>Routes</th>
              <th style={thStyle}>Cluster</th>
              <th style={thStyle}>NATS</th>
              <th style={thStyle}>Age</th>
            </tr>
          </thead>
          <tbody>
            {projects.map(p => (
              <tr
                key={p.metadata.name}
                onClick={() => onViewDetail(p.metadata.name)}
                style={{ cursor: 'pointer', borderBottom: '1px solid #eee' }}
                onMouseEnter={e => (e.currentTarget.style.background = '#f9f9ff')}
                onMouseLeave={e => (e.currentTarget.style.background = '')}
              >
                <td style={tdStyle}><strong>{p.metadata.name}</strong></td>
                <td style={tdStyle}><PhaseBadge phase={p.status?.phase} /></td>
                <td style={{ ...tdStyle, color: '#666' }}>{p.spec.routes?.length ?? 0}</td>
                <td style={{ ...tdStyle, color: '#666' }}>{health(p.status?.cluster)}</td>
                <td style={{ ...tdStyle, color: '#666' }}>{health(p.status?.nats)}</td>
                <td style={{ ...tdStyle, color: '#666', fontSize: 12 }}>{age(p.metadata.creationTimestamp)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

function health(c?: { ready?: number; total?: number }): string {
  if (!c) return '—'
  return `${c.ready ?? 0}/${c.total ?? 0}`
}

function PhaseBadge({ phase }: { phase?: string }) {
  const colors: Record<string, { bg: string; text: string }> = {
    Ready:        { bg: '#dcfce7', text: '#16a34a' },
    Provisioning: { bg: '#fef9c3', text: '#d97706' },
    Degraded:     { bg: '#fee2e2', text: '#dc2626' },
    Deleting:     { bg: '#e5e7eb', text: '#6b7280' },
  }
  const c = colors[phase ?? ''] ?? { bg: '#f3f4f6', text: '#6b7280' }
  return (
    <span style={{ background: c.bg, color: c.text, padding: '2px 8px',
                   borderRadius: 12, fontSize: 12, fontWeight: 600 }}>
      {phase ?? 'Unknown'}
    </span>
  )
}

function age(ts?: string): string {
  if (!ts) return '—'
  return new Date(ts).toLocaleString('en-US', { dateStyle: 'short', timeStyle: 'short' })
}

const thStyle: React.CSSProperties = { padding: '8px 12px', fontWeight: 600, fontSize: 13 }
const tdStyle: React.CSSProperties = { padding: '10px 12px', verticalAlign: 'middle' }
const newBtnStyle: React.CSSProperties = {
  marginLeft: 'auto', padding: '6px 12px', fontSize: 13, background: '#1d4ed8',
  color: '#fff', border: 'none', borderRadius: 6, cursor: 'pointer',
}
