<template>
  <Modal :model-value="open" :title="t('update.title')" @close="$emit('close')">
    <div class="space-y-3 min-w-[380px] max-w-[480px]">
      <!-- 检查中 -->
      <div v-if="loading" class="flex items-center justify-center gap-2 text-muted text-sm py-8">
        <span class="inline-block w-4 h-4 border-2 border-line border-t-brand rounded-full animate-spin" />
        {{ t('update.checking') }}
      </div>

      <!-- 检查失败 -->
      <div v-else-if="error" class="text-center py-8">
        <p class="text-sm text-danger mb-3">{{ t('update.checkFailed') }}: {{ error }}</p>
        <button class="btn btn-ghost btn-sm" @click="refresh">{{ t('update.retry') }}</button>
      </div>

      <template v-else-if="info">
        <!-- 有更新 -->
        <template v-if="info.has_update">
          <div class="flex items-center gap-3 rounded-lg border border-line p-3">
            <div class="flex-1">
              <p class="text-xs text-muted">{{ t('update.current') }}</p>
              <p class="text-lg font-semibold font-mono">{{ info.current }}</p>
            </div>
            <Icon name="arrowRight" size="16" class="text-muted shrink-0" />
            <div class="flex-1">
              <p class="text-xs text-brand">{{ t('update.latest') }}</p>
              <p class="text-lg font-semibold font-mono text-brand">{{ info.latest }}</p>
            </div>
          </div>
          <div class="flex items-center justify-between gap-2">
            <p v-if="info.release?.published_at" class="text-xs text-muted">
              {{ t('update.publishedAt') }}: {{ formatDate(info.release.published_at) }}
            </p>
            <p v-if="info.install_type" class="text-xs text-muted ml-auto shrink-0">
              {{ t('update.installType') }}: {{ t('update.install_' + info.install_type) }}
            </p>
          </div>

          <!-- 不可安装提示(镜像未发布/平台无资产)→ 禁用更新按钮 -->
          <div
            v-if="info.installable === false"
            class="rounded-lg border border-warn/40 bg-warn/10 p-2.5 text-xs text-warn flex items-center gap-1.5"
          >
            <Icon name="alert" size="14" class="shrink-0" />
            {{ info.not_installable_reason }}
          </div>

          <!-- 更新内容:分类解析结果 -->
          <div v-if="info.notes?.length" class="space-y-2">
            <div v-for="sec in info.notes" :key="sec.type" class="rounded-lg border border-line p-3">
              <p class="text-xs font-semibold mb-1.5 flex items-center gap-1.5">
                <span>{{ noteIcon(sec.type) }}</span> {{ t('update.sec_' + sec.type) }}
              </p>
              <ul class="space-y-1">
                <li v-for="(item, i) in sec.items" :key="i" class="text-[12px] text-muted flex gap-1.5">
                  <span class="shrink-0">•</span><span class="break-all leading-relaxed">{{ item }}</span>
                </li>
              </ul>
            </div>
          </div>
          <!-- 分类解析失败 → 回退原始 Release Notes -->
          <div
            v-else-if="info.notes_raw && info.release?.body"
            class="code-panel border border-line rounded-lg p-3 max-h-56 overflow-y-auto text-[12px] leading-relaxed whitespace-pre-wrap text-muted"
          >
            {{ info.release.body }}
          </div>
          <!-- 无分类也无原始解析标记(老版本接口) → 直接显示 body -->
          <div
            v-else-if="info.release?.body"
            class="code-panel border border-line rounded-lg p-3 max-h-56 overflow-y-auto text-[12px] leading-relaxed whitespace-pre-wrap text-muted"
          >
            {{ info.release.body }}
          </div>

          <div class="flex items-center justify-between pt-1">
            <a
              v-if="info.release?.html_url"
              :href="info.release.html_url"
              target="_blank"
              rel="noopener"
              class="text-xs text-brand hover:underline"
            >
              {{ t('update.viewOnGithub') }}
            </a>
            <span />
            <button class="btn btn-brand btn-sm" :disabled="applying || info.installable === false" @click="apply">
              <span v-if="applying" class="inline-block w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin" />
              {{ applying ? t('update.applying') : t('update.apply') }}
            </button>
          </div>
        </template>

        <!-- 无更新 -->
        <div v-else class="text-center py-8">
          <Icon name="check" size="30" class="text-ok mx-auto mb-2" />
          <p class="text-sm font-medium">{{ t('update.upToDate') }} <span class="font-mono">{{ info.current }}</span></p>
          <button class="btn btn-ghost btn-sm mt-3" @click="refresh">{{ t('common.refresh') }}</button>
        </div>
      </template>

      <!-- 更新中:实时阶段进度 -->
      <div
        v-if="applying"
        class="rounded-lg border border-brand/30 bg-brand/5 p-3 text-sm text-brand flex items-center gap-2"
      >
        <span class="inline-block w-4 h-4 border-2 border-brand/30 border-t-brand rounded-full animate-spin shrink-0" />
        {{ phase ? t('update.phase_' + phase) : t('update.applying') }}
      </div>

      <!-- 更新失败 -->
      <div
        v-if="phaseError"
        class="rounded-lg border border-danger/30 bg-danger/5 p-3 text-sm text-danger leading-relaxed"
      >
        <p class="font-medium mb-1">{{ t('update.phase_failed') }} — {{ t('update.failedKeepOld') }}</p>
        <p class="text-[12px] break-all">{{ phaseError }}</p>
      </div>

      <!-- 更新已启动(面板重启中) -->
      <div
        v-if="applied"
        class="rounded-lg border border-brand/30 bg-brand/5 p-3 text-sm text-brand flex items-center gap-2"
      >
        <span class="inline-block w-4 h-4 border-2 border-brand/30 border-t-brand rounded-full animate-spin shrink-0" />
        {{ t('update.started') }}
      </div>
    </div>
  </Modal>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from './Icon.vue'
