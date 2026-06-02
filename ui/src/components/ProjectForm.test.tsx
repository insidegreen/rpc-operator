import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ProjectForm } from './ProjectForm'

describe('ProjectForm', () => {
  it('submits name + optional sizing', async () => {
    const onCreate = vi.fn().mockResolvedValue(undefined)
    render(<ProjectForm onCreate={onCreate} onClose={() => {}} />)
    await userEvent.type(screen.getByLabelText(/Project name/i), 'orders')
    await userEvent.type(screen.getByLabelText(/Cluster instances/i), '2')
    await userEvent.type(screen.getByLabelText(/NATS storage/i), '20Gi')
    await userEvent.click(screen.getByRole('button', { name: /Create project/i }))
    expect(onCreate).toHaveBeenCalledWith('orders', {
      cluster: { instances: 2 },
      nats: { storage: '20Gi' },
    })
  })

  it('requires a DNS-1123 name', async () => {
    const onCreate = vi.fn()
    render(<ProjectForm onCreate={onCreate} onClose={() => {}} />)
    await userEvent.type(screen.getByLabelText(/Project name/i), 'Bad_Name')
    await userEvent.click(screen.getByRole('button', { name: /Create project/i }))
    expect(onCreate).not.toHaveBeenCalled()
    expect(screen.getByText(/DNS-1123 label/i)).toBeInTheDocument()
  })
})
