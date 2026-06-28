export type DurationUnit = 'minutes' | 'hours' | 'days'

export interface DurationParts {
  value: number
  unit: DurationUnit
}

// Sum the h/m components of a Go duration string into total minutes.
function toMinutes(s: string): number {
  let total = 0
  const re = /(\d+)(h|m)/g
  let match: RegExpExecArray | null
  while ((match = re.exec(s)) !== null) {
    const n = parseInt(match[1], 10)
    total += match[2] === 'h' ? n * 60 : n
  }
  return total
}

export function parseDuration(s: string): DurationParts {
  const mins = toMinutes(s)
  if (mins > 0 && mins % 1440 === 0) return { value: mins / 1440, unit: 'days' }
  if (mins > 0 && mins % 60 === 0) return { value: mins / 60, unit: 'hours' }
  return { value: mins, unit: 'minutes' }
}

export function formatDuration({ value, unit }: DurationParts): string {
  if (unit === 'days') return `${value * 24}h`  // Go has no day unit
  if (unit === 'hours') return `${value}h`
  return `${value}m`
}

export function durationToMs(s: string): number {
  return toMinutes(s) * 60_000
}

export function humanizeDuration(s: string): string {
  const { value, unit } = parseDuration(s)
  const singular = unit.slice(0, -1)  // 'minute' | 'hour' | 'day'
  return `${value} ${value === 1 ? singular : unit}`
}

export function humanizeRemaining(ms: number): string {
  if (ms <= 0) return ''
  const totalSec = Math.floor(ms / 1000)
  const h = Math.floor(totalSec / 3600)
  const m = Math.floor((totalSec % 3600) / 60)
  const sec = totalSec % 60
  const parts: string[] = []
  if (h > 0) parts.push(`${h}h`)
  if (h > 0 || m > 0) parts.push(`${m}m`)
  parts.push(`${sec}s`)
  return parts.join(' ')
}
