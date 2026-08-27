<template>
  <div class="status-page">
    <!-- 操作栏(仿 3x-ui OverviewActionBar) -->
    <div class="action-bar">
      <div class="ab-left">
        <span class="ab-version">{{ t('app.version') }}</span>
        <span class="ab-badge" :class="dockerOk ? 'ok' : 'err'">
          <span class="ab-dot" />
          {{ dockerOk ? t('status.dockerOk') : t('status.dockerDown') }}
        </span>
        <span class="ab-badge" :class="licenseActive ? 'pro' : 'free'">
          {{ licenseActive ? t('license.pro') : t('license.community') }}
        </span>
      </div>
      <div class="ab-right">
        <button type="button" class="ab-btn" @click="openHistory">
          <Icon name="stats" size="14" /> {{ t('status.systemHistory') }}
        </button>
        <button type="button" class="ab-btn" @click="openConfig">
          <Icon name="edit" size="14" /> {{ t('status.config') }}
        </button>
        <button type="button" class="ab-btn" @click="openLogs">
          <Icon name="terminal" size="14" /> {{ t('status.logs') }}
        </button>
        <button type="button" class="ab-btn" @click="openBackup">
          <Icon name="download" size="14" /> {{ t('status.backup') }}
        </button>
        <button type="button" class="ab-btn" @click="panelRestart">
          <Icon name="restart" size="14" /> {{ t('status.restart') }}
        </button>
        <button type="button" class="ab-btn" @click="refreshAll">
          <Icon name="refresh" size="14" /> {{ t('common.refresh') }}
        </button>
      </div>
    </div>

    <!-- 健康警告条(仿 3x-ui ov-health) -->
    <div v-if="health" class="health-bar" :style="{ color: health.color }">
      <span class="health-mark" :style="{ background: health.color }" />
      {{ health.text }}
    </div>

    <!-- 统计入口:容器 / 运行中 / 镜像 / 卷 -->
    <div class="stat-row">
      <button type="button" class="stat-chip" @click="$router.push('/containers')">
        <Icon name="container" size="15" /> {{ t('nav.containers') }}
        <b>{{ counts.total }}</b>
      </button>
      <button type="button" class="stat-chip" @click="$router.push('/containers')">
        <Icon name="play" size="15" /> {{ t('status.running') }}
        <b class="ok">{{ counts.running }}</b>
      </button>
      <button type="button" class="stat-chip" @click="$router.push('/images')">
        <Icon name="image" size="15" /> {{ t('nav.images') }}
        <b>{{ counts.images }}</b>
      </button>
      <button type="button" class="stat-chip" @click="$router.push('/volumes')">
        <Icon name="volume" size="15" /> {{ t('nav.volumes') }}
        <b>{{ counts.volumes }}</b>
      </button>
    </div>

    <hr class="rule" />

    <!-- 四张系统状态卡(仿 3x-ui VitalTile:环形 + 均值/峰值 + 趋势) -->
    <div class="vitals">
      <div class="vital-card">
        <div class="vital-head">
          <Icon name="cpu" size="16" class="vital-icon" />
          <span class="vital-label">{{ t('dashboard.cpuUsage') }}</span>
        </div>
        <CircularGauge
          :value="mon.cpu_pct"
          color="#ec4899"
          :sub="cpuSub"
        />
        <div class="vital-foot">
          <span>{{ t('dashboard.avg') }} <b>{{ avg(hist.cpu).toFixed(0) }}%</b></span>
          <span>{{ t('dashboard.peak') }} <b>{{ peak(hist.cpu).toFixed(0) }}%</b></span>
        </div>
        <div class="vital-spark">
          <MiniChart :s1="hist.cpu" color1="#ec4899" :height="36" />
        </div>
      </div>

      <div class="vital-card">
        <div class="vital-head">
          <Icon name="memory" size="16" class="vital-icon" />
          <span class="vital-label">{{ t('dashboard.memUsage') }}</span>
        </div>
        <CircularGauge
          :value="memPct"
          color="#a78bfa"
          :sub="memSub"
        />
        <div class="vital-foot">
          <span>{{ t('dashboard.avg') }} <b>{{ avg(hist.mem).toFixed(0) }}%</b></span>
          <span>{{ t('dashboard.peak') }} <b>{{ peak(hist.mem).toFixed(0) }}%</b></span>
        </div>
        <div class="vital-spark">
          <MiniChart :s1="hist.mem" color1="#a78bfa" :height="36" />
        </div>
      </div>

      <div class="vital-card">
        <div class="vital-head">
          <Icon name="stats" size="16" class="vital-icon" />
          <span class="vital-label">{{ t('dashboard.load') }}</span>
        </div>
        <CircularGauge
          :value="loadPct"
          color="#fbbf24"
          :display="load1 == null ? '-' : load1.toFixed(2)"
          :unit="t('dashboard.loadUnit')"
          :sub="loadText"
        />
        <div class="vital-foot">
          <span>{{ t('dashboard.cores') }} <b>{{ host?.cpu_cores ?? '-' }}</b></span>
          <span>{{ t('status.load15') }} <b>{{ load2 == null ? '-' : load2.toFixed(2) }}</b></span>
        </div>
        <div class="vital-spark">
          <MiniChart :s1="hist.load" color1="#fbbf24" :height="36" />
        </div>
      </div>

      <div class="vital-card">
        <div class="vital-head">
          <Icon name="drive" size="16" class="vital-icon" />
          <span class="vital-label">{{ t('dashboard.diskUsage') }}</span>
        </div>
        <CircularGauge
          :value="diskPct"
          color="#34d399"
          :sub="diskSub"
        />
        <div class="vital-foot">
          <span>{{ t('status.free') }} <b>{{ freeDisk }}</b></span>
          <span>{{ t('dashboard.avg') }} <b>{{ avg(hist.disk).toFixed(0) }}%</b></span>
        </div>
        <div class="vital-spark">
          <MiniChart :s1="hist.disk" color1="#34d399" :height="36" />
        </div>
      </div>
    </div>

    <!-- 中部:网络吞吐 + 容器 IO(仿 3x-ui ThroughputCard) -->
    <div class="mid-row">
      <div class="card mid-card">
        <div class="card-head">
          <Icon name="network" size="15" class="text-brand" />
          <h3>{{ t('dashboard.network') }}</h3>
          <div class="ml-auto flex items-center gap-3 text-[12px]">
            <span class="legend"><i class="dot down" />{{ t('dashboard.down') }} <b class="info">{{ netRate.rx }}</b></span>
            <span class="legend"><i class="dot up" />{{ t('dashboard.up') }} <b class="purple">{{ netRate.tx }}</b></span>
          </div>
        </div>
        <div class="chart-box">
          <MiniChart :s1="netHistory.rx" :s2="netHistory.tx" color1="#60a5fa" color2="#c084fc" :height="110" />
        </div>
      </div>

      <div class="card mid-card">
        <div class="card-head">
          <Icon name="drive" size="15" class="text-brand" />
          <h3>{{ t('dashboard.io') }}</h3>
          <div class="ml-auto flex items-center gap-3 text-[12px]">
            <span class="legend"><i class="dot read" />{{ t('dashboard.read') }} <b class="ok">{{ ioRate.read }}</b></span>
            <span class="legend"><i class="dot write" />{{ t('dashboard.write') }} <b class="warn">{{ ioRate.write }}</b></span>
          </div>
        </div>
        <div class="chart-box">
          <MiniChart :s1="ioHistory.read" :s2="ioHistory.write" color1="#34d399" color2="#fbbf24" :height="110" />
        </div>
      </div>
    </div>

    <!-- 系统信息条(仿 3x-ui SystemStrip) -->
    <div class="sys-strip">
      <div class="sys-item"><Icon name="box" size="13" class="text-muted" /><span class="sys-label">{{ t('dashboard.hostname') }}</span><b>{{ host?.hostname || '-' }}</b></div>
      <div class="sys-item"><Icon name="drive" size="13" class="text-muted" /><span class="sys-label">{{ t('dashboard.os') }}</span><b class="truncate">{{ host?.os || '-' }}</b></div>
      <div class="sys-item"><Icon name="cpu" size="13" class="text-muted" /><span class="sys-label">{{ t('dashboard.kernel') }}</span><b class="truncate">{{ host?.kernel || '-' }}</b></div>
      <div class="sys-item"><Icon name="clock" size="13" class="text-muted" /><span class="sys-label">{{ t('dashboard.uptime') }}</span><b>{{ uptimeText }}</b></div>
      <div class="sys-item"><Icon name="stats" size="13" class="text-muted" /><span class="sys-label">{{ t('dashboard.serverTime') }}</span><b class="font-mono">{{ serverTimeText }}</b></div>
      <div class="sys-item"><Icon name="cpu" size="13" class="text-muted" /><span class="sys-label">{{ t('dashboard.cpuModel') }}</span><b class="truncate">{{ host?.cpu_model || '-' }}</b></div>
      <div class="sys-item"><Icon name="info" size="13" class="text-muted" /><span class="sys-label">{{ t('dashboard.arch') }}</span><b>{{ host?.arch || '-' }}</b></div>
    </div>

    <!-- 底部:运行中容器 + 最近事件 -->
    <div class="bottom-row">
      <div class="card xl:col-span-2">
        <div class="card-head px-5 pt-4 pb-2">
          <Icon name="container" size="16" class="text-brand" />
          <h3>{{ t('dashboard.runningContainers') }}</h3>
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
              <td class="td font-medium">{{ containerName(c) }}</td>
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

      <div class="card p-5 flex flex-col min-h-[220px]">
        <div class="flex items-center gap-2 mb-4">
          <Icon name="stats" size="16" class="text-brand" />
          <h3>{{ t('dashboard.recentEvents') }}</h3>
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

    <!-- ============ 日志弹窗(仿 3x-ui LogModal) ============ -->
    <div v-if="logsOpen" class="modal-mask" @click.self="logsOpen = false">
      <div class="modal-box w-full max-w-2xl">
        <div class="modal-head">
          <h3>{{ t('status.logs') }}</h3>
          <button type="button" class="modal-close" @click="logsOpen = false"><Icon name="x" size="16" /></button>
        </div>
        <pre class="logs-view">{{ logsText }}</pre>
        <div class="modal-foot">
          <button class="btn btn-ghost btn-sm" @click="loadLogs"><Icon name="refresh" size="12" /> {{ t('common.refresh') }}</button>
        </div>
      </div>
    </div>

    <!-- ============ 配置弹窗(仿 3x-ui ConfigModal) ============ -->
    <div v-if="configOpen" class="modal-mask" @click.self="configOpen = false">
      <div class="modal-box w-full max-w-2xl">
        <div class="modal-head">
          <h3>{{ t('status.config') }}</h3>
          <div class="flex items-center gap-2">
            <button class="btn btn-ghost btn-sm" @click="copyConfig">
              <Icon name="copy" size="12" /> {{ t('common.copy') }}
            </button>
            <button type="button" class="modal-close" @click="configOpen = false"><Icon name="x" size="16" /></button>
          </div>
        </div>
        <pre class="logs-view">{{ configText }}</pre>
      </div>
    </div>

    <!-- ============ 备份与恢复弹窗(仿 3x-ui BackupModal) ============ -->
    <div v-if="backupOpen" class="modal-mask" @click.self="backupOpen = false">
      <div class="modal-box w-full max-w-md">
        <div class="modal-head">
          <h3>{{ t('status.backupTitle') }}</h3>
          <button type="button" class="modal-close" @click="backupOpen = false"><Icon name="x" size="16" /></button>
        </div>
        <div class="space-y-4 p-5">
          <button class="btn btn-brand w-full" @click="downloadBackup">
            <Icon name="download" size="14" /> {{ t('status.downloadBackup') }}
          </button>
          <div
            class="rounded-xl border-2 border-dashed border-line hover:border-brand/60 transition-all cursor-pointer flex flex-col items-center justify-center gap-2 py-8 bg-surface2/40"
            @click="backupInput?.click()"
          >
            <Icon name="upload" size="26" class="text-muted" />
            <p class="text-[13px]">{{ t('status.restore') }}</p>
            <p class="text-[11px] text-muted">{{ t('status.chooseBackupFile') }}</p>
            <input ref="backupInput" type="file" accept=".tar.gz,.gz" class="hidden" @change="restoreBackup" />
          </div>
          <p class="text-[11px] text-muted">{{ t('status.restoreConfirm') }}</p>
        </div>
      </div>
    </div>

    <!-- ============ 系统历史弹窗(仿 3x-ui SystemHistoryModal) ============ -->
    <div v-if="historyOpen" class="modal-mask" @click.self="historyOpen = false">
      <div class="modal-box w-full max-w-2xl">
        <div class="modal-head">
          <h3>{{ t('status.systemHistory') }}</h3>
          <button type="button" class="modal-close" @click="historyOpen = false"><Icon name="x" size="16" /></button>
        </div>
        <div class="p-5 space-y-5">
          <div>
            <div class="flex items-center justify-between text-[12px] mb-1">
              <span class="text-muted">{{ t('dashboard.cpuUsage') }}</span>
              <span class="font-mono">{{ (mon.cpu_pct ?? 0).toFixed(1) }}%</span>
            </div>
            <MiniChart :s1="hist.cpu" color1="#ec4899" :height="60" />
          </div>
          <div>
            <div class="flex items-center justify-between text-[12px] mb-1">
              <span class="text-muted">{{ t('dashboard.memUsage') }}</span>
              <span class="font-mono">{{ memPct.toFixed(1) }}%</span>
            </div>
            <MiniChart :s1="hist.mem" color1="#a78bfa" :height="60" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import StatusBadge from '../components/StatusBadge.vue'
