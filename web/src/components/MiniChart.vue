<template>
  <div class="mc-wrap" :style="{ height: props.height + 'px' }" @mousemove="onMove" @mouseleave="tip.show = false">
    <svg :viewBox="`0 0 ${W} ${H}`" preserveAspectRatio="none" class="w-full h-full block">
      <!-- 网格线(4 等分虚线) -->
      <line v-for="i in 3" :key="i" x1="0" :y1="(H / 4) * i" :x2="W" :y2="(H / 4) * i" stroke="var(--dm-line)" stroke-width="0.5" stroke-dasharray="3 3" opacity="0.5" />

      <!-- y 轴刻度文字 -->
      <g v-if="props.showAxes">
        <text v-for="ty in yTicks" :key="'y' + ty" :x="1" :y="yPos(ty) + 3" font-size="3.2" fill="var(--dm-muted)" text-anchor="start">{{ fmtY(ty) }}</text>
      </g>

      <!-- x 轴时间标签 -->
      <g v-if="props.showAxes">
        <text
          v-for="(ti, i) in xTicks"
          :key="'x' + i"
          :x="xPos(ti.index)"
          :y="H - 1"
          font-size="3.2"
          fill="var(--dm-muted)"
          :text-anchor="ti.anchor"
        >{{ ti.label }}</text>
      </g>

      <!-- 参考线(均值线/当前值线) -->
      <g v-for="(rl, i) in props.refLines" :key="'rl' + i">
        <line :x1="0" :y1="yPos(rl.y)" :x2="W" :y2="yPos(rl.y)" :stroke="rl.color || props.color1" stroke-width="0.6" :stroke-dasharray="rl.dash || '4 3'" />
        <text v-if="rl.label" :x="W - 1" :y="yPos(rl.y) - 1" font-size="3" :fill="rl.color || props.color1" font-weight="600" text-anchor="end">{{ rl.label }}</text>
      </g>

      <!-- 面积渐变填充 -->
      <polygon v-if="s1.length > 1 && props.fill" :points="fillPoints(s1)" fill="url(#grad1)" />
      <polygon v-if="s2.length > 1 && props.fill" :points="fillPoints(s2)" fill="url(#grad2)" />

      <!-- 折线 -->
      <path v-if="s1.length > 1" :d="linePath(s1)" fill="none" :stroke="props.color1" :stroke-width="props.strokeWidth" stroke-linejoin="round" stroke-linecap="round" />
      <path v-if="s2.length > 1" :d="linePath(s2)" fill="none" :stroke="props.color2" :stroke-width="props.strokeWidth" stroke-linejoin="round" stroke-linecap="round" />

      <defs>
        <linearGradient id="grad1" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" :stop-color="props.color1" :stop-opacity="props.fill" />
          <stop offset="100%" :stop-color="props.color1" stop-opacity="0" />
        </linearGradient>
        <linearGradient id="grad2" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" :stop-color="props.color2" :stop-opacity="props.fill" />
          <stop offset="100%" :stop-color="props.color2" stop-opacity="0" />
        </linearGradient>
      </defs>
    </svg>

    <div v-if="!s1.length && !s2.length" class="absolute inset-0 flex items-center justify-center text-[12px] text-muted">
      {{ props.emptyText }}
    </div>

    <!-- tooltip(仿 3x-ui Sparkline) -->
    <div
      v-if="props.showTooltip && tip.show"
      class="mc-tip"
      :style="{ left: tip.x + 'px', top: tip.y + 'px' }"
    >
      <div class="mc-tip-label">{{ tip.label }}</div>
      <div v-for="(row, i) in tip.rows" :key="i" class="mc-tip-row">
        <span v-if="tip.rows.length > 1" class="mc-tip-dot" :style="{ background: row.color }" />
        <span v-if="tip.rows.length > 1" class="mc-tip-name">{{ row.name }}</span>
        <span class="mc-tip-val">{{ row.text }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, reactive } from 'vue'

const props = defineProps({
  /** 第一条序列(数值数组) */
  s1: { type: Array, default: () => [] },
  /** 第二条序列 */
  s2: { type: Array, default: () => [] },
  color1: { type: String, default: '#60a5fa' },
  color2: { type: String, default: '#ec4899' },
  emptyText: { type: String, default: '' },
  /** 图表高度(px) */
  height: { type: Number, default: 40 },
  /** 面积填充透明度(0 = 不填充) */
  fill: { type: Number, default: 0 },
  /** 折线宽度 */
  strokeWidth: { type: Number, default: 1.6 },
  /** 平滑曲线 */
  smooth: { type: Boolean, default: false },
  /** 参考线 [{ y, label?, color?, dash? }] */
  refLines: { type: Array, default: () => [] },
  /** x 轴时间标签(与序列等长) */
  labels: { type: Array, default: () => [] },
  /** 显示坐标轴 */
  showAxes: { type: Boolean, default: false },
  /** 悬停 tooltip */
  showTooltip: { type: Boolean, default: false },
  /** 系列名(tooltip 用) */
  name1: { type: String, default: '' },
  name2: { type: String, default: '' },
  /** y 值格式化 */
  yFormatter: { type: Function, default: null },
  /** y 域上限(null = 自动 = 最大值 * 1.1) */
  valueMax: { type: Number, default: null },
  valueMin: { type: Number, default: 0 },
  /** x 轴刻度数量 */
  tickCountX: { type: Number, default: 4 },
})

const W = 100
const H = computed(() => props.height)

