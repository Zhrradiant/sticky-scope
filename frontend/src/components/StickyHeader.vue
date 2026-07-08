<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useProjectsStore } from '@/stores/projects'
import { useChangesStore } from '@/stores/changes'
import { useUiStore } from '@/stores/ui'
import { App } from '@/api'
import { WindowMinimise, Quit } from '../../wailsjs/runtime/runtime'
import ConfirmDialog from './ConfirmDialog.vue'
import { currentLocale, setLocale, i18n } from '@/i18n'
import type { Lang } from '@/types'
import compressIcon from '@/assets/compress-solid-full.svg'
import expandIcon from '@/assets/expand-solid-full.svg'

const projects = useProjectsStore()
const changes = useChangesStore()
const ui = useUiStore()
const { projects: list, activeId, addingProject, addProgress } = storeToRefs(projects)
const { isExpanded, isCollapsed } = storeToRefs(ui)
const { busy } = storeToRefs(changes)

const active = computed(() => list.value.find((p) => p.id === activeId.value) ?? null)
const projDisplayName = computed(() => {
  const name = active.value?.name
  if (!name) return i18n.global.t('header.noProject')
  return name.length > 12 ? name.slice(0, 12) + '…' : name
})
const hasChanges = computed(() => (changes.changeSetFor(projects.activeId)?.totalFiles ?? 0) > 0)

// ---- project dropdown ----
const dropdownOpen = ref(false)
const projRef = ref<HTMLElement | null>(null)

function toggleDropdown() {
  dropdownOpen.value = !dropdownOpen.value
}

function onDocClick(e: MouseEvent) {
  if (projRef.value && !projRef.value.contains(e.target as Node)) {
    dropdownOpen.value = false
  }
  // Close settings panel when clicking outside both the panel and the ⚙ trigger button.
  if (settOpen.value) {
    const inPanel = settPanelRef.value?.contains(e.target as Node)
    const inBtn = settBtnRef.value?.contains(e.target as Node)
    if (!inPanel && !inBtn) settOpen.value = false
  }
  if (moreRef.value && !moreRef.value.contains(e.target as Node)) {
    moreOpen.value = false
  }
}

onMounted(() => document.addEventListener('click', onDocClick))
onUnmounted(() => document.removeEventListener('click', onDocClick))

function selectProject(id: string) {
  projects.select(id)
  dropdownOpen.value = false
}

// ---- confirm / expand ----
async function confirm() {
  const id = projects.activeId
  if (!id) return
  await changes.confirmAll(id)
}

function toggleExpand() {
  const expand = !isExpanded.value
  ui.setExpanded(expand)
  App.SetCompactMode(expand)
}

function toggleCollapsed() {
  const collapse = !isCollapsed.value
  // Close any open panels before collapsing — they'd overflow the tiny window.
  dropdownOpen.value = false
  settOpen.value = false
  moreOpen.value = false
  ui.setCollapsed(collapse)
  App.SetCollapsedMode(collapse)
  if (!collapse) {
    // Exiting collapsed mode: restore previous window size
    App.SetCompactMode(isExpanded.value)
  }
}

async function spawnStickyForProject(projectPath: string) {
  dropdownOpen.value = false
  await App.SpawnStickyNote(projectPath)
}

async function addProject() {
  await projects.pickAndAdd()
  dropdownOpen.value = false
}

// ---- remove project ----
const removePrompt = ref(false)

// ---- need-project alert (alert-mode dialog) ----
const needProjectAlert = ref(false)

async function removeCurrentProject() {
  const p = active.value
  if (!p) return
  dropdownOpen.value = false
  removePrompt.value = true
}

async function onRemoveConfirmed() {
  removePrompt.value = false
  const p = active.value
  if (!p) return
  await projects.remove(p.id)
}

function onRemoveCancelled() {
  removePrompt.value = false
}

// ---- deep rescan ----
function doDeepRescan() {
  if (projects.activeId) changes.deepRescan(projects.activeId)
}

