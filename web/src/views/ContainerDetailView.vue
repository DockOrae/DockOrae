<template>
  <div class="space-y-4 fade-up">
    <!-- 头部 -->
    <div class="card px-5 py-4">
      <div class="flex items-center gap-3 flex-wrap">
        <button class="btn btn-ghost btn-sm" @click="$router.back()"><Icon name="x" size="13" /> {{ t('common.back') }}</button>
        <div class="min-w-0">
          <div class="flex items-center gap-2.5">
            <h2 class="text-base font-semibold truncate">{{ name }}</h2>
            <StatusBadge v-if="inspect" :state="inspect.State?.Status" />
          </div>
          <div class="text-[11px] text-muted font-mono mt-0.5">{{ shortId(id) }} · {{ inspect?.Config?.Image || '' }}</div>
        </div>
        <div class="ml-auto flex items-center gap-1.5 flex-wrap">
          <button v-if="status !== 'running'" class="btn btn-ok btn-sm" @click="act('start')"><Icon name="play" size="13" /> {{ t('common.start') }}</button>
          <button v-if="status === 'running'" class="btn btn-ghost btn-sm" @click="act('stop')"><Icon name="stop" size="13" /> {{ t('common.stop') }}</button>
          <button class="btn btn-ghost btn-sm" @click="act('restart')"><Icon name="restart" size="13" /> {{ t('common.restart') }}</button>
          <button v-if="status === 'running'" class="btn btn-ghost btn-sm" @click="act('pause')"><Icon name="pause" size="13" /> {{ t('common.pause') }}</button>
          <button v-if="status === 'paused'" class="btn btn-ok btn-sm" @click="act('unpause')"><Icon name="play" size="13" /> {{ t('common.unpause') }}</button>
          <button class="btn btn-danger btn-sm" @click="remove"><Icon name="trash" size="13" /> {{ t('common.delete') }}</button>
        </div>
      </div>
    </div>

    <!-- 标签页 -->
    <div class="flex gap-1 border-b border-line">
      <button
        v-for="tabItem in tabs"
        :key="tabItem.key"
        class="px-4 py-2.5 text-[13px] font-medium rounded-t-lg transition-colors -mb-px border-b-2"
        :class="tab === tabItem.key ? 'text-brand border-brand' : 'text-muted hover:text-text border-transparent'"
        @click="tab = tabItem.key"
      >
        {{ t(tabItem.labelKey) }}
      </button>
    </div>

    <DetailOverview v-if="tab === 'overview' && inspect" :data="inspect" />
    <DetailLogs v-else-if="tab === 'logs'" :id="id" />
    <DetailStats v-else-if="tab === 'stats'" :id="id" />
    <DetailTerminal v-else-if="tab === 'terminal'" :id="id" />
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '../components/Icon.vue'
import StatusBadge from '../components/StatusBadge.vue'
import DetailOverview from './container/DetailOverview.vue'
import DetailLogs from './container/DetailLogs.vue'
import DetailStats from './container/DetailStats.vue'
import DetailTerminal from './container/DetailTerminal.vue'
import { api } from '../api'
import { containerName, shortId } from '../util'
import { useConfirm } from '../confirm'
import { toastErr, toastOk } from '../toast'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const id = computed(() => route.params.id)
const confirm = useConfirm()

const inspect = ref(null)
const status = ref('')
const tab = ref('overview')
let timer = null

const tabs = [
  { key: 'overview', labelKey: 'containerDetail.tabOverview' },
  { key: 'logs', labelKey: 'containerDetail.tabLogs' },
  { key: 'stats', labelKey: 'containerDetail.tabStats' },
  { key: 'terminal', labelKey: 'containerDetail.tabTerminal' },
]

const name = computed(() => containerName(inspect.value) || id.value)

async function load() {
  try {
    inspect.value = await api(`/containers/${id.value}`)
    status.value = inspect.value?.State?.Status || ''
  } catch (e) {
    toastErr(e.message)
  }
}

async function act(action) {
  try {
    await api(`/containers/${id.value}/${action}`, { method: 'POST' })
    toastOk({ start: t('containerDetail.toastStarted'), stop: t('containerDetail.toastStopped'), restart: t('containerDetail.toastRestarted'), pause: t('containerDetail.toastPaused'), unpause: t('containerDetail.toastResumed') }[action])
    load()
  } catch (e) {
    toastErr(e.message)
  }
}

async function remove() {
  const ok = await confirm(t('containerDetail.confirmDelete', { name: name.value }), {
    title: t('containerDetail.confirmDeleteTitle'),
    confirmText: t('common.delete'),
  })
  if (!ok) return
  try {
    await api(`/containers/${id.value}?force=true`, { method: 'DELETE' })
    toastOk(t('common.deleted'))
    router.push('/containers')
  } catch (e) {
    toastErr(e.message)
  }
}

onMounted(() => {
  load()
  timer = setInterval(load, 5000)
})
onBeforeUnmount(() => clearInterval(timer))
watch(id, () => {
  tab.value = 'overview'
  load()
})
</script>
