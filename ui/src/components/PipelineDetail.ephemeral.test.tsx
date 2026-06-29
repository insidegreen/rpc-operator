import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { PipelineDetail } from './PipelineDetail'
import type { Pipeline } from '../types'

const base: Pipeline = {
  apiVersion: 'rpc.operator.io/v1alpha1',
  kind: 'Pipeline',
  metadata: { name: 'oneshot', namespace: 'default' },
  spec: { rawYAML: '', ephemeral: { ttlAfterSuccess: '1h', ttlAfterFailure: '72h' } },
  status: { phase: 'Stopped', completionTime: '2026-06-28T12:00:00Z', completionResult: 'Succeeded' },
}

describe('PipelineDetail — ephemeral', () => {
  it('hides Run for a finished ephemeral pipeline but keeps Edit', () => {
    render(
      <PipelineDetail
        pipeline={base}
        showLogs={false}
        onEdit={vi.fn()}
        onBack={vi.fn()}
        onRun={vi.fn()}
        onStop={vi.fn()}
      />,
    )
    expect(screen.queryByRole('button', { name: 'Run' })).toBeNull()
    expect(screen.getByRole('button', { name: 'Edit' })).toBeInTheDocument()
    expect(screen.getByText('Succeeded')).toBeInTheDocument()
  })
})
