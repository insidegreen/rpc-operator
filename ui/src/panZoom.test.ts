import { describe, it, expect } from 'vitest'
import { zoomAtPoint, stepZoom, computeFit, ZOOM_BOUNDS } from './panZoom'

describe('zoomAtPoint', () => {
  it('keeps the world point under the cursor fixed', () => {
    const cursor = { x: 100, y: 50 }
    const next = zoomAtPoint({ k: 1, tx: 0, ty: 0 }, cursor, 2, ZOOM_BOUNDS)
    expect(next.k).toBe(2)
    // screen position of the same world point must not move: world*k + t
    const worldX = (cursor.x - 0) / 1
    const worldY = (cursor.y - 0) / 1
    expect(worldX * next.k + next.tx).toBeCloseTo(cursor.x)
    expect(worldY * next.k + next.ty).toBeCloseTo(cursor.y)
  })

  it('clamps k to the bounds', () => {
    expect(zoomAtPoint({ k: 1, tx: 0, ty: 0 }, { x: 0, y: 0 }, 100, ZOOM_BOUNDS).k).toBe(3)
    expect(zoomAtPoint({ k: 1, tx: 0, ty: 0 }, { x: 0, y: 0 }, 0.001, ZOOM_BOUNDS).k).toBe(0.25)
  })
})

describe('stepZoom', () => {
  it('zooms around the viewport centre', () => {
    const viewport = { w: 400, h: 200 }
    const next = stepZoom({ k: 1, tx: 0, ty: 0 }, viewport, 2, ZOOM_BOUNDS)
    expect(next.k).toBe(2)
    // centre point (200,100) stays fixed
    expect(200 * next.k + next.tx).toBeCloseTo(200)
    expect(100 * next.k + next.ty).toBeCloseTo(100)
  })
})

describe('computeFit', () => {
  it('scales content to fit and centres it', () => {
    const next = computeFit({ w: 200, h: 100 }, { w: 400, h: 400 }, ZOOM_BOUNDS)
    expect(next.k).toBe(2)                 // min(400/200, 400/100)=2, within bounds
    expect(next.tx).toBeCloseTo(0)         // (400 - 200*2)/2
    expect(next.ty).toBeCloseTo(100)       // (400 - 100*2)/2
  })

  it('clamps the fit scale to the max bound', () => {
    expect(computeFit({ w: 10, h: 10 }, { w: 400, h: 400 }, ZOOM_BOUNDS).k).toBe(3)
  })

  it('returns identity when sizes are non-positive', () => {
    expect(computeFit({ w: 200, h: 100 }, { w: 0, h: 0 }, ZOOM_BOUNDS)).toEqual({ k: 1, tx: 0, ty: 0 })
  })
})
