<template>
  <div class="space-y-4 fade-up flex flex-col h-full">
    <!-- 工具栏 -->
    <div class="flex flex-wrap items-center gap-3">
      <div class="relative">
        <span class="absolute left-3 top-1/2 -translate-y-1/2 text-muted"><Icon name="search" size="14" /></span>
        <input v-model="keyword" class="input !w-64 !pl-9" :placeholder="t('appStore.searchPh')" />
      </div>
      <button class="btn btn-ghost btn-sm ml-auto" :disabled="syncing" @click="syncApps">
        <Icon name="refresh" size="13" :class="syncing ? 'animate-spin' : ''" /> {{ t('appStore.sync') }}
      </button>
      <div class="text-[12px] text-muted">{{ t('appStore.count', { count: filtered.length }) }}</div>
    </div>

    <!-- 分类(1Panel 同款:顶部横向标签) -->
    <div class="flex flex-wrap items-center gap-1.5">
      <button class="px-2.5 py-1 rounded-full text-[12px] transition-colors" :class="cat === '' ? 'bg-brand text-white font-medium' : 'bg-surface2 text-muted hover:text-text'" @click="cat = ''">
        {{ t('appStore.all') }}
      </button>
      <button v-for="c in categories" :key="c" class="px-2.5 py-1 rounded-full text-[12px] transition-colors" :class="cat === c ? 'bg-brand text-white font-medium' : 'bg-surface2 text-muted hover:text-text'" @click="cat = c">
        {{ c }}
      </button>
      <span class="w-px h-4 bg-line mx-1.5" />
      <button class="px-2.5 py-1 rounded-full text-[12px] transition-colors" :class="cat === '__installed' ? 'bg-brand text-white font-medium' : 'bg-surface2 text-muted hover:text-text'" @click="cat = '__installed'">
        {{ t('appStore.installed') }} ({{ installedCount }})
      </button>
      <button class="px-2.5 py-1 rounded-full text-[12px] transition-colors" :class="cat === '__updatable' ? 'bg-brand text-white font-medium' : 'bg-surface2 text-muted hover:text-text'" @click="cat = '__updatable'">
        {{ t('appStore.updatable') }} ({{ updatableCount }})
      </button>
    </div>

    <!-- 应用网格(4 列,滚动浏览全部) -->
    <div class="flex-1 flex flex-col min-h-0">
        <div v-if="!filtered.length" class="panel p-10 text-center text-muted text-sm flex-1">{{ t('appStore.empty') }}</div>
        <div v-else class="grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5 flex-1 min-h-0 overflow-y-auto content-start items-start pb-2">
          <div v-for="a in filtered" :key="a.key" class="panel border border-line app-card" @click="openDetail(a)">
            <div class="app-image">
              <img v-if="!iconFailed[a.key]" :src="iconUrl(a.key)" :alt="a.name" loading="lazy" @error="iconFailed[a.key] = true" />
              <span v-else class="emoji-fallback">{{ a.icon }}</span>
            </div>
            <div class="app-content">
              <div class="content-top">
                <span class="app-title truncate">{{ a.name }}</span>
                <span class="flex flex-col items-end gap-1">
                  <span v-if="a.installed" class="installed-tag">{{ t('appStore.installedTag') }}</span>
                  <span v-if="a.update_available" class="update-tag">{{ t('appStore.updatable') }}</span>
                </span>
              </div>
              <div class="content-middle">
                <span class="app-description">{{ a.description }}</span>
              </div>
              <div class="content-bottom">
                <span class="cat-tag">{{ a.category }}</span>
                <button v-if="!a.installed" class="btn btn-brand btn-xs" @click.stop="openDetail(a)"><Icon name="download" size="11" /> {{ t('appStore.install') }}</button>
                <span v-else class="flex items-center gap-1.5">
                  <button :class="a.update_available ? 'btn btn-warn btn-xs' : 'btn btn-ghost btn-xs'" @click.stop="quickUpgrade(a)"><Icon name="refresh" size="11" /> {{ t('appStore.upgrade') }}</button>
                  <span class="text-[11px] text-ok">✓</span>
                </span>
              </div>
            </div>
          </div>
        </div>

      </div>

    <!-- 详情 / 安装 -->
    <Modal :model-value="!!detail" size="2xl" :title="detail ? detail.icon + ' ' + detail.name : ''" @close="detail = null">
      <div v-if="detail" class="space-y-3">
        <p class="text-xs text-muted">{{ detail.description }}</p>

        <!-- 已安装:管理操作 -->
        <div v-if="detail.installed" class="rounded-lg border border-line bg-surface2/50 p-3 space-y-2">
          <p class="text-xs" style="color:#22c55e">✓ {{ t('appStore.installedMsg') }}</p>
          <p v-if="detail.update_available" class="text-xs text-warn"><Icon name="refresh" size="12" /> {{ t('appStore.updateAvailableMsg') }}</p>
          <div class="flex gap-2 flex-wrap">
            <button class="btn btn-ghost btn-sm" @click="goCompose"><Icon name="container" size="13" /> {{ t('appStore.openStack') }}</button>
            <button :class="detail.update_available ? 'btn btn-warn btn-sm' : 'btn btn-ghost btn-sm'" :disabled="busy" @click="upgrade"><Icon name="refresh" size="13" /> {{ t('appStore.upgrade') }}</button>
            <button class="btn btn-danger btn-sm" :disabled="busy" @click="uninstall"><Icon name="trash" size="13" /> {{ t('appStore.uninstall') }}</button>
          </div>
        </div>

        <!-- 未安装:参数表单 -->
        <template v-else>
          <div v-for="p in detail.params" :key="p.key" class="space-y-1">
            <label class="label">{{ paramLabel(p) }} <span v-if="p.required" class="text-danger">*</span></label>
            <input v-if="p.type === 'text' || p.type === 'number'" v-model="params[p.key]" :type="p.type === 'number' ? 'number' : 'text'" class="input" />
            <div v-else-if="p.type === 'password'" class="flex gap-2 items-center">
              <input v-model="params[p.key]" :type="showPwd[p.key] ? 'text' : 'password'" class="input flex-1" :placeholder="p.random ? t('appStore.autoGenerate') : ''" />
              <button class="btn btn-icon btn-xs shrink-0" :title="t('appStore.togglePwd')" @click="showPwd[p.key] = !showPwd[p.key]"><Icon :name="showPwd[p.key] ? 'eyeOff' : 'eye'" size="13" /></button>
              <button v-if="p.random" class="btn btn-ghost btn-xs shrink-0" @click="params[p.key] = genRandom()">{{ t('appStore.generate') }}</button>
            </div>
            <select v-else-if="p.type === 'select'" v-model="params[p.key]" class="input">
              <option v-for="o in p.options" :key="o.value" :value="o.value">{{ o.label }}</option>
            </select>
            <label v-else-if="p.type === 'checkbox'" class="flex items-center gap-2 text-xs cursor-pointer select-none w-fit">
              <input type="checkbox" class="accent-brand w-4 h-4" :checked="params[p.key] === 'true'" @change="params[p.key] = $event.target.checked ? 'true' : 'false'" />
              <span>{{ paramLabel(p) }}</span>
            </label>
            <textarea v-else-if="p.type === 'textarea'" v-model="params[p.key]" rows="3" class="input font-mono text-[12px]" spellcheck="false" />
            <p v-if="p.hint" class="text-[11px] text-muted">{{ t('appStore.hint_' + p.hint) }}</p>
          </div>

          <!-- 编辑 compose 文件(1Panel 同款) -->
          <label class="flex items-center gap-2 text-xs cursor-pointer select-none w-fit pt-1">
            <input v-model="editCompose" type="checkbox" class="accent-brand w-4 h-4" />
            <span>{{ t('appStore.editCompose') }}</span>
          </label>
          <div v-if="editCompose" class="space-y-1">
            <div class="flex items-center justify-between">
              <label class="label">{{ t('appStore.composePreview') }}</label>
              <button v-if="composeDirty" class="btn btn-ghost btn-xs" @click="refreshPreview"><Icon name="refresh" size="11" /> {{ t('appStore.refreshPreview') }}</button>
            </div>
            <textarea v-model="composeText" rows="16" class="input font-mono text-[12px]" spellcheck="false" />
            <p class="text-[11px] text-warn">{{ t('appStore.composeWarn') }}</p>
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
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import Icon from '../components/Icon.vue'
import Modal from '../components/Modal.vue'
import { api } from '../api'
import { useConfirm } from '../confirm'
import { toastErr, toastOk } from '../toast'

