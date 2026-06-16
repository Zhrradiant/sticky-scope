import { onMounted, onUnmounted } from 'vue'
import { EventsOn, EventsOff, model } from '@/api'
import { EVENTS, type AddProgress } from '@/types'
import { useChangesStore } from '@/stores/changes'
import { useProjectsStore } from '@/stores/projects'

export function useWailsEvents() {
  const changes = useChangesStore()
  const projects = useProjectsStore()

  onMounted(() => {
    EventsOn(EVENTS.changes, (cs: model.ChangeSet) => {
      changes.setChangeSet(cs)
      if (cs.projectId === projects.activeId) {
        void changes.refreshSelectedDiff(cs.projectId)
      }
    })
    EventsOn(EVENTS.addProgress, (p: AddProgress) => {
      projects.setAddProgress(p)
    })
  })

  onUnmounted(() => {
    EventsOff(EVENTS.changes)
    EventsOff(EVENTS.addProgress)
  })
}
