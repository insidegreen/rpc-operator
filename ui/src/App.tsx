import { useEffect, useMemo, useState } from 'react'
import { listCatalog } from './api'
import { PipelineEditor } from './components/PipelineEditor'
import { DeployBar } from './components/DeployBar'
import type { CatalogComponent, PipelineSpec } from './types'

const DEFAULT_SPEC: PipelineSpec = {
  input: {
    type: 'generate',
    config: { mapping: 'root = "hello world"', interval: '1s', count: 5 },
  },
  processors: [{ type: 'mapping', config: 'root = content().uppercase()' }],
  output: { type: 'stdout', config: {} },
}

export default function App() {
  const [namespace, setNamespace] = useState('rpc-operator-poc')
  const [name, setName] = useState('my-pipeline')
  const [spec, setSpec] = useState<PipelineSpec>(DEFAULT_SPEC)
  const [catalog, setCatalog] = useState<CatalogComponent[]>([])

  useEffect(() => {
    listCatalog().then(setCatalog).catch(console.error)
  }, [])

  const catalogCache = useMemo(
    () => new Map(catalog.map(c => [c.category + '/' + c.name, c])),
    [catalog],
  )

  return (
    <div
      style={{
        maxWidth: 1200,
        margin: '0 auto',
        padding: 24,
        fontFamily: 'system-ui, sans-serif',
      }}
    >
      <h1 style={{ fontSize: 22, marginBottom: 4 }}>RPC Operator — Pipeline Editor</h1>
      <p style={{ color: '#666', marginBottom: 24, fontSize: 14 }}>
        Redpanda Connect Pipelines konfigurieren und in Kubernetes deployen.
      </p>

      <div style={{ display: 'flex', gap: 16, marginBottom: 24 }}>
        <label style={{ flex: 1 }}>
          Namespace
          <input
            value={namespace}
            onChange={e => setNamespace(e.target.value)}
            style={inputStyle}
          />
        </label>
        <label style={{ flex: 2 }}>
          Pipeline-Name
          <input value={name} onChange={e => setName(e.target.value)} style={inputStyle} />
        </label>
      </div>

      <PipelineEditor spec={spec} catalogCache={catalogCache} onChange={setSpec} />
      <DeployBar namespace={namespace} name={name} spec={spec} />
    </div>
  )
}

const inputStyle: React.CSSProperties = {
  display: 'block',
  width: '100%',
  marginTop: 4,
  padding: '6px 10px',
  border: '1px solid #ccc',
  borderRadius: 4,
  fontSize: 14,
  boxSizing: 'border-box',
}
