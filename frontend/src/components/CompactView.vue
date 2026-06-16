<script setup lang="ts">
import FileList from './FileList.vue'

const emit = defineEmits<{ (e: 'select', path: string): void }>()
</script>

<template>
  <div class="compact-view">
    <FileList compact @select="emit('select', $event)" />
  </div>
</template>

<style scoped>
.compact-view {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
</style>
