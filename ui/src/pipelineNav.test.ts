import { describe, it, expect } from 'vitest'
import { pipelineBackTarget } from './pipelineNav'

describe('pipelineBackTarget', () => {
  it('routes a project origin back to the project detail', () => {
    expect(pipelineBackTarget({ kind: 'project', name: 'orders' }))
      .toEqual({ section: 'projects', projectsView: 'detail' })
  })

  it('routes a cluster origin back to the cluster detail', () => {
    expect(pipelineBackTarget({ kind: 'cluster', name: 'c1' }))
      .toEqual({ section: 'clusters', clustersView: 'detail' })
  })

  it('routes the default (pipeline-list) origin back to the pipeline list', () => {
    expect(pipelineBackTarget({ kind: 'pipelines' }))
      .toEqual({ section: 'pipelines', pipelinesView: 'list' })
  })
})