// ---- more menu (compact mode) ----
const moreOpen = ref(false)
const moreRef = ref<HTMLElement | null>(null)

function toggleMore() {
  moreOpen.value = !moreOpen.value
}

function onMoreDeepRescan() {
  moreOpen.value = false
  doDeepRescan()
}

function onMoreSettings() {
  moreOpen.value = false
  openSettings()
}

function onMoreExpand() {
  moreOpen.value = false
  toggleExpand()
}

// ---- settings panel ----
const settOpen = ref(false)
const settPanelRef = ref<HTMLElement | null>(null)
const settBtnRef = ref<HTMLElement | null>(null)
const useGitignore = ref(true)
const extraPatterns = ref('')
const defaultPatterns = ref('')
const showDefaultPatterns = ref(false)
const locale = ref<Lang>(currentLocale())

// Load the global shared default patterns from the backend into the textarea.
async function loadDefaultPatterns() {
  try {
    const s = await App.GetSettings()
    defaultPatterns.value = (s.defaultPatterns ?? []).join('\n')
  } catch {
    defaultPatterns.value = ''
  }
}

// Load every field in the settings panel from current source-of-truth. Called
// on every open so the panel always reflects the latest persisted state
// (after a save, a reset, or a project switch) instead of a stale snapshot.
async function syncSettingsState() {
  const p = active.value
  if (p) {
    useGitignore.value = p.useGitignore
    extraPatterns.value = (p.ignore ?? []).join('\n')
  }
  await loadDefaultPatterns()
}

watch(() => active.value, () => {
  // If the panel is open while the active project changes, refresh the
  // per-project fields live so the user never sees stale content.
  if (settOpen.value && active.value) {
    useGitignore.value = active.value.useGitignore
    extraPatterns.value = (active.value.ignore ?? []).join('\n')
  }
})

function openSettings() {
  settOpen.value = true
  void syncSettingsState()
}

async function applySettings() {
  const id = projects.activeId
  if (!id) {
    needProjectAlert.value = true
    return
  }
  const extras = extraPatterns.value.split('\n').map(s => s.trim()).filter(s => s.length > 0)
  const defs = defaultPatterns.value.split('\n').map(s => s.trim()).filter(s => s.length > 0)
  // Persist per-project extra patterns and the global shared defaults together.
  await projects.updateIgnore(id, extras, useGitignore.value)
  await App.UpdateDefaultPatterns(defs)
  settOpen.value = false
}

// Restore the global shared default patterns to the factory preset and refresh
// the textarea immediately so the new content is visible without reopening.
async function resetDefaultPatterns() {
  try {
    await App.ResetDefaultPatterns()
  } finally {
    // Always reload from the backend so the textarea converges to the truth,
    // whether the reset succeeded or threw.
    await loadDefaultPatterns()
  }
}

function switchLocale(l: Lang) {
  locale.value = l
  setLocale(l)
}
</script>

