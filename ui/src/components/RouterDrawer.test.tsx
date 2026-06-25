import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { RouterDrawer } from './RouterDrawer'

const pipelines = ['ingest', 'warehouse', 'alert']

describe('RouterDrawer', () => {
  it('submits a new route with from + one target', async () => {
    const onSave = vi.fn()
    render(<RouterDrawer pipelines={pipelines} onSave={onSave} onClose={() => {}} />)
    await userEvent.type(screen.getByLabelText(/Route name/i), 'fan')
    await userEvent.selectOptions(screen.getByLabelText(/From/i), 'ingest')
    await userEvent.selectOptions(screen.getByLabelText(/Target 1/i), 'warehouse')
    await userEvent.click(screen.getByRole('button', { name: /Save router/i }))
    expect(onSave).toHaveBeenCalledWith({
      name: 'fan', from: 'ingest', to: [{ pipeline: 'warehouse' }],
    })
  })

  it('rejects an empty route name', async () => {
    const onSave = vi.fn()
    render(<RouterDrawer pipelines={pipelines} onSave={onSave} onClose={() => {}} />)
    await userEvent.selectOptions(screen.getByLabelText(/From/i), 'ingest')
    await userEvent.selectOptions(screen.getByLabelText(/Target 1/i), 'warehouse')
    await userEvent.click(screen.getByRole('button', { name: /Save router/i }))
    expect(onSave).not.toHaveBeenCalled()
    expect(screen.getByText(/name is required/i)).toBeInTheDocument()
  })

  it('pre-fills when editing an existing route', () => {
    render(<RouterDrawer pipelines={pipelines} onSave={() => {}} onClose={() => {}}
      route={{ name: 'fan', from: 'ingest', to: [{ pipeline: 'alert', when: 'this.level=="high"' }] }} />)
    expect(screen.getByLabelText(/Route name/i)).toHaveValue('fan')
    expect(screen.getByDisplayValue('this.level=="high"')).toBeInTheDocument()
  })

  it('saves an optional target label', async () => {
    const onSave = vi.fn()
    render(<RouterDrawer pipelines={pipelines} onSave={onSave} onClose={() => {}} />)
    await userEvent.type(screen.getByLabelText(/Route name/i), 'fan')
    await userEvent.selectOptions(screen.getByLabelText(/From/i), 'ingest')
    await userEvent.selectOptions(screen.getByLabelText(/Target 1/i), 'alert')
    await userEvent.type(screen.getByLabelText(/Label 1/i), 'High-Priority EU')
    await userEvent.click(screen.getByRole('button', { name: /Save router/i }))
    expect(onSave).toHaveBeenCalledWith({
      name: 'fan', from: 'ingest', to: [{ pipeline: 'alert', label: 'High-Priority EU' }],
    })
  })

  it('omits an empty label from the saved target', async () => {
    const onSave = vi.fn()
    render(<RouterDrawer pipelines={pipelines} onSave={onSave} onClose={() => {}} />)
    await userEvent.type(screen.getByLabelText(/Route name/i), 'fan')
    await userEvent.selectOptions(screen.getByLabelText(/From/i), 'ingest')
    await userEvent.selectOptions(screen.getByLabelText(/Target 1/i), 'warehouse')
    await userEvent.click(screen.getByRole('button', { name: /Save router/i }))
    expect(onSave).toHaveBeenCalledWith({
      name: 'fan', from: 'ingest', to: [{ pipeline: 'warehouse' }],
    })
  })
})
