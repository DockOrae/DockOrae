<template>
  <div class="space-y-4">
    <div class="grid grid-cols-2 lg:grid-cols-4 gap-3">
      <Card class="p-4">
        <div class="text-[11px] text-muted mb-1">{{ t('stats.cpuUsage') }}</div>
        <div class="text-2xl font-bold" :style="{ color: cpuColor }">{{ last.cpu_pct ?? 0 }}%</div>
      </Card>
      <Card class="p-4">
        <div class="text-[11px] text-muted mb-1">{{ t('stats.memory') }}</div>
        <div class="text-2xl font-bold text-emerald-400">{{ fmt(last.mem_usage) }}</div>
        <div class="text-[11px] text-muted mt-0.5">/ {{ fmt(last.mem_limit) }} · {{ last.mem_pct ?? 0 }}%</div>
      </Card>
      <Card class="p-4">
        <div class="text-[11px] text-muted mb-1">{{ t('stats.netRx') }}</div>
        <div class="text-2xl font-bold text-sky-400">{{ rate(last.net_rx) }}</div>
      </Card>
      <Card class="p-4">
        <div class="text-[11px] text-muted mb-1">{{ t('stats.netTx') }}</div>
        <div class="text-2xl font-bold text-purple-400">{{ rate(last.net_tx) }}</div>
      </Card>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-3">
      <Card class="p-4">
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs text-muted">{{ t('stats.cpuTrend') }}</span>
          <span class="text-xs font-semibold text-brand">{{ last.cpu_pct ?? 0 }}%</span>
        </div>
        <Sparkline :data="cpu" color="#ec4899" :height="90" :max="100" />
      </Card>
      <Card class="p-4">
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs text-muted">{{ t('stats.memTrend') }}</span>
          <span class="text-xs font-semibold text-emerald-400">{{ last.mem_pct ?? 0 }}%</span>
        </div>
        <Sparkline :data="mem" color="#34d399" :height="90" :max="100" />
      </Card>
      <Card class="p-4">
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs text-muted">{{ t('stats.netTrend') }}</span>
          <span class="text-xs text-muted">RX {{ rate(last.net_rx) }} · TX {{ rate(last.net_tx) }}</span>
        </div>
        <Sparkline :data="net" color="#60a5fa" :height="90" />
      </Card>
    </div>

    <p v-if="!connected" class="text-center text-xs text-muted">{{ t('stats.connecting') }}</p>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Sparkline from '../../components/Sparkline.vue'
import { wsUrl } from '../../api'
import { formatBytes } from '../../util'
import { Card } from '@/components/ui/card'

const { t } = useI18n()
const props = defineProps({ id: { type: String, required: true } })

const cpu = ref([])
const mem = ref([])
const net = ref([])
const last = ref({})
const connected = ref(false)
let ws = null
let prev = null
let timer = null

const rate = (v) => (v == null ? '-' : formatBytes(v) + '/s')
const fmt = (v) => (v == null ? '-' : formatBytes(v))
const cpuColor = computed(() => {
  const v = last.value.cpu_pct || 0
  return v > 80 ? '#f87171' : v > 50 ? '#fbbf24' : '#34d399'
})

function connect() {
  try {
    ws = new WebSocket(wsUrl(`/containers/${props.id}/stats`))
    ws.onopen = () => (connected.value = true)
    ws.onmessage = (ev) => {
      try {
        const s = JSON.parse(ev.data)
        last.value = s
        const now = Date.now()
        if (prev) {
          const dt = (now - prev.t) / 1000
          const rxRate = dt > 0 ? (s.net_rx - prev.net_rx) / dt : 0
          const txRate = dt > 0 ? (s.net_tx - prev.net_tx) / dt : 0
          const rateSum = (rxRate + txRate) / 2
          net.value.push(rateSum > 0 ? rateSum : 0)
        } else {
          net.value.push(0)
        }
        cpu.value.push(s.cpu_pct)
        mem.value.push(s.mem_pct)
        for (const a of [cpu, mem, net]) if (a.value.length > 60) a.value.shift()
        prev = { t: now, net_rx: s.net_rx, net_tx: s.net_tx }
      } catch { /* ignore */ }
    }
    ws.onclose = () => {
      connected.value = false
      ws = null
    }
    ws.onerror = () => ws?.close()
  } catch { /* ignore */ }
}

onMounted(() => {
  connect()
  timer = setInterval(() => {
    if (!ws) connect()
  }, 5000)
})

onBeforeUnmount(() => {
  clearInterval(timer)
  ws?.close()
})
</script>
