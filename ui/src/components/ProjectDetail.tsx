import { useEffect, useState, useCallback } from 'react'
import { getProject, listPipelines, updateProject } from '../api'
import type { PipelineProject, ProjectRoute, ProjectRouteStatus } from '../types'
import { buildTopology, computeLayout, type TopoNode } from '../topology'
import { TopologyCanvas } from './TopologyCanvas'
import { RouterDrawer } from './RouterDrawer'

interface Props {
  namespace: string
  name: string
  readOnly: boolean
  onBack: () => void
  onOpenPipeline: (pipeline: string) => void
  /** Opens the pipeline editor with projectRef pre-filled. */
  onAddPipeline: (project: string) => void
}

const subjectOf = (project: string, route: string) => `rpc.${project}.${route}`
const streamOf = (project: string, route: string) => `rpc-${project}-${route}`

export function ProjectDetail({ namespace, name, readOnly, onBack, onOpenPipeline, onAddPipeline }: Props) {
  const [project, setProject] = useState<PipelineProject>()
  const [members, setMembers] = useState<string[]>([])
  const [error, setError] = useState<string>()
  const [selectedId, setSelectedId] = useState<string | null>(null)
  // null = creating a new route, a route = editing, undefined = drawer closed.
  const [drawerRoute, setDrawerRoute] = useState<ProjectRoute | null | undefined>(undefined)

  const load = useCallback(() => {
    getProject(namespace, name)
      .then(p => { setProject(p); setError(undefined) })
      .catch(e => setError((e as Error).message))
    // Pipelines attached via projectRef — shown on the map even when unrouted.
    listPipelines(namespace)
      .then(ps => setMembers(ps.filter(p => p.spec.projectRef?.name === name).map(p => p.metadata.name)))
      .catch(() => setMembers([]))
  }, [namespace, name])

  useEffect(() => {
    load()
    const id = setInterval(load, 15_000)
    return () => clearInterval(id)
  }, [load])

  if (error) return (
    <div>
      <button onClick={onBack} style={backLinkStyle}>← Back</button>
      <p style={{ color: 'red' }}>Error: {error}</p>
    </div>
  )
  if (!project) return <p style={{ color: '#888' }}>Loading project…</p>

  const routes = project.spec.routes ?? []
  const topo = computeLayout(buildTopology(project, members))
  const selectedNode = topo.nodes.find(n => n.id === selectedId) ?? null
  const pipelineNames = topo.nodes.filter(n => n.kind === 'pipeline').map(n => n.id)
  // Surface the operator's verdict: any False status condition is a problem the
  // user must act on (e.g. RoutesValid=False → fix the offending pipeline's I/O).
  const problems = (project.status?.conditions ?? []).filter(c => c.status === 'False')
  const degraded = project.status?.phase === 'Degraded'

  async function saveRoute(updated: ProjectRoute) {
    if (!project) return
    const existing = routes.filter(r => r.name !== updated.name)
    const nextSpec = { ...project.spec, routes: [...existing, updated] }
    try {
      await updateProject(namespace, name, nextSpec, project.metadata.resourceVersion)
      setDrawerRoute(undefined)
      load()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  async function deleteRoute(routeName: string) {
    if (!project) return
    if (!window.confirm(`Delete router "${routeName}"? Affected pipelines will be re-rendered.`)) return
    const nextSpec = { ...project.spec, routes: routes.filter(r => r.name !== routeName) }
    try {
      await updateProject(namespace, name, nextSpec, project.metadata.resourceVersion)
      setSelectedId(null)
      load()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16 }}>
        <button onClick={onBack} style={backLinkStyle}>← Back</button>
        <h2 style={{ margin: 0, fontSize: 18 }}>{name}</h2>
        <span style={{ fontSize: 13, color: '#888' }}>{project.status?.phase ?? 'Unknown'}</span>
        {!readOnly && (
          <div style={{ marginLeft: 'auto', display: 'flex', gap: 8 }}>
            <button onClick={() => onAddPipeline(name)} style={toolbarBtnStyle}>+ Pipeline</button>
            <button onClick={() => setDrawerRoute(null)} style={toolbarBtnStyle}>+ Router</button>
          </div>
        )}
      </div>

      {problems.length > 0 && (
        <div style={degraded ? problemBannerStyle : infoBannerStyle}>
          <strong>{degraded ? 'Project degraded' : 'Project provisioning'}</strong>
          <ul style={{ margin: '6px 0 0', paddingLeft: 18 }}>
            {problems.map(c => (
              <li key={c.type}>
                <span style={{ fontWeight: 600 }}>{c.type}</span>
                {c.reason ? ` (${c.reason})` : ''}{c.message ? `: ${c.message}` : ''}
              </li>
            ))}
          </ul>
        </div>
      )}

      <div style={{ display: 'flex', gap: 16, alignItems: 'flex-start' }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          {topo.nodes.length === 0 ? (
            <p style={{ color: '#888' }}>
              No pipelines or routes yet. {readOnly ? '' : 'Attach a pipeline with “+ Pipeline”, then use “+ Router” to wire them together.'}
            </p>
          ) : (
            <>
              {routes.length === 0 && (
                <p style={{ color: '#888', fontSize: 13, marginTop: 0 }}>
                  No routes yet. {readOnly ? '' : 'Use “+ Router” to wire these pipelines together.'}
                </p>
              )}
              <TopologyCanvas topology={topo} selectedId={selectedId} onSelect={setSelectedId} />
            </>
          )}
        </div>

        <aside style={panelStyle}>
          {!selectedNode ? (
            <p style={{ color: '#888', fontSize: 13 }}>Select a node to see details.</p>
          ) : selectedNode.kind === 'router' ? (
            <RouterPanel
              project={name}
              route={routes.find(r => r.name === selectedNode.routeName)!}
              status={project.status?.routes?.find(rs => rs.name === selectedNode.routeName)}
              readOnly={readOnly}
              onEdit={r => setDrawerRoute(r)}
              onDelete={deleteRoute}
            />
          ) : (
            <PipelinePanel node={selectedNode} routes={routes} onOpen={onOpenPipeline} />
          )}
        </aside>
      </div>

      {drawerRoute !== undefined && (
        <RouterDrawer
          pipelines={pipelineNames}
          route={drawerRoute ?? undefined}
          onSave={saveRoute}
          onClose={() => setDrawerRoute(undefined)}
        />
      )}
    </div>
  )
}

function RouterPanel({ project, route, status, readOnly, onEdit, onDelete }: {
  project: string; route: ProjectRoute; status?: ProjectRouteStatus; readOnly: boolean
  onEdit: (r: ProjectRoute) => void; onDelete: (name: string) => void
}) {
  const failed = (status?.conditions ?? []).filter(c => c.status === 'False')
  return (
    <div style={{ fontSize: 13 }}>
      <h3 style={panelTitleStyle}>Router: {route.name}</h3>
      <Row label="Subject" value={subjectOf(project, route.name)} />
      <Row label="Stream" value={streamOf(project, route.name)} />
      <Row label="Stream status" value={status?.phase || 'not provisioned'} />
      <Row label="Producer" value={route.from} />
      {failed.map(c => (
        <div key={c.type} style={routeProblemStyle}>{c.reason ? `${c.reason}: ` : ''}{c.message}</div>
      ))}
      <div style={{ marginTop: 10, fontWeight: 600 }}>Targets</div>
      <table style={{ width: '100%', fontSize: 12, marginTop: 4 }}>
        <tbody>
          {route.to.map(t => (
            <tr key={t.pipeline}>
              <td style={{ padding: '3px 0' }}>{t.pipeline}</td>
              <td style={{ padding: '3px 0', color: '#64748b' }}>{t.when ? `when: ${t.when}` : '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {!readOnly && (
        <div style={{ display: 'flex', gap: 8, marginTop: 16 }}>
          <button onClick={() => onEdit(route)} style={toolbarBtnStyle}>Edit router</button>
          <button onClick={() => onDelete(route.name)} style={deleteBtnStyle}>Delete router</button>
        </div>
      )}
    </div>
  )
}

function PipelinePanel({ node, routes, onOpen }: {
  node: TopoNode; routes: ProjectRoute[]; onOpen: (p: string) => void
}) {
  const incoming = routes.filter(r => r.to.some(t => t.pipeline === node.id)).map(r => r.name)
  const outgoing = routes.filter(r => r.from === node.id).map(r => r.name)
  const role = outgoing.length && incoming.length ? 'middle'
    : outgoing.length ? 'source'
    : incoming.length ? 'sink' : 'standalone'
  return (
    <div style={{ fontSize: 13 }}>
      <h3 style={panelTitleStyle}>Pipeline: {node.id}</h3>
      <Row label="Role" value={role} />
      <Row label="Incoming routes" value={incoming.join(', ') || '—'} />
      <Row label="Outgoing routes" value={outgoing.join(', ') || '—'} />
      <div style={{ marginTop: 16 }}>
        <button onClick={() => onOpen(node.id)} style={toolbarBtnStyle}>Open pipeline</button>
      </div>
    </div>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div style={{ display: 'flex', gap: 8, padding: '3px 0' }}>
      <span style={{ color: '#888', minWidth: 110 }}>{label}</span>
      <span style={{ fontFamily: 'monospace', wordBreak: 'break-all' }}>{value}</span>
    </div>
  )
}

const backLinkStyle: React.CSSProperties = {
  border: 'none', background: 'none', cursor: 'pointer', fontSize: 14, color: '#3b82f6',
}
const panelStyle: React.CSSProperties = {
  width: 300, flexShrink: 0, border: '1px solid #eee', borderRadius: 8, padding: 16, background: '#fff',
}
const panelTitleStyle: React.CSSProperties = { margin: '0 0 12px', fontSize: 14 }
const toolbarBtnStyle: React.CSSProperties = {
  padding: '5px 10px', fontSize: 12, background: '#1d4ed8', color: '#fff',
  border: 'none', borderRadius: 6, cursor: 'pointer',
}
const deleteBtnStyle: React.CSSProperties = {
  padding: '5px 10px', fontSize: 12, background: '#fff', color: '#dc2626',
  border: '1px solid #fca5a5', borderRadius: 6, cursor: 'pointer',
}
const problemBannerStyle: React.CSSProperties = {
  background: '#fef2f2', color: '#b91c1c', border: '1px solid #fca5a5',
  borderRadius: 8, padding: '10px 14px', fontSize: 13, marginBottom: 16,
}
const infoBannerStyle: React.CSSProperties = {
  background: '#fffbeb', color: '#92400e', border: '1px solid #fcd34d',
  borderRadius: 8, padding: '10px 14px', fontSize: 13, marginBottom: 16,
}
const routeProblemStyle: React.CSSProperties = {
  background: '#fef2f2', color: '#b91c1c', border: '1px solid #fca5a5',
  borderRadius: 6, padding: '6px 8px', fontSize: 12, marginTop: 8,
}
