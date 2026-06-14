import yaml from 'js-yaml'
import type { Pipeline } from './types'

export interface CacheUse {
  pipeline: string
  cache: string
  operators: string[]
}

/**
 * Recursively collect (resource, operator) pairs from `cache:` mappings that
 * carry BOTH fields — i.e. the Redpanda Connect `cache` processor. The cache
 * input (resource, no operator), cache output (uses `target`), and `cached`
 * processor (`cache` is a string) all fail this test and are skipped.
 */
function collect(node: unknown, out: Array<{ resource: string; operator: string }>): void {
  if (Array.isArray(node)) {
    for (const item of node) collect(item, out)
    return
  }
  if (node && typeof node === 'object') {
    const obj = node as Record<string, unknown>
    const c = obj.cache
    if (c && typeof c === 'object' && !Array.isArray(c)) {
      const cc = c as Record<string, unknown>
      if (typeof cc.resource === 'string' && typeof cc.operator === 'string') {
        out.push({ resource: cc.resource, operator: cc.operator })
      }
    }
    for (const v of Object.values(obj)) collect(v, out)
  }
}

/**
 * Detect which pipelines use which cache resource, and with which operators.
 * Only rawYAML pipelines are scanned: the component catalog has no `cache`
 * processor, so structured pipelines cannot reference a cache.
 */
export function detectCacheUses(pipelines: Pipeline[]): CacheUse[] {
  const grouped = new Map<string, Map<string, Set<string>>>() // pipeline -> cache -> operators
  for (const p of pipelines) {
    const rawYAML = p.spec.rawYAML
    if (!rawYAML) continue
    let doc: unknown
    try {
      doc = yaml.load(rawYAML)
    } catch {
      continue
    }
    const pairs: Array<{ resource: string; operator: string }> = []
    collect(doc, pairs)
    if (pairs.length === 0) continue
    let byCache = grouped.get(p.metadata.name)
    if (!byCache) { byCache = new Map(); grouped.set(p.metadata.name, byCache) }
    for (const { resource, operator } of pairs) {
      let ops = byCache.get(resource)
      if (!ops) { ops = new Set(); byCache.set(resource, ops) }
      ops.add(operator)
    }
  }
  const uses: CacheUse[] = []
  for (const [pipeline, byCache] of grouped) {
    for (const [cache, ops] of byCache) {
      uses.push({ pipeline, cache, operators: [...ops].sort() })
    }
  }
  return uses
}
