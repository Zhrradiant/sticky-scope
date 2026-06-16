<script setup lang="ts">
import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useChangesStore } from '@/stores/changes'

const changes = useChangesStore()
const { fileDiff, diffLoading, selectedPath } = storeToRefs(changes)

const fd = computed(() => fileDiff.value)

function sign(kind: string): string {
  return kind === 'add' ? '+' : kind === 'del' ? '-' : ' '
}
</script>

<template>
  <section class="diff">
    <div v-if="!selectedPath" class="placeholder">
      <div class="ph-icon">⌗</div>
      <p>{{ $t('diff.placeholder') }}</p>
    </div>

    <template v-else>
      <header class="head">
        <span class="fpath" :title="fd?.path">{{ fd?.path }}</span>
        <span class="meta">
          <span v-if="fd?.added" class="stat-add">+{{ fd.added }}</span>
          <span v-if="fd?.removed" class="stat-del">-{{ fd.removed }}</span>
        </span>
      </header>

      <div class="scroll">
        <p v-if="diffLoading" class="note">{{ $t('common.loading') }}</p>

        <template v-else-if="fd">
          <p v-if="fd.binary" class="note">⛓ {{ $t('diff.binary') }}<span v-if="fd.message"> · {{ fd.message }}</span></p>
          <p v-else-if="fd.message && !fd.hunks.length" class="note">{{ fd.message }}</p>
          <p v-else-if="!fd.hunks.length" class="note">{{ $t('diff.noContent') }}</p>

          <template v-else>
            <p v-if="fd.truncated" class="banner">{{ $t('diff.truncatedBanner') }}</p>
            <div class="code">
              <template v-for="(h, hi) in fd.hunks" :key="hi">
                <div class="hunk-h">{{ h.header }}</div>
                <div v-for="(l, li) in h.lines" :key="hi + '-' + li" class="line" :class="l.kind">
                  <span class="ln old">{{ l.oldLine || '' }}</span>
                  <span class="ln new">{{ l.newLine || '' }}</span>
                  <span class="sign">{{ sign(l.kind) }}</span>
                  <span class="content">{{ l.content }}</span>
                </div>
              </template>
            </div>
          </template>
        </template>
      </div>
    </template>
  </section>
</template>

<style scoped>
.diff {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg);
}
.placeholder {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--faint);
}
.ph-icon {
  font-size: 44px;
  margin-bottom: 10px;
  opacity: 0.5;
}
.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}
.fpath {
  font-family: var(--mono);
  font-size: 12.5px;
  color: var(--text-strong);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.meta {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
  font-family: var(--mono);
  font-size: 12px;
}
.scroll {
  flex: 1;
  overflow: auto;
}
.note {
  color: var(--muted);
  padding: 20px;
  font-size: 13px;
}
.banner {
  margin: 0;
  padding: 7px 16px;
  background: rgba(184, 134, 11, 0.12);
  color: var(--modified);
  font-size: 12px;
  border-bottom: 1px solid var(--border);
}
.code {
  font-family: var(--mono);
  font-size: 13px;
  line-height: 1.6;
  min-width: max-content;
  padding-bottom: 30px;
}
.hunk-h {
  color: var(--accent);
  background: rgba(200, 132, 60, 0.07);
  padding: 2px 12px;
  user-select: text;
}
.line {
  display: flex;
  white-space: pre;
}
.line.add {
  background: var(--add-bg);
}
.line.del {
  background: var(--del-bg);
}
.ln {
  flex-shrink: 0;
  width: 48px;
  text-align: right;
  padding: 0 8px;
  color: var(--faint);
  user-select: none;
  border-right: 1px solid var(--border);
}
.line.add .ln.new {
  background: var(--add-gutter);
}
.line.del .ln.old {
  background: var(--del-gutter);
}
.sign {
  flex-shrink: 0;
  width: 18px;
  text-align: center;
  user-select: none;
}
.line.add .sign {
  color: var(--add);
}
.line.del .sign {
  color: var(--del);
}
.content {
  flex: 1;
  padding-right: 16px;
  user-select: text;
}
</style>
