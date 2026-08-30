<template>
  <div class="status-page ov-page">
    <!-- ============ 操作栏(仿 3x-ui OverviewActionBar:状态胶囊 + 版本 + 分组按钮) ============ -->
    <div class="ov-bar">
      <span class="ov-state" :data-state="dockerOk ? 'running' : 'stop'">
        <span class="ov-state-dot" :style="{ color: dockerOk ? '#34d399' : '#f87171' }" />
        <span>{{ t('status.dockerStatus') }} · {{ dockerOk ? t('status.dockerOk') : t('status.dockerDown') }}</span>
        <button v-if="host?.docker_version" type="button" class="ov-state-version ov-mono">{{ dockerVersion }}</button>
      </span>
      <span class="ab-badge" :class="licenseActive ? 'pro' : 'free'">
        {{ licenseActive ? t('license.pro') : t('license.community') }}
      </span>

      <div class="ov-bar-actions">
        <button type="button" class="ov-bar-btn primary" @click="panelRestart">
          <Icon name="restart" size="14" /> {{ t('status.restart') }}
        </button>
        <span class="ov-bar-sep" />
        <button type="button" class="ov-bar-btn" @click="openLogs"><Icon name="terminal" size="14" /> {{ t('status.logs') }}</button>
        <button type="button" class="ov-bar-btn" @click="openConfig"><Icon name="edit" size="14" /> {{ t('status.config') }}</button>
        <button type="button" class="ov-bar-btn" @click="openBackup"><Icon name="download" size="14" /> {{ t('status.backup') }}</button>
        <span class="ov-bar-sep" />
        <button type="button" class="ov-bar-btn" @click="openHistory"><Icon name="stats" size="14" /> {{ t('status.systemHistory') }}</button>
      </div>
    </div>

    <!-- 健康警告条(仿 3x-ui ov-health) -->
    <div v-if="health" class="ov-health" :style="{ color: health.color }">
      <span class="ov-health-mark" />
      {{ health.text }}
    </div>

    <hr class="ov-rule" />

    <!-- ============ 四张状态卡(仿 3x-ui VitalTile:大数字 + detail + 均值/峰值 + 趋势) ============ -->
    <div class="ov-vitals">
      <Card v-for="v in vitals" :key="v.label" class="ov-tile">
        <div class="ov-tile-head">
          <span class="ov-tile-icon"><Icon :name="v.icon" size="15" /></span>
          <span class="ov-kicker">{{ v.label }}</span>
        </div>
        <div class="ov-tile-value">
          <span class="ov-tile-number">{{ v.percent.toFixed(1) }}</span>
          <span class="ov-tile-unit">%</span>
        </div>
        <div class="ov-tile-detail">{{ v.detail }}</div>
        <div class="ov-tile-foot">
          <span>{{ v.footLeft }}</span>
          <span>{{ v.footRight }}</span>
        </div>
        <div class="ov-tile-chart">
          <MiniChart :s1="v.data" :color1="v.color" :height="62" :fill="0.3" :stroke-width="1.5" :value-max="100" :ref-lines="meanRef(v.data, v.color)" />
        </div>
      </Card>
    </div>

    <!-- ============ 中部:网络吞吐(仿 3x-ui ThroughputCard)+ 容器卡(仿 ConnectionsCard) ============ -->
    <div class="ov-mid">
      <!-- 吞吐卡 -->
      <Card class="ov-tile ov-wide">
        <div class="ov-wide-head">
          <div>
            <div class="ov-kicker">{{ t('dashboard.overallSpeed') }}</div>
            <div class="ov-sub">{{ t('dashboard.throughputSub') }} · {{ t('dashboard.peak') }} {{ netPeakText }}</div>
          </div>
          <div class="ov-wide-legend">
            <div class="ov-legend-label">
              <Icon name="arrowUp" size="12" class="text-brand" />
              {{ t('dashboard.up') }}
              <span class="ov-legend-num">{{ netRate.tx }}</span>
            </div>
            <div class="ov-legend-label">
              <Icon name="arrowDown" size="12" class="text-muted" />
              {{ t('dashboard.down') }}
              <span class="ov-legend-num">{{ netRate.rx }}</span>
            </div>
          </div>
        </div>
        <div class="ov-wide-chart">
          <MiniChart
            :s1="netHistory.tx"
            :s2="netHistory.rx"
            color1="#ec4899"
            color2="#8b93a7"
            :height="170"
            :fill="0.24"
            :stroke-width="1.75"
            :show-tooltip="true"
            :labels="labels"
            name1="↑"
            name2="↓"
            :y-formatter="fmtRate"
            :ref-lines="netRefLines"
          />
        </div>
        <div class="ov-wide-foot">
          <div>
            <div class="ov-kicker">{{ t('dashboard.sent') }}</div>
            <div class="ov-foot-value">{{ sentTotal }}</div>
          </div>
          <span class="ov-foot-sep" />
          <div>
            <div class="ov-kicker">{{ t('dashboard.received') }}</div>
            <div class="ov-foot-value">{{ recvTotal }}</div>
          </div>
          <span class="ov-foot-sep" />
          <div>
            <div class="ov-kicker">{{ t('dashboard.avgWindow') }}</div>
            <div class="ov-foot-value">
              <span class="ov-foot-part">↑ {{ avgTx }}</span> <span class="ov-foot-part">↓ {{ avgRx }}</span>
            </div>
          </div>
        </div>
      </Card>

      <!-- 磁盘 IO 卡(仿 3x-ui ConnectionsCard 风格) -->
      <Card class="ov-tile ov-wide">
        <div class="ov-wide-head">
          <div>
            <div class="ov-kicker">{{ t('dashboard.io') }}</div>
            <div class="ov-sub">{{ t('dashboard.peak') }} {{ ioPeakText }}</div>
          </div>
          <div class="ov-wide-legend">
            <div class="ov-legend-label">
              <span class="ov-swatch" style="background: #34d399" />
              {{ t('dashboard.read') }}
              <span class="ov-legend-num">{{ ioRate.read }}</span>
            </div>
            <div class="ov-legend-label">
              <span class="ov-swatch" style="background: #fbbf24" />
              {{ t('dashboard.write') }}
              <span class="ov-legend-num">{{ ioRate.write }}</span>
            </div>
          </div>
        </div>
        <div class="ov-wide-chart">
          <MiniChart
            :s1="ioHistory.read"
            :s2="ioHistory.write"
            color1="#34d399"
            color2="#fbbf24"
            :height="170"
            :fill="0.24"
            :stroke-width="1.5"
            :show-tooltip="true"
            :labels="dockerLabels"
            :y-formatter="fmtRate"
            :ref-lines="ioRefLines"
          />
        </div>
      </Card>
    </div>

    <!-- ============ Docker 统计三卡:容器 / 镜像 / 卷(仿 3x-ui ConnectionsCard 风格) ============ -->
    <div class="ov-docker">
      <!-- 容器卡(数量 + 图标,点击跳转容器列表) -->
      <Card class="ov-tile ov-wide clickable count-card" @click="$router.push('/containers')">
        <div class="count-card-head">
          <div class="ov-kicker">{{ t('nav.containers') }}</div>
          <span class="count-card-icon"><Icon name="container" size="28" /></span>
        </div>
        <div class="count-card-num">{{ counts.total }}</div>
        <div class="count-card-sub">{{ t('dashboard.running') }} {{ counts.running }} · {{ t('dashboard.stopped') }} {{ counts.total - counts.running }}</div>
      </Card>

      <!-- 镜像卡(数量 + 图标,点击跳转镜像列表) -->
      <Card class="ov-tile ov-wide clickable count-card" @click="$router.push('/images')">
        <div class="count-card-head">
          <div class="ov-kicker">{{ t('nav.images') }}</div>
          <span class="count-card-icon"><Icon name="image" size="28" /></span>
        </div>
        <div class="count-card-num">{{ counts.images }}</div>
        <div class="count-card-sub">{{ t('dashboard.totalSize') }} {{ imageSizeText }}</div>
      </Card>

      <!-- 卷卡(数量 + 图标,点击跳转卷列表) -->
      <Card class="ov-tile ov-wide clickable count-card" @click="$router.push('/volumes')">
        <div class="count-card-head">
          <div class="ov-kicker">{{ t('nav.volumes') }}</div>
          <span class="count-card-icon"><Icon name="volume" size="28" /></span>
        </div>
        <div class="count-card-num">{{ counts.volumes }}</div>
        <div class="count-card-sub">{{ t('dashboard.mounted') }} {{ mountedVolumes }} · {{ t('dashboard.unmounted') }} {{ counts.volumes - mountedVolumes }}</div>
      </Card>
    </div>

    <!-- ============ 系统信息条(仿 3x-ui SystemStrip:3 列) ============ -->
    <Card class="ov-strip">
      <div class="ov-strip-grid">
        <div class="ov-strip-cell">
          <div class="ov-kicker ov-kicker-icon">
            <Icon name="clock" size="13" />
            {{ t('dashboard.uptime') }}
          </div>
          <div class="ov-strip-split">
            <div>
              <div class="ov-strip-sub">{{ t('dashboard.os') }}</div>
              <div class="ov-strip-value">{{ uptimeText }}</div>
            </div>
            <span class="ov-strip-split-sep" />
            <div>
              <div class="ov-strip-sub">{{ t('dashboard.serverTime') }}</div>
              <div class="ov-strip-value ov-mono time-v">{{ serverTimeText }}</div>
            </div>
          </div>
        </div>

        <div class="ov-strip-cell">
          <div class="ov-kicker ov-kicker-icon">
            <Icon name="box" size="13" />
            {{ t('dashboard.panel') }}
          </div>
          <div class="ov-strip-split">
            <div>
              <div class="ov-strip-sub">{{ t('dashboard.memory') }}</div>
              <div class="ov-strip-value">{{ panelMem }}</div>
            </div>
            <span class="ov-strip-split-sep" />
            <div>
              <div class="ov-strip-sub">{{ t('dashboard.threads') }}</div>
              <div class="ov-strip-value">{{ panelThreads }}</div>
            </div>
          </div>
        </div>

        <div class="ov-strip-cell">
          <div class="ov-kicker ov-kicker-icon">
            <Icon name="globe" size="13" />
            {{ t('dashboard.ipAddresses') }}
            <button type="button" class="ip-toggle" :title="t('dashboard.toggleIpVisibility')" @click="showIp = !showIp">
              <Icon :name="showIp ? 'eye' : 'eyeOff'" size="14" />
            </button>
          </div>
          <div class="ov-ip" :class="{ 'ip-hidden': !showIp }">
            <div class="ov-mono">{{ ipv4 || 'N/A' }}</div>
            <div class="ov-mono ov-ip-v6">{{ ipv6 || 'N/A' }}</div>
          </div>
        </div>
      </div>
    </Card>

    <!-- ============ 日志弹窗(仿 3x-ui LogModal:行数 + 自动更新 + 下载) ============ -->
    <Modal :model-value="logsOpen" size="2xl" :title="t('status.logs')" @close="logsOpen = false">
      <div class="log-toolbar">
        <Button variant="ghost" size="sm" :title="t('common.refresh')" @click="loadLogs">
          <Icon name="refresh" size="12" :class="{ 'animate-spin': logLoading }" />
        </Button>
        <Select v-model="logRows" style="width: 110px">
          <SelectTrigger class="!h-8 !text-xs" style="width: 110px"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem v-for="n in [20, 50, 100, 500, 1000]" :key="n" :value="String(n)">{{ n }}</SelectItem>
          </SelectContent>
        </Select>
        <label class="log-check"><input type="checkbox" v-model="logAuto" /> {{ t('status.autoUpdate') }}</label>
        <Button variant="brand" size="sm" class="ml-auto" @click="downloadLogs"><Icon name="download" size="12" /> {{ t('common.download') }}</Button>
      </div>
      <pre class="logs-view">{{ logsText }}</pre>
    </Modal>

    <!-- ============ 配置弹窗(仿 3x-ui ConfigModal:下载 + 复制) ============ -->
    <Modal :model-value="configOpen" size="xl" :title="t('status.config')" @close="configOpen = false">
      <pre class="logs-view">{{ configText }}</pre>
      <template #footer>
        <Button variant="ghost" size="sm" @click="downloadConfig"><Icon name="download" size="12" /> {{ t('common.download') }}</Button>
        <Button variant="brand" size="sm" @click="copyConfig"><Icon name="copy" size="12" /> {{ t('common.copy') }}</Button>
      </template>
    </Modal>

    <!-- ============ 备份与恢复弹窗(仿 3x-ui BackupModal:导出/导入列表) ============ -->
    <Modal :model-value="backupOpen" size="lg" :title="t('status.backupTitle')" @close="backupOpen = false">
      <div class="backup-list">
        <div class="backup-item">
          <div class="backup-meta">
            <div class="backup-title">{{ t('status.exportDatabase') }}</div>
            <div class="backup-description">{{ t('status.exportDatabaseDesc') }}</div>
          </div>
          <Button variant="brand" :title="t('status.exportDatabase')" @click="downloadBackup"><Icon name="download" size="14" /></Button>
        </div>
        <div class="backup-item">
          <div class="backup-meta">
            <div class="backup-title">{{ t('status.importDatabase') }}</div>
            <div class="backup-description">{{ t('status.importDatabaseDesc') }}</div>
          </div>
          <Button variant="brand" :title="t('status.importDatabase')" @click="backupInput?.click()"><Icon name="upload" size="14" /></Button>
        </div>
      </div>
      <input ref="backupInput" type="file" accept=".tar.gz,.gz" class="hidden" @change="restoreBackup" />
      <p class="text-[11px] text-muted px-5 pb-4">{{ t('status.restoreConfirm') }}</p>
    </Modal>

    <!-- ============ 系统历史弹窗(仿 3x-ui SystemHistoryModal:Tabs + 图) ============ -->
    <Modal :model-value="historyOpen" size="xl" :title="t('status.systemHistory')" @close="historyOpen = false">
      <div class="h-tabs px-4 pt-3">
        <button
          v-for="tab in historyTabs"
          :key="tab.key"
          type="button"
          class="h-tab"
          :class="{ active: historyTab === tab.key }"
          @click="historyTab = tab.key"
        >
          <Icon :name="tab.icon" size="13" class="inline mr-1 align-[-2px]" /> {{ t(tab.labelKey) }}
        </button>
      </div>
      <div class="p-5">
        <div class="flex items-center justify-between text-[12px] mb-2">
          <span class="text-muted">{{ historyChart.title }}</span>
          <span class="font-mono">{{ historyChart.current }}</span>
        </div>
        <MiniChart
          :s1="historyChart.s1"
          :s2="historyChart.s2"
          :color1="historyChart.color1"
          :color2="historyChart.color2"
          :height="200"
          :fill="0.24"
          :stroke-width="1.5"
          :show-tooltip="true"
          :labels="labels"
          :y-formatter="historyChart.fmt"
          :value-max="historyChart.valueMax"
          :ref-lines="historyChart.refLines"
        />
      </div>
    </Modal>
  </div>
