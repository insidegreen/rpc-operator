import { describe, it, expect, afterEach, beforeAll, afterAll, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { setupServer } from 'msw/node'
import { http, HttpResponse } from 'msw'

// jsdom can't run the lazy Monaco chunk / web workers; mock it to a plain textarea.
vi.mock('../utils/monacoSetup', () => ({ setupMonaco: () => Promise.resolve() }))
vi.mock('@monaco-editor/react', () => ({
  default: ({ value, onChange }: { value?: string; onChange?: (v: string | undefined) => void }) => (
    <textarea aria-label="raw-yaml" value={value} onChange={e => onChange?.(e.target.value)} />
  ),
}))

import { PipelineEditor } from './PipelineEditor'

const server = setupServer(
  http.get('/api/v1/namespaces/default/pipelineclusters', () => HttpResponse.json({ items: [] })),
  http.get('/api/v1/namespaces/default/pipelineprojects', () => HttpResponse.json({ items: [] })),
)
beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('PipelineEditor — ephemeral', () => {
  it('includes spec.ephemeral in the deploy payload when toggled on', async () => {
    let captured: unknown = null
    server.use(http.post('/api/v1/namespaces/default/pipelines', async ({ request }) => {
      captured = await request.json()
      return HttpResponse.json({ metadata: { name: 'np', namespace: 'default' }, spec: {} })
    }))

    render(<PipelineEditor namespace="default" onBack={() => {}} onSaved={() => {}} />)
    await screen.findByRole('checkbox')  // EphemeralEditor toggle present

    await userEvent.type(screen.getByRole('textbox', { name: /Pipeline name/i }), 'np')
    await userEvent.click(screen.getByRole('checkbox'))
    await userEvent.click(screen.getByRole('button', { name: /Deploy/i }))

    await waitFor(() => expect(captured).not.toBeNull())
    expect((captured as { spec: { ephemeral?: unknown } }).spec.ephemeral)
      .toEqual({ ttlAfterSuccess: '1h', ttlAfterFailure: '72h' })
  })
})
