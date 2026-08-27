<template>
  <div class="flex flex-col items-center">
    <div class="relative" :style="{ width: size + 'px', height: size + 'px' }">
      <svg :width="size" :height="size" viewBox="0 0 120 120">
        <!-- 背景环 -->
        <circle cx="60" cy="60" r="50" fill="none" stroke="var(--dm-surface2)" stroke-width="10" />
        <!-- 进度环 -->
        <circle
          cx="60"
          cy="60"
          r="50"
          fill="none"
          :stroke="color"
          stroke-width="10"
          stroke-linecap="round"
          :stroke-dasharray="circumference"
          :stroke-dashoffset="circumference * (1 - clamped / 100)"
          transform="rotate(-90 60 60)"
          class="gauge-bar"
        />
      </svg>
      <div class="absolute inset-0 flex flex-col items-center justify-center">
        <div class="text-xl font-bold leading-none" :style="{ color }">{{ display }}</div>
        <div v-if="unit" class="text-[10px] text-muted mt-0.5">{{ unit }}</div>
      </div>
    </div>
    <div class="text-[13px] font-semibold mt-2">{{ label }}</div>
    <div v-if="sub" class="text-[11px] text-muted mt-0.5 truncate max-w-[130px]">{{ sub }}</div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  /** 0-100 的百分比 */
  value: { type: Number, default: 0 },
  label: { type: String, default: '' },
  sub: { type: String, default: '' },
  unit: { type: String, default: '%' },
  color: { type: String, default: '#60a5fa' },
  size: { type: Number, default: 120 },
  /** 中心显示的自定义文本(默认取 value) */
  display: { type: [String, Number], default: null },
  /** 小数位 */
  digits: { type: Number, default: 1 },
})

const circumference = Math.round(2 * Math.PI * 50 * 100) / 100
const clamped = computed(() => Math.min(100, Math.max(0, props.value || 0)))
const center = computed(() => {
  if (props.display !== null && props.display !== '') return props.display
  return props.value == null ? '-' : props.value.toFixed(props.digits)
})
</script>

<style scoped>
.gauge-bar {
  transition: stroke-dashoffset 0.6s ease;
}
</style>
