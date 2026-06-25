import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { TopologyCanvas } from './TopologyCanvas'
import { buildTopology, computeLayout } from '../topology'
import type { PipelineProject } from '../types'

const project: PipelineProject = {
  metadata: { name: 'orders', namespace: 'default' },
  spec: { routes: [{ name: 'fan', from: 'ingest', to: [{ pipeline: 'warehouse' }] }] },
}

describe('TopologyCanvas', () => {
  it('renders one labelled box per node', () => {
    const topo = computeLayout(buildTopology(project))
    render(<TopologyCanvas topology={topo} selectedId={null} onSelect={() => {}} />)
    expect(screen.getByText('ingest')).toBeInTheDocument()
    expect(screen.getByText('warehouse')).toBeInTheDocument()
    expect(screen.getByText('fan')).toBeInTheDocument()       // router pill label
  })

  it('marks the selected node', () => {
    const topo = computeLayout(buildTopology(project))
    const { container } = render(
      <TopologyCanvas topology={topo} selectedId="ingest" onSelect={() => {}} />)
    expect(container.querySelector('[data-selected="true"]')).toBeTruthy()
  })

  it('wraps content in a transform group', () => {
    const topo = computeLayout(buildTopology(project))
    const { container } = render(<TopologyCanvas topology={topo} selectedId={null} onSelect={() => {}} />)
    const g = container.querySelector('svg > g[transform]')
    expect(g).toBeTruthy()
    expect(g!.getAttribute('transform')).toMatch(/translate\(.*\) scale\(.*\)/)
  })
})

describe('TopologyCanvas caches', () => {
  const withCache: PipelineProject = {
    metadata: { name: 'orders', namespace: 'default' },
    spec: {
      routes: [{ name: 'fan', from: 'ingest', to: [{ pipeline: 'warehouse' }] }],
      cacheResources: [{ name: 'shared', natsKV: {} }],
    },
  }

  it('renders the cache node label and the operator label on its edge', () => {
    const topo = computeLayout(buildTopology(withCache, ['ingest', 'warehouse'],
      [{ pipeline: 'warehouse', cache: 'shared', operators: ['get', 'set'] }]))
    const { container } = render(<TopologyCanvas topology={topo} selectedId={null} onSelect={() => {}} />)
    expect(screen.getByText('shared')).toBeInTheDocument()
    expect(screen.getByText('get, set')).toBeInTheDocument()
    // dashed cache edge present
    expect(container.querySelector('path[stroke-dasharray]')).toBeTruthy()
  })

  it('renders the "output" label on a cache output edge', () => {
    const topo = computeLayout(buildTopology(withCache, ['ingest', 'warehouse'],
      [{ pipeline: 'warehouse', cache: 'shared', operators: ['output'] }]))
    render(<TopologyCanvas topology={topo} selectedId={null} onSelect={() => {}} />)
    expect(screen.getByText('output')).toBeInTheDocument()
  })

  it('renders a cacheLink arc with its level label', () => {
    const proj: PipelineProject = {
      metadata: { name: 'orders', namespace: 'default' },
      spec: { cacheResources: [{ name: 'leveled', config: { multilevel: ['hot'] } }, { name: 'hot', config: { memory: {} } }] },
    }
    const topo = computeLayout(buildTopology(proj, [], [], [{ from: 'leveled', to: 'hot', level: 1 }]))
    render(<TopologyCanvas topology={topo} selectedId={null} onSelect={() => {}} />)
    expect(screen.getByText('L1')).toBeInTheDocument()
  })
})

