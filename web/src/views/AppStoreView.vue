<template>
  <div class="space-y-4 fade-up">
    <!-- 工具栏 -->
    <div class="flex flex-wrap items-center gap-3">
      <div class="relative">
        <span class="absolute left-3 top-1/2 -translate-y-1/2 text-muted"><Icon name="search" size="14" /></span>
        <input v-model="keyword" class="input !w-64 !pl-9" :placeholder="t('appStore.searchPh')" />
      </div>
      <div class="ml-auto text-[12px] text-muted">{{ t('appStore.count', { count: filtered.length }) }}</div>
    </div>

    <div class="flex gap-4">
      <!-- 分类侧栏 -->
      <div class="w-36 shrink-0 space-y-0.5">
        <button class="w-full text-left px-3 py-2 rounded-lg text-[13px] transition-colors" :class="cat === '' ? 'bg-brand/10 text-brand font-medium' : 'text-muted hover:bg-surface2'" @click="cat = ''">
          {{ t('appStore.all') }}
        </button>
        <button v-for="c in categories" :key="c" class="w-full text-left px-3 py-2 rounded-lg text-[13px] transition-colors" :class="cat === c ? 'bg-brand/10 text-brand font-medium' : 'text-muted hover:bg-surface2'" @click="cat = c">
          {{ t('appStore.cat_' + c) }}
        </button>
        <div class="h-px bg-line my-2" />
        <button class="w-full text-left px-3 py-2 rounded-lg text-[13px] transition-colors" :class="cat === '__installed' ? 'bg-brand/10 text-brand font-medium' : 'text-muted hover:bg-surface2'" @click="cat = '__installed'">
          {{ t('appStore.installed') }} ({{ installedCount }})
        </button>
      </div>

      <!-- 应用网格 -->
      <div class="flex-1">
        <div v-if="!filtered.length" class="panel p-10 text-center text-muted text-sm">{{ t('appStore.empty') }}</div>
        <div v-else class="grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
          <div v-for="a in filtered" :key="a.key" class="panel p-4 cursor-pointer hover:border-brand/40 transition-colors relative" @click="openDetail(a)">
            <span v-if="a.installed" class="absolute top-2 right-2 text-[10px] px-1.5 py-0.5 rounded-full" style="color:#22c55e;background:rgba(34,197,94,.12);border:1px solid rgba(34,197,94,.3)">
              {{ t('appStore.installedTag') }}
            </span>
            <div class="text-2xl mb-2">{{ a.icon }}</div>
            <div class="text-sm font-semibold truncate">{{ a.name }}</div>
            <div class="text-[11px] text-muted mt-1 line-clamp-2 h-8 overflow-hidden">{{ a.description }}</div>
            <div class="text-[10px] text-muted mt-2 font-mono">{{ a.ports.join(', ') }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 详情 / 安装 -->
    <Modal :model-value="!!detail" :title="detail ? detail.icon + ' ' + detail.name : ''" @close="detail = null">
      <div v-if="detail" class="space-y-3 max-w-[420px]">
        <p class="text-xs text-muted">{{ detail.description }}</p>
        <div class="text-[11px] text-muted font-mono">
          {{ t('appStore.ports') }}: {{ detail.ports.join(', ') }}
        </div>

        <!-- 已安装:管理操作 -->
        <div v-if="detail.installed" class="rounded-lg border border-line bg-surface2/50 p-3 space-y-2">
          <p class="text-xs" style="color:#22c55e">✓ {{ t('appStore.installedMsg') }}</p>
          <div class="flex gap-2 flex-wrap">
            <button class="btn btn-ghost btn-sm" @click="goCompose"><Icon name="container" size="13" /> {{ t('appStore.openStack') }}</button>
            <button class="btn btn-ghost btn-sm" :disabled="busy" @click="upgrade"><Icon name="refresh" size="13" /> {{ t('appStore.upgrade') }}</button>
            <button class="btn btn-danger btn-sm" :disabled="busy" @click="uninstall"><Icon name="trash" size="13" /> {{ t('appStore.uninstall') }}</button>
          </div>
        </div>

        <!-- 未安装:参数表单 -->
        <template v-else>
          <div v-for="p in detail.params" :key="p.key" class="space-y-1">
            <label class="label">{{ prettyKey(p.key) }} <span v-if="p.required" class="text-danger">*</span></label>
            <input v-if="p.type !== 'select'" v-model="params[p.key]" :type="p.type === 'password' ? 'password' : 'text'" class="input" />
            <select v-else v-model="params[p.key]" class="input">
              <option v-for="o in p.options" :key="o" :value="o">{{ o }}</option>
            </select>
          </div>
          <p v-if="installErr" class="text-xs text-danger">{{ installErr }}</p>
          <div v-if="installing" class="flex items-center gap-2 text-xs text-brand">
            <span class="inline-block w-3 h-3 border-2 border-brand/30 border-t-brand rounded-full animate-spin" />
            {{ t('appStore.installing') }}
          </div>
        </template>
      </div>
      <template #footer>
        <template v-if="detail && !detail.installed">
          <button class="btn btn-ghost btn-sm" @click="detail = null">{{ t('common.cancel') }}</button>
          <button class="btn btn-brand btn-sm" :disabled="installing || !validParams" @click="install">
            <Icon name="download" size="13" /> {{ t('appStore.install') }}
          </button>
        </template>
        <button v-else class="btn btn-ghost btn-sm" @click="detail = null">{{ t('common.close') }}</button>
      </template>
    </Modal>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import Icon from '../components/Icon.vue'
import Modal from '../components/Modal.vue'
import { api } from '../api'
import { useConfirm } from '../confirm'
import { toastErr, toastOk } from '../toast'

const { t } = useI18n()
const router = useRouter()
const apps = ref([])
const categories = ref([])
const keyword = ref('')
const cat = ref('')
const detail = ref(null)
const params = reactive({})
const installing = ref(false)
const installErr = ref('')
const busy = ref(false)
const confirm = useConfirm()

const filtered = computed(() => {
  let list = apps.value
  if (cat.value === '__installed') list = list.filter((a) => a.installed)
  else if (cat.value) list = list.filter((a) => a.category === cat.value)
  if (keyword.value) {
    const k = keyword.value.toLowerCase()
    list = list.filter((a) => a.name.toLowerCase().includes(k) || a.description.toLowerCase().includes(k) || a.key.includes(k))
  }
  return list
})

const installedCount = computed(() => apps.value.filter((a) => a.installed).length)

const validParams = computed(() => {
  if (!detail.value) return false
  return (detail.value.params || []).every((p) => !p.required || (params[p.key] || '').trim())
})

function prettyKey(k) {
  return k.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())
}