<template>
  <header class="sticky-header drag-region">
    <div class="left-group no-drag">
      <!-- Collapse toggle -->
      <button
        v-if="active"
        class="collapse-btn"
        @click="toggleCollapsed"
        :title="isCollapsed ? $t('header.expandFromTray') : $t('header.collapseToTray')"
      >
        <img :src="isCollapsed ? compressIcon : expandIcon" class="collapse-icon" alt="" />
      </button>

      <!-- Project dropdown -->
      <div ref="projRef" class="proj-drop">
        <button class="proj-btn" @click="isCollapsed ? null : toggleDropdown()" :disabled="!list.length || isCollapsed">
          <span class="proj-name">{{ projDisplayName }}</span>
          <span class="arrow">▾</span>
        </button>
        <div v-if="dropdownOpen && !isCollapsed" class="dropdown-menu">
          <div class="proj-list-scroll">
            <div
              v-for="p in list"
              :key="p.id"
              class="dropdown-item proj-row"
              :class="{ active: p.id === activeId }"
              @click="selectProject(p.id)"
            >
              <span class="dot" :class="p.available ? 'on' : 'bad'"></span>
              <span class="item-name">{{ p.name }}</span>
              <button
                class="share-btn"
                @click.stop="spawnStickyForProject(p.path)"
                :title="$t('header.pinSticky')"
              >↗</button>
            </div>
          </div>
          <div class="dropdown-sep"></div>
          <div class="dropdown-item" @click="!addingProject && addProject()" :class="{ disabled: addingProject }">
            <template v-if="addingProject && addProgress">
              <div class="add-prog">
                <span class="add-prog-label">{{ $t('header.adding') }}</span>
                <div class="add-prog-bar">
                  <div class="add-prog-fill" :style="{ width: addProgress.total > 0 ? (addProgress.current / addProgress.total * 100) + '%' : '100%' }"></div>
                </div>
                <span class="add-prog-num" v-if="addProgress.total > 0">{{ addProgress.current }}/{{ addProgress.total }}</span>
              </div>
            </template>
            <template v-else>{{ $t('header.addProject') }}</template>
          </div>
          <div class="dropdown-sep" v-if="active"></div>
          <div v-if="active" class="dropdown-item danger" @click="removeCurrentProject">
            ✕ {{ $t('header.removeProject') }}
          </div>
        </div>
      </div>
    </div>

    <!-- Actions -->
    <div class="actions no-drag">
      <button v-if="isExpanded && !isCollapsed" class="sm primary confirm-btn" :disabled="busy || !hasChanges" @click="confirm" :title="$t('header.confirmAll')">
        {{ $t('header.confirm') }}
      </button>

      <!-- Expanded mode: individual action buttons -->
      <template v-if="isExpanded && !isCollapsed">
        <button class="sm ghost expand-btn" @click="toggleExpand" :title="$t('header.collapse')">↙</button>
        <button class="sm ghost icon-btn" :disabled="busy || !active" @click="doDeepRescan" :title="$t('header.deepRescan')">⟳</button>
        <button ref="settBtnRef" class="sm ghost icon-btn" @click.stop="settOpen ? (settOpen = false) : openSettings()" :title="$t('header.settings')">⚙</button>
      </template>

      <!-- Compact mode: "···" overflow menu -->
      <div v-else-if="!isCollapsed" ref="moreRef" class="more-drop no-drag">
        <button class="sm ghost icon-btn more-btn" @click="toggleMore" :title="$t('header.more')">···</button>
        <div v-if="moreOpen" class="dropdown-menu more-menu">
          <div class="dropdown-item" @click.stop="onMoreDeepRescan">
            <span class="more-icon">⟳</span>
            <span>{{ $t('header.deepRescan') }}</span>
          </div>
          <div class="dropdown-item" @click.stop="onMoreSettings">
            <span class="more-icon">⚙</span>
            <span>{{ $t('header.settings') }}</span>
          </div>
          <template v-if="active">
            <div class="dropdown-sep"></div>
            <div class="dropdown-item" @click.stop="onMoreExpand">
              <span class="more-icon">↗</span>
              <span>{{ $t('header.expand') }}</span>
            </div>
          </template>
        </div>
      </div>

      <!-- Window controls -->
      <button v-if="!isCollapsed" class="sm ghost icon-btn" @click="WindowMinimise" :title="$t('header.minimize')">─</button>
      <button v-if="!isCollapsed" class="sm ghost icon-btn win-close" @click="Quit" :title="$t('header.close')">✕</button>
    </div>

    <!-- Settings panel (shared between expanded and compact modes) -->
    <div v-if="settOpen" ref="settPanelRef" class="sett-panel-global no-drag">
      <h4>{{ $t('settings.title') }}</h4>
      <!-- Language toggle -->
      <div class="lang-row">
        <span class="lbl-lang">{{ $t('settings.language') }}</span>
        <div class="lang-toggles">
          <button class="lang-btn" :class="{ active: locale === 'zh' }" @click="switchLocale('zh')">{{ $t('settings.langZh') }}</button>
          <button class="lang-btn" :class="{ active: locale === 'en' }" @click="switchLocale('en')">{{ $t('settings.langEn') }}</button>
        </div>
      </div>
      <label class="chk">
        <input type="checkbox" v-model="useGitignore" />
        <span>{{ $t('settings.useGitignore') }}</span>
      </label>
      <p class="lbl">{{ $t('settings.extraPatterns') }}</p>
      <textarea v-model="extraPatterns" rows="4" :placeholder="$t('settings.extraPlaceholder')"></textarea>
      <p class="lbl toggle-lbl" @click="showDefaultPatterns = !showDefaultPatterns">
        <span class="toggle-arrow" :class="{ open: showDefaultPatterns }">▸</span>
        {{ $t('settings.defaultPatterns') }}
        <button class="sm ghost icon-btn reset-defaults-btn" @click.stop="resetDefaultPatterns" :title="$t('settings.resetDefaults')" :aria-label="$t('settings.resetDefaults')">⟲</button>
      </p>
      <textarea v-if="showDefaultPatterns" v-model="defaultPatterns" rows="5" :placeholder="$t('settings.defaultPlaceholder')"></textarea>
      <div class="sett-actions">
        <button class="sm" @click="settOpen = false">{{ $t('common.cancel') }}</button>
        <button class="sm primary" @click="applySettings">{{ $t('common.save') }}</button>
      </div>
    </div>

    <!-- Need-project alert -->
    <ConfirmDialog
      v-if="needProjectAlert"
      mode="alert"
      :title="$t('common.notice')"
      :message="$t('settings.needProject')"
      @confirm="needProjectAlert = false"
    />

    <!-- Remove project confirmation dialog -->
    <ConfirmDialog
      v-if="removePrompt"
      :title="$t('header.removeProject')"
      :message="$t('header.confirmRemove')"
      :confirm-text="$t('header.removeProject')"
      :cancel-text="$t('common.cancel')"
      @confirm="onRemoveConfirmed"
      @cancel="onRemoveCancelled"
    />
  </header>