// ---------- 归一化(与 3x-ui Sparkline 一致:valueMax 固定域或自动 max*1.1) ----------
const norm = computed(() => {
  const a = props.s1.slice(-120)
  const b = props.s2.slice(-120)
  const n = Math.max(a.length, b.length)
  let max = props.valueMin
  if (props.valueMax != null) max = props.valueMax
  else {
    for (const v of [...a, ...b]) if (Number.isFinite(v) && v > max) max = v
    if (max <= props.valueMin) max = props.valueMin + 1
    max = max * 1.1
  }
  // 截齐到同一长度(与 label 对齐)
  const pad = (arr) => {
    const s = arr.slice(-n)
    return new Array(n - s.length).fill(null).concat(s)
  }
  return { a: pad(a), b: pad(b), max, n }
})

const yPos = (v) => H.value - 2 - (clamp(v, props.valueMin, norm.value.max) / (norm.value.max - props.valueMin || 1)) * (H.value - 8)
const xPos = (i) => (norm.value.n === 1 ? 0 : (i / (norm.value.n - 1)) * W)
const clamp = (v, lo, hi) => (v == null ? lo : Math.min(hi, Math.max(lo, v)))
const s1 = computed(() => norm.value.a)
const s2 = computed(() => norm.value.b)

// ---------- 路径 ----------
function smoothPath(arr) {
  // Catmull-Rom → cubic Bezier 平滑
  const pts = arr.map((v, i) => [xPos(i), yPos(v)])
  if (pts.length < 3) return 'M' + pts.map((p) => `${p[0]},${p[1]}`).join(' L')
  let d = `M${pts[0][0]},${pts[0][1]}`
  for (let i = 0; i < pts.length - 1; i++) {
    const p0 = pts[Math.max(0, i - 1)]
    const p1 = pts[i]
    const p2 = pts[i + 1]
    const p3 = pts[Math.min(pts.length - 1, i + 2)]
    const c1 = [p1[0] + (p2[0] - p0[0]) / 6, p1[1] + (p2[1] - p0[1]) / 6]
    const c2 = [p2[0] - (p3[0] - p1[0]) / 6, p2[1] - (p3[1] - p1[1]) / 6]
    d += ` C${c1[0]},${c1[1]} ${c2[0]},${c2[1]} ${p2[0]},${p2[1]}`
  }
  return d
}
function linePath(arr) {
  if (arr.length < 2) return ''
  if (!props.smooth) return 'M' + arr.map((v, i) => `${xPos(i)},${yPos(v)}`).join(' L')
  return smoothPath(arr)
}
function fillPoints(arr) {
  const pts = arr.map((v, i) => [xPos(i), yPos(v)])
  return `${pts.map((p) => `${p[0]},${p[1]}`).join(' ')} ${xPos(pts.length - 1)},${H.value - 2} ${xPos(0)},${H.value - 2}`
}

// ---------- 坐标轴 ----------
const yTicks = computed(() => {
  if (!props.showAxes) return []
  const [lo, hi] = [props.valueMin, norm.value.max]
  if (props.valueMax === 100 && props.valueMin === 0) return [0, 25, 50, 75, 100]
  return Array.from({ length: 5 }, (_, i) => lo + ((hi - lo) * i) / 4)
})
const xTicks = computed(() => {
  if (!props.showAxes || !norm.value.n) return []
  const m = Math.max(2, props.tickCountX)
  const idxs = Array.from({ length: m }, (_, i) => Math.round((i * (norm.value.n - 1)) / (m - 1)))
  return idxs.map((i, k) => ({
    index: i,
    label: String(props.labels[i] ?? ''),
    anchor: k === 0 ? 'start' : k === m - 1 ? 'end' : 'middle',
  }))
})
const fmtY = (v) => (props.yFormatter ? props.yFormatter(v) : String(Math.round(v)))

// ---------- tooltip ----------
const tip = reactive({ show: false, x: 0, y: 0, label: '', rows: [] })
function onMove(ev) {
  if (!props.showTooltip || !norm.value.n) return
  const rect = ev.currentTarget.getBoundingClientRect()
  const ratio = W / rect.width
  const mx = (ev.clientX - rect.left) * ratio
  const idx = clamp(Math.round((mx / W) * (norm.value.n - 1)), 0, norm.value.n - 1)
  const label = props.labels[idx] ?? ''
  const rows = []
  if (s1.value[idx] != null) rows.push({ name: props.name1, color: props.color1, text: (props.yFormatter || fmtY)(s1.value[idx]) })
  if (s2.value[idx] != null) rows.push({ name: props.name2, color: props.color2, text: (props.yFormatter || fmtY)(s2.value[idx]) })
  tip.show = true
  tip.label = String(label)
  tip.rows = rows
  const w = rect.width
  const px = ev.clientX - rect.left
  tip.x = px + 12 > w - 90 ? px - 90 - 8 : px + 12
  tip.y = ev.clientY - rect.top - 40
  if (tip.y < 0) tip.y = ev.clientY - rect.top + 12
}
</script>

<style scoped>
.mc-wrap {
  position: relative;
  width: 100%;
}
.mc-wrap svg {
  display: block;
  width: 100%;
  height: 100%;
}
.mc-tip {
  position: absolute;
  top: 0;
  left: 0;
  pointer-events: none;
  white-space: nowrap;
  z-index: 5;
  padding: 6px 10px;
  border: 1px solid var(--dm-line);
  border-radius: 6px;
  background: var(--dm-card);
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.18);
  color: var(--dm-text);
  font-size: 12px;
}
.mc-tip-label {
  margin-bottom: 4px;
  color: var(--dm-muted);
  font-size: 11px;
}
.mc-tip-row {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  line-height: 16px;
}
.mc-tip-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex: none;
}
.mc-tip-name {
  color: var(--dm-muted);
}
.mc-tip-val {
  margin-left: auto;
}
</style>
