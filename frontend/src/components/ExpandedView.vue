<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import FileList from './FileList.vue'
import DiffViewer from './DiffViewer.vue'

const emit = defineEmits<{ (e: 'select', path: string): void }>()

// ---- resizable split ----
const leftPanel = ref<HTMLElement | null>(null)
const divider = ref<HTMLElement | null>(null)
const leftWidth = ref(320)
const dragging = ref(false)

function onDividerDown(e: MouseEvent) {
  e.preventDefault()
  dragging.value = true
}

function onMouseMove(e: MouseEvent) {
  if (!dragging.value) return
  const parent = leftPanel.value?.parentElement
  if (!parent) return
  const rect = parent.getBoundingClientRect()
  let w = e.clientX - rect.left
  // clamp: min 200px, max 50% of parent
  w = Math.max(200, Math.min(w, rect.width * 0.5))
  leftWidth.value = w
}

function onMouseUp() {
  dragging.value = false
}

onMounted(() => {
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
})
onUnmounted(() => {
  document.removeEventListener('mousemove', onMouseMove)
  document.removeEventListener('mouseup', onMouseUp)
})
</script>

<template>
  <div class="expanded-view" :class="{ dragging }">
    <div ref="leftPanel" class="left-panel" :style="{ width: leftWidth + 'px' }">
      <FileList @select="emit('select', $event)" />
    </div>
    <div
      ref="divider"
      class="divider"
      @mousedown="onDividerDown"
    ></div>
    <DiffViewer />
  </div>
</template>

<style scoped>
.expanded-view {
  flex: 1;
  min-height: 0;
  display: flex;
  user-select: none;
}
.expanded-view.dragging {
  cursor: col-resize;
}

.left-panel {
  flex-shrink: 0;
  min-width: 200px;
  max-width: 50%;
  display: flex;
}

.divider {
  flex-shrink: 0;
  width: 4px;
  cursor: col-resize;
  background: transparent;
  transition: background 0.15s;
  position: relative;
}
.divider:hover,
.expanded-view.dragging .divider {
  background: var(--accent);
  opacity: 0.5;
}
</style>