import CircularGauge from '../components/CircularGauge.vue'
import MiniChart from '../components/MiniChart.vue'
import Icon from '../components/Icon.vue'
import { api, getToken, wsUrl } from '../api'
import { containerName, humanPorts, formatBytes } from '../util'
import { toastErr, toastOk } from '../toast'
import { licenseActive } from '../store'

const { t } = useI18n()

// ---------- 基础数据 ----------
const containers = ref([])
const images = ref(0)
const volumes = ref(0)
const host = ref(null)
const dockerOk = ref(false)

// ---------- 监控 ----------
const mon = ref({ cpu_pct: 0, mem: null, load: null, disk: null, net: { rx: 0, tx: 0 }, io: { read: 0, write: 0 } })
const netHistory = ref({ rx: [], tx: [] })
const ioHistory = ref({ read: [], write: [] })
const hist = ref({ cpu: [], mem: [], load: [], disk: [] })
const prevNet = ref(null)
const prevIo = ref(null)
const prevTs = ref(null)
const netRate = ref({ rx: '0 B/s', tx: '0 B/s' })
const ioRate = ref({ read: '0 B/s', write: '0 B/s' })
let monTimer = null
let clockTimer = null

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
const ports = (c) => humanPorts(c.Ports)

// ---------- 圆环计算 ----------
const load1 = computed(() => (mon.value.load ? mon.value.load[0] : null))
const load2 = computed(() => (mon.value.load ? mon.value.load[1] : null))
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
const freeDisk = computed(() => {
  const d = mon.value.disk
  return d ? fmtBytes(Math.max(0, d.total - d.used)) : '-'
})
const cpuSub = computed(() => {
  const cores = host.value?.cpu_cores
  return cores ? `${cores} ${t('dashboard.cores')}` : ''
})
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

