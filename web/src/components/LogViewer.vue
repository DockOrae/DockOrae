<template>
  <div class="flex flex-col h-full">
    <div class="flex items-center gap-2 mb-2 flex-wrap">
      <Select v-model="tail" class="!w-28" :disabled="connected">
        <SelectTrigger class="!h-8 !text-xs !py-1"><SelectValue /></SelectTrigger>
        <SelectContent>
          <SelectItem :value="String(200)">{{ t('logViewer.lastLines', { n: 200 }) }}</SelectItem>
          <SelectItem :value="String(500)">{{ t('logViewer.lastLines', { n: 500 }) }}</SelectItem>
          <SelectItem :value="String(1000)">{{ t('logViewer.lastLines', { n: 1000 }) }}</SelectItem>
          <SelectItem :value="String(5000)">{{ t('logViewer.lastLines', { n: 5000 }) }}</SelectItem>
        </SelectContent>
      </Select>
      <Button variant="ghost" size="sm" @click="toggleFollow">
        <Icon :name="connected ? 'pause' : 'play'" size="13" />
        {{ connected ? t('logViewer.pause') : t('logViewer.resume') }}
      </Button>
      <Button variant="ghost" size="sm" @click="clear">
        <Icon name="x" size="13" /> {{ t('logViewer.clear') }}
      </Button>
      <Button variant="ghost" size="sm" @click="download">
        <Icon name="download" size="13" /> {{ t('logViewer.download') }}
      </Button>
      <label class="flex items-center gap-1.5 text-xs text-muted ml-auto cursor-pointer select-none">
        <input v-model="autoScroll" type="checkbox" class="accent-[#ec4899]" />
        {{ t('logViewer.autoScroll') }}
      </label>
    </div>
    <div
      ref="box"
      class="flex-1 code-panel border border-line rounded-lg overflow-auto p-3 font-mono text-[12.5px] leading-relaxed min-h-[200px]"
      @scroll="onScroll"
    >
      <template v-if="!lines.length">
        <p class="text-muted/70">{{ connected ? t('logViewer.waiting') : t('logViewer.paused') }}</p>
      </template>
      <div v-for="(l, i) in lines" :key="i" class="whitespace-pre-wrap break-all" v-html="l" />
    </div>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from './Icon.vue'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { wsUrl } from '../api'

const { t } = useI18n()
const props = defineProps({
  stream: { type: String, required: true }, // 相对路径,如 /containers/abc/logs
  follow: { type: Boolean, default: true },
})

const box = ref(null)
const lines = ref([])
const plain = ref([])
const connected = ref(false)
const autoScroll = ref(true)
const tail = ref('500')
let ws = null
let seq = 0

function connect() {
  disconnect()
  const url = wsUrl(props.stream) + `&tail=${tail.value}`
  ws = new WebSocket(url)
  ws.onopen = () => {
    connected.value = true
    if (props.follow) autoScroll.value = true
  }
  ws.onmessage = (ev) => {
    const text = typeof ev.data === 'string' ? ev.data : ''
    const parts = text.split('\n')
    if (parts.length) {
      for (const p of parts) {
        lines.value.push(ansiToHtml(p))
        plain.value.push(p)
      }
      if (lines.value.length > 6000) {
        lines.value.splice(0, lines.value.length - 6000)
        plain.value.splice(0, plain.value.length - 6000)
      }
      if (autoScroll.value) scrollBottom()
    }
  }
  ws.onclose = () => {
    connected.value = false
    ws = null
  }
  ws.onerror = () => ws?.close()
}

function disconnect() {
  if (ws) {
    ws.onclose = null
    ws.close()
    ws = null
  }
  connected.value = false
}

function toggleFollow() {
  if (connected.value) disconnect()
  else connect()
}

function clear() {
  lines.value = []
  plain.value = []
}

function download() {
  const blob = new Blob([plain.value.join('\n')], { type: 'text/plain;charset=utf-8' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = `logs-${Date.now()}.log`
  a.click()
  URL.revokeObjectURL(a.href)
}

function scrollBottom() {
  requestAnimationFrame(() => {
    if (box.value) box.value.scrollTop = box.value.scrollHeight
  })
}

function onScroll() {
  const el = box.value
  if (!el) return
  autoScroll.value = el.scrollTop + el.clientHeight >= el.scrollHeight - 60
}

watch(tail, () => {
  if (connected.value) {
    // tail 变更重连
    connect()
  }
})
watch(autoScroll, (v) => v && scrollBottom())

onMounted(() => {
  if (props.follow) connect()
})
onBeforeUnmount(() => disconnect())

// ---------------- ANSI -> HTML(先转义再上色,防注入) ----------------
const FG = { 30: '#8b93a7', 31: '#f87171', 32: '#34d399', 33: '#fbbf24', 34: '#60a5fa', 35: '#f472b6', 36: '#22d3ee', 37: '#e5e7eb' }
const FG_B = { 90: '#6b7280', 91: '#ef4444', 92: '#22c55e', 93: '#eab308', 94: '#3b82f6', 95: '#ec4899', 96: '#06b6d4', 97: '#f9fafb' }
const BG = { 40: '#1f2937', 41: '#7f1d1d', 42: '#14532d', 43: '#713f12', 44: '#1e3a8a', 45: '#701a75', 46: '#155e75', 47: '#374151' }

function ansiToHtml(t) {
  const esc = t.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
  let out = ''
  let open = 0
  const re = /\x1b\[([0-9;]*)m/g
  let last = 0
  let m
  while ((m = re.exec(esc))) {
    out += esc.slice(last, m.index)
    last = re.lastIndex
    for (const c of (m[1] || '0').split(';')) {
      if (!c || c === '0') {
        out += '</span>'.repeat(open)
        open = 0
      } else if (c === '1') {
        out += '<span style="font-weight:700">'
        open++
      } else if (FG[c]) {
        out += `<span style="color:${FG[c]}">`
        open++
      } else if (FG_B[c]) {
        out += `<span style="color:${FG_B[c]}">`
        open++
      } else if (BG[c]) {
        out += `<span style="background:${BG[c]}">`
        open++
      }
    }
  }
  out += esc.slice(last) + '</span>'.repeat(open)
  return out
}
void seq
</script>
