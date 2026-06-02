import { describe, it, expect, afterEach, beforeAll, afterAll } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { setupServer } from 'msw/node'
import { http, HttpResponse } from 'msw'
import { ProjectDetail } from './ProjectDetail'
import type { PipelineProject } from '../types'

const orders: PipelineProject = {
  metadata: { name: 'orders', namespace: 'default' },
  spec: { routes: [{ name: 'fan', from: 'ingest', to: [{ pipeline: 'warehouse' }] }] },
  status: { phase: 'Ready', cluster: { name: 'orders-cluster', ready: 1, total: 1 } },
}

const server = setupServer(
  http.get('/api/v1/namespaces/default/pipelineprojects/orders', () => HttpResponse.json(orders)),
)
beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('ProjectDetail', () => {
  it('loads and renders the topology + side panel for a selected node', async () => {
    render(<ProjectDetail namespace="default" name="orders" readOnly={false}
      onBack={() => {}} onOpenPipeline={() => {}} onAddPipeline={() => {}} />)
    await waitFor(() => expect(screen.getByText('ingest')).toBeInTheDocument())
    await userEvent.click(screen.getByText('fan'))               // select the router node
    expect(screen.getByText(/Subject/i)).toBeInTheDocument()     // router side panel
    expect(screen.getByText('rpc.orders.fan')).toBeInTheDocument()
  })

  it('hides + Router in read-only mode', async () => {
    render(<ProjectDetail namespace="default" name="orders" readOnly={true}
      onBack={() => {}} onOpenPipeline={() => {}} onAddPipeline={() => {}} />)
    await waitFor(() => expect(screen.getByText('ingest')).toBeInTheDocument())
    expect(screen.queryByRole('button', { name: /\+ Router/i })).toBeNull()
  })
})
