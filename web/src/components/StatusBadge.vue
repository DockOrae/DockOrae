<template>
  <Badge :variant="variant" :class="cls">
    <span class="w-[0.45rem] h-[0.45rem] rounded-full shrink-0" :style="{ background: color }" />
    {{ label }}
  </Badge>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Badge } from '@/components/ui/badge'

const { t } = useI18n()
const props = defineProps({
  state: { type: String, default: '' },
})

const MAP = {
  running: { color: '#34d399', labelKey: 'status.running', variant: 'success' },
  exited: { color: '#8b93a7', labelKey: 'status.exited', variant: 'default' },
  stopped: { color: '#8b93a7', labelKey: 'status.stopped', variant: 'default' },
  created: { color: '#60a5fa', labelKey: 'status.created', variant: 'info' },
  restarting: { color: '#fbbf24', labelKey: 'status.restarting', variant: 'warning' },
  paused: { color: '#fbbf24', labelKey: 'status.paused', variant: 'warning' },
  dead: { color: '#f87171', labelKey: 'status.dead', variant: 'destructive' },
  removing: { color: '#fbbf24', labelKey: 'status.removing', variant: 'warning' },
  running_full: { color: '#34d399', labelKey: 'status.runningFull', variant: 'success' },
  partial: { color: '#fbbf24', labelKey: 'status.partial', variant: 'warning' },
}

const meta = computed(() => MAP[props.state] || { color: '#8b93a7', labelKey: null, variant: 'default' })
const variant = computed(() => meta.value.variant)
const color = computed(() => meta.value.color)
const cls = computed(() => (props.state ? '' : 'opacity-80'))
const label = computed(() => (meta.value.labelKey ? t(meta.value.labelKey) : props.state || '-'))
</script>