describe('TopologyCanvas route-edge labels', () => {
  const make = (to: Array<{ pipeline: string; when?: string; label?: string }>) =>
    computeLayout(buildTopology({
      metadata: { name: 'orders', namespace: 'default' },
      spec: { routes: [{ name: 'fan', from: 'ingest', to }] },
    }))

  it('shows the label (truncated) when a label is set', () => {
    const topo = make([{ pipeline: 'alert', when: 'this.level == "high"', label: 'High-Priority EU' }])
    render(<TopologyCanvas topology={topo} selectedId={null} onSelect={() => {}} />)
    expect(screen.getByText('High-Priority EU')).toBeInTheDocument()
    expect(screen.queryByText(/^when:/)).not.toBeInTheDocument()
  })

  it('falls back to when: when no label is set', () => {
    const topo = make([{ pipeline: 'alert', when: 'this.level == "high"' }])
    render(<TopologyCanvas topology={topo} selectedId={null} onSelect={() => {}} />)
    expect(screen.getByText(/^when:this\.level/)).toBeInTheDocument()
  })

  it('renders no edge label when neither label nor when is set', () => {
    const topo = make([{ pipeline: 'warehouse' }])
    render(<TopologyCanvas topology={topo} selectedId={null} onSelect={() => {}} />)
    expect(screen.queryByText(/^when:/)).not.toBeInTheDocument()
  })

  it('exposes the full when expression via an SVG <title> tooltip', () => {
    const topo = make([{ pipeline: 'alert', when: 'this.level == "high"', label: 'High-Priority EU' }])
    const { container } = render(<TopologyCanvas topology={topo} selectedId={null} onSelect={() => {}} />)
    const titles = Array.from(container.querySelectorAll('title')).map(t => t.textContent)
    expect(titles).toContain('High-Priority EU — this.level == "high"')
  })
})

describe('TopologyCanvas zoom', () => {
  it('renders the zoom control bar', () => {
    const topo = computeLayout(buildTopology(project))
    render(<TopologyCanvas topology={topo} selectedId={null} onSelect={() => {}} />)
    expect(screen.getByLabelText('Zoom in')).toBeInTheDocument()
    expect(screen.getByLabelText('Zoom out')).toBeInTheDocument()
    expect(screen.getByLabelText('Fit to view')).toBeInTheDocument()
  })

  it('changes the transform scale on wheel-up and restores it on Fit', () => {
    const topo = computeLayout(buildTopology(project))
    const { container } = render(<TopologyCanvas topology={topo} selectedId={null} onSelect={() => {}} />)
    const viewport = screen.getByTestId('topology-viewport')
    const transformOf = () => container.querySelector('svg > g[transform]')!.getAttribute('transform')!
    const before = transformOf()
    fireEvent.wheel(viewport, { deltaY: -10, clientX: 0, clientY: 0 })
    expect(transformOf()).not.toBe(before)
    expect(transformOf()).toMatch(/scale\(1\.1\b/)
    fireEvent.click(screen.getByLabelText('Fit to view'))
    expect(transformOf()).toBe('translate(0,0) scale(1)')   // jsdom viewport is 0×0 → identity fit
  })

  it('still selects a node on click (pan does not swallow it)', () => {
    const topo = computeLayout(buildTopology(project))
    const onSelect = vi.fn()
    render(<TopologyCanvas topology={topo} selectedId={null} onSelect={onSelect} />)
    fireEvent.click(screen.getByText('ingest'))
    expect(onSelect).toHaveBeenCalledWith('ingest')
  })

  it('pans the transform when dragging the empty canvas', () => {
    const topo = computeLayout(buildTopology(project))
    const { container } = render(<TopologyCanvas topology={topo} selectedId={null} onSelect={() => {}} />)
    const svg = container.querySelector('svg')!
    const transformOf = () => container.querySelector('svg > g[transform]')!.getAttribute('transform')!
    // pointerdown on the svg background (no data-node ancestor), then move 40,25
    fireEvent.pointerDown(svg, { clientX: 10, clientY: 10, pointerId: 1 })
    fireEvent.pointerMove(svg, { clientX: 50, clientY: 35, pointerId: 1 })
    fireEvent.pointerUp(svg, { pointerId: 1 })
    expect(transformOf()).toBe('translate(40,25) scale(1)')
  })

  it('does not pan when the drag starts on a node', () => {
    const topo = computeLayout(buildTopology(project))
    const { container } = render(<TopologyCanvas topology={topo} selectedId={null} onSelect={() => {}} />)
    const transformOf = () => container.querySelector('svg > g[transform]')!.getAttribute('transform')!
    const node = container.querySelector('[data-node]')!
    fireEvent.pointerDown(node, { clientX: 10, clientY: 10, pointerId: 1 })
    fireEvent.pointerMove(container.querySelector('svg')!, { clientX: 50, clientY: 35, pointerId: 1 })
    fireEvent.pointerUp(container.querySelector('svg')!, { pointerId: 1 })
    expect(transformOf()).toBe('translate(0,0) scale(1)')
  })
})