const { t, locale } = useI18n()
const router = useRouter()
const apps = ref([])
const categories = ref([])
const keyword = ref('')
const cat = ref('')
const detail = ref(null)
const params = reactive({})
const showPwd = reactive({})
const installing = ref(false)
const installErr = ref('')
const busy = ref(false)
const syncing = ref(false)
const editCompose = ref(false)
const composeText = ref('')
const composeDirty = ref(false)
const confirm = useConfirm()

const filtered = computed(() => {
  let list = apps.value
  if (cat.value === '__installed') list = list.filter((a) => a.installed)
  else if (cat.value === '__updatable') list = list.filter((a) => a.update_available)
  else if (cat.value) list = list.filter((a) => a.category === cat.value)
  if (keyword.value) {
    const k = keyword.value.toLowerCase()
    list = list.filter((a) => a.name.toLowerCase().includes(k) || a.description.toLowerCase().includes(k) || a.key.includes(k))
  }
  return list
})

const iconFailed = reactive({})

function iconUrl(key) {
  return `/api/apps/icon/${key}`
}

const installedCount = computed(() => apps.value.filter((a) => a.installed).length)
const updatableCount = computed(() => apps.value.filter((a) => a.update_available).length)

const validParams = computed(() => {
  if (!detail.value) return false
  return (detail.value.params || []).every((p) => !p.required || (params[p.key] || '').trim())
})

