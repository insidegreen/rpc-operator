export interface ViewTransform { k: number; tx: number; ty: number }
export interface Size { w: number; h: number }
export interface Point { x: number; y: number }
export interface Bounds { min: number; max: number }

export const ZOOM_BOUNDS: Bounds = { min: 0.25, max: 3 }

function clamp(v: number, lo: number, hi: number): number {
  return Math.min(hi, Math.max(lo, v))
}

/** Scale by `factor` around viewport-space `cursor`; the world point under the cursor stays fixed. */
export function zoomAtPoint(state: ViewTransform, cursor: Point, factor: number, bounds: Bounds): ViewTransform {
  const k = clamp(state.k * factor, bounds.min, bounds.max)
  const worldX = (cursor.x - state.tx) / state.k
  const worldY = (cursor.y - state.ty) / state.k
  return { k, tx: cursor.x - worldX * k, ty: cursor.y - worldY * k }
}

/** Scale by `factor` around the viewport centre (for +/- buttons). */
export function stepZoom(state: ViewTransform, viewport: Size, factor: number, bounds: Bounds): ViewTransform {
  return zoomAtPoint(state, { x: viewport.w / 2, y: viewport.h / 2 }, factor, bounds)
}

/** Scale `content` to fit `viewport` (clamped to bounds) and centre it. */
export function computeFit(content: Size, viewport: Size, bounds: Bounds): ViewTransform {
  if (content.w <= 0 || content.h <= 0 || viewport.w <= 0 || viewport.h <= 0) {
    return { k: 1, tx: 0, ty: 0 }
  }
  const k = clamp(Math.min(viewport.w / content.w, viewport.h / content.h), bounds.min, bounds.max)
  return { k, tx: (viewport.w - content.w * k) / 2, ty: (viewport.h - content.h * k) / 2 }
}
