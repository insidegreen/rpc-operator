import React, { useEffect, useState } from 'react'
import { listPipelines, deletePipeline } from '../api'
import type { Pipeline } from '../types'

interface Props {
  namespace: string
  onEdit: (pipeline: Pipeline) => void
  onNew: () => void
}

export function PipelineList({ namespace, onEdit, onNew }: Props) {
  const [pipelines, setPipelines] = useState<Pipeline[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string>()
  const [expanded, setExpanded] = useState<Set<string>>(new Set())

  function load() {
    listPipelines(namespace)
      .then(items => { setPipelines(items); setError(undefined) })
      .catch(e => setError((e as Error).message))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load()
    const id = setInterval(load, 10_000)
    return () => clearInterval(id)
  }, [namespace])

  function toggleExpand(name: string, e: React.MouseEvent) {
    e.stopPropagation()
    setExpanded(prev => {
      const next = new Set(prev)
      next.has(name) ? next.delete(name) : next.add(name)
      return next
    })
  }

  async function handleDelete(p: Pipeline, e: React.MouseEvent) {
    e.stopPropagation()
    if (!confirm(`Pipeline "${p.metadata.name}" löschen?`)) return
    await deletePipeline(p.metadata.namespace, p.metadata.name).catch(console.error)
    load()
  }

  if (loading) return <p style={{ color: '#888' }}>Lade Pipelines…</p>
  if (error)   return <p style={{ color: 'red' }}>Fehler: {error}</p>

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ margin: 0, fontSize: 18 }}>Pipelines — {namespace}</h2>
        <button onClick={onNew} style={newBtnStyle}>+ Neue Pipeline</button>
      </div>
      {pipelines.length === 0 ? (
        <p style={{ color: '#888' }}>Keine Pipelines in diesem Namespace.</p>
      ) : (
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 14 }}>
          <thead>
            <tr style={{ background: '#f5f5f5', textAlign: 'left' }}>
              <th style={thStyle}>Name</th>
              <th style={thStyle}>Status</th>
              <th style={thStyle}>Pod</th>
              <th style={thStyle}>Letztes Update</th>
              <th style={thStyle}></th>
              <th style={thStyle}></th>
            </tr>
          </thead>
          <tbody>
            {pipelines.map(p => (
              <React.Fragment key={p.metadata.name}>
                <tr
                  onClick={() => onEdit(p)}
                  style={{ cursor: 'pointer', borderBottom: expanded.has(p.metadata.name) ? 'none' : '1px solid #eee' }}
                  onMouseEnter={e => (e.currentTarget.style.background = '#f9f9ff')}
                  onMouseLeave={e => (e.currentTarget.style.background = '')}
                >
                  <td style={tdStyle}><strong>{p.metadata.name}</strong></td>
                  <td style={tdStyle}><PhaseBadge phase={p.status?.phase} /></td>
                  <td style={{ ...tdStyle, color: '#666', fontFamily: 'monospace', fontSize: 12 }}>
                    {p.status?.podName ?? '—'}
                  </td>
                  <td style={{ ...tdStyle, color: '#666', fontSize: 12 }}>
                    {lastUpdated(p)}
                  </td>
                  <td style={tdStyle}>
                    <button
                      onClick={e => toggleExpand(p.metadata.name, e)}
                      title="Conditions anzeigen"
                      style={{ border: 'none', background: 'none', cursor: 'pointer', fontSize: 13, color: '#555' }}
                    >
                      {expanded.has(p.metadata.name) ? '▼' : '▶'}
                    </button>
                  </td>
                  <td style={{ ...tdStyle, textAlign: 'right' }}>
                    <button
                      onClick={e => handleDelete(p, e)}
                      style={{ color: '#c00', border: 'none', background: 'none', cursor: 'pointer', fontSize: 13 }}
                    >
                      Löschen
                    </button>
                  </td>
                </tr>
                {expanded.has(p.metadata.name) && (
                  <tr>
                    <td colSpan={6} style={{ padding: '0 12px 12px 32px', background: '#fafafa', borderBottom: '1px solid #eee' }}>
                      <ConditionsPanel conditions={p.status?.conditions} />
                    </td>
                  </tr>
                )}
              </React.Fragment>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

function PhaseBadge({ phase }: { phase?: string }) {
  const colors: Record<string, { bg: string; text: string }> = {
    Running: { bg: '#dcfce7', text: '#16a34a' },
    Failed:  { bg: '#fee2e2', text: '#dc2626' },
    Pending: { bg: '#fef9c3', text: '#d97706' },
    Stopped: { bg: '#f3f4f6', text: '#6b7280' },
  }
  const c = colors[phase ?? ''] ?? { bg: '#f3f4f6', text: '#6b7280' }
  return (
    <span style={{
      background: c.bg, color: c.text,
      padding: '2px 8px', borderRadius: 12, fontSize: 12, fontWeight: 600,
    }}>
      {phase ?? 'Unknown'}
    </span>
  )
}

type Condition = NonNullable<NonNullable<Pipeline['status']>['conditions']>[number]

function ConditionsPanel({ conditions }: { conditions?: Condition[] }) {
  if (!conditions || conditions.length === 0) {
    return <p style={{ margin: '8px 0', color: '#888', fontSize: 13 }}>Keine Conditions vorhanden.</p>
  }
  return (
    <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13, marginTop: 8 }}>
      <thead>
        <tr style={{ color: '#666', textAlign: 'left' }}>
          <th style={{ padding: '4px 8px', fontWeight: 600 }}>Type</th>
          <th style={{ padding: '4px 8px', fontWeight: 600 }}>Status</th>
          <th style={{ padding: '4px 8px', fontWeight: 600 }}>Reason</th>
          <th style={{ padding: '4px 8px', fontWeight: 600 }}>Message</th>
          <th style={{ padding: '4px 8px', fontWeight: 600 }}>Seit</th>
        </tr>
      </thead>
      <tbody>
        {conditions.map((c, i) => (
          <tr key={i} style={{ borderTop: '1px solid #eee' }}>
            <td style={{ padding: '4px 8px', fontFamily: 'monospace' }}>{c.type}</td>
            <td style={{ padding: '4px 8px' }}>
              <ConditionStatusBadge status={c.status} />
            </td>
            <td style={{ padding: '4px 8px', color: '#555' }}>{c.reason ?? '—'}</td>
            <td style={{ padding: '4px 8px', color: '#555', maxWidth: 400, wordBreak: 'break-word' }}>
              {c.message ?? '—'}
            </td>
            <td style={{ padding: '4px 8px', color: '#888', whiteSpace: 'nowrap' }}>
              {c.lastTransitionTime
                ? new Date(c.lastTransitionTime).toLocaleString('de-DE', { dateStyle: 'short', timeStyle: 'short' })
                : '—'}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

function ConditionStatusBadge({ status }: { status: string }) {
  const styles: Record<string, { color: string; label: string }> = {
    True:    { color: '#16a34a', label: '✓ True' },
    False:   { color: '#dc2626', label: '✗ False' },
    Unknown: { color: '#d97706', label: '? Unknown' },
  }
  const s = styles[status] ?? { color: '#6b7280', label: status }
  return <span style={{ color: s.color, fontWeight: 600 }}>{s.label}</span>
}

function lastUpdated(p: Pipeline): string {
  const times = (p.status?.conditions ?? [])
    .map(c => c.lastTransitionTime)
    .filter(Boolean) as string[]
  const ts = times.length > 0
    ? times.reduce((a, b) => (a > b ? a : b))
    : p.metadata.creationTimestamp
  if (!ts) return '—'
  return new Date(ts).toLocaleString('de-DE', { dateStyle: 'short', timeStyle: 'short' })
}

const thStyle: React.CSSProperties = { padding: '8px 12px', fontWeight: 600, fontSize: 13 }
const tdStyle: React.CSSProperties = { padding: '10px 12px', verticalAlign: 'middle' }
const newBtnStyle: React.CSSProperties = {
  padding: '6px 16px', background: '#3b82f6', color: '#fff',
  border: 'none', borderRadius: 4, cursor: 'pointer', fontSize: 14,
}