// ---------- 健康检查(仿 3x-ui:超阈值警告) ----------
const health = computed(() => {
  const items = [
    { name: t('dashboard.cpuUsage'), value: mon.value.cpu_pct },
    { name: t('dashboard.memUsage'), value: memPct.value },
    { name: t('dashboard.diskUsage'), value: diskPct.value },
  ]
  const crit = items.filter((i) => i.value >= 90)
  if (crit.length) {
    return { text: t('status.healthCritical', { list: crit.map((i) => `${i.name} ${i.value.toFixed(0)}%`).join(', ') }), color: '#ef4444' }
  }
  const warm = items.filter((i) => i.value >= 75)
  if (warm.length) {
    return { text: t('status.healthWarm', { list: warm.map((i) => `${i.name} ${i.value.toFixed(0)}%`).join(', ') }), color: '#f59e0b' }
  }
  return null
})

const fmtBytes = (n) => formatBytes(n, 1)
const fmtRate = (n) => (n == null ? '-' : formatBytes(n, 1) + '/s')
const avg = (arr) => (arr.length ? arr.reduce((a, b) => a + b, 0) / arr.length : 0)
const peak = (arr) => (arr.length ? Math.max(...arr) : 0)

// ---------- 加载 ----------
async function loadBase() {
  try {
    const [info, cs, imgs, vols] = await Promise.all([
      api('/system/info'),
      api('/containers'),
      api('/images'),
      api('/volumes'),
    ])
    dockerOk.value = true
    containers.value = cs
    images.value = imgs.length
    volumes.value = vols.length
    if (!host.value) {
      try {
        const h = await api('/system/host')
        host.value = h
        if (h.server_time) timeOffset = h.server_time * 1000 - Date.now()
      } catch { /* ignore */ }
    }
  } catch {
    dockerOk.value = false
  }
}

