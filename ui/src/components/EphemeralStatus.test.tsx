import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { EphemeralStatus } from './EphemeralStatus'

const T0 = '2026-06-28T12:00:00Z'
const t0ms = new Date(T0).getTime()

describe('EphemeralStatus', () => {
  it('shows the retention descriptor before completion', () => {
    render(<EphemeralStatus ephemeral={{ ttlAfterSuccess: '1h', ttlAfterFailure: '72h' }} />)
    expect(screen.getByText(/cleans up 1 hour after success/i)).toBeInTheDocument()
    expect(screen.getByText(/3 days after failure/i)).toBeInTheDocument()
  })

  it('shows result badge and live countdown after success', () => {
    render(
      <EphemeralStatus
        ephemeral={{ ttlAfterSuccess: '1h', ttlAfterFailure: '72h' }}
        completionTime={T0}
        completionResult="Succeeded"
        now={() => t0ms + 30 * 60 * 1000}  // 30 min after completion → 30 min left
      />,
    )
    expect(screen.getByText('Succeeded')).toBeInTheDocument()
    expect(screen.getByText(/cleans up in 30m 0s/i)).toBeInTheDocument()
  })

  it('shows cleanup pending once the deadline has passed', () => {
    render(
      <EphemeralStatus
        ephemeral={{ ttlAfterSuccess: '1h', ttlAfterFailure: '72h' }}
        completionTime={T0}
        completionResult="Succeeded"
        now={() => t0ms + 2 * 60 * 60 * 1000}  // 2h later, 1h TTL → expired
      />,
    )
    expect(screen.getByText(/cleanup pending/i)).toBeInTheDocument()
  })
})