</template>

<script setup>
import { computed, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import MiniChart from '../components/MiniChart.vue'
import Icon from '../components/Icon.vue'
import Modal from '../components/Modal.vue'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { api, getToken, entrancePath } from '../api'
import { formatBytes } from '../util'
import { toastErr, toastOk } from '../toast'
import { useConfirm } from '../confirm'
import { licenseActive } from '../store'

const { t } = useI18n()
const askConfirm = useConfirm()

// ---------- 基础数据 ----------
const containers = ref([])
const images = ref(0)
const imagesSize = ref(0)
const volumes = ref(0)
const mountedVolumes = ref(0)
const host = ref(null)
const dockerOk = ref(false)
const dockerVersion = computed(() => (host.value?.docker_version ? 'v' + host.value.docker_version : '?'))

// ---------- 监控 ----------
const mon = ref({ cpu_pct: 0, mem: null, load: null, swap: null, disk: null, panel: null, publicIP: null, net: { rx: 0, tx: 0 }, io: { read: 0, write: 0 } })
const netHistory = ref({ rx: [], tx: [] }) // 速率序列(B/s)
const ioHistory = ref({ read: [], write: [] }) // 速率序列(B/s)
const hist = ref({ cpu: [], mem: [], swap: [], disk: [], containerCount: [], imageCount: [], volumeCount: [] })
const labels = ref([]) // 3s 采样时间标签(吞吐/IO)
const dockerLabels = ref([]) // 15s 采样时间标签(容器/镜像/卷)
const netRate = ref({ rx: '0 B/s', tx: '0 B/s' })
const ioRate = ref({ read: '0 B/s', write: '0 B/s' })
let monTimer = null
let dockerTimer = null
let clockTimer = null

let timeOffset = 0
const serverTimeText = ref('-')

const counts = computed(() => ({
  total: containers.value.length,
  running: containers.value.filter((c) => c.State === 'running').length,
  images: images.value,
  volumes: volumes.value,
}))

// ---------- 计算值 ----------
const swapPct = computed(() => mon.value.swap?.pct ?? 0)
const swapSub = computed(() => {
  const s = mon.value.swap
  return s ? `${fmtBytes(s.used)} / ${fmtBytes(s.total)}` : '-'
})
const memPct = computed(() => mon.value.mem?.pct ?? 0)
const memSub = computed(() => {
  const m = mon.value.mem
  return m ? `${fmtBytes(m.used)} / ${fmtBytes(m.total)}` : '-'
})
const memUsedText = computed(() => (mon.value.mem ? `${fmtBytes(mon.value.mem.used)} / ${fmtBytes(mon.value.mem.total)}` : '-'))
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
const imageSizeText = computed(() => fmtBytes(imagesSize.value))
const imageAvgText = computed(() => (images.value ? fmtBytes(imagesSize.value / images.value) : '-'))
const panelMem = computed(() => (mon.value.panel ? fmtBytes(mon.value.panel.mem) : '-'))
const panelThreads = computed(() => (mon.value.panel ? String(mon.value.panel.threads) : '-'))
const sentTotal = computed(() => fmtBytes(mon.value.net?.tx_total ?? 0))
const recvTotal = computed(() => fmtBytes(mon.value.net?.rx_total ?? 0))
const avgTx = computed(() => fmtRate(avg(netHistory.value.tx)))
const avgRx = computed(() => fmtRate(avg(netHistory.value.rx)))
const netPeakText = computed(() => fmtRate(peak(netHistory.value.tx) > peak(netHistory.value.rx) ? peak(netHistory.value.tx) : peak(netHistory.value.rx)))
const ioPeakText = computed(() => fmtRate(Math.max(peak(ioHistory.value.read), peak(ioHistory.value.write))))

// ---------- 四张状态卡(仿 3x-ui VitalTile) ----------
const vitals = computed(() => [
  {
    icon: 'cpu', label: t('dashboard.cpuUsage'), percent: mon.value.cpu_pct, color: '#ec4899',
    detail: cpuSub.value, data: hist.value.cpu,
    footLeft: `${t('dashboard.avg')} ${avg(hist.value.cpu).toFixed(0)}%`,
    footRight: `${t('dashboard.peak')} ${peak(hist.value.cpu).toFixed(0)}%`,
  },
  {
    icon: 'memory', label: t('dashboard.memUsage'), percent: memPct.value, color: '#a78bfa',
    detail: memSub.value, data: hist.value.mem,
    footLeft: `${t('dashboard.avg')} ${avg(hist.value.mem).toFixed(0)}%`,
    footRight: `${t('dashboard.peak')} ${peak(hist.value.mem).toFixed(0)}%`,
  },
  {
    icon: 'swap', label: t('dashboard.swap'), percent: swapPct.value, color: '#fbbf24',
    detail: swapSub.value, data: hist.value.swap,
    footLeft: `${t('dashboard.avg')} ${avg(hist.value.swap).toFixed(0)}%`,
    footRight: `${t('dashboard.peak')} ${peak(hist.value.swap).toFixed(0)}%`,
  },
  {
    icon: 'drive', label: t('dashboard.storage'), percent: diskPct.value, color: '#34d399',
    detail: diskSub.value, data: hist.value.disk,
    footLeft: `${t('status.free')} ${freeDisk.value}`,
    footRight: `${t('dashboard.avg')} ${avg(hist.value.disk).toFixed(0)}%`,
  },
])

// 均值参考线(仿 3x-ui VitalTile referenceLines)
function meanRef(data, color) {
  if (data.length < 2) return []
  return [{ y: avg(data), dash: '3 4', color: 'color-mix(in srgb, var(--dm-muted) 55%, transparent)' }]
}
const netRefLines = computed(() => [
  { y: parseRate(netRate.value.rx), color: '#8b93a7', dash: '2 4' },
  { y: parseRate(netRate.value.tx), color: '#ec4899', dash: '2 4' },
])
const ioRefLines = computed(() => [
  { y: parseRate(ioRate.value.read), color: '#34d399', dash: '2 4' },
  { y: parseRate(ioRate.value.write), color: '#fbbf24', dash: '2 4' },
])

// ---------- 健康检查(仿 3x-ui:≥90 红 / ≥75 黄) ----------
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
const fmtRate = (v) => (v == null ? '-' : formatBytes(v, 1) + '/s')
const fmtInt = (v) => Math.round(v).toLocaleString()
const avg = (arr) => (arr.length ? arr.reduce((a, b) => a + b, 0) / arr.length : 0)
const peak = (arr) => (arr.length ? Math.max(...arr) : 0)
function parseRate(s) {
  const m = String(s).match(/^([\d.]+)/)
  const unit = String(s).match(/([KMG]?B)\/s$/)
  if (!m) return 0
  const mult = unit ? { B: 1, KB: 1024, MB: 1024 ** 2, GB: 1024 ** 3 }[unit[1]] || 1 : 1
  return Number(m[1]) * mult
}

// ---------- 加载 ----------
async function loadBase() {
  try {
    const [cs, imgs, vols] = await Promise.all([api('/containers'), api('/images'), api('/volumes')])
    dockerOk.value = true
    containers.value = cs
    images.value = imgs.length
    imagesSize.value = imgs.reduce((a, b) => a + (Number(b.Size) || 0), 0)
    volumes.value = vols.length
    mountedVolumes.value = new Set(
      (cs || []).flatMap((c) => (c.Mounts || []).filter((m) => m.Type === 'volume').map((m) => m.Name).filter(Boolean),
    )).size
    if (!host.value) {
      try {
        const h = await api('/system/host')
        host.value = h
        if (h.server_time) timeOffset = h.server_time * 1000 - Date.now()
      } catch { /* ignore */ }
    }
    // 容器/镜像/卷趋势(15s 节奏)
    pushHistory(hist.value.containerCount, containers.value.length)
    pushHistory(hist.value.imageCount, images.value)
    pushHistory(hist.value.volumeCount, volumes.value)
    pushLabel(dockerLabels.value)
  } catch {
    dockerOk.value = false
  }
}

async function loadMonitor() {
  try {
    const m = await api('/system/monitor')
    // 速率由后端 8 秒采样差分直接给出(B/s),前端不再做累计值差分(避免缓存导致的 0 锯齿)
    const rx = m.net?.rx_rate ?? 0
    const tx = m.net?.tx_rate ?? 0
    const rd = m.io?.read_rate ?? 0
    const wr = m.io?.write_rate ?? 0
    netRate.value.rx = fmtRate(rx)
    netRate.value.tx = fmtRate(tx)
    pushHistory(netHistory.value.rx, rx)
    pushHistory(netHistory.value.tx, tx)
    ioRate.value.read = fmtRate(rd)
    ioRate.value.write = fmtRate(wr)
    pushHistory(ioHistory.value.read, rd)
    pushHistory(ioHistory.value.write, wr)

    mon.value = m
    pushHistory(hist.value.cpu, m.cpu_pct)
    pushHistory(hist.value.mem, m.mem?.pct ?? 0)
    pushHistory(hist.value.swap, m.swap?.pct ?? 0)
    pushHistory(hist.value.disk, m.disk?.pct ?? 0)
    pushLabel(labels.value)
  } catch { /* ignore */ }
}

function pushHistory(arr, v) {
  arr.push(v)
  if (arr.length > 120) arr.shift()
}
function pushLabel(arr) {
  arr.push(new Date(Date.now() + timeOffset).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }))
  if (arr.length > 120) arr.shift()
}

