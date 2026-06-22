<script setup lang="ts">
import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useProjectsStore } from '@/stores/projects'
import { useChangesStore } from '@/stores/changes'
import { model } from '@/api'

const projects = useProjectsStore()
const changes = useChangesStore()
const { busy } = storeToRefs(changes)

const cs = computed<model.ChangeSet | null>(() => changes.changeSetFor(projects.activeId))
const hasChanges = computed(() => (cs.value?.totalFiles ?? 0) > 0)

async function confirmAll() {
  if (projects.activeId) await changes.confirmAll(projects.activeId)
}
</script>

<template>
  <footer class="collapsed-footer">
    <button
      class="sync-btn"
      :disabled="busy || !hasChanges"
      @click="confirmAll"
      :title="$t('header.confirmAll')"
    >{{ $t('header.confirm') }}</button>
    <span class="stat-add" v-if="cs?.totalAdded">+{{ cs.totalAdded }}</span>
    <span class="stat-del" v-if="cs?.totalRemoved">-{{ cs.totalRemoved }}</span>
    <span class="info">{{ cs?.totalFiles ?? 0 }} {{ $t('status.files') }}</span>
  </footer>
</template>

<style scoped>
.collapsed-footer {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  border-top: 1px solid var(--border);
  flex-shrink: 0;
  font-size: 11px;
  font-family: var(--mono);
  background: var(--bg);
}
.sync-btn {
  flex-shrink: 0;
  background: var(--accent);
  border: 1px solid var(--accent);
  border-radius: 8px;
  padding: 2px 8px;
  font-size: 11px;
  font-weight: 600;
  color: #FFF;
  cursor: pointer;
  line-height: 1.3;
  font-family: var(--sans);
}
.sync-btn:hover:not(:disabled) {
  background: var(--accent-hover);
  border-color: var(--accent-hover);
}
.sync-btn:disabled {
  opacity: 0.35;
  cursor: default;
}
.collapsed-footer .info {
  color: var(--muted);
  margin-left: auto;
}
</style>
