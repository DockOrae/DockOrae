<template>
  <Teleport to="body">
    <div class="fixed top-4 right-4 z-[70] flex flex-col gap-2 items-end">
      <TransitionGroup name="toast">
        <div
          v-for="t in toastState.items"
          :key="t.id"
          class="toast-in flex items-center gap-2.5 px-4 py-2.5 rounded-xl border shadow-lg shadow-black/30 bg-surface text-sm"
          :style="{ borderColor: borderColor(t.type) }"
        >
          <Icon :name="iconOf(t.type)" :size="15" :style="{ color: colorOf(t.type) }" />
          <span class="text-text">{{ t.message }}</span>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup>
import Icon from './Icon.vue'
import { toastState } from '../toast'

function colorOf(type) {
  return { success: '#34d399', error: '#f87171', info: '#60a5fa' }[type] || '#8b93a7'
}
function borderColor(type) {
  return { success: 'rgba(52,211,153,.35)', error: 'rgba(248,113,113,.35)', info: 'rgba(96,165,250,.35)' }[type] || '#232a38'
}
function iconOf(type) {
  return { success: 'check', error: 'alert', info: 'info' }[type] || 'info'
}
</script>
