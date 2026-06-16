<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useProjectsStore } from '@/stores/projects'
import { useChangesStore } from '@/stores/changes'
import { useUiStore } from '@/stores/ui'
import { useWailsEvents } from '@/composables/useWailsEvents'
import StickyHeader from '@/components/StickyHeader.vue'
import CompactView from '@/components/CompactView.vue'
import ExpandedView from '@/components/ExpandedView.vue'
import EmptyGuide from '@/components/EmptyGuide.vue'
import { App } from '@/api'

const projects = useProjectsStore()
const changes = useChangesStore()
const ui = useUiStore()
const { active } = storeToRefs(projects)
const { isExpanded } = storeToRefs(ui)

useWailsEvents()

onMounted(async () => {
  await projects.load()
})

watch(
  () => projects.activeId,
  async (id) => {
    changes.resetForProject()
    if (id) {
      await changes.fetchChanges(id)
    }
  },
  { immediate: true },
)

function onFileSelect(_path: string) {
  // In compact mode, clicking a file expands to show diff
  if (!isExpanded.value) {
    ui.setExpanded(true)
    App.SetCompactMode(true)
  }
}
</script>

<template>
  <div class="sticky-note">
    <StickyHeader />

    <!-- No project: show guide -->
    <EmptyGuide v-if="!active" />

    <!-- Has project, compact mode -->
    <CompactView v-else-if="!isExpanded" @select="onFileSelect" />

    <!-- Has project, expanded mode -->
    <ExpandedView v-else />
  </div>
</template>

<style scoped>
.sticky-note {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: var(--bg);
  border-radius: var(--sticky-radius);
  overflow: hidden;
}
</style>