import Modal from './Modal.vue'
import { api } from '../api'
import { useConfirm } from '../confirm'
import { formatDate } from '../util'
import { toastErr, toastOk } from '../toast'

const { t } = useI18n()
const props = defineProps({ open: Boolean })
const emit = defineEmits(['close'])
const confirm = useConfirm()

const info = ref(null)
const loading = ref(false)
const error = ref('')
const applying = ref(false)
const applied = ref(false)
const phase = ref('') // 当前更新阶段
const phaseError = ref('')

// Release Notes 分类图标(与后端 note section type 对应)
const NOTE_ICONS = {
  features: '✨',
  bug_fixes: '🐛',
  improvements: '🔧',
  security: '🔒',
  breaking_changes: '⚠️',
}
const noteIcon = (type) => NOTE_ICONS[type] || '📋'

async function refresh() {
  loading.value = true
  error.value = ''
  applied.value = false
  applying.value = false
  phase.value = ''
  phaseError.value = ''
  try {
    info.value = await api('/update/check')
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

// 轮询 /update/status 获取实时进度,每 1s:
//  - 到 done(compose 已接管)或 restarting(二进制即将重启)返回该状态
//  - failed 返回状态(带 error)
//  - 请求失败(面板重启中/进程退出)返回 null,由调用方转轮询版本接口确认新版本上线
async function pollStatus() {
  for (let i = 0; i < 300; i++) {
    await new Promise((r) => setTimeout(r, 1000))
    let s
    try {
      s = await api('/update/status')
    } catch {
      return null // 面板重启中
    }
    phase.value = s.phase
    if (s.phase === 'failed') return s
    if (s.phase === 'done' || s.phase === 'restarting') return s
  }
  return null
}

// 更新后面板重启/容器重建,轮询 check 直到新版本上线(最多 60 秒)。
// Health Check + Version Verify:无错误 && 确认无更新 && 当前版本已非 unknown(真实版本已上线)。
async function waitDone() {
  for (let i = 0; i < 12; i++) {
    await new Promise((r) => setTimeout(r, 5000))
    try {
      const r = await api('/update/check')
      // 新版已上线:检查成功、无更新提示、当前版本是真实版本(非 unknown)
      if (!r.error && r.has_update === false && r.current && r.current !== 'unknown') return true
    } catch { /* 面板重启中,忽略 */ }
  }
  return false
}

async function apply() {
  if (!info.value) return
  const ok = await confirm(
    `${t('update.applyConfirm', { version: info.value.latest })}\n${t('update.installType')}: ${t('update.install_' + info.value.install_type)}`,
    {
      title: t('update.applyTitle'),
      danger: true,
      confirmText: t('update.apply'),
    }
  )
  if (!ok) return
  applying.value = true
  applied.value = false
  phase.value = ''
  phaseError.value = ''
  try {
    // 异步启动,立即返回;进度轮询 /update/status
    await api('/update/apply', { method: 'POST' })
    const s = await pollStatus()
    if (s && s.phase === 'failed') {
      phaseError.value = s.error || t('update.phase_failed')
      applying.value = false
      return
    }
    // restarting(二进制)/ done(compose)/ 面板已重启(null) → 等新版本上线
    applying.value = false
    applied.value = true
    toastOk(t('update.started'))
    if (await waitDone()) {
      toastOk(t('update.completed'))
      info.value = null
      await refresh()
    } else {
      toastErr(t('update.timeout'))
    }
  } catch (e) {
    applying.value = false
    toastErr(e.message)
  }
}

watch(
  () => props.open,
  (v) => {
    if (v) refresh()
  }
)
</script>
