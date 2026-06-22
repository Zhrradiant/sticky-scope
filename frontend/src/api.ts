// Central access point for the Wails-generated bindings, so components import
// from "@/api" rather than reaching into ../../wailsjs with brittle paths.
import * as App from '../wailsjs/go/main/App'
import { model } from '../wailsjs/go/models'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'

// OpenFileLocation is added here rather than in the auto-generated bindings
// because it was added to the Go backend after the last wails generate run.
// The next wails build will regenerate App.d.ts / App.js to include it.
export function OpenFileLocation(id: string, path: string): Promise<void> {
  return (window as any)['go']['main']['App']['OpenFileLocation'](id, path)
}

export { App, model, EventsOn, EventsOff }
