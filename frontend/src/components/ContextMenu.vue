<script setup lang="ts">
/**
 * ContextMenu — generic right-click popup menu.
 *
 * Usage:
 *   1. Call useContextMenu() in the parent component.
 *   2. Bind the returned reactive state to this component's props.
 *   3. Wire @contextmenu on the target element to openMenu(e, items).
 *
 * The menu auto-closes on click-outside, Escape, and after any item action.
 */
import { computed, onMounted, onUnmounted, type PropType } from 'vue'
import type { ContextMenuItem } from '@/composables/useContextMenu'

const props = defineProps({
  visible: { type: Boolean, required: true },
  x: { type: Number, required: true },
  y: { type: Number, required: true },
  items: { type: Array as PropType<ContextMenuItem[]>, required: true },
})

const emit = defineEmits<{
  (e: 'close'): void
}>()

function onItemClick(item: ContextMenuItem) {
  if (item.disabled) return
  item.action()
  emit('close')
}

// ---- auto-position: keep the menu inside the viewport ----
const style = computed(() => {
  // Use a fixed max to avoid measuring before the menu is rendered;
  // after mount CSS will re-flow. We apply a rough clamp so the menu
  // doesn't go off-screen.
  const maxW = typeof window !== 'undefined' ? window.innerWidth - 220 : 800
  const maxH = typeof window !== 'undefined' ? window.innerHeight - 160 : 600
  let left = props.x
  let top = props.y
  if (left > maxW) left = maxW
  if (top > maxH) top = maxH
  return { left: left + 'px', top: top + 'px' }
})

// ---- close on Escape ----
function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.visible) {
    emit('close')
  }
}

onMounted(() => {
  document.addEventListener('keydown', onKeyDown)
})
onUnmounted(() => {
  document.removeEventListener('keydown', onKeyDown)
})
</script>

<template>
  <Teleport to="body">
    <!-- backdrop catches clicks outside -->
    <div v-if="visible" class="ctx-backdrop" @click="emit('close')" @contextmenu.prevent="emit('close')" />
    <div v-if="visible" class="ctx-menu" :style="style" @click.stop>
      <template v-for="(item, idx) in items" :key="idx">
        <div
          class="ctx-item"
          :class="{ disabled: item.disabled, danger: item.danger }"
          @click="onItemClick(item)"
        >
          <span v-if="item.icon" class="ctx-icon">{{ item.icon }}</span>
          <span class="ctx-label">{{ item.label }}</span>
        </div>
        <div v-if="item.separator" class="ctx-sep" />
      </template>
    </div>
  </Teleport>
</template>

<style scoped>
.ctx-backdrop {
  position: fixed;
  inset: 0;
  z-index: 999;
}

.ctx-menu {
  position: fixed;
  z-index: 1000;
  min-width: 180px;
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: 12px;
  box-shadow: 0 6px 24px rgba(61, 50, 38, 0.15);
  padding: 6px;
  overflow: hidden;
}

.ctx-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 12px;
  border-radius: 8px;
  cursor: pointer;
  font-size: 13px;
  color: var(--text);
  transition: background 0.1s;
  white-space: nowrap;
}

.ctx-item:hover:not(.disabled) {
  background: var(--panel-2);
}

.ctx-item.disabled {
  opacity: 0.4;
  cursor: default;
}

.ctx-item.danger {
  color: var(--del);
}

.ctx-item.danger:hover:not(.disabled) {
  background: rgba(192, 91, 77, 0.10);
}

.ctx-icon {
  flex-shrink: 0;
  width: 16px;
  text-align: center;
  font-size: 13px;
}

.ctx-label {
  flex: 1;
}

.ctx-sep {
  height: 1px;
  background: var(--border);
  margin: 4px 8px;
}
</style>