async function loadMonitor() {
  try {
    const m = await api('/system/monitor')
    const now = Date.now()
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
    pushHistory(hist.value.cpu, m.cpu_pct)
    pushHistory(hist.value.mem, m.mem?.pct ?? 0)
    pushHistory(hist.value.load, m.load?.[0] ?? 0)
    pushHistory(hist.value.disk, m.disk?.pct ?? 0)
  } catch { /* ignore */ }
}

function pushHistory(arr, v) {
  arr.push(v)
  if (arr.length > 60) arr.shift()
}

function refreshAll() {
  loadBase()
  loadMonitor()
}

// ---------- 操作栏弹窗(仿 3x-ui) ----------
const logsOpen = ref(false)
const logsText = ref('')
const configOpen = ref(false)
const configText = ref('')
const backupOpen = ref(false)
const historyOpen = ref(false)
const backupInput = ref(null)

function openLogs() {
  logsOpen.value = true
  loadLogs()
}
async function loadLogs() {
  try {
    const r = await api('/system/logs')
    logsText.value = (r.logs || []).join('\n') || '-'
  } catch (e) {
    logsText.value = e.message
  }
}

function openConfig() {
  configOpen.value = true
  api('/system/config')
    .then((r) => {
      configText.value = typeof r === 'string' ? r : JSON.stringify(r, null, 2)
    })
    .catch((e) => (configText.value = e.message))
}
function copyConfig() {
  navigator.clipboard?.writeText(configText.value).then(() => toastOk(t('common.copied'))).catch(() => {})
}