// ---------- 公网 IP(仿 3x-ui status.publicIP:随 monitor 轮询携带,默认隐藏,眼睛切换) ----------
const showIp = ref(false)
const ipv4 = computed(() => mon.value.publicIP?.ipv4 || '')
const ipv6 = computed(() => mon.value.publicIP?.ipv6 || '')

function refreshAll() {
  loadBase()
  loadMonitor()
}

// ---------- 操作栏弹窗(仿 3x-ui LogModal / ConfigModal / BackupModal) ----------
const logsOpen = ref(false)
const logsText = ref('')
const logRows = ref('50')
const logAuto = ref(false)
const logLoading = ref(false)
let logTimer = null
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
  logLoading.value = true
  try {
    const r = await api('/system/logs?lines=' + logRows.value)
    logsText.value = (r.logs || []).join('\n') || '-'
  } catch (e) {
    logsText.value = e.message
  } finally {
    logLoading.value = false
  }
}
function downloadLogs() {
  const blob = new Blob([logsText.value], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'docker-manager.log'
  a.click()
  URL.revokeObjectURL(url)
}
watch(logAuto, (v) => {
  clearInterval(logTimer)
  if (v && logsOpen.value) {
    logTimer = setInterval(loadLogs, 5000)
  }
})
watch(logsOpen, (v) => {
  if (!v) {
    clearInterval(logTimer)
    logAuto.value = false
  }
})

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
function downloadConfig() {
  const blob = new Blob([configText.value], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'config.json'
  a.click()
  URL.revokeObjectURL(url)
}

function openBackup() {
  backupOpen.value = true
}
async function downloadBackup() {
  try {
    const resp = await fetch(entrancePath('/api/system/backup'), {
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
  const ok = await askConfirm(t('status.restoreConfirm'), { title: t('status.restore'), danger: true, confirmText: t('status.restore') })
  if (!ok) return
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

const historyTab = ref('cpu')
const historyTabs = [
  { key: 'cpu', icon: 'cpu', labelKey: 'dashboard.cpuUsage' },
  { key: 'mem', icon: 'memory', labelKey: 'dashboard.memUsage' },
  { key: 'net', icon: 'network', labelKey: 'dashboard.network' },
  { key: 'io', icon: 'drive', labelKey: 'dashboard.io' },
  { key: 'disk', icon: 'stats', labelKey: 'dashboard.storage' },
]
const historyChart = computed(() => {
  switch (historyTab.value) {
    case 'mem':
      return {
        title: `${t('dashboard.memUsage')} / ${t('dashboard.swap')}`,
        current: `${memPct.value.toFixed(1)}% / ${swapPct.value.toFixed(1)}%`,
        s1: hist.value.mem, s2: hist.value.swap, color1: '#a78bfa', color2: '#fbbf24',
        fmt: (v) => v.toFixed(1) + '%', valueMax: 100,
        refLines: meanRef(hist.value.mem, '#a78bfa'),
      }
    case 'net':
      return {
        title: `${t('dashboard.up')} / ${t('dashboard.down')}`,
        current: `${netRate.value.tx} / ${netRate.value.rx}`,
        s1: netHistory.value.tx, s2: netHistory.value.rx, color1: '#ec4899', color2: '#8b93a7',
        fmt: fmtRate, valueMax: null,
        refLines: netRefLines.value,
      }
    case 'io':
      return {
        title: `${t('dashboard.read')} / ${t('dashboard.write')}`,
        current: `${ioRate.value.read} / ${ioRate.value.write}`,
        s1: ioHistory.value.read, s2: ioHistory.value.write, color1: '#34d399', color2: '#fbbf24',
        fmt: fmtRate, valueMax: null,
        refLines: ioRefLines.value,
      }
    case 'disk':
      return {
        title: t('dashboard.storage'),
        current: `${diskPct.value.toFixed(1)}%`,
        s1: hist.value.disk, s2: [], color1: '#34d399', color2: '',
        fmt: (v) => v.toFixed(1) + '%', valueMax: 100,
        refLines: meanRef(hist.value.disk, '#34d399'),
      }
    default:
      return {
        title: t('dashboard.cpuUsage'),
        current: `${(mon.value.cpu_pct ?? 0).toFixed(1)}%`,
        s1: hist.value.cpu, s2: [], color1: '#ec4899', color2: '',
        fmt: (v) => v.toFixed(1) + '%', valueMax: 100,
        refLines: meanRef(hist.value.cpu, '#ec4899'),
      }
  }
})

async function panelRestart() {
  const ok = await askConfirm(t('status.restartConfirm'), { title: t('status.restart'), danger: true, confirmText: t('common.restart') })
  if (!ok) return
  api('/system/restart', { method: 'POST' })
    .then(() => toastOk(t('status.restarting')))
    .catch((e) => toastErr(e.message))
}

// ---------- 定时器 ----------
function startTimers() {
  refreshAll()
  monTimer = setInterval(loadMonitor, 3000)
  dockerTimer = setInterval(loadBase, 15000)
  clockTimer = setInterval(() => {
    serverTimeText.value = new Date(Date.now() + timeOffset).toLocaleTimeString()
  }, 1000)
}

onMounted(() => {
  startTimers()
})
onActivated(() => {
  if (!monTimer) startTimers()
})
onDeactivated(() => {
  clearInterval(monTimer)
  monTimer = null
  clearInterval(dockerTimer)
  dockerTimer = null
  clearInterval(clockTimer)
  clockTimer = null
})
onBeforeUnmount(() => {
  clearInterval(monTimer)
  clearInterval(dockerTimer)
  clearInterval(clockTimer)
  clearInterval(logTimer)
})
</script>

<style scoped>
/* ============ 3x-ui IndexPage ov-* 样式移植(变量映射到本面板主题) ============ */
.status-page {
  --ov-accent: var(--color-brand);
  --ov-line: var(--dm-line);
  --ov-label: var(--dm-muted);
  --ov-faint: color-mix(in srgb, var(--dm-muted) 82%, transparent);
  --ov-gap: 12px;
  --ov-pad: 20px;
}

.ov-page {
  display: flex;
  flex-direction: column;
  gap: var(--ov-gap);
}

/* ---------- 操作栏 ---------- */
.ov-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.ov-state {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 4px 12px;
  border: 1px solid var(--ov-line);
  border-radius: 999px;
  font-size: 13px;
  color: var(--dm-text);
}
.ov-state-dot {
  position: relative;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
  flex: none;
}
.ov-state[data-state='running'] .ov-state-dot::after {
  content: '';
  position: absolute;
  inset: -1px;
  border-radius: 50%;
  border: 1px solid currentColor;
  animation: ovPulse 1.6s infinite ease-out;
}
@keyframes ovPulse {
  0% { transform: scale(0.9); opacity: 0.5; }
  100% { transform: scale(2.4); opacity: 0; }
}
@media (prefers-reduced-motion: reduce) {
  .ov-state[data-state='running'] .ov-state-dot::after {
    animation: none;
  }
}
.ov-state-version {
  padding: 0;
  border: 0;
  background: transparent;
  font: inherit;
  cursor: pointer;
  color: var(--ov-label);
  transition: color 0.2s;
}
.ov-state-version:hover {
  color: var(--ant-color-primary, var(--color-brand));
}
.ov-bar-actions {
  margin-inline-start: auto;
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}
@media (max-width: 768px) {
  .ov-bar-actions {
    margin-inline-start: 0;
    width: 100%;
    justify-content: space-between;
  }
}
.ov-bar-sep {
  width: 1px;
  height: 20px;
  background: var(--ov-line);
  margin: 0 4px;
}
.ov-bar-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 10px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--ov-label);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s;
}
.ov-bar-btn:hover {
  color: var(--color-brand);
  background: color-mix(in srgb, var(--color-brand) 8%, transparent);
}
.ov-bar-btn.primary {
  border-color: color-mix(in srgb, var(--color-brand) 55%, transparent);
  color: var(--color-brand);
  font-weight: 500;
}
.ov-bar-btn.primary:hover {
  background: color-mix(in srgb, var(--color-brand) 12%, transparent);
  border-color: var(--color-brand);
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
.ab-badge.pro { color: #ec4899; }
.ab-badge.free { color: var(--dm-muted); }

/* ---------- 健康条 / 分隔线 ---------- */
.ov-health {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12.5px;
}
.ov-health-mark {
  width: 14px;
  height: 1px;
  background: currentColor;
  flex: none;
}
.ov-rule {
  height: 1px;
  border: 0;
  margin: 0;
  background: linear-gradient(to right, transparent, var(--ov-line) 48px, var(--ov-line) calc(100% - 48px), transparent);
}

/* ---------- 通用 ---------- */
.ov-kicker {
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--ov-label);
}
.ov-kicker-icon {
  display: flex;
  align-items: center;
  gap: 7px;
}
.ov-sub {
  font-size: 12.5px;
  margin-top: 4px;
  color: var(--ov-faint);
}
.ov-mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.card {
  background: var(--dm-surface);
  border: 1px solid var(--dm-line);
  border-radius: 14px;
}

/* ---------- vitals 四卡 ---------- */
.ov-vitals {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: var(--ov-gap);
}
@media (max-width: 1100px) {
  .ov-vitals { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
@media (max-width: 560px) {
  .ov-vitals { grid-template-columns: minmax(0, 1fr); }
}
.ov-tile {
  overflow: hidden;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.ov-tile.clickable {
  cursor: pointer;
}

/* ---------- 容器/镜像/卷数量卡(大图标 + 数量,点击跳转) ---------- */
.count-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 22px;
  min-height: 170px;
  justify-content: center;
}
.count-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.count-card-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border-radius: 16px;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--color-brand) 20%, transparent),
    color-mix(in srgb, var(--color-brand) 6%, transparent)
  );
  color: var(--color-brand);
  box-shadow: 0 4px 16px color-mix(in srgb, var(--color-brand) 20%, transparent);
  transition: transform 0.15s ease;
}
.count-card:hover .count-card-icon {
  transform: scale(1.06);
}
.count-card-num {
  font-size: 48px;
  font-weight: 700;
  line-height: 1.05;
  letter-spacing: -0.02em;
  color: var(--dm-text);
  font-variant-numeric: tabular-nums;
}
.count-card-sub {
  font-size: 12px;
  color: var(--ov-faint);
}
.ov-tile:hover {
  border-color: color-mix(in srgb, var(--color-brand) 40%, var(--dm-line));
  box-shadow: 0 4px 18px rgba(0, 0, 0, 0.06);
}
.ov-tile-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px var(--ov-pad) 0;
  color: var(--ov-accent);
}
.ov-tile-icon {
  display: inline-flex;
  font-size: 15px;
}
.ov-tile-value {
  display: flex;
  align-items: baseline;
  gap: 4px;
  padding: 12px var(--ov-pad) 0;
}
.ov-tile-number {
  font-size: 34px;
  font-weight: 600;
  line-height: 1;
  letter-spacing: -0.02em;
  color: var(--dm-text);
  font-variant-numeric: tabular-nums;
}
.ov-tile-unit {
  font-size: 14px;
  color: var(--ov-label);
}
.ov-tile-detail {
  padding: 5px var(--ov-pad) 0;
  font-size: 12px;
  color: var(--ov-label);
}
.ov-tile-foot {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  padding: 14px var(--ov-pad) 0;
  font-size: 10px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--ov-faint);
}
.ov-tile-chart {
  margin-top: 6px;
}

/* ---------- 吞吐 + 容器(mid 2fr/1fr) ---------- */
.ov-mid {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(0, 1fr);
  gap: var(--ov-gap);
}
@media (max-width: 1100px) {
  .ov-mid { grid-template-columns: minmax(0, 1fr); }
}

/* ---------- Docker 三卡(镜像/卷/IO) ---------- */
.ov-docker {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--ov-gap);
}
@media (max-width: 1100px) {
  .ov-docker { grid-template-columns: minmax(0, 1fr); }
}

