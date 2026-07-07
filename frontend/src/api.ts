// Central access point for the Wails-generated bindings, so components import
// from "@/api" rather than reaching into ../../wailsjs with brittle paths.
import * as App from '../wailsjs/go/main/App'
import { model } from '../wailsjs/go/models'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'

// Some Go bindings are surfaced through hand-wrappers here (calling into the
// same Wails runtime registry) rather than via the re-exported `App` namespace
// below. This is the established place for methods added outside of a
// `wails generate` run; if a future generate adds one to App.js, the matching
// wrapper here becomes a harmless duplicate and can be dropped.
export function OpenFileLocation(id: string, path: string): Promise<void> {
  return (window as any)['go']['main']['App']['OpenFileLocation'](id, path)
}

export function StartAllMonitoring(): Promise<void> {
  return (window as any)['go']['main']['App']['StartAllMonitoring']()
}

export { App, model, EventsOn, EventsOff }
