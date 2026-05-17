import { useState, useEffect } from 'react'
import { listCatalog } from '../api'
import type { CatalogComponent } from '../types'

interface Props {
  category: 'inputs' | 'processors' | 'outputs'
  onSelect: (component: CatalogComponent) => void
  onClose: () => void
}

export function ComponentPicker({ category, onSelect, onClose }: Props) {
  const [components, setComponents] = useState<CatalogComponent[]>([])
  const [query, setQuery] = useState('')

  useEffect(() => {
    listCatalog().then(all => setComponents(all.filter(c => c.category === category)))
  }, [category])

  const filtered = components.filter(
    c =>
      c.name.toLowerCase().includes(query.toLowerCase()) ||
      c.summary.toLowerCase().includes(query.toLowerCase()),
  )

  return (
    <div style={overlayStyle}>
      <div style={modalStyle}>
        <h3 style={{ margin: '0 0 12px' }}>Add component ({category})</h3>
        <input
          autoFocus
          placeholder="Search…"
          value={query}
          onChange={e => setQuery(e.target.value)}
          style={{ width: '100%', marginBottom: 8, padding: '6px 8px', boxSizing: 'border-box' }}
        />
        <ul style={{ listStyle: 'none', padding: 0, maxHeight: 320, overflowY: 'auto', margin: 0 }}>
          {filtered.map(c => (
            <li key={c.name} style={itemStyle} onClick={() => onSelect(c)}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <strong>{c.name}</strong>
                {c.status !== 'stable' && (
                  <span style={badgeStyle('#888', '#f0f0f0')}>{c.status}</span>
                )}
                {c.bodyKind === 'composite' && (
                  <span style={badgeStyle('#3b5', '#e8fff0')}>composite</span>
                )}
              </div>
              <p style={{ margin: '3px 0 0', fontSize: 12, color: '#666' }}>{c.summary}</p>
            </li>
          ))}
          {filtered.length === 0 && (
            <li style={{ color: '#999', padding: 8 }}>No results.</li>
          )}
        </ul>
        <button onClick={onClose} style={{ marginTop: 12 }}>
          Cancel
        </button>
      </div>
    </div>
  )
}

function badgeStyle(color: string, bg: string): React.CSSProperties {
  return { fontSize: 11, color, background: bg, padding: '1px 6px', borderRadius: 10 }
}

const overlayStyle: React.CSSProperties = {
  position: 'fixed',
  inset: 0,
  background: 'rgba(0,0,0,0.4)',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  zIndex: 100,
}
const modalStyle: React.CSSProperties = {
  background: '#fff',
  borderRadius: 8,
  padding: 24,
  width: 500,
  boxShadow: '0 4px 24px rgba(0,0,0,0.2)',
}
const itemStyle: React.CSSProperties = {
  padding: '8px 12px',
  cursor: 'pointer',
  borderRadius: 4,
  marginBottom: 4,
  border: '1px solid #eee',
}
