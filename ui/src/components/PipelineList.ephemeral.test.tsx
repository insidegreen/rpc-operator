import { describe, it, expect, beforeAll, afterEach, afterAll } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { setupServer } from 'msw/node'
import { http, HttpResponse } from 'msw'
import { PipelineList } from './PipelineList'

const server = setupServer(
  http.get('/api/v1/namespaces/default/pipelines', () => HttpResponse.json({ items: [
    { metadata: { name: 'oneshot', namespace: 'default' },
      spec: { ephemeral: { ttlAfterSuccess: '1h', ttlAfterFailure: '72h' } },
      status: { phase: 'Running' } },
    { metadata: { name: 'longrunner', namespace: 'default' },
      spec: {}, status: { phase: 'Running' } },
  ] })),
)
beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('PipelineList — ephemeral tag', () => {
  it('tags only ephemeral pipelines', async () => {
    render(<PipelineList namespace="default" onViewDetail={() => {}} />)
    await waitFor(() => expect(screen.getByText('oneshot')).toBeInTheDocument())
    const tags = screen.getAllByText('ephemeral')
    expect(tags).toHaveLength(1)
  })
})