</template>

<style scoped>
.sticky-header {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: var(--sticky-header-height);
  padding: 0 12px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--border);
  background: var(--bg);
}
.proj-drop { position: relative; }
.left-group {
  display: flex;
  align-items: center;
  gap: 2px;
}
.collapse-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  padding: 0;
  cursor: pointer;
  flex-shrink: 0;
  transform: translateY(2px);
}
.collapse-btn:hover {
  background: transparent;
  border-color: transparent;
}
.collapse-icon {
  width: 14px;
  height: 14px;
  opacity: 0.4;
  transform: scale(0.8);
  transition: opacity 0.15s, transform 0.15s;
}
.collapse-btn:hover .collapse-icon {
  opacity: 0.7;
  transform: scale(0.85);
}
.proj-btn {
  display: flex; align-items: center; gap: 6px;
  background: transparent; border: 1px solid transparent; border-radius: 10px;
  padding: 5px 10px; font-size: 13px; font-weight: 600; color: var(--text-strong); cursor: pointer;
}
.proj-btn:hover { background: var(--panel-2); border-color: var(--border); }
.proj-name { max-width: 160px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.arrow { font-size: 10px; color: var(--muted); }

.dropdown-menu {
  position: absolute; top: 100%; left: 0; margin-top: 4px;
  background: var(--elevated); border: 1px solid var(--border-strong); border-radius: 12px;
  box-shadow: 0 4px 16px rgba(61,50,38,0.12); min-width: 200px; z-index: 100; padding: 6px;
}
.proj-list-scroll { max-height: 280px; overflow-y: auto; }
.dropdown-item {
  display: flex; align-items: center; gap: 8px; padding: 8px 12px; border-radius: 8px;
  cursor: pointer; font-size: 13px; color: var(--text); transition: background 0.1s;
}
.dropdown-item:hover { background: var(--panel-2); }
.dropdown-item.active { color: var(--accent); font-weight: 600; }
.dropdown-item .item-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; }
.dropdown-sep { height: 1px; background: var(--border); margin: 4px 8px; }
.dropdown-item.danger { color: var(--del); }
.dropdown-item.danger:hover { background: rgba(192, 91, 77, 0.08); }
.dropdown-item.disabled { opacity: 0.5; cursor: default; }
.dropdown-item.disabled:hover { background: transparent; }

