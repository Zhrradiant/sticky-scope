<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import { useProjectsStore } from '@/stores/projects'
import { useChangesStore } from '@/stores/changes'
import { splitPath } from '@/types'
import { model, OpenFileLocation } from '@/api'
import { useContextMenu } from '@/composables/useContextMenu'
import type { ContextMenuItem } from '@/composables/useContextMenu'
import ContextMenu from '@/components/ContextMenu.vue'

const props = withDefaults(defineProps<{ compact?: boolean }>(), { compact: false })
const emit = defineEmits<{ (e: 'select', path: string): void }>()

const { t } = useI18n()
const projects = useProjectsStore()
const changes = useChangesStore()
const { selectedPath, busy } = storeToRefs(changes)

const cs = computed<model.ChangeSet | null>(() => changes.changeSetFor(projects.activeId))
const files = computed(() => cs.value?.files ?? [])
const hasChanges = computed(() => (cs.value?.totalFiles ?? 0) > 0)
// "loading" only when this project has never received a ChangeSet yet. If we
// already have one (e.g. cached from a previous visit) we show it immediately
// even while a refresh is in flight, instead of flashing a loading state.
const loading = computed(() =>
  changes.isLoading(projects.activeId) && !cs.value,
)

// ---- context menu ----
const ctx = useContextMenu()

function onContextMenu(e: MouseEvent, filePath: string) {
  const items: ContextMenuItem[] = [
    {
      label: t('files.openLocation'),
      action: () => {
        if (projects.activeId) void OpenFileLocation(projects.activeId, filePath)
      },
    },
  ]
  ctx.openMenu(e, items)
}

function statusDot(s: string): string {
  if (s === 'added') return '●'
  if (s === 'deleted') return '○'
  return '◉'
}

function statusTitle(s: string): string {
  if (s === 'added') return 'added'
  if (s === 'deleted') return 'deleted'
  return 'modified'
}

function pick(path: string) {
  emit('select', path)
  if (projects.activeId) void changes.selectFile(projects.activeId, path)
}

async function confirmAll() {
  if (projects.activeId) await changes.confirmAll(projects.activeId)
}
</script>

<template>
  <section class="filelist" :class="{ compact: compact }">
    <header class="head" v-if="!compact">
      <span class="title">{{ $t('files.title') }}</span>
      <span class="num" v-if="files.length">{{ files.length }}</span>
    </header>

    <div class="scroll">
      <p v-if="loading" class="loading">{{ $t('common.loading') }}</p>
      <p v-else-if="!files.length" class="empty">{{ $t('files.empty') }}</p>

      <div
        v-for="f in files"
        :key="f.path"
        class="item"
        :class="{ active: f.path === selectedPath }"
        @click="pick(f.path)"
        @contextmenu="onContextMenu($event, f.path)"
      >
        <span class="st" :class="f.status" :title="statusTitle(f.status)">{{ statusDot(f.status) }}</span>
        <span class="path">
          <template v-if="compact">
            <span class="name">{{ splitPath(f.path).name }}</span>
          </template>
          <template v-else>
            <span class="dir">{{ splitPath(f.path).dir }}</span>
            <span class="name">{{ splitPath(f.path).name }}</span>
          </template>
        </span>
        <span v-if="f.binary" class="tag bin">{{ $t('files.binary') }}</span>
        <span v-else class="counts">
          <span v-if="f.added" class="stat-add">+{{ f.added }}</span>
          <span v-if="f.removed" class="stat-del">-{{ f.removed }}</span>
        </span>
      </div>

      <p v-if="cs?.truncated && !compact" class="trunc">{{ $t('status.truncated') }}</p>
    </div>

    <!-- Compact footer -->
    <footer v-if="compact" class="mini-footer">
      <button class="sync-btn" :disabled="busy || loading || !hasChanges" @click="confirmAll" :title="$t('header.confirmAll')">{{ $t('header.confirm') }}</button>
      <template v-if="loading">
        <span class="info">{{ $t('common.loading') }}</span>
      </template>
      <template v-else>
        <span class="stat-add" v-if="cs?.totalAdded">+{{ cs.totalAdded }}</span>
        <span class="stat-del" v-if="cs?.totalRemoved">-{{ cs.totalRemoved }}</span>
        <span class="info">{{ cs?.totalFiles ?? 0 }} {{ $t('status.files') }}</span>
      </template>
    </footer>

    <ContextMenu
      :visible="ctx.visible.value"
      :x="ctx.x.value"
      :y="ctx.y.value"
      :items="ctx.items.value"
      @close="ctx.closeMenu()"
    />
  </section>
</template>

<style scoped>
.filelist {
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.filelist:not(.compact) {
  flex: 1;
  min-width: 0;
  border-right: 1px solid var(--border);
  background: var(--panel);
}
.filelist.compact {
  flex: 1;
  background: transparent;
}
.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}
.title {
  font-size: 12.5px;
  font-weight: 600;
  color: var(--muted);
}
.num {
  font-size: 12px;
  color: var(--faint);
}
.scroll {
  flex: 1;
  overflow-y: auto;
  padding: 6px;
}
.filelist.compact .scroll {
  padding: 4px 8px;
}
.empty {
  color: var(--add);
  text-align: center;
  padding: 28px 10px;
  font-size: 13px;
}
.filelist.compact .empty {
  padding: 20px 10px;
}
.loading {
  color: var(--muted);
  text-align: center;
  padding: 28px 10px;
  font-size: 13px;
  animation: pulse 1.4s ease-in-out infinite;
}
.filelist.compact .loading {
  padding: 20px 10px;
}
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.45; }
}
.item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 6px;
  cursor: pointer;
}
.item:hover {
  background: var(--panel-2);
}
.item.active {
  background: var(--elevated);
}
.st {
  flex-shrink: 0;
  width: 16px;
  text-align: center;
  font-size: 14px;
}
.st.added {
  color: var(--add);
}
.st.deleted {
  color: var(--del);
}
.st.modified {
  color: var(--modified);
}
.path {
  flex: 1;
  min-width: 0;
  font-size: 12.5px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: var(--mono);
}
.dir {
  color: var(--faint);
}
.name {
  color: var(--text-strong);
}
.counts {
  flex-shrink: 0;
  font-size: 12px;
  font-family: var(--mono);
  display: flex;
  gap: 7px;
}
.tag.bin {
  flex-shrink: 0;
}
.trunc {
  color: var(--modified);
  font-size: 12px;
  text-align: center;
  padding: 10px;
}
.mini-footer {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  border-top: 1px solid var(--border);
  flex-shrink: 0;
  font-size: 11px;
  font-family: var(--mono);
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
.mini-footer .info {
  color: var(--muted);
  margin-left: auto;
}
</style>
