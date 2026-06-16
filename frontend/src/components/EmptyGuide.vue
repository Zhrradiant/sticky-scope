<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useProjectsStore } from '@/stores/projects'
import logoImg from '@/assets/images/logo-universal.png'

const projects = useProjectsStore()
const { addingProject, addProgress } = storeToRefs(projects)
</script>

<template>
  <div class="guide">
    <div class="guide-card">
      <img class="logo" :src="logoImg" :alt="$t('app.name')" />
      <button v-if="!addingProject" class="primary" @click="projects.pickAndAdd()">+ {{ $t('sidebar.addProject') }}</button>
      <div v-if="addingProject && addProgress" class="add-prog add-prog--standalone">
        <span class="add-prog-label">{{ $t('header.adding') }}</span>
        <div class="add-prog-bar">
          <div class="add-prog-fill" :style="{ width: addProgress.total > 0 ? (addProgress.current / addProgress.total * 100) + '%' : '100%' }"></div>
        </div>
        <span class="add-prog-num" v-if="addProgress.total > 0">{{ addProgress.current }}/{{ addProgress.total }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.guide {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}
.guide-card {
  text-align: center;
  padding: 20px;
}
.guide-card .logo {
  display: block;
  width: 96px;
  height: 96px;
  margin: 0 auto 18px;
  object-fit: contain;
}

/* standalone progress bar (outside button, matches dropdown style) */
.add-prog--standalone {
  margin-top: 12px;
  width: 220px;
  margin-left: auto;
  margin-right: auto;
}
</style>
