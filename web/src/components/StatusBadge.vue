<template>
  <span class="badge" :style="style">
    <span class="dot" :style="{ background: color }" />
    {{ label }}
  </span>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const props = defineProps({
  state: { type: String, default: '' },
})

const MAP = {
  running: { color: '#34d399', labelKey: 'status.running' },
  exited: { color: '#8b93a7', labelKey: 'status.exited' },
  stopped: { color: '#8b93a7', labelKey: 'status.stopped' },
  created: { color: '#60a5fa', labelKey: 'status.created' },
  restarting: { color: '#fbbf24', labelKey: 'status.restarting' },
  paused: { color: '#fbbf24', labelKey: 'status.paused' },
  dead: { color: '#f87171', labelKey: 'status.dead' },
  removing: { color: '#fbbf24', labelKey: 'status.removing' },
  running_full: { color: '#34d399', labelKey: 'status.runningFull' },
  partial: { color: '#fbbf24', labelKey: 'status.partial' },
}

const meta = computed(() => MAP[props.state] || { color: '#8b93a7', labelKey: null })
const style = computed(() => ({
  color: meta.value.color,
  background: hexToRgba(meta.value.color, 0.12),
  border: `1px solid ${hexToRgba(meta.value.color, 0.3)}`,
}))
const label = computed(() => (meta.value.labelKey ? t(meta.value.labelKey) : props.state || '-'))

function hexToRgba(hex, a) {
  const n = parseInt(hex.slice(1), 16)
  return `rgba(${(n >> 16) & 255},${(n >> 8) & 255},${n & 255},${a})`
}
</script>
