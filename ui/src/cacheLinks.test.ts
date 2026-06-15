import { describe, it, expect } from 'vitest'
import { detectCacheLinks } from './cacheLinks'
import type { ProjectCacheResource } from './types'

describe('detectCacheLinks', () => {
  it('detects ordered links from a multilevel config', () => {
    const caches: ProjectCacheResource[] = [
      { name: 'leveled', config: { multilevel: ['hot', 'kv'] } },
      { name: 'hot', config: { memory: {} } },
      { name: 'kv', natsKV: {} },
    ]
    expect(detectCacheLinks(caches)).toEqual([
      { from: 'leveled', to: 'hot', level: 1 },
      { from: 'leveled', to: 'kv', level: 2 },
    ])
  })

  it('finds a multilevel nested under another key', () => {
    const caches: ProjectCacheResource[] = [
      { name: 'wrap', config: { something: { multilevel: ['a'] } } },
    ]
    expect(detectCacheLinks(caches)).toEqual([{ from: 'wrap', to: 'a', level: 1 }])
  })

  it('skips inline multilevel entries (objects, not labels)', () => {
    const caches: ProjectCacheResource[] = [
      { name: 'inline', config: { multilevel: [{ label: '', memory: {} }, { label: '', redis: { url: 'x' } }] } },
    ]
    expect(detectCacheLinks(caches)).toEqual([])
  })

  it('yields nothing for managed natsKV or non-multilevel custom configs', () => {
    const caches: ProjectCacheResource[] = [
      { name: 'kv', natsKV: {} },
      { name: 'r', config: { redis: { url: 'x' } } },
    ]
    expect(detectCacheLinks(caches)).toEqual([])
  })

  it('still emits a link to an undeclared layer', () => {
    const caches: ProjectCacheResource[] = [
      { name: 'leveled', config: { multilevel: ['ghost'] } },
    ]
    expect(detectCacheLinks(caches)).toEqual([{ from: 'leveled', to: 'ghost', level: 1 }])
  })
})
