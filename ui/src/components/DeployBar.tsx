import { useState } from 'react'
import { validatePipeline, getPipeline, createPipeline, updatePipeline } from '../api'
import type { PipelineSpec, ValidationError } from '../types'

interface Props {
  namespace: string
  name: string
  spec: PipelineSpec
}

export function DeployBar({ namespace, name, spec }: Props) {
  const [status, setStatus] = useState<'idle' | 'validating' | 'deploying' | 'done' | 'error'>(
    'idle',
  )
  const [validationErrors, setValidationErrors] = useState<ValidationError[]>([])
  const [message, setMessage] = useState<string>()

  async function handleDeploy() {
    if (!name || !namespace) {
      setMessage('Name und Namespace sind Pflichtfelder.')
      setStatus('error')
      return
    }

    setStatus('validating')
    setValidationErrors([])
    setMessage(undefined)

    try {
      const v = await validatePipeline(namespace, name, spec)
      if (!v.valid) {
        setValidationErrors(v.errors ?? [])
        setStatus('error')
        return
      }
    } catch (e) {
      setMessage('Validierung fehlgeschlagen: ' + (e as Error).message)
      setStatus('error')
      return
    }

    setStatus('deploying')
    try {
      let existing: { metadata?: { resourceVersion?: string } } | null = null
      try {
        existing = await getPipeline(namespace, name)
      } catch (e: unknown) {
        if ((e as { status?: number }).status !== 404) throw e
      }

      const result = existing
        ? await updatePipeline(namespace, name, spec, existing.metadata?.resourceVersion)
        : await createPipeline(namespace, name, spec)

      setMessage(
        `Pipeline "${result.metadata.name}" deployed. Phase: ${result.status?.phase ?? 'Pending'}`,
      )
      setStatus('done')
    } catch (e) {
      setMessage('Deploy fehlgeschlagen: ' + (e as Error).message)
      setStatus('error')
    }
  }

  const busy = status === 'validating' || status === 'deploying'

  return (
    <div style={{ marginTop: 24, padding: 16, background: '#f0f4f8', borderRadius: 8 }}>
      <button
        onClick={handleDeploy}
        disabled={busy || !name}
        style={{ padding: '8px 24px', fontSize: 15, cursor: busy ? 'wait' : 'pointer' }}
      >
        {status === 'validating'
          ? 'Validiert…'
          : status === 'deploying'
            ? 'Deployt…'
            : 'Deploy'}
      </button>

      {message && (
        <p style={{ marginTop: 8, color: status === 'done' ? 'green' : 'red' }}>{message}</p>
      )}

      {validationErrors.length > 0 && (
        <ul style={{ marginTop: 8, color: 'red', paddingLeft: 20 }}>
          {validationErrors.map((e, i) => (
            <li key={i}>
              <code>{e.path}</code>: {e.message}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