function openBackup() {
  backupOpen.value = true
}
async function downloadBackup() {
  try {
    const resp = await fetch('/api/system/backup', {
      headers: { Authorization: 'Bearer ' + (getToken() || '') },
    })
    if (!resp.ok) throw new Error(resp.statusText)
    const blob = await resp.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'docker-manager-backup.tar.gz'
    a.click()
    URL.revokeObjectURL(url)
  } catch (e) {
    toastErr(e.message)
  }
}
async function restoreBackup(ev) {
  const file = ev.target.files?.[0]
  ev.target.value = ''
  if (!file) return
  if (!confirm(t('status.restoreConfirm'))) return
  const fd = new FormData()
  fd.append('file', file)
  try {
    const r = await api('/system/restore', { method: 'POST', body: fd })
    toastOk(t('status.restored'))
    if (r.needRestart) setTimeout(panelRestart, 1500)
  } catch (e) {
    toastErr(e.message)
  }
}

function openHistory() {
  historyOpen.value = true
}

function panelRestart() {
  if (!confirm(t('status.restartConfirm'))) return
  api('/system/restart', { method: 'POST' })
    .then(() => toastOk(t('status.restarting')))
    .catch((e) => toastErr(e.message))
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
  refreshAll()
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

<style scoped>
.status-page {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

/* ---------- 操作栏 ---------- */
.action-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.ab-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.ab-version {
  font-size: 12px;
  font-weight: 600;
  padding: 4px 10px;
  border-radius: 999px;
  background: var(--dm-surface2);
  border: 1px solid var(--dm-line);
  color: var(--dm-muted);
}
.ab-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  padding: 4px 10px;
  border-radius: 999px;
  border: 1px solid var(--dm-line);
  background: var(--dm-surface2);
}
.ab-badge.ok { color: #34d399; }
.ab-badge.err { color: #f87171; }
.ab-badge.pro { color: #ec4899; }
.ab-badge.free { color: var(--dm-muted); }
.ab-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: currentColor;
}
.ab-right {
  margin-left: auto;
}
.ab-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 8px;
  border: 1px solid var(--dm-line);
  background: var(--dm-surface);
  color: var(--dm-muted);
  font-size: 12.5px;
  cursor: pointer;
  transition: all 0.15s;
}
.ab-btn:hover {
  color: var(--color-brand);
  border-color: var(--color-brand);
}

/* ---------- 健康条 ---------- */
.health-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  padding: 9px 14px;
  border-radius: 10px;
  background: color-mix(in srgb, currentColor 8%, transparent);
  border: 1px solid color-mix(in srgb, currentColor 25%, transparent);
}
.health-mark {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

/* ---------- 统计入口 ---------- */
.stat-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 10px;
}
.stat-chip {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 14px;
  border-radius: 12px;
  border: 1px solid var(--dm-line);
  background: var(--dm-surface);
  color: var(--dm-muted);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s;
}
.stat-chip:hover {
  border-color: var(--color-brand);
  color: var(--dm-text);
}
.stat-chip b {
  margin-left: auto;
  font-size: 16px;
  color: var(--dm-text);
}
.stat-chip b.ok { color: #34d399; }

.rule {
  border: none;
  border-top: 1px solid var(--dm-line);
  margin: 2px 0;
}

/* ---------- VitalTile 四卡 ---------- */
.vitals {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(230px, 1fr));
  gap: 12px;
}
.vital-card {
  background: var(--dm-surface);
  border: 1px solid var(--dm-line);
  border-radius: 14px;
  padding: 14px 16px 10px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}
.vital-head {
  display: flex;
  align-items: center;
  gap: 7px;
  align-self: flex-start;
}
.vital-icon {
  color: var(--color-brand);
}
.vital-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--dm-text);
}
.vital-foot {
  display: flex;
  justify-content: space-between;
  width: 100%;
  font-size: 11.5px;
  color: var(--dm-muted);
  padding: 2px 4px 0;
}
.vital-foot b {
  color: var(--dm-text);
  font-weight: 600;
}
.vital-spark {
  width: 100%;
  margin-top: 2px;
}

