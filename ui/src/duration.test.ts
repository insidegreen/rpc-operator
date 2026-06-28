import { describe, it, expect } from 'vitest'
import {
  parseDuration, formatDuration, durationToMs, humanizeDuration, humanizeRemaining,
} from './duration'

describe('parseDuration', () => {
  it.each([
    ['1h', { value: 1, unit: 'hours' }],
    ['72h', { value: 3, unit: 'days' }],
    ['90m', { value: 90, unit: 'minutes' }],
    ['1h30m', { value: 90, unit: 'minutes' }],
    ['120m', { value: 2, unit: 'hours' }],
    ['45m', { value: 45, unit: 'minutes' }],
    ['2h', { value: 2, unit: 'hours' }],
  ] as const)('normalizes %s', (input, expected) => {
    expect(parseDuration(input)).toEqual(expected)
  })
})

describe('formatDuration', () => {
  it.each([
    [{ value: 1, unit: 'hours' }, '1h'],
    [{ value: 3, unit: 'days' }, '72h'],
    [{ value: 90, unit: 'minutes' }, '90m'],
    [{ value: 2, unit: 'days' }, '48h'],
  ] as const)('serializes %o', (input, expected) => {
    expect(formatDuration(input)).toBe(expected)
  })

  it('round-trips through parse', () => {
    expect(formatDuration(parseDuration('72h'))).toBe('72h')
  })
})

describe('durationToMs', () => {
  it('converts h and m to ms', () => {
    expect(durationToMs('1h')).toBe(3_600_000)
    expect(durationToMs('1h30m')).toBe(5_400_000)
  })
})

describe('humanizeDuration', () => {
  it.each([
    ['1h', '1 hour'],
    ['72h', '3 days'],
    ['90m', '90 minutes'],
    ['2h', '2 hours'],
  ] as const)('humanizes %s', (input, expected) => {
    expect(humanizeDuration(input)).toBe(expected)
  })
})

describe('humanizeRemaining', () => {
  it('formats hours/minutes/seconds', () => {
    expect(humanizeRemaining((58 * 60 + 12) * 1000)).toBe('58m 12s')
    expect(humanizeRemaining((3600 + 120 + 3) * 1000)).toBe('1h 2m 3s')
  })
  it('returns empty for non-positive', () => {
    expect(humanizeRemaining(0)).toBe('')
    expect(humanizeRemaining(-5000)).toBe('')
  })
})
