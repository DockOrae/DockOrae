<template>
  <svg :width="width" :height="height" :viewBox="`0 0 ${width} ${height}`" class="block">
    <defs>
      <linearGradient :id="gid" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" :stop-color="color" stop-opacity="0.25" />
        <stop offset="100%" :stop-color="color" stop-opacity="0" />
      </linearGradient>
    </defs>
    <polygon v-if="points.length > 1" :points="areaPoints" :fill="`url(#${gid})`" />
    <polyline
      v-if="points.length > 1"
      :points="points"
      fill="none"
      :stroke="color"
      stroke-width="1.6"
      stroke-linejoin="round"
      stroke-linecap="round"
    />
  </svg>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  data: { type: Array, default: () => [] },
  color: { type: String, default: '#ec4899' },
  width: { type: Number, default: 300 },
  height: { type: Number, default: 80 },
  max: { type: Number, default: 0 },
})

const gid = `g${Math.random().toString(36).slice(2, 9)}`

const points = computed(() => {
  const d = props.data
  if (!d.length) return ''
  const w = props.width
  const h = props.height
  const max = props.max > 0 ? props.max : Math.max(...d, 1)
  const min = Math.min(...d, 0)
  const range = max - min || 1
  const step = d.length > 1 ? w / (d.length - 1) : w
  return d
    .map((v, i) => `${(i * step).toFixed(1)},${(h - 3 - ((v - min) / range) * (h - 8)).toFixed(1)}`)
    .join(' ')
})

const areaPoints = computed(() => {
  const p = points.value
  if (!p) return ''
  const last = p.split(' ').pop()
  return `${p} ${last.split(',')[0]},${props.height} 0,${props.height}`
})
</script>
