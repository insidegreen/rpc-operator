import type { EphemeralSpec } from '../types'
import { parseDuration, formatDuration, type DurationUnit } from '../duration'

interface Props {
  value: EphemeralSpec | undefined
  onChange: (next: EphemeralSpec | undefined) => void
}

const DEFAULTS: EphemeralSpec = { ttlAfterSuccess: '1h', ttlAfterFailure: '72h' }
const UNITS: DurationUnit[] = ['minutes', 'hours', 'days']

export function EphemeralEditor({ value, onChange }: Props) {
  const enabled = !!value

  function setTtl(field: keyof EphemeralSpec, raw: string) {
    onChange({ ...(value ?? DEFAULTS), [field]: raw })
  }

  return (
    <div style={boxStyle}>
      <label style={headerStyle}>
        <input
          type="checkbox"
          checked={enabled}
          onChange={e => onChange(e.target.checked ? { ...DEFAULTS } : undefined)}
        />
        {' '}Ephemeral — delete this pipeline after it finishes
      </label>
      {enabled && value && (
        <div style={{ marginTop: 8 }}>
          <TtlRow label="Keep after success"
                  duration={value.ttlAfterSuccess ?? '1h'}
                  onChange={s => setTtl('ttlAfterSuccess', s)} />
          <TtlRow label="Keep after failure"
                  duration={value.ttlAfterFailure ?? '72h'}
                  onChange={s => setTtl('ttlAfterFailure', s)} />
          <p style={hintStyle}>
            For sub-hour precision like 1h30m, switch the pipeline to YAML mode.
          </p>
        </div>
      )}
    </div>
  )
}

function TtlRow({ label, duration, onChange }: {
  label: string; duration: string; onChange: (s: string) => void
}) {
  const { value, unit } = parseDuration(duration)
  return (
    <div style={rowStyle}>
      <span style={labelStyle}>{label}</span>
      <input
        type="number"
        min={1}
        value={value}
        onChange={e => onChange(formatDuration({ value: Math.max(1, parseInt(e.target.value, 10) || 1), unit }))}
        style={numStyle}
      />
      <select
        value={unit}
        onChange={e => onChange(formatDuration({ value, unit: e.target.value as DurationUnit }))}
        style={selStyle}
      >
        {UNITS.map(u => <option key={u} value={u}>{u}</option>)}
      </select>
    </div>
  )
}

const boxStyle: React.CSSProperties = {
  border: '1px solid #dde', borderRadius: 6, padding: 12, marginTop: 12, background: '#fafafa',
}
const headerStyle: React.CSSProperties = {
  fontWeight: 600, fontSize: 14, color: '#334', display: 'flex', alignItems: 'center', gap: 6,
}
const rowStyle: React.CSSProperties = {
  display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6,
}
const labelStyle: React.CSSProperties = { fontSize: 13, color: '#444', width: 140 }
const numStyle: React.CSSProperties = {
  padding: '4px 8px', border: '1px solid #ccc', borderRadius: 4, fontSize: 13, width: 72,
}
const selStyle: React.CSSProperties = {
  padding: '4px 8px', border: '1px solid #ccc', borderRadius: 4, fontSize: 13,
}
const hintStyle: React.CSSProperties = { margin: '6px 0 0', fontSize: 11, color: '#888' }