function paramLabel(p) {
  return (locale.value || '').startsWith('zh') ? p.label_zh || p.label_en : p.label_en || p.label_zh
}

function genRandom() {
  const chars = 'abcdef0123456789'
  let s = ''
  for (let i = 0; i < 16; i++) s += chars[Math.floor(Math.random() * chars.length)]
  return s
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

async function syncApps() {
  syncing.value = true
  try {
    await api('/apps/sync', { method: 'POST' })
    toastOk(t('appStore.syncDone'))
    await load()
  } catch (e) {
    toastErr(e.message)
  } finally {
    syncing.value = false
  }
}

async function openDetail(a) {
  installErr.value = ''
  editCompose.value = false
  composeText.value = ''
  composeDirty.value = false
  try {
    const d = await api(`/apps/${a.key}`)
    detail.value = { ...a, ...d }
    for (const p of (d.params || [])) params[p.key] = p.default || ''
  } catch (e) {
    toastErr(e.message)
  }
}

// 编辑 compose:勾选后拉取渲染预览预填
watch(editCompose, async (on) => {
  if (!on || !detail.value) return
  await refreshPreview()
})

// 参数变化后标记预览过期(提醒刷新)
watch(
  () => ({ ...params }),
  () => {
    if (editCompose.value) composeDirty.value = true
  },
  { deep: true }
)

async function refreshPreview() {
  try {
    const r = await api(`/apps/${detail.value.key}/preview`, { method: 'POST', json: { params: { ...params } } })
    composeText.value = r.yaml || ''
    composeDirty.value = false
  } catch (e) {
    toastErr(e.message)
  }
}

async function install() {
  installing.value = true
  installErr.value = ''
  try {
    await api(`/apps/${detail.value.key}/install`, {
      method: 'POST',
      json: { params: { ...params }, yaml: editCompose.value ? composeText.value : '' },
    })
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

// 已安装卡片上的快速更新按钮
async function quickUpgrade(a) {
  try {
    await api(`/apps/${a.key}/upgrade`, { method: 'POST' })
    toastOk(t('appStore.toastUpgraded'))
  } catch (e) {
    toastErr(e.message)
  }
}

function goCompose() {
  const key = detail.value.key
  detail.value = null
  router.push(`/compose/${key}`)
}

onMounted(load)
</script>

<style scoped>
/* 应用卡片:左图右文(1Panel AppCard 同款) */
.app-card {
  display: flex;
  padding: 12px;
  cursor: pointer;
  overflow: hidden;
  min-height: 126px;
  transition: border-color 0.15s;
}
.app-card:hover {
  border-color: var(--color-brand);
}
.app-image {
  flex: 0 0 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform 0.1s;
}
.app-card:hover .app-image {
  transform: scale(1.2);
}
.app-image img {
  width: 48px;
  height: 48px;
  object-fit: cover;
  border-radius: 8px;
}
.emoji-fallback {
  font-size: 28px;
  line-height: 1;
}
.app-content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  padding-left: 12px;
}
.content-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 6px;
}
.app-title {
  font-size: 14px;
  font-weight: 600;
}
.installed-tag {
  font-size: 10px;
  padding: 1px 8px;
  border-radius: 999px;
  white-space: nowrap;
  color: #22c55e;
  background: rgba(34, 197, 94, 0.12);
  border: 1px solid rgba(34, 197, 94, 0.3);
}
.update-tag {
  font-size: 10px;
  padding: 1px 8px;
  border-radius: 999px;
  white-space: nowrap;
  color: #fbbf24;
  background: rgba(251, 191, 36, 0.12);
  border: 1px solid rgba(251, 191, 36, 0.35);
}
.content-middle {
  flex: 1;
  margin: 4px 0;
  overflow: hidden;
}
.app-description {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  font-size: 11px;
  color: var(--color-muted);
  line-height: 1.4;
  height: 2.8em;
}
.content-bottom {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 6px;
  margin-top: 6px;
}
.cat-tag {
  font-size: 10px;
  padding: 1px 8px;
  border-radius: 999px;
  background: rgba(236, 72, 153, 0.1);
  color: var(--color-brand);
  white-space: nowrap;
}
</style>
