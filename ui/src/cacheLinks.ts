import type { ProjectCacheResource } from './types'

export interface CacheLink {
  from: string   // composing cache name
  to: string     // referenced layer cache name
  level: number  // 1-based position in the multilevel array
}

/**
 * Walk a config value for a `multilevel` array and record (from → entry, level)
 * for each STRING entry. Object entries (the inline multilevel form) reference
 * no shared resource and are skipped. Recurses so a nested `multilevel` is found.
 */
function collect(node: unknown, from: string, out: CacheLink[]): void {
  if (Array.isArray(node)) {
    for (const item of node) collect(item, from, out)
    return
  }
  if (node && typeof node === 'object') {
    const obj = node as Record<string, unknown>
    const ml = obj.multilevel
    if (Array.isArray(ml)) {
      ml.forEach((entry, i) => {
        if (typeof entry === 'string') out.push({ from, to: entry, level: i + 1 })
      })
    }
    for (const v of Object.values(obj)) collect(v, from, out)
  }
}

/**
 * Detect multilevel cache→cache links among declared cache resources. Only each
 * resource's `config` is scanned; managed (natsKV) caches and configs without a
 * `multilevel` array yield nothing.
 */
export function detectCacheLinks(caches: ProjectCacheResource[]): CacheLink[] {
  const out: CacheLink[] = []
  for (const c of caches) {
    if (c.config === undefined) continue
    collect(c.config, c.name, out)
  }
  return out
}