async function load() {
  try {
    const r = await api('/apps')
    apps.value = r.apps || []
    categories.value = r.categories || []
  } catch (e) {
    toastErr(e.message)
  }
}

async function openDetail(a) {
  installErr.value = ''
  try {
    const d = await api(`/apps/${a.key}`)
    detail.value = { ...a, ...d }
    for (const p of (d.params || [])) params[p.key] = p.default || ''
  } catch (e) {
    toastErr(e.message)
  }
}

async function install() {
  installing.value = true
  installErr.value = ''
  try {
    await api(`/apps/${detail.value.key}/install`, { method: 'POST', json: { params: { ...params } } })
    toastOk(t('appStore.toastInstalled', { name: detail.value.name }))
    detail.value = null
    load()
  } catch (e) {
    installErr.value = e.message
  } finally {
    installing.value = false
  }
}

async function uninstall() {
  const ok = await confirm(t('appStore.confirmUninstall', { name: detail.value.name }), {
    title: t('appStore.uninstall'),
    confirmText: t('appStore.uninstall'),
  })
  if (!ok) return
  busy.value = true
  try {
    await api(`/apps/${detail.value.key}/uninstall`, { method: 'POST' })
    toastOk(t('common.deleted'))
    detail.value = null
    load()
  } catch (e) {
    toastErr(e.message)
  } finally {
    busy.value = false
  }
}

async function upgrade() {
  busy.value = true
  try {
    await api(`/apps/${detail.value.key}/upgrade`, { method: 'POST' })
    toastOk(t('appStore.toastUpgraded'))
  } catch (e) {
    toastErr(e.message)
  } finally {
    busy.value = false
  }
}

function goCompose() {
  const key = detail.value.key
  detail.value = null
  router.push(`/compose/${key}`)
}

onMounted(load)
</script>
