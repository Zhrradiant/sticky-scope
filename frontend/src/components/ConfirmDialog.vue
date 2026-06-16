<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  title: string
  message: string
  mode?: 'confirm' | 'alert'
  confirmText?: string
  cancelText?: string
}>(), {
  mode: 'confirm',
})

const emit = defineEmits<{
  (e: 'confirm'): void
  (e: 'cancel'): void
}>()

const isAlert = computed(() => props.mode === 'alert')

function onOverlayClick() {
  // In alert mode the overlay also dismisses (same as OK).
  if (isAlert.value) {
    emit('confirm')
  } else {
    emit('cancel')
  }
}
</script>

<template>
  <div class="overlay" @click.self="onOverlayClick">
    <div class="dialog">
      <h3>{{ title }}</h3>
      <p>{{ message }}</p>
      <div class="btns">
        <button v-if="!isAlert" class="sm" @click="emit('cancel')">
          {{ cancelText || t('common.cancel') }}
        </button>
        <button
          class="sm"
          :class="isAlert ? '' : 'danger'"
          @click="emit('confirm')"
        >
          {{ isAlert ? t('common.ok') : (confirmText || t('common.confirm')) }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  z-index: 200;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(61, 50, 38, 0.2);
  backdrop-filter: blur(2px);
}
.dialog {
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: 16px;
  box-shadow: 0 8px 32px rgba(61, 50, 38, 0.16);
  width: 320px;
  max-width: calc(100vw - 40px);
  padding: 20px 22px 18px;
}
.dialog h3 {
  margin: 0 0 10px;
  font-size: 15px;
  font-weight: 700;
  color: var(--text-strong);
}
.dialog p {
  margin: 0 0 20px;
  font-size: 13px;
  color: var(--text);
  line-height: 1.5;
}
.btns {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