/* ---------- 中部卡 ---------- */
.mid-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 12px;
}
.card {
  background: var(--dm-surface);
  border: 1px solid var(--dm-line);
  border-radius: 14px;
}
.mid-card {
  padding: 14px 16px;
}
.card-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  flex-wrap: wrap;
}
.card-head h3 {
  font-size: 13.5px;
  font-weight: 600;
  color: var(--dm-text);
}
.legend {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--dm-muted);
}
.legend .dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
}
.dot.down { background: #60a5fa; }
.dot.up { background: #c084fc; }
.dot.read { background: #34d399; }
.dot.write { background: #fbbf24; }
.legend .info { color: #60a5fa; }
.legend .purple { color: #c084fc; }
.legend .ok { color: #34d399; }
.legend .warn { color: #fbbf24; }
.chart-box {
  background: var(--dm-surface2);
  border: 1px solid var(--dm-line);
  border-radius: 10px;
  padding: 8px;
}

/* ---------- 系统信息条 ---------- */
.sys-strip {
  display: flex;
  align-items: center;
  gap: 18px;
  flex-wrap: wrap;
  padding: 10px 16px;
  border-radius: 12px;
  border: 1px solid var(--dm-line);
  background: var(--dm-surface);
  font-size: 12px;
}
.sys-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.sys-item b {
  color: var(--dm-text);
  font-weight: 600;
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.sys-label {
  color: var(--dm-muted);
}

/* ---------- 底部 ---------- */
.bottom-row {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 12px;
}
@media (max-width: 1100px) {
  .bottom-row {
    grid-template-columns: 1fr;
  }
}

/* ---------- 弹窗(仿 3x-ui Modal) ---------- */
.modal-mask {
  position: fixed;
  inset: 0;
  z-index: 80;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(4px);
  padding: 16px;
}
.modal-box {
  background: var(--dm-surface);
  border: 1px solid var(--dm-line);
  border-radius: 14px;
  box-shadow: 0 24px 64px rgba(0, 0, 0, 0.35);
  overflow: hidden;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
}
.modal-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  border-bottom: 1px solid var(--dm-line);
  font-size: 14px;
  font-weight: 600;
  color: var(--dm-text);
}
.modal-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--dm-muted);
  cursor: pointer;
}
.modal-close:hover {
  color: var(--dm-text);
  background: var(--dm-surface2);
}
.modal-foot {
  display: flex;
  justify-content: flex-end;
  padding: 10px 18px;
  border-top: 1px solid var(--dm-line);
}
.logs-view {
  flex: 1;
  min-height: 320px;
  max-height: 60vh;
  overflow: auto;
  padding: 14px 16px;
  margin: 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 11.5px;
  line-height: 1.6;
  color: var(--dm-text);
  background: var(--dm-surface2);
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
