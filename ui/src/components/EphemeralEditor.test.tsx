import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { EphemeralEditor } from './EphemeralEditor'

describe('EphemeralEditor', () => {
  it('emits backend defaults when toggled on', async () => {
    const onChange = vi.fn()
    render(<EphemeralEditor value={undefined} onChange={onChange} />)
    await userEvent.click(screen.getByRole('checkbox'))
    expect(onChange).toHaveBeenCalledWith({ ttlAfterSuccess: '1h', ttlAfterFailure: '72h' })
  })

  it('emits undefined when toggled off', async () => {
    const onChange = vi.fn()
    render(<EphemeralEditor value={{ ttlAfterSuccess: '1h', ttlAfterFailure: '72h' }} onChange={onChange} />)
    await userEvent.click(screen.getByRole('checkbox'))
    expect(onChange).toHaveBeenCalledWith(undefined)
  })

  it('round-trips a unit change through the duration string', async () => {
    const onChange = vi.fn()
    render(<EphemeralEditor value={{ ttlAfterSuccess: '1h', ttlAfterFailure: '72h' }} onChange={onChange} />)
    // The success row shows value 1, unit "hours"; switch it to days.
    const selects = screen.getAllByRole('combobox')
    await userEvent.selectOptions(selects[0], 'days')
    expect(onChange).toHaveBeenCalledWith({ ttlAfterSuccess: '24h', ttlAfterFailure: '72h' })
  })
})
