// Central access point for the Wails-generated bindings, so components import
// from "@/api" rather than reaching into ../../wailsjs with brittle paths.
import * as App from '../wailsjs/go/main/App'
import { model } from '../wailsjs/go/models'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'

export { App, model, EventsOn, EventsOff }
