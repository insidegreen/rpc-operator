import { useEffect, useState } from 'react'
import { listClusters } from '../api'
import type { PipelineCluster } from '../types'

interface Props {
  namespace: string
  onViewDetail: (name: string) => void
}

export function ClusterList({ namespace, onViewDetail }: Props) {
  const [clusters, setClusters] = useState<PipelineCluster[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string>()

  useEffect(() => {
    let cancelled = false
    function load() {
      listClusters(namespace)
        .then(items => { if (!cancelled) { setClusters(items); setError(undefined) } })
        .catch(e => {
          if (!cancelled) {
            if ((e as { status?: number }).status === 403) {
              setClusters([])
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

  if (loading) return <p style={{ color: '#888' }}>Loading clusters…</p>
  if (error)   return <p style={{ color: 'red' }}>Error: {error}</p>

  return (
    <div>
      <h2 style={{ margin: '0 0 16px', fontSize: 18 }}>Clusters — {namespace}</h2>
      {clusters.length === 0 ? (
        <p style={{ color: '#888' }}>No clusters in this namespace.</p>
      ) : (
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 14 }}>
          <thead>
            <tr style={{ background: '#f5f5f5', textAlign: 'left' }}>
              <th style={thStyle}>Name</th>
              <th style={thStyle}>Phase</th>
              <th style={thStyle}>Ready</th>
              <th style={thStyle}>Age</th>
            </tr>
          </thead>
          <tbody>
            {clusters.map(c => (
              <tr
                key={c.metadata.name}
                onClick={() => onViewDetail(c.metadata.name)}
                style={{ cursor: 'pointer', borderBottom: '1px solid #eee' }}
                onMouseEnter={e => (e.currentTarget.style.background = '#f9f9ff')}
                onMouseLeave={e => (e.currentTarget.style.background = '')}
              >
                <td style={tdStyle}><strong>{c.metadata.name}</strong></td>
                <td style={tdStyle}><PhaseBadge phase={c.status?.phase} /></td>
                <td style={{ ...tdStyle, color: '#666' }}>
                  {(c.status?.readyReplicas ?? 0)}/{c.spec.replicas ?? 0}
                </td>
                <td style={{ ...tdStyle, color: '#666', fontSize: 12 }}>{age(c.metadata.creationTimestamp)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

function PhaseBadge({ phase }: { phase?: string }) {
  const colors: Record<string, { bg: string; text: string }> = {
    Ready:    { bg: '#dcfce7', text: '#16a34a' },
    Pending:  { bg: '#fef9c3', text: '#d97706' },
    Degraded: { bg: '#fee2e2', text: '#dc2626' },
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
