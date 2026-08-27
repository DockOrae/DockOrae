<template>
  <div class="relative w-full h-full">
    <svg :viewBox="`0 0 ${W} ${H}`" preserveAspectRatio="none" class="w-full h-full">
      <!-- 网格线 -->
      <line v-for="i in 3" :key="i" x1="0" :y1="(H / 4) * i" :x2="W" :y2="(H / 4) * i" stroke="var(--dm-line)" stroke-width="0.5" stroke-dasharray="3 3" opacity="0.5" />
      <polyline v-if="s1.length > 1" :points="points(s1)" fill="none" :stroke="color1" stroke-width="1.6" stroke-linejoin="round" stroke-linecap="round" />
      <polyline v-if="s2.length > 1" :points="points(s2)" fill="none" :stroke="color2" stroke-width="1.6" stroke-linejoin="round" stroke-linecap="round" />
    </svg>
    <div v-if="!s1.length && !s2.length" class="absolute inset-0 flex items-center justify-center text-[12px] text-muted">
      {{ emptyText }}
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  /** 第一条序列(数值数组) */
  s1: { type: Array, default: () => [] },
  /** 第二条序列 */
  s2: { type: Array, default: () => [] },
  color1: { type: String, default: '#60a5fa' },
  color2: { type: String, default: '#ec4899' },
  emptyText: { type: String, default: '' },
})

const W = 100
const H = 40

function points(series) {
  const n = series.length
  const step = W / (n - 1)
  const max = Math.max(...series, 1)
  return series
    .map((v, i) => `${(i * step).toFixed(2)},${(H - 4 - (v / max) * (H - 8)).toFixed(2)}`)
    .join(' ')
}
</script>
