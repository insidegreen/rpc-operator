import { describe, it, expect, beforeAll, afterEach, afterAll, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { setupServer } from 'msw/node'
import { http, HttpResponse } from 'msw'

// PipelineEditor lazy-loads Monaco + the RPK schema, neither of which jsdom
// can run. Mock both so the editor renders as a plain textarea.
vi.mock('./utils/monacoSetup', () => ({ setupMonaco: () => Promise.resolve() }))
vi.mock('@monaco-editor/react', () => ({
  default: ({ value, onChange }: { value?: string; onChange?: (v: string | undefined) => void }) => (
    <textarea aria-label="raw-yaml" value={value} onChange={e => onChange?.(e.target.value)} />
  ),
}))

import App from './App'
import type { Pipeline, PipelineProject } from './types'

const project: PipelineProject = {
  metadata: { name: 'myproj', namespace: 'default' },
  spec: { routes: [] },
}

function pipeline(name: string): Pipeline {
  return {
    metadata: { name, namespace: 'default', resourceVersion: '1' },
    spec: { rawYAML: `# ${name}\ninput:\n  stdin: {}\n`, projectRef: { name: 'myproj' } },
  } as Pipeline
}

const alpha = pipeline('alpha')
const bravo = pipeline('bravo')
const pipelines: Record<string, Pipeline> = { alpha, bravo }

const server = setupServer(
  http.get('/api/v1/auth/config', () =>
    HttpResponse.json({ oidcEnabled: false })),
  http.get('/api/v1/auth/whoami', () =>
    HttpResponse.json({ user: 'tester', readOnly: false })),
  http.get('/api/v1/namespaces', () => HttpResponse.json({ namespaces: ['default'] })),
  // List endpoints are namespace-agnostic so the pre-listNamespaces poll on the
  // initial namespace is satisfied too; the UI switches to `default` immediately.
  http.get('/api/v1/namespaces/:ns/pipelineclusters', () =>
    HttpResponse.json({ items: [] })),
  http.get('/api/v1/namespaces/:ns/pipelineprojects', () =>
    HttpResponse.json({ items: [project] })),
  http.get('/api/v1/namespaces/default/pipelineprojects/myproj', () =>
    HttpResponse.json(project)),
  http.get('/api/v1/namespaces/:ns/pipelines', () =>
    HttpResponse.json({ items: [alpha, bravo] })),
  // Namespace-level connection poll (any namespace, incl. the pre-listNamespaces default).
  http.get('/api/v1/namespaces/:ns/pipelines/connections', () =>
    HttpResponse.json({ connections: {} })),
  http.get('/api/v1/namespaces/default/pipelines/:name', ({ params }) => {
    const p = pipelines[params.name as string]
    return p ? HttpResponse.json(p) : new HttpResponse(null, { status: 404 })
  }),
)
beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

async function editFromMap(user: ReturnType<typeof userEvent.setup>, name: string) {
  await user.click(await screen.findByText(name))            // select the pipeline node
  await user.click(await screen.findByText('Edit pipeline')) // open the editor
}

// F50.4-adjacent regression: editing a pipeline from the project (tactical) map,
// returning, then editing a *different* pipeline must open the newly selected one
// — not the previously edited one. Reproduces the stale-mount bug where the editor
// re-mounted with the prior pipeline's props before the new load resolved.
describe('App — edit pipeline from the project map', () => {
  it('opens the newly selected pipeline, not the previously edited one', async () => {
    const user = userEvent.setup()
    render(<App />)

    // Navigate: Projects → myproj map.
    await user.click(await screen.findByText('Projects'))
    await user.click(await screen.findByText('myproj'))

    // Edit alpha → editor shows alpha.
    await editFromMap(user, 'alpha')
    await waitFor(() =>
      expect(screen.getByLabelText(/Pipeline name/)).toHaveValue('alpha'))

    // Back to the map, then edit bravo → editor must show bravo.
    await user.click(screen.getByText('← Back'))
    await editFromMap(user, 'bravo')
    await waitFor(() =>
      expect(screen.getByLabelText(/Pipeline name/)).toHaveValue('bravo'))
  })
})
