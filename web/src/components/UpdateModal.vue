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
              <p class="text-lg font-semibold font-mono">v{{ info.current }}</p>
            </div>
            <Icon name="arrowRight" size="16" class="text-muted shrink-0" />
            <div class="flex-1">
              <p class="text-xs text-brand">{{ t('update.latest') }}</p>
              <p class="text-lg font-semibold font-mono text-brand">v{{ info.latest }}</p>
            </div>
          </div>
          <p v-if="info.release?.published_at" class="text-xs text-muted">
            {{ t('update.publishedAt') }}: {{ formatDate(info.release.published_at) }}
          </p>
          <div
            v-if="info.release?.body"
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
            <button class="btn btn-brand btn-sm" :disabled="applying" @click="apply">
              <span v-if="applying" class="inline-block w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin" />
              {{ applying ? t('update.applying') : t('update.apply') }}
            </button>
          </div>
        </template>

        <!-- 无更新 -->
        <div v-else class="text-center py-8">
          <Icon name="check" size="30" class="text-ok mx-auto mb-2" />
          <p class="text-sm font-medium">{{ t('update.upToDate') }} <span class="font-mono">v{{ info.current }}</span></p>
          <button class="btn btn-ghost btn-sm mt-3" @click="refresh">{{ t('common.refresh') }}</button>
        </div>
      </template>

      <!-- 更新已启动 -->
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

async function refresh() {
  loading.value = true
  error.value = ''
  applied.value = false
  try {
    info.value = await api('/update/check')
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

// 更新后面板容器重建,轮询 check 直到新版本上线(最多 60 秒)
async function waitDone() {
  for (let i = 0; i < 12; i++) {
    await new Promise((r) => setTimeout(r, 5000))
    try {
      const r = await api('/update/check')
      // 必须无错误且确认无更新(新版已上线)才算完成;检查失败不算
      if (!r.error && r.has_update === false) return true
    } catch { /* 面板重启中,忽略 */ }
  }
  return false
}

async function apply() {
  if (!info.value) return
  const ok = await confirm(t('update.applyConfirm', { version: 'v' + info.value.latest }), {
    title: t('update.applyTitle'),
    danger: true,
    confirmText: t('update.apply'),
  })
  if (!ok) return
  applying.value = true
  try {
    await api('/update/apply', { method: 'POST' })
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
