import { describe, it, expect, afterEach, beforeAll, afterAll } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { setupServer } from 'msw/node'
import { http, HttpResponse } from 'msw'
import { ProjectList } from './ProjectList'
import type { PipelineProject } from '../types'

const orders: PipelineProject = {
  metadata: { name: 'orders', namespace: 'default' },
  spec: { routes: [{ name: 'ingest', from: 'a', to: [{ pipeline: 'b' }] }] },
  status: { phase: 'Ready', cluster: { ready: 1, total: 1 }, nats: { ready: 1, total: 1 } },
}

const server = setupServer()
beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('ProjectList', () => {
  it('renders a project row', async () => {
    server.use(http.get('/api/v1/namespaces/default/pipelineprojects', () =>
      HttpResponse.json({ items: [orders] })))
    render(<ProjectList namespace="default" onViewDetail={() => {}} onNew={() => {}} />)
    await waitFor(() => expect(screen.getByText('orders')).toBeInTheDocument())
    expect(screen.getByText('Ready')).toBeInTheDocument()
  })

  it('shows the empty state', async () => {
    server.use(http.get('/api/v1/namespaces/default/pipelineprojects', () =>
      HttpResponse.json({ items: [] })))
    render(<ProjectList namespace="default" onViewDetail={() => {}} onNew={() => {}} />)
    await waitFor(() =>
      expect(screen.getByText(/No projects in this namespace/i)).toBeInTheDocument())
  })

  it('treats 403 as empty (no error banner)', async () => {
    server.use(http.get('/api/v1/namespaces/default/pipelineprojects', () =>
      HttpResponse.json({ error: 'forbidden' }, { status: 403 })))
    render(<ProjectList namespace="default" onViewDetail={() => {}} onNew={() => {}} />)
    await waitFor(() =>
      expect(screen.getByText(/No projects in this namespace/i)).toBeInTheDocument())
  })
})
