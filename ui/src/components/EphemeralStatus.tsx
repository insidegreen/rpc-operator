import { useEffect, useState } from 'react'
import type { EphemeralSpec } from '../types'
import { humanizeDuration, humanizeRemaining, durationToMs } from '../duration'

interface Props {
  ephemeral: EphemeralSpec
  completionTime?: string
  completionResult?: 'Succeeded' | 'Failed'
  /** Injectable clock for deterministic tests; defaults to Date.now. */
  now?: () => number
}

export function EphemeralStatus({ ephemeral, completionTime, completionResult, now = Date.now }: Props) {
  const completed = !!completionTime
  const [, setTick] = useState(0)

  useEffect(() => {
    if (!completed) return
    const id = setInterval(() => setTick(t => t + 1), 1000)
    return () => clearInterval(id)
  }, [completed])

  const success = ephemeral.ttlAfterSuccess ?? '1h'
  const failure = ephemeral.ttlAfterFailure ?? '72h'

  if (!completed) {
    return (
      <div style={boxStyle}>
        <strong>Ephemeral</strong> · cleans up {humanizeDuration(success)} after success,{' '}
        {humanizeDuration(failure)} after failure.
      </div>
    )
  }

  const ttl = completionResult === 'Failed' ? failure : success
  const deadline = new Date(completionTime!).getTime() + durationToMs(ttl)
  const remaining = humanizeRemaining(deadline - now())

  return (
    <div style={boxStyle}>
      <ResultBadge result={completionResult} />
      <span style={{ marginLeft: 8, fontSize: 13 }}>
        {remaining ? `cleans up in ${remaining}` : 'cleanup pending…'}
      </span>
    </div>
  )
}

function ResultBadge({ result }: { result?: 'Succeeded' | 'Failed' }) {
  const c = result === 'Failed'
    ? { bg: '#fee2e2', text: '#dc2626' }
    : { bg: '#dcfce7', text: '#16a34a' }
  return (
    <span style={{ background: c.bg, color: c.text, padding: '2px 8px', borderRadius: 10, fontSize: 12, fontWeight: 600 }}>
      {result ?? 'Succeeded'}
    </span>
  )
}

const boxStyle: React.CSSProperties = {
  background: '#fafafa', border: '1px solid #eee', borderRadius: 6,
  padding: 12, marginBottom: 16, fontSize: 13, color: '#444',
}