.ov-wide-head {
  display: flex;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 16px;
  padding: var(--ov-pad) var(--ov-pad) 0;
}
.ov-wide-head-stack {
  flex-direction: column;
  gap: 0;
}
.ov-wide-legend {
  margin-inline-start: auto;
  display: flex;
  gap: 22px;
  text-align: end;
}
.ov-legend-label {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
  font-size: 11px;
  color: var(--ov-label);
}
.ov-legend-num {
  font-size: 13px;
  font-weight: 600;
  color: var(--dm-text);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.ov-conn-total {
  display: flex;
  align-items: baseline;
  gap: 6px;
  margin-top: 12px;
}
.ov-conn-legend {
  display: flex;
  gap: 16px;
  padding: 16px var(--ov-pad) 0;
}
.ov-conn-legend > div {
  flex: 1 1 0;
}
.ov-conn-legend .ov-legend-label {
  justify-content: flex-start;
}
.ov-swatch {
  width: 14px;
  height: 2px;
  flex: none;
}
.ov-wide-chart {
  padding: 12px 8px 0;
}
.ov-wide-foot {
  display: flex;
  gap: 16px;
  margin: 12px var(--ov-pad) 0;
  padding: 14px 0 var(--ov-pad);
  border-top: 1px solid var(--ov-line);
}
.ov-wide-foot > div {
  flex: 1 1 0;
}
.ov-foot-sep {
  width: 1px;
  background: var(--ov-line);
}
.ov-foot-value {
  font-size: 18px;
  font-weight: 600;
  margin-top: 4px;
  color: var(--dm-text);
  font-variant-numeric: tabular-nums;
}
.ov-foot-part {
  display: inline-block;
  white-space: nowrap;
}
@media (max-width: 560px) {
  .ov-wide-foot {
    flex-direction: column;
    gap: 10px;
  }
  .ov-wide-foot .ov-foot-sep {
    display: none;
  }
}

/* ---------- 系统信息条(3 列) ---------- */
.ov-strip-grid {
  display: grid;
  grid-template-columns: minmax(max-content, 1.2fr) minmax(max-content, 1.2fr) minmax(0, 1.6fr);
  gap: 16px;
  padding: var(--ov-pad);
}
@media (max-width: 1439px) {
  .ov-strip-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
}
@media (max-width: 1100px) {
  .ov-strip-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
@media (max-width: 560px) {
  .ov-strip-grid { grid-template-columns: minmax(0, 1fr); }
}
@media (min-width: 1440px) {
  .ov-strip-cell + .ov-strip-cell {
    border-inline-start: 1px solid var(--ov-line);
    padding-inline-start: 16px;
  }
}
.ov-strip-value {
  font-size: 19px;
  font-weight: 600;
  margin-top: 6px;
  color: var(--dm-text);
  font-variant-numeric: tabular-nums;
}
.ov-strip-value.sm {
  font-size: 15px;
}
.ov-strip-value.time-v {
  font-size: 15px;
}
.ov-strip-split {
  display: flex;
  align-items: stretch;
  gap: 14px;
}
.ov-strip-split-sep {
  width: 1px;
  background: var(--ov-line);
  margin-top: 8px;
}
.ov-strip-sub {
  font-size: 10px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  margin-top: 8px;
  color: var(--ov-faint);
}
.ov-strip-sub + .ov-strip-value {
  margin-top: 2px;
}

/* ---------- IP 地址(眼睛切换,仿 3x-ui) ---------- */
.ip-toggle {
  margin-inline-start: auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--ov-faint);
  cursor: pointer;
  transition: color 0.2s;
}
.ip-toggle:hover {
  color: var(--color-brand);
}
.ov-ip {
  margin-top: 7px;
  font-size: 13px;
  overflow-wrap: anywhere;
  transition: filter 0.2s ease;
}
.ov-ip-v6 {
  margin-top: 3px;
  color: var(--ov-label);
}
.ip-hidden {
  filter: blur(6px);
}

/* ---------- 弹窗(仿 3x-ui LogModal / BackupModal / SystemHistoryModal) ---------- */
.h-tabs {
  display: flex;
  gap: 4px;
  border-bottom: 1px solid var(--dm-line);
  padding: 0 4px;
}
.h-tab {
  position: relative;
  padding: 10px 14px;
  border: none;
  background: transparent;
  color: var(--dm-muted);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: color 0.2s;
}
.h-tab:hover {
  color: var(--dm-text);
}
.h-tab.active {
  color: var(--color-brand);
}
.h-tab.active::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  bottom: -1px;
  height: 2px;
  border-radius: 2px 2px 0 0;
  background: var(--color-brand);
}
.log-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--dm-line);
}
.log-check {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12.5px;
  color: var(--dm-muted);
  cursor: pointer;
  user-select: none;
}
.log-check input {
  accent-color: var(--color-brand);
}
.backup-list {
  padding: 6px 16px;
}
.backup-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 14px 0;
  border-bottom: 1px solid var(--dm-line);
}
.backup-item:last-of-type {
  border-bottom: none;
}
.backup-meta {
  flex: 1;
  min-width: 0;
}
.backup-title {
  font-size: 13.5px;
  font-weight: 600;
  color: var(--dm-text);
}
.backup-description {
  font-size: 11.5px;
  color: var(--dm-muted);
  margin-top: 3px;
  line-height: 1.45;
}

/* ---------- 弹窗 ---------- */
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
