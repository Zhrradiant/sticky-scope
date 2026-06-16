// Frontend-only types and helpers.

export const EVENTS = {
  changes: 'changes:updated',
  addProgress: 'add:progress',
} as const

export type Lang = 'zh' | 'en'

// AddProgress mirrors the Go model.AddProgress emitted during project addition.
export interface AddProgress {
  message: string
  current: number
  total: number
}

// splitPath separates a path into its directory and basename.
export function splitPath(p: string): { dir: string; name: string } {
  const idx = p.lastIndexOf('/')
  if (idx < 0) return { dir: '', name: p }
  return { dir: p.slice(0, idx + 1), name: p.slice(idx + 1) }
}
