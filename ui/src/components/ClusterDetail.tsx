import { useCallback, useEffect, useState } from 'react'
import { getCluster, getClusterInstances, getClusterMetrics } from '../api'
import type { ClusterDistribution, PipelineCluster } from '../types'
import { MetricsGraph } from './MetricsGraph'
import { ClusterInstances } from './ClusterInstances'
import { ClusterScaleControl } from './ClusterScaleControl'

interface Props {
  namespace: string
  name: string
  readOnly: boolean
  onBack: () => void
  /** Chip click → open the pipeline's detail (cross-section jump). */
  onOpenPipeline: (name: string) => void
}

export function ClusterDetail({ namespace, name, readOnly, onBack, onOpenPipeline }: Props) {
  const [cluster, setCluster] = useState<PipelineCluster | null>(null)
  const [dist, setDist] = useState<ClusterDistribution | null>(null)
  const [error, setError] = useState<string>()

  const reload = useCallback(() => {
    Promise.all([getCluster(namespace, name), getClusterInstances(namespace, name)])
      .then(([c, d]) => { setCluster(c); setDist(d); setError(undefined) })
      .catch(e => setError((e as Error).message))
  }, [namespace, name])

  useEffect(() => {
    reload()
    const id = setInterval(reload, 15_000)
    return () => clearInterval(id)
  }, [reload])

  if (error)   return <p style={{ color: 'red' }}>Error: {error}</p>
  if (!cluster) return <p style={{ color: '#888' }}>Loading cluster…</p>

  const desired = cluster.spec.replicas ?? 0
  const ready = cluster.status?.readyReplicas ?? 0

  return (
    <div>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 20 }}>
        <button onClick={onBack} style={backBtnStyle}>← Clusters</button>
        <h2 style={{ margin: 0, fontSize: 18 }}>{cluster.metadata.name}</h2>
        <span style={{ color: '#888', fontSize: 13 }}>{cluster.metadata.namespace}</span>
        <PhaseBadge phase={cluster.status?.phase} />
        <span style={{ fontSize: 13, color: '#555' }}>{ready}/{desired} ready</span>
        {!readOnly && (
          <div style={{ marginLeft: 'auto' }}>
            <ClusterScaleControl
              namespace={namespace}
              name={name}
              replicas={desired}
              onScaled={reload}
            />
          </div>
        )}
      </div>

      {/* Aggregate metrics (full-width, top) */}
      <MetricsGraph
        fetchMetrics={(q, start, end) => getClusterMetrics(namespace, name, q, start, end)}
        isRunning={ready > 0}
        idleLabel="No ready instances."
      />

      {/* Instances (full-width, below) */}
      {dist && <ClusterInstances distribution={dist} onOpenPipeline={onOpenPipeline} />}
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

const backBtnStyle: React.CSSProperties = {
  border: 'none', background: 'none', cursor: 'pointer', fontSize: 14, color: '#3b82f6',
}
