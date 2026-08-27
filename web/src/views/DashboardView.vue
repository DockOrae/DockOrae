<template>
  <div class="space-y-6 fade-up">
    <!-- 统计卡片 -->
    <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
      <StatCard icon="container" :label="t('dashboard.totalContainers')" :value="counts.total" :sub="t('dashboard.allStates')" color="#60a5fa" bg="rgba(96,165,250,.12)" @click="$router.push('/containers')" />
      <StatCard icon="play" :label="t('dashboard.running')" :value="counts.running" :sub="t('dashboard.running')" color="#34d399" bg="rgba(52,211,153,.12)" @click="$router.push('/containers')" />
      <StatCard icon="image" :label="t('dashboard.images')" :value="counts.images" :sub="t('dashboard.images')" color="#ec4899" bg="rgba(236,72,153,.12)" @click="$router.push('/images')" />
      <StatCard icon="volume" :label="t('dashboard.volumes')" :value="counts.volumes" :sub="t('dashboard.volumes')" color="#fbbf24" bg="rgba(251,191,36,.12)" @click="$router.push('/volumes')" />
    </div>

    <!-- 圆形仪表:负载 / CPU / 内存 / 磁盘(参照 1Panel 概述) -->
    <div class="card p-5">
      <div class="flex items-center gap-2 mb-4">
        <Icon name="stats" size="16" class="text-brand" />
        <h2 class="text-sm font-semibold">{{ t('dashboard.usage') }}</h2>
      </div>
      <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
        <!-- 负载 -->
        <div class="flex flex-col items-center py-2">
          <CircularGauge
            :value="loadPct"
            :label="t('dashboard.load')"
            :sub="loadText"
            color="#ec4899"
            :display="load1 == null ? '-' : load1.toFixed(2)"
            :unit="t('dashboard.loadUnit')"
          />
        </div>
        <!-- CPU -->
        <div class="flex flex-col items-center py-2">
          <CircularGauge
            :value="mon.cpu_pct"
            :label="t('dashboard.cpuUsage')"
            :sub="cpuSub"
            color="#60a5fa"
          />
        </div>
        <!-- 内存 -->
        <div class="flex flex-col items-center py-2">
          <CircularGauge
            :value="memPct"
            :label="t('dashboard.memUsage')"
            :sub="memSub"
            color="#34d399"
          />
        </div>
        <!-- 磁盘 -->
        <div class="flex flex-col items-center py-2">
          <CircularGauge
            :value="diskPct"
            :label="t('dashboard.diskUsage')"
            :sub="diskSub"
            color="#fbbf24"
          />
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 xl:grid-cols-3 gap-4">
      <!-- 实时监控(网络 / 磁盘 IO) -->
      <div class="card p-5 xl:col-span-2">
        <div class="flex items-center gap-2 mb-4 flex-wrap">
          <Icon name="stats" size="16" class="text-brand" />
          <h2 class="text-sm font-semibold">{{ t('dashboard.monitor') }}</h2>
          <div class="ml-auto flex items-center gap-1 bg-surface2/70 border border-line rounded-lg p-0.5">
            <button
              class="px-3 py-1 text-[12px] font-medium rounded-md transition-colors"
              :class="monitorTab === 'net' ? 'bg-brand text-white' : 'text-muted hover:text-text'"
              @click="monitorTab = 'net'"
            >
              {{ t('dashboard.network') }}
            </button>
            <button
              class="px-3 py-1 text-[12px] font-medium rounded-md transition-colors"
              :class="monitorTab === 'io' ? 'bg-brand text-white' : 'text-muted hover:text-text'"
              @click="monitorTab = 'io'"
            >
              {{ t('dashboard.io') }}
            </button>
          </div>
        </div>

        <!-- 速率 / 累计标签 -->
        <div class="flex flex-wrap gap-2 mb-3">
          <template v-if="monitorTab === 'net'">
            <span class="px-2.5 py-1 rounded-md bg-surface2/70 border border-line text-[11px]">
              <span class="text-muted">{{ t('dashboard.down') }}:</span>
              <span class="text-info font-semibold">{{ netRate.rx }}</span>
            </span>
            <span class="px-2.5 py-1 rounded-md bg-surface2/70 border border-line text-[11px]">
              <span class="text-muted">{{ t('dashboard.up') }}:</span>
              <span class="text-purple-400 font-semibold">{{ netRate.tx }}</span>
            </span>
            <span class="px-2.5 py-1 rounded-md bg-surface2/70 border border-line text-[11px]">
              <span class="text-muted">{{ t('dashboard.netTotal') }}:</span>
              <span class="font-semibold">{{ fmtBytes(net.rx) }} ↓ / {{ fmtBytes(net.tx) }} ↑</span>
            </span>
          </template>
          <template v-else>
            <span class="px-2.5 py-1 rounded-md bg-surface2/70 border border-line text-[11px]">
              <span class="text-muted">{{ t('dashboard.read') }}:</span>
              <span class="text-emerald-400 font-semibold">{{ ioRate.read }}</span>
            </span>
            <span class="px-2.5 py-1 rounded-md bg-surface2/70 border border-line text-[11px]">
              <span class="text-muted">{{ t('dashboard.write') }}:</span>
              <span class="text-amber-400 font-semibold">{{ ioRate.write }}</span>
            </span>
            <span class="px-2.5 py-1 rounded-md bg-surface2/70 border border-line text-[11px]">
              <span class="text-muted">{{ t('dashboard.ioTotal') }}:</span>
              <span class="font-semibold">{{ fmtBytes(io.read) }} ⬇ / {{ fmtBytes(io.write) }} ⬆</span>
            </span>
          </template>
        </div>

        <div class="h-40 bg-surface2/40 border border-line rounded-lg p-3">
          <MiniChart
            v-if="monitorTab === 'net'"
            :s1="netHistory.rx"
            :s2="netHistory.tx"
            color1="#60a5fa"
            color2="#c084fc"
            :empty-text="t('dashboard.noData')"
          />
          <MiniChart
            v-else
            :s1="ioHistory.read"
            :s2="ioHistory.write"
            color1="#34d399"
            color2="#fbbf24"
            :empty-text="t('dashboard.noData')"
          />
        </div>
      </div>

      <!-- 系统信息 -->
      <div class="card p-5">
        <div class="flex items-center gap-2 mb-4">
          <Icon name="cpu" size="16" class="text-brand" />
          <h2 class="text-sm font-semibold">{{ t('dashboard.systemInfo') }}</h2>
        </div>
        <div v-if="host" class="space-y-2.5 text-[13px]">
          <div class="flex items-center gap-2"><Icon name="box" size="14" class="text-muted shrink-0" /><span class="text-muted shrink-0">{{ t('dashboard.hostname') }}</span><span class="ml-auto font-medium truncate">{{ host.hostname || '-' }}</span></div>
          <div class="flex items-center gap-2"><Icon name="drive" size="14" class="text-muted shrink-0" /><span class="text-muted shrink-0">{{ t('dashboard.os') }}</span><span class="ml-auto font-medium truncate">{{ host.os || '-' }}</span></div>
          <div class="flex items-center gap-2"><Icon name="network" size="14" class="text-muted shrink-0" /><span class="text-muted shrink-0">{{ t('dashboard.kernel') }}</span><span class="ml-auto font-medium truncate">{{ host.kernel || '-' }}</span></div>
          <div class="flex items-center gap-2"><Icon name="clock" size="14" class="text-muted shrink-0" /><span class="text-muted shrink-0">{{ t('dashboard.uptime') }}</span><span class="ml-auto font-medium truncate">{{ uptimeText }}</span></div>
          <div class="flex items-center gap-2"><Icon name="info" size="14" class="text-muted shrink-0" /><span class="text-muted shrink-0">{{ t('dashboard.serverTime') }}</span><span class="ml-auto font-medium font-mono text-[12px]">{{ serverTimeText }}</span></div>
          <div class="flex items-center gap-2"><Icon name="stats" size="14" class="text-muted shrink-0" /><span class="text-muted shrink-0">{{ t('dashboard.load') }}</span><span class="ml-auto font-medium font-mono text-[12px]">{{ loadText }}</span></div>
          <div class="flex items-center gap-2"><Icon name="cpu" size="14" class="text-muted shrink-0" /><span class="text-muted shrink-0">{{ t('dashboard.cpuModel') }}</span><span class="ml-auto font-medium truncate">{{ host.cpu_model || '-' }}</span></div>
          <div class="flex items-center gap-2"><Icon name="cpu" size="14" class="text-muted shrink-0" /><span class="text-muted shrink-0">{{ t('dashboard.cores') }}</span><span class="ml-auto font-medium">{{ host.cpu_cores ?? '-' }}</span></div>
          <div class="flex items-center gap-2"><Icon name="memory" size="14" class="text-muted shrink-0" /><span class="text-muted shrink-0">{{ t('dashboard.memTotal') }}</span><span class="ml-auto font-medium">{{ fmtBytes(host.mem_total) }}</span></div>
          <div class="flex items-center gap-2"><Icon name="drive" size="14" class="text-muted shrink-0" /><span class="text-muted shrink-0">{{ t('dashboard.arch') }}</span><span class="ml-auto font-medium">{{ host.arch || '-' }}</span></div>
        </div>
        <p v-else class="text-[13px] text-muted flex items-center gap-2">
          <span class="inline-block w-4 h-4 border-2 border-brand/30 border-t-brand rounded-full animate-spin" />
          {{ t('dashboard.loadingDocker') }}
        </p>
      </div>
    </div>

    <div class="grid grid-cols-1 xl:grid-cols-3 gap-4">
      <!-- 运行中容器 -->
      <div class="card xl:col-span-2">
        <div class="flex items-center gap-2 px-5 pt-4 pb-2">
          <Icon name="container" size="16" class="text-brand" />
          <h2 class="text-sm font-semibold">{{ t('dashboard.runningContainers') }}</h2>
          <router-link to="/containers" class="ml-auto link text-[12px]">{{ t('dashboard.viewAll') }}</router-link>
        </div>
        <table class="table">
          <thead>
            <tr>
              <th class="th">{{ t('containers.thName') }}</th>
              <th class="th">{{ t('containers.thImage') }}</th>
              <th class="th">{{ t('containers.thPorts') }}</th>
              <th class="th">{{ t('containers.thStatus') }}</th>
              <th class="th w-24">{{ t('containers.thActions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="c in runningContainers" :key="c.Id" class="cursor-pointer" @click="$router.push('/containers/' + c.Id)">
              <td class="td font-medium">{{ name(c) }}</td>
              <td class="td text-muted">{{ c.Image }}</td>
              <td class="td text-muted text-[12px]">{{ ports(c) }}</td>
              <td class="td"><StatusBadge :state="c.State" /></td>
              <td class="td" @click.stop>
                <button class="btn btn-icon btn-sm" :title="c.State === 'running' ? t('common.stop') : t('common.start')" @click="quick(c)">
                  <Icon :name="c.State === 'running' ? 'stop' : 'play'" size="13" />
                </button>
              </td>
            </tr>
            <tr v-if="!runningContainers.length">
              <td colspan="5" class="td text-center text-muted py-8">{{ t('dashboard.noRunningContainers') }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 最近事件 -->
      <div class="card p-5 flex flex-col min-h-[220px]">
        <div class="flex items-center gap-2 mb-4">
          <Icon name="stats" size="16" class="text-brand" />
          <h2 class="text-sm font-semibold">{{ t('dashboard.recentEvents') }}</h2>
          <span class="ml-auto text-[11px]" :class="wsOn ? 'text-ok' : 'text-muted'">{{ wsOn ? t('dashboard.live') : t('dashboard.notConnected') }}</span>
        </div>
        <div class="flex-1 overflow-y-auto max-h-56 space-y-1.5 pr-1">
          <div v-if="!events.length" class="text-[12px] text-muted">{{ t('dashboard.noEvents') }}</div>
          <div v-for="(e, i) in events" :key="i" class="flex items-center gap-2.5 text-[12px]">
            <span class="dot shrink-0" :style="{ background: e.color }" />
            <span class="text-muted shrink-0 w-10">{{ e.label }}</span>
            <span class="truncate text-text/90">{{ e.name }}</span>
            <span class="ml-auto text-muted/70 shrink-0">{{ e.time }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import StatCard from '../components/StatCard.vue'
import StatusBadge from '../components/StatusBadge.vue'
import CircularGauge from '../components/CircularGauge.vue'
import MiniChart from '../components/MiniChart.vue'
import Icon from '../components/Icon.vue'
import { api, wsUrl } from '../api'
import { containerName, humanPorts, formatBytes } from '../util'
import { toastErr } from '../toast'

const { t } = useI18n()

// ---------- 基础数据 ----------
const containers = ref([])
const images = ref(0)
const volumes = ref(0)
const host = ref(null)

// ---------- 监控 ----------
const monitorTab = ref('net')
const mon = ref({ cpu_pct: 0, mem: null, load: null, disk: null, net: { rx: 0, tx: 0 }, io: { read: 0, write: 0 } })
const netHistory = ref({ rx: [], tx: [] })
const ioHistory = ref({ read: [], write: [] })
const prevNet = ref(null)
const prevIo = ref(null)
const prevTs = ref(null)
const netRate = ref({ rx: '0 B/s', tx: '0 B/s' })
const ioRate = ref({ read: '0 B/s', write: '0 B/s' })
let monTimer = null
let clockTimer = null

// 服务器时间偏移
let timeOffset = 0
const serverTimeText = ref('-')

// ---------- 事件 ----------
const events = ref([])
const wsOn = ref(false)
let ws = null
let deactivated = false

const counts = computed(() => ({
  total: containers.value.length,
  running: containers.value.filter((c) => c.State === 'running').length,
  images: images.value,
  volumes: volumes.value,
}))
const runningContainers = computed(() => containers.value.filter((c) => c.State === 'running').slice(0, 12))
const name = (c) => containerName(c)
const ports = (c) => humanPorts(c.Ports)

// ---------- 圆环计算 ----------
const load1 = computed(() => (mon.value.load ? mon.value.load[0] : null))
const loadPct = computed(() => {
  const l = load1.value
  const cores = host.value?.cpu_cores || 1
  if (l == null) return 0
  return Math.min(100, (l / cores) * 100)
})
const loadText = computed(() => {
  const l = mon.value.load
  return l ? l.map((x) => x.toFixed(2)).join(' / ') : '-'
})
const memPct = computed(() => mon.value.mem?.pct ?? 0)
const memSub = computed(() => {
  const m = mon.value.mem
  return m ? `${fmtBytes(m.used)} / ${fmtBytes(m.total)}` : '-'
})
const diskPct = computed(() => mon.value.disk?.pct ?? 0)
const diskSub = computed(() => {
  const d = mon.value.disk
  return d ? `${fmtBytes(d.used)} / ${fmtBytes(d.total)}` : '-'
})
const cpuSub = computed(() => {
  const cores = host.value?.cpu_cores
  return cores ? `${cores} ${t('dashboard.cores')}` : ''
})
const net = computed(() => mon.value.net)
const io = computed(() => mon.value.io)
const uptimeText = computed(() => {
  const s = host.value?.uptime
  if (!s) return '-'
  const d = Math.floor(s / 86400)
  const h = Math.floor((s % 86400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  if (d > 0) return `${d}${t('time.daysShort')} ${h}${t('time.hoursShort')} ${m}${t('time.minShort')}`
  if (h > 0) return `${h}${t('time.hoursShort')} ${m}${t('time.minShort')}`
  return `${m}${t('time.minShort')}`
})
const fmtBytes = (n) => formatBytes(n, 1)
const fmtRate = (n) => (n == null ? '-' : formatBytes(n, 1) + '/s')

// ---------- 加载 ----------
async function loadBase() {
  try {
    const [info, cs, imgs, vols] = await Promise.all([
      api('/system/info'),
      api('/containers'),
      api('/images'),
      api('/volumes'),
    ])
    containers.value = cs
    images.value = imgs.length
    volumes.value = vols.length
    if (!host.value) {
      try {
        const h = await api('/system/host')
        host.value = h
        if (h.server_time) timeOffset = h.server_time * 1000 - Date.now()
      } catch { /* 监控接口在无权限环境可能失败 */ }
    }
  } catch {
    /* Docker 不可达:统计卡保持 0,监控数据自然为空 */
  }
}

async function loadMonitor() {
  try {
    const m = await api('/system/monitor')
    const now = Date.now()
    // 速率 = 累计差值 / 时间差
    if (prevTs.value && prevNet.value) {
      const dt = (now - prevTs.value) / 1000
      if (dt > 0) {
        netRate.value.rx = fmtRate(Math.max(0, (m.net.rx - prevNet.value.rx) / dt))
        netRate.value.tx = fmtRate(Math.max(0, (m.net.tx - prevNet.value.tx) / dt))
      }
    }
    if (prevTs.value && prevIo.value) {
      const dt = (now - prevTs.value) / 1000
      if (dt > 0) {
        ioRate.value.read = fmtRate(Math.max(0, (m.io.read - prevIo.value.read) / dt))
        ioRate.value.write = fmtRate(Math.max(0, (m.io.write - prevIo.value.write) / dt))
      }
    }
    prevNet.value = { rx: m.net.rx, tx: m.net.tx }
    prevIo.value = { read: m.io.read, write: m.io.write }
    prevTs.value = now

    mon.value = m
    pushHistory(netHistory.value.rx, m.net.rx)
    pushHistory(netHistory.value.tx, m.net.tx)
    pushHistory(ioHistory.value.read, m.io.read)
    pushHistory(ioHistory.value.write, m.io.write)
  } catch { /* 忽略,下轮重试 */ }
}

function pushHistory(arr, v) {
  arr.push(v)
  if (arr.length > 60) arr.shift()
}

// ---------- 事件 WS ----------
function connectEvents() {
  try {
    ws = new WebSocket(wsUrl('/ws/events'))
    ws.onopen = () => (wsOn.value = true)
    ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data)
        if (msg.type !== 'event') return
        const d = msg.data
        const color = EVENT_COLOR[d.action] || '#8b93a7'
        events.value.unshift({
          color,
          label: t('events.' + d.action).startsWith('events.') ? d.action : t('events.' + d.action),
          name: d.actor_attributes?.name || d.id || d.from || '-',
          time: timeAgoI18n(d.time),
        })
        if (events.value.length > 30) events.value.pop()
      } catch { /* ignore */ }
    }
    ws.onclose = () => {
      wsOn.value = false
      if (!deactivated) setTimeout(connectEvents, 3000)
    }
    ws.onerror = () => ws?.close()
  } catch { /* ignore */ }
}

const EVENT_COLOR = {
  start: '#34d399', stop: '#f87171', die: '#f87171', destroy: '#f87171', create: '#60a5fa',
  restart: '#fbbf24', pause: '#fbbf24', unpause: '#60a5fa', kill: '#f87171', oom: '#f87171',
  pull: '#fbbf24', push: '#60a5fa', rename: '#60a5fa', attach: '#8b93a7', detach: '#8b93a7',
  exec_start: '#60a5fa', health_status: '#60a5fa',
}

function timeAgoI18n(ts) {
  if (!ts) return '-'
  const s = Math.floor(Date.now() / 1000) - ts
  if (s < 60) return t('time.justNow')
  if (s < 3600) return t('time.minutesAgo', { n: Math.floor(s / 60) })
  if (s < 86400) return t('time.hoursAgo', { n: Math.floor(s / 3600) })
  return t('time.daysAgo', { n: Math.floor(s / 86400) })
}

async function quick(c) {
  try {
    const action = c.State === 'running' ? 'stop' : 'start'
    await api(`/containers/${c.Id}/${action}`, { method: 'POST' })
    loadBase()
  } catch (e) {
    toastErr(e.message)
  }
}

function startTimers() {
  loadBase()
  loadMonitor()
  monTimer = setInterval(loadMonitor, 3000)
  clockTimer = setInterval(() => {
    serverTimeText.value = new Date(Date.now() + timeOffset).toLocaleTimeString()
  }, 1000)
}

onMounted(() => {
  startTimers()
  connectEvents()
})
onActivated(() => {
  deactivated = false
  if (!monTimer) startTimers()
  if (!ws) connectEvents()
})
onDeactivated(() => {
  deactivated = true
  clearInterval(monTimer)
  monTimer = null
  clearInterval(clockTimer)
  clockTimer = null
  ws?.close()
  ws = null
})
onBeforeUnmount(() => {
  clearInterval(monTimer)
  clearInterval(clockTimer)
  ws?.close()
})
</script>
