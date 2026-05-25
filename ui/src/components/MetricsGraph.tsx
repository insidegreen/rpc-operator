import { useEffect, useState, useCallback } from 'react'
import {
  LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, Legend,
} from 'recharts'
import type { MetricQuery, MetricsResponse } from '../types'

interface Props {
  /** Fetches one metric series for [startSec, endSec]. Pipeline detail closes
   *  over getMetrics(ns, name, …); cluster detail over getClusterMetrics(…). */
  fetchMetrics: (query: MetricQuery, startSec: number, endSec: number) => Promise<MetricsResponse>
  /** When false, the chart shows a resting placeholder and does not poll. */
  isRunning: boolean
  /** Text for the resting placeholder. Defaults to the pipeline wording. */
  idleLabel?: string
}

interface ChartPoint {
  t: number
  throughput?: number
  errors?: number
}

const WINDOW_SEC = 30 * 60 // 30 minutes

export function MetricsGraph({ fetchMetrics, isRunning, idleLabel = 'Pipeline is not running.' }: Props) {
  const [data, setData] = useState<ChartPoint[]>([])
  const [unavailable, setUnavailable] = useState(false)
  const [error, setError] = useState<string | undefined>()

  const load = useCallback(async () => {
    const now = Math.floor(Date.now() / 1000)
    const start = now - WINDOW_SEC

    try {
      const [throughput, errors] = await Promise.all([
        fetchMetrics('throughput', start, now),
        fetchMetrics('error_rate', start, now),
      ])
      setUnavailable(false)
      setError(undefined)

      const byT = new Map<number, ChartPoint>()
      for (const dp of throughput.datapoints) {
        byT.set(dp.t, { t: dp.t, throughput: dp.v })
      }
      for (const dp of errors.datapoints) {
        const existing = byT.get(dp.t) ?? { t: dp.t }
        byT.set(dp.t, { ...existing, errors: dp.v })
      }
      const sorted = Array.from(byT.values()).sort((a, b) => a.t - b.t)
      setData(sorted)
    } catch (e: unknown) {
      const status = (e as { status?: number }).status
      if (status === 503) {
        setUnavailable(true)
      } else if (status === 409) {
        setData([])
      } else {
        setError((e as Error).message)
      }
    }
  }, [fetchMetrics])

  useEffect(() => {
    if (!isRunning) return
    load()
    const id = setInterval(load, 10_000)
    return () => clearInterval(id)
  }, [isRunning, load])

  const sectionStyle: React.CSSProperties = {
    background: '#fafafa',
    border: '1px solid #eee',
    borderRadius: 6,
    padding: 16,
    marginBottom: 16,
  }

  if (unavailable) {
    return (
      <div style={sectionStyle}>
        <h3 style={{ margin: '0 0 8px', fontSize: 14, fontWeight: 600 }}>Metrics</h3>
        <p style={{ color: '#888', fontSize: 13, margin: 0 }}>
          Prometheus not configured. Start the operator with <code>--prometheus-url</code>.
        </p>
      </div>
    )
  }

  if (!isRunning) {
    return (
      <div style={sectionStyle}>
        <h3 style={{ margin: '0 0 8px', fontSize: 14, fontWeight: 600 }}>Metrics</h3>
        <p style={{ color: '#888', fontSize: 13, margin: 0 }}>{idleLabel}</p>
      </div>
    )
  }

  return (
    <div style={sectionStyle}>
      <h3 style={{ margin: '0 0 12px', fontSize: 14, fontWeight: 600 }}>Metrics (last 30 min)</h3>
      {error && <p style={{ color: '#c00', fontSize: 13 }}>Error: {error}</p>}
      {data.length === 0 && !error ? (
        <p style={{ color: '#888', fontSize: 13, margin: 0 }}>No data.</p>
      ) : (
        <ResponsiveContainer width="100%" height={200}>
          <LineChart data={data}>
            <XAxis
              dataKey="t"
              tickFormatter={(t: number) => new Date(t * 1000).toLocaleTimeString()}
              minTickGap={60}
            />
            <YAxis unit=" msg/s" width={80} />
            <Tooltip
              labelFormatter={(label) => new Date((label as number) * 1000).toLocaleTimeString()}
              formatter={(v) => typeof v === 'number' ? [`${v.toFixed(2)} msg/s`] : [String(v)]}
            />
            <Legend />
            <Line
              type="monotone" dataKey="throughput" name="Throughput"
              stroke="#2563eb" dot={false} strokeWidth={2}
            />
            <Line
              type="monotone" dataKey="errors" name="Errors"
              stroke="#dc2626" dot={false} strokeWidth={2}
            />
          </LineChart>
        </ResponsiveContainer>
      )}
    </div>
  )
}