.proj-row { cursor: pointer; }
.share-btn {
  opacity: 0;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 2px 6px;
  font-size: 13px;
  color: var(--accent);
  cursor: pointer;
  transition: opacity 0.12s, background 0.12s;
  flex-shrink: 0;
  line-height: 1;
}
.share-btn:hover {
  background: var(--accent);
  color: #FFF;
  border-color: var(--accent);
}
.dropdown-item:hover .share-btn {
  opacity: 1;
}

.dot { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }
.dot.on { background: var(--add); box-shadow: 0 0 6px var(--add); }
.dot.bad { background: var(--del); }

.actions { display: flex; align-items: center; gap: 5px; }
.confirm-btn { font-weight: 600; }
.icon-btn { font-size: 15px; padding: 4px 8px; color: var(--muted); }
.expand-btn { font-size: 16px; color: var(--accent); padding: 4px 8px; }

/* ---- window controls ---- */
.win-close:hover:not(:disabled) {
  color: var(--del);
}

/* ---- more menu (compact mode) ---- */
.more-drop { position: relative; }
.more-btn { font-size: 16px; letter-spacing: 1px; }
.more-menu { right: 0; left: auto; min-width: 180px; }
.more-icon { font-size: 14px; width: 20px; text-align: center; flex-shrink: 0; }

/* ---- settings panel (shared) ---- */
.sett-panel-global {
  position: absolute; top: var(--sticky-header-height); right: 12px; margin-top: 4px;
  background: var(--elevated); border: 1px solid var(--border-strong); border-radius: 14px;
  box-shadow: 0 4px 20px rgba(61,50,38,0.14); width: 280px; z-index: 100; padding: 14px 16px;
}
.sett-panel-global h4 { margin: 0 0 12px; font-size: 14px; color: var(--text-strong); }
.chk { display: flex; align-items: center; gap: 8px; cursor: pointer; color: var(--text); font-size: 13px; margin-bottom: 12px; }
.lbl { font-size: 12px; color: var(--muted); margin: 0 0 4px; }
textarea {
  width: 100%; box-sizing: border-box;
  font-family: var(--mono); font-size: 12px;
  background: var(--panel-2); color: var(--text); border: 1px solid var(--border);
  border-radius: 8px; padding: 8px 10px; resize: vertical; outline: none;
}
textarea:focus { border-color: var(--accent); }
.sett-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 10px; }
.lang-row { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.lbl-lang { font-size: 12px; color: var(--muted); }
.lang-toggles { display: flex; gap: 4px; }
.lang-btn {
  font-size: 12px; padding: 3px 10px; border-radius: 8px;
  border: 1px solid var(--border); background: transparent; color: var(--text); cursor: pointer;
}
.lang-btn.active { background: var(--accent); color: #FFF; border-color: var(--accent); }
.toggle-lbl { cursor: pointer; user-select: none; display: flex; align-items: center; gap: 4px; }
.toggle-lbl:hover { color: var(--accent); }
.toggle-arrow { display: inline-block; font-size: 10px; width: 12px; transition: transform 0.15s; }
.toggle-arrow.open { transform: rotate(90deg); }
.reset-defaults-btn { margin-left: auto; font-size: 15px; line-height: 1; padding: 2px 6px; }
.reset-defaults-btn:hover { color: var(--accent); }
</style>
